package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/httpx"
	platformlogger "github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
	platformmetrics "github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/metrics"
)

const maxPayloadBytes = 1 << 20

// TaskService define as operações de tarefas expostas pelo handler.
type TaskService interface {
	Create(ctx context.Context, userID int64, in domain.TaskInput) (*domain.Task, error)
	List(ctx context.Context, query domain.ListQuery) ([]domain.Task, error)
	Get(ctx context.Context, id, userID int64) (*domain.Task, error)
	Update(ctx context.Context, id, userID int64, in domain.TaskInput) (*domain.Task, error)
	Delete(ctx context.Context, id, userID int64) error
	Move(ctx context.Context, move domain.Move) (*domain.Task, error)
	History(ctx context.Context, id, userID int64, limit, offset int) ([]domain.HistoryEntry, error)
}

// Handler expõe os endpoints HTTP de tarefas e health.
type Handler struct {
	svc     TaskService
	deps    map[string]func(context.Context) error
	log     *slog.Logger
	metrics counter
	secret  string
}

type counter interface{ Inc(...string) }

// New cria o handler com suas dependências.
func New(svc TaskService, deps map[string]func(context.Context) error, log *slog.Logger, metrics counter, secret string) *Handler {
	return &Handler{svc: svc, deps: deps, log: log, metrics: metrics, secret: secret}
}

// Routes monta o mux com health público e rotas de tarefas protegidas por JWT.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)

	protected := func(operation string, next http.Handler) http.Handler {
		return auth.Middleware(h.secret, func(*http.Request) {
			h.metrics.Inc(operation, "unauthorized")
		})(next)
	}
	mux.Handle("GET /tasks", protected("list", http.HandlerFunc(h.List)))
	mux.Handle("POST /tasks", protected("create", http.HandlerFunc(h.Create)))
	mux.Handle("GET /tasks/{id}", protected("get", http.HandlerFunc(h.Get)))
	mux.Handle("PUT /tasks/{id}", protected("update", http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /tasks/{id}", protected("delete", http.HandlerFunc(h.Delete)))
	mux.Handle("PATCH /tasks/{id}/move", protected("move", http.HandlerFunc(h.Move)))
	mux.Handle("GET /tasks/{id}/history", protected("history", http.HandlerFunc(h.History)))
	return platformmetrics.RoutePattern(mux)
}

type taskPayload struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	Priority    string       `json:"priority"`
	DueDate     *domain.Date `json:"due_date"`
}

type taskResponse struct {
	ID          int64        `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	Priority    string       `json:"priority"`
	DueDate     *domain.Date `json:"due_date"`
	Position    float64      `json:"position"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type taskMutationResponse struct {
	ID int64 `json:"id"`
}

func newTaskResponse(task domain.Task) taskResponse {
	return taskResponse{
		ID: task.ID, Title: task.Title, Description: task.Description,
		Status: task.Status, Priority: task.Priority, DueDate: task.DueDate,
		Position: task.Position, CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
	}
}

func (p taskPayload) input() domain.TaskInput {
	return domain.TaskInput{Title: p.Title, Description: p.Description, Status: p.Status, Priority: p.Priority, DueDate: p.DueDate}
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

// List devolve as tarefas do usuário autenticado com filtros, ordenação e paginação.
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
	query, err := listQuery(r)
	if err != nil {
		h.metrics.Inc("list", "invalid_filter")
		httpx.WriteError(w, http.StatusBadRequest, "invalid filter")
		return
	}
	query.UserID, query.Limit, query.Offset = uid, limit, offset
	tasks, err := h.svc.List(r.Context(), query)
	if errors.Is(err, domain.ErrInvalidInput) {
		h.metrics.Inc("list", "invalid_filter")
		httpx.WriteError(w, http.StatusBadRequest, "invalid filter")
		return
	}
	if err != nil {
		h.metrics.Inc("list", "error")
		h.logError(r.Context(), "list tasks", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	responses := make([]taskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = newTaskResponse(task)
	}
	h.metrics.Inc("list", "success")
	httpx.WriteJSON(w, http.StatusOK, responses)
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
	task, err := h.svc.Create(r.Context(), uid, p.input())
	if errors.Is(err, domain.ErrInvalidInput) {
		h.metrics.Inc("create", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid title, status, priority or due date")
		return
	}
	if err != nil {
		h.metrics.Inc("create", "error")
		h.logError(r.Context(), "create task", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.metrics.Inc("create", "success")
	if task != nil {
		platformlogger.WithContext(r.Context(), h.log).InfoContext(r.Context(), "task_created", "task_id", task.ID, "user_id", uid, "status", task.Status, "priority", task.Priority)
	}
	httpx.WriteJSON(w, http.StatusCreated, taskMutationResponse{ID: task.ID})
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
	httpx.WriteJSON(w, http.StatusOK, newTaskResponse(*task))
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
	task, err := h.svc.Update(r.Context(), id, uid, p.input())
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
	if task != nil {
		platformlogger.WithContext(r.Context(), h.log).InfoContext(r.Context(), "task_updated", "task_id", task.ID, "user_id", uid, "status", task.Status, "priority", task.Priority)
	}
	httpx.WriteJSON(w, http.StatusOK, taskMutationResponse{ID: task.ID})
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
	platformlogger.WithContext(r.Context(), h.log).InfoContext(r.Context(), "task_deleted", "task_id", id, "user_id", uid)
	w.WriteHeader(http.StatusNoContent)
}

type movePayload struct {
	Status   string `json:"status"`
	BeforeID *int64 `json:"before_id"`
	AfterID  *int64 `json:"after_id"`
}

func (h *Handler) Move(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		h.metrics.Inc("move", "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseID(r)
	if err != nil {
		h.metrics.Inc("move", "invalid_id")
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p movePayload
	if err := decodePayload(w, r, &p); err != nil {
		h.metrics.Inc("move", "invalid_payload")
		writePayloadError(w, err)
		return
	}
	task, err := h.svc.Move(r.Context(), domain.Move{ID: id, UserID: uid, Status: p.Status, BeforeID: p.BeforeID, AfterID: p.AfterID})
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.metrics.Inc("move", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid move")
	case errors.Is(err, domain.ErrNotFound):
		h.metrics.Inc("move", "not_found")
		httpx.WriteError(w, http.StatusNotFound, "task not found")
	case errors.Is(err, domain.ErrConflict):
		h.metrics.Inc("move", "conflict")
		httpx.WriteError(w, http.StatusConflict, "board out of date")
	case err != nil:
		h.metrics.Inc("move", "error")
		h.logError(r.Context(), "move task", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	default:
		h.metrics.Inc("move", "success")
		if task != nil {
			platformlogger.WithContext(r.Context(), h.log).InfoContext(r.Context(), "task_moved", "task_id", task.ID, "user_id", uid, "status", task.Status, "position", task.Position)
		}
		httpx.WriteJSON(w, http.StatusOK, newTaskResponse(*task))
	}
}

// History devolve o histórico de alterações de uma tarefa do usuário autenticado.
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r)
	if !ok {
		h.metrics.Inc("history", "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseHistoryID(r)
	if err != nil {
		h.metrics.Inc("history", "invalid_id")
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		h.metrics.Inc("history", "invalid_pagination")
		httpx.WriteError(w, http.StatusBadRequest, "invalid pagination")
		return
	}
	entries, err := h.svc.History(r.Context(), id, uid, limit, offset)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		h.metrics.Inc("history", "not_found")
		httpx.WriteError(w, http.StatusNotFound, "task not found")
	case err != nil:
		h.metrics.Inc("history", "error")
		h.logError(r.Context(), "task history", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	default:
		if entries == nil {
			entries = []domain.HistoryEntry{}
		}
		h.metrics.Inc("history", "success")
		httpx.WriteJSON(w, http.StatusOK, entries)
	}
}

// parseHistoryID lê o ID da rota /tasks/{id}/history, inclusive fora do mux.
func parseHistoryID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	if raw == "" {
		raw = path.Base(path.Dir(r.URL.Path))
	}
	return strconv.ParseInt(raw, 10, 64)
}

// listQuery lê filtros e ordenação; a validação dos valores fica no serviço.
func listQuery(r *http.Request) (domain.ListQuery, error) {
	params := r.URL.Query()
	query := domain.ListQuery{
		Status:   params.Get("status"),
		Priority: params.Get("priority"),
		Sort:     params.Get("sort"),
		Order:    params.Get("order"),
	}
	if raw := params.Get("overdue"); raw != "" {
		overdue, err := strconv.ParseBool(raw)
		if err != nil {
			return query, fmt.Errorf("parse overdue: %w", err)
		}
		query.Overdue = overdue
	}
	return query, nil
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
	platformlogger.WithContext(ctx, h.log).ErrorContext(ctx, operation, "error", err)
}

// Health responde o estado do serviço, do PostgreSQL e do RabbitMQ.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	httpx.Health(w, r, h.deps, func(name string, err error) {
		h.logError(r.Context(), "health dependency failed: "+name, err)
	})
}
