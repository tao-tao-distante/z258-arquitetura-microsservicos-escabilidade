package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/httpx"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
	platformmetrics "github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/metrics"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/middleware"
)

const maxPayloadBytes = 16 << 10

// AuthService define as operações de autenticação expostas pelo handler.
type AuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, string, error)
	Profile(ctx context.Context, userID int64) (*domain.User, error)
	UpdateName(ctx context.Context, userID int64, name string) (*domain.User, error)
	ChangePassword(ctx context.Context, userID int64, current, next string) error
}

// Handler expõe os endpoints HTTP de registro, login, perfil e health.
type Handler struct {
	svc     AuthService
	secret  string
	deps    map[string]func(context.Context) error
	log     *slog.Logger
	limiter *middleware.RateLimiter
	metrics counter
}

type counter interface{ Inc(...string) }

type noopCounter struct{}

func (noopCounter) Inc(...string) {}

// New cria o handler com o serviço, as checagens de dependência e o logger informados.
func New(svc AuthService, deps map[string]func(context.Context) error, logs ...*slog.Logger) *Handler {
	log := slog.Default()
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	}
	h := &Handler{svc: svc, deps: deps, log: log, metrics: noopCounter{}}
	h.limiter = middleware.NewRateLimiter(10, time.Minute, time.Now).OnLimit(func(r *http.Request) {
		h.metrics.Inc(limitedOperation(r.URL.Path), "rate_limited")
	})
	return h
}

// WithAuthMetrics habilita contagem de tentativas por operação e resultado.
func (h *Handler) WithAuthMetrics(metrics counter) *Handler {
	if metrics != nil {
		h.metrics = metrics
	}
	return h
}

// WithJWT define o segredo usado para validar o Bearer token das rotas de perfil.
func (h *Handler) WithJWT(secret string) *Handler {
	h.secret = secret
	return h
}

func limitedOperation(path string) string {
	switch path {
	case "/register":
		return "register"
	case "/me/password":
		return "change_password"
	default:
		return "login"
	}
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	ID    int64  `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// Routes monta o mux com rate limit em registro/login/troca de senha, perfil protegido por JWT e
// correlação global.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /register", h.limiter.Handler(http.HandlerFunc(h.Register)))
	mux.Handle("POST /login", h.limiter.Handler(http.HandlerFunc(h.Login)))
	mux.HandleFunc("GET /health", h.Health)
	mux.Handle("GET /me", h.protected("get_profile", h.GetProfile))
	mux.Handle("PATCH /me", h.protected("update_profile", h.UpdateProfile))
	mux.Handle("PUT /me/password", h.limiter.Handler(h.protected("change_password", h.ChangePassword)))
	return platformmetrics.RoutePattern(mux)
}

// Register cria um usuário e devolve o JWT emitido.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	c, status, err := decodeCredentials(w, r)
	if err != nil {
		h.metrics.Inc("register", "invalid_payload")
		httpx.WriteError(w, status, "invalid payload")
		return
	}
	user, token, err := h.svc.Register(r.Context(), c.Email, c.Password)
	if errors.Is(err, domain.ErrEmailTaken) {
		h.metrics.Inc("register", "email_taken")
		httpx.WriteError(w, http.StatusConflict, "email already taken")
		return
	}
	if errors.Is(err, domain.ErrInvalidInput) {
		h.metrics.Inc("register", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid email or password")
		return
	}
	if err != nil {
		h.logInternal(r.Context(), err)
		h.metrics.Inc("register", "error")
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.metrics.Inc("register", "success")
	httpx.WriteJSON(w, http.StatusCreated, authResponse{ID: user.ID, Email: user.Email, Name: user.Name, Token: token})
}

// Login autentica o usuário e devolve o JWT emitido.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	c, status, err := decodeCredentials(w, r)
	if err != nil {
		h.metrics.Inc("login", "invalid_payload")
		httpx.WriteError(w, status, "invalid payload")
		return
	}
	user, token, err := h.svc.Login(r.Context(), c.Email, c.Password)
	if errors.Is(err, domain.ErrInvalidCredentials) {
		h.metrics.Inc("login", "invalid_credentials")
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if errors.Is(err, domain.ErrInvalidInput) {
		h.metrics.Inc("login", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid email or password")
		return
	}
	if err != nil {
		h.logInternal(r.Context(), err)
		h.metrics.Inc("login", "error")
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.metrics.Inc("login", "success")
	httpx.WriteJSON(w, http.StatusOK, authResponse{ID: user.ID, Email: user.Email, Name: user.Name, Token: token})
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, int, error) {
	var c credentials
	if status, err := decodePayload(w, r, &c); err != nil {
		return credentials{}, status, err
	}
	if c.Email == "" || c.Password == "" {
		return credentials{}, http.StatusBadRequest, errors.New("credentials required")
	}
	return c, 0, nil
}

func (h *Handler) logInternal(ctx context.Context, err error) {
	logger.WithContext(ctx, h.log).ErrorContext(ctx, "internal_error", "error", err)
}

// Health responde o estado do serviço e do PostgreSQL.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	httpx.Health(w, r, h.deps, nil)
}
