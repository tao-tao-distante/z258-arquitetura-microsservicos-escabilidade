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
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/middleware"
)

const maxPayloadBytes = 16 << 10

// AuthService define as operações de autenticação expostas pelo handler.
type AuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, string, error)
}

// Handler expõe os endpoints HTTP de registro, login e health.
type Handler struct {
	svc     AuthService
	deps    map[string]func(context.Context) error
	log     *slog.Logger
	limiter *middleware.RateLimiter
}

// New cria o handler com o serviço, as checagens de dependência e o logger informados.
func New(svc AuthService, deps map[string]func(context.Context) error, logs ...*slog.Logger) *Handler {
	log := slog.Default()
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	}
	return &Handler{svc: svc, deps: deps, log: log, limiter: middleware.NewRateLimiter(10, time.Minute, time.Now)}
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	ID    int64  `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	Token string `json:"token"`
}

// Routes monta o mux com rate limit em registro/login e correlação global.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /register", h.limiter.Handler(http.HandlerFunc(h.Register)))
	mux.Handle("POST /login", h.limiter.Handler(http.HandlerFunc(h.Login)))
	mux.HandleFunc("GET /health", h.Health)
	return middleware.Correlation(mux)
}

// Register cria um usuário e devolve o JWT emitido.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	c, status, err := decodeCredentials(w, r)
	if err != nil {
		httpx.WriteError(w, status, "invalid payload")
		return
	}
	user, token, err := h.svc.Register(r.Context(), c.Email, c.Password)
	if errors.Is(err, domain.ErrEmailTaken) {
		httpx.WriteError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid email or password")
			return
		}
		h.logInternal(r.Context(), err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, authResponse{ID: user.ID, Email: user.Email, Token: token})
}

// Login autentica o usuário e devolve o JWT emitido.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	c, status, err := decodeCredentials(w, r)
	if err != nil {
		httpx.WriteError(w, status, "invalid payload")
		return
	}
	user, token, err := h.svc.Login(r.Context(), c.Email, c.Password)
	if errors.Is(err, domain.ErrInvalidCredentials) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid email or password")
			return
		}
		h.logInternal(r.Context(), err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, authResponse{ID: user.ID, Email: user.Email, Token: token})
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, int, error) {
	var c credentials
	if err := httpx.DecodeJSON(w, r, &c, maxPayloadBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return credentials{}, http.StatusRequestEntityTooLarge, err
		}
		return credentials{}, http.StatusBadRequest, err
	}
	if c.Email == "" || c.Password == "" {
		return credentials{}, http.StatusBadRequest, errors.New("credentials required")
	}
	return c, 0, nil
}

func (h *Handler) logInternal(ctx context.Context, err error) {
	logger.WithCorrelation(ctx, h.log).ErrorContext(ctx, "internal_error", "error", err)
}

// Health responde o estado do serviço e do PostgreSQL.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	httpx.Health(w, r, h.deps, nil)
}
