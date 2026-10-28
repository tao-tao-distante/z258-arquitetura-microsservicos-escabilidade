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

type AuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, string, error)
}

type Handler struct {
	svc     AuthService
	deps    map[string]func(context.Context) error
	log     *slog.Logger
	limiter *middleware.RateLimiter
}

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

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /register", h.limiter.Handler(http.HandlerFunc(h.Register)))
	mux.Handle("POST /login", h.limiter.Handler(http.HandlerFunc(h.Login)))
	mux.HandleFunc("GET /health", h.Health)
	return middleware.Correlation(mux)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	c, status, err := decodeCredentials(w, r)
	if err != nil {
		writeError(w, status, "invalid payload")
		return
	}
	user, token, err := h.svc.Register(r.Context(), c.Email, c.Password)
	if errors.Is(err, domain.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid email or password")
			return
		}
		h.logInternal(r.Context(), err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{ID: user.ID, Email: user.Email, Token: token})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	c, status, err := decodeCredentials(w, r)
	if err != nil {
		writeError(w, status, "invalid payload")
		return
	}
	user, token, err := h.svc.Login(r.Context(), c.Email, c.Password)
	if errors.Is(err, domain.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid email or password")
			return
		}
		h.logInternal(r.Context(), err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{ID: user.ID, Email: user.Email, Token: token})
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

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	deps := map[string]string{}
	status := http.StatusOK
	overall := "ok"
	for name, check := range h.deps {
		if err := check(r.Context()); err != nil {
			deps[name] = "error"
			status = http.StatusServiceUnavailable
			overall = "degraded"
			continue
		}
		deps[name] = "ok"
	}
	writeJSON(w, status, map[string]any{"status": overall, "dependencies": deps})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	httpx.WriteJSON(w, status, payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	httpx.WriteError(w, status, msg)
}
