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

// TaskService define as operações de tarefas expostas pelo handler.
type TaskService interface {
	Create(ctx context.Context, userID int64, title, description, status string) (*domain.Task, error)
	List(ctx context.Context, userID int64, limit, offset int) ([]domain.Task, error)
	Get(ctx context.Context, id, userID int64) (*domain.Task, error)
	Update(ctx context.Context, id, userID int64, title, description, status string) (*domain.Task, error)
	Delete(ctx context.Context, id, userID int64) error
}

// Handler expõe os endpoints HTTP de tarefas e health.
type Handler struct {
	svc     TaskService
	deps    map[string]func(context.Context) error
	log     *slog.Logger
	metrics counter
}

type counter interface{ Inc(...string) }

type noopCounter struct{}

func (noopCounter) Inc(...string) {}

// New cria o handler com o serviço, as checagens de dependência e o logger informados.
func New(svc TaskService, deps map[string]func(context.Context) error, logs ...*slog.Logger) *Handler {
	log := slog.Default()
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	}
	return &Handler{svc: svc, deps: deps, log: log, metrics: noopCounter{}}
}

// WithTaskMetrics habilita contagem das operações de tarefa.
func (h *Handler) WithTaskMetrics(metrics counter) *Handler {
	if metrics != nil {
		h.metrics = metrics
	}
	return h
}

// Routes monta o mux com health público e rotas de tarefas protegidas por JWT.
func (h *Handler) Routes(jwtSecret string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)

	protected := func(operation string, next http.Handler) http.Handler {
		return auth.Middleware(jwtSecret, func(*http.Request) {
			h.metrics.Inc(operation, "unauthorized")
		})(next)
	}
	mux.Handle("GET /tasks", protected("list", http.HandlerFunc(h.List)))
	mux.Handle("POST /tasks", protected("create", http.HandlerFunc(h.Create)))
	mux.Handle("GET /tasks/{id}", protected("get", http.HandlerFunc(h.Get)))
	mux.Handle("PUT /tasks/{id}", protected("update", http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /tasks/{id}", protected("delete", http.HandlerFunc(h.Delete)))
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

// List devolve as tarefas do usuário autenticado com paginação.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		h.metrics.Inc("list", "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		h.metrics.Inc("list", "invalid_pagination")
		httpx.WriteError(w, http.StatusBadRequest, "invalid pagination")
		return
	}
	tasks, err := h.svc.List(r.Context(), uid, limit, offset)
	if err != nil {
		h.metrics.Inc("list", "error")
		h.logError(r.Context(), "list tasks", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	h.metrics.Inc("list", "success")
	httpx.WriteJSON(w, http.StatusOK, tasks)
}

// Create cria uma tarefa para o usuário autenticado.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		h.metrics.Inc("create", "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var p taskPayload
	if err := decodePayload(w, r, &p); err != nil {
		h.metrics.Inc("create", "invalid_payload")
		writePayloadError(w, err)
		return
	}
	task, err := h.svc.Create(r.Context(), uid, p.Title, p.Description, p.Status)
	if errors.Is(err, domain.ErrInvalidInput) {
		h.metrics.Inc("create", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid title or status")
		return
	}
	if err != nil {
		h.metrics.Inc("create", "error")
		h.logError(r.Context(), "create task", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.metrics.Inc("create", "success")
	httpx.WriteJSON(w, http.StatusCreated, task)
}

// Get devolve uma tarefa do usuário autenticado.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		h.metrics.Inc("get", "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseID(r)
	if err != nil {
		h.metrics.Inc("get", "invalid_id")
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := h.svc.Get(r.Context(), id, uid)
	if errors.Is(err, domain.ErrNotFound) {
		h.metrics.Inc("get", "not_found")
		httpx.WriteError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.metrics.Inc("get", "error")
		h.logError(r.Context(), "get task", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.metrics.Inc("get", "success")
	httpx.WriteJSON(w, http.StatusOK, task)
}

// Update atualiza uma tarefa do usuário autenticado.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		h.metrics.Inc("update", "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseID(r)
	if err != nil {
		h.metrics.Inc("update", "invalid_id")
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p taskPayload
	if err := decodePayload(w, r, &p); err != nil {
		h.metrics.Inc("update", "invalid_payload")
		writePayloadError(w, err)
		return
	}
	task, err := h.svc.Update(r.Context(), id, uid, p.Title, p.Description, p.Status)
	if errors.Is(err, domain.ErrInvalidInput) {
		h.metrics.Inc("update", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid task")
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		h.metrics.Inc("update", "not_found")
		httpx.WriteError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.metrics.Inc("update", "error")
		h.logError(r.Context(), "update task", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.metrics.Inc("update", "success")
	httpx.WriteJSON(w, http.StatusOK, task)
}

// Delete remove uma tarefa do usuário autenticado.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		h.metrics.Inc("delete", "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseID(r)
	if err != nil {
		h.metrics.Inc("delete", "invalid_id")
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	err = h.svc.Delete(r.Context(), id, uid)
	if errors.Is(err, domain.ErrNotFound) {
		h.metrics.Inc("delete", "not_found")
		httpx.WriteError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.metrics.Inc("delete", "error")
		h.logError(r.Context(), "delete task", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.metrics.Inc("delete", "success")
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
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "payload too large")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "invalid payload")
}

func (h *Handler) logError(ctx context.Context, operation string, err error) {
	platformlogger.WithCorrelation(ctx, h.log).ErrorContext(ctx, operation, "error", err)
}

// Health responde o estado do serviço, do PostgreSQL e do RabbitMQ.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	httpx.Health(w, r, h.deps, func(name string, err error) {
		h.logError(r.Context(), "health dependency failed: "+name, err)
	})
}
