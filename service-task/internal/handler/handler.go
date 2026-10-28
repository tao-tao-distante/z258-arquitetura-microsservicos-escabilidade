package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strconv"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/httpx"
	platformlogger "github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
)

const maxPayloadBytes = 1 << 20

type TaskService interface {
	Create(ctx context.Context, userID int64, title, description string) (*domain.Task, error)
	List(ctx context.Context, userID int64, limit, offset int) ([]domain.Task, error)
	Get(ctx context.Context, id, userID int64) (*domain.Task, error)
	Update(ctx context.Context, id, userID int64, title, description, status string) (*domain.Task, error)
	Delete(ctx context.Context, id, userID int64) error
}

type Handler struct {
	svc  TaskService
	deps map[string]func(context.Context) error
	log  *slog.Logger
}

func New(svc TaskService, deps map[string]func(context.Context) error) *Handler {
	return &Handler{svc: svc, deps: deps, log: slog.Default()}
}

func NewWithLogger(svc TaskService, deps map[string]func(context.Context) error, log *slog.Logger) *Handler {
	return &Handler{svc: svc, deps: deps, log: log}
}

func (h *Handler) Routes(jwtSecret string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)

	protected := auth.Middleware(jwtSecret)
	mux.Handle("GET /tasks", protected(http.HandlerFunc(h.List)))
	mux.Handle("POST /tasks", protected(http.HandlerFunc(h.Create)))
	mux.Handle("GET /tasks/{id}", protected(http.HandlerFunc(h.Get)))
	mux.Handle("PUT /tasks/{id}", protected(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /tasks/{id}", protected(http.HandlerFunc(h.Delete)))
	return mux
}

type taskPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func userIDFrom(r *http.Request) (int64, bool) {
	claims := auth.FromContext(r.Context())
	if claims == nil {
		return 0, false
	}
	return claims.UserID, true
}

func parseID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	if raw == "" {
		raw = path.Base(r.URL.Path)
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pagination")
		return
	}
	tasks, err := h.svc.List(r.Context(), uid, limit, offset)
	if err != nil {
		h.logError(r.Context(), "list tasks", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var p taskPayload
	if err := decodePayload(w, r, &p); err != nil {
		writePayloadError(w, err)
		return
	}
	task, err := h.svc.Create(r.Context(), uid, p.Title, p.Description)
	if errors.Is(err, domain.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if err != nil {
		h.logError(r.Context(), "create task", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := h.svc.Get(r.Context(), id, uid)
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.logError(r.Context(), "get task", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p taskPayload
	if err := decodePayload(w, r, &p); err != nil {
		writePayloadError(w, err)
		return
	}
	task, err := h.svc.Update(r.Context(), id, uid, p.Title, p.Description, p.Status)
	if errors.Is(err, domain.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid task")
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.logError(r.Context(), "update task", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	err = h.svc.Delete(r.Context(), id, uid)
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.logError(r.Context(), "delete task", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pagination(r *http.Request) (int, int, error) {
	limit, offset := 50, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, errors.New("invalid limit")
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, errors.New("invalid offset")
		}
	}
	return limit, offset, nil
}

func decodePayload(w http.ResponseWriter, r *http.Request, dst any) error {
	return httpx.DecodeJSON(w, r, dst, maxPayloadBytes)
}

func writePayloadError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "payload too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid payload")
}

func (h *Handler) logError(ctx context.Context, operation string, err error) {
	platformlogger.WithCorrelation(ctx, h.log).ErrorContext(ctx, operation, "error", err)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	deps := map[string]string{}
	status := http.StatusOK
	overall := "ok"
	for name, check := range h.deps {
		if err := check(r.Context()); err != nil {
			h.logError(r.Context(), "health dependency failed: "+name, err)
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
