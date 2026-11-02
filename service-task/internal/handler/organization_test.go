package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
)

func TestCreatePassesPriorityAndDueDate(t *testing.T) {
	svc := &fakeService{task: &domain.Task{ID: 1}}
	body := bytes.NewBufferString(`{"title":"x","description":"d","status":"doing","priority":"high","due_date":"2026-11-15"}`)
	rec := httptest.NewRecorder()

	newTestHandler(svc, nil).Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", body).WithContext(ctxWithClaims()))

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "x", svc.input.Title)
	require.Equal(t, "d", svc.input.Description)
	require.Equal(t, "doing", svc.input.Status)
	require.Equal(t, "high", svc.input.Priority)
	require.Equal(t, "2026-11-15", svc.input.DueDate.String())
}

func TestUpdatePassesNullDueDate(t *testing.T) {
	svc := &fakeService{task: &domain.Task{ID: 1}}
	body := bytes.NewBufferString(`{"title":"x","priority":"low","due_date":null}`)
	rec := httptest.NewRecorder()

	newTestHandler(svc, nil).Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/1", body).WithContext(ctxWithClaims()))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "low", svc.input.Priority)
	require.Nil(t, svc.input.DueDate)
}

func TestTaskResponseIncludesPriorityAndDueDate(t *testing.T) {
	due, err := domain.ParseDate("2026-11-15")
	require.NoError(t, err)
	svc := &fakeService{task: &domain.Task{ID: 1, Priority: "urgent", DueDate: &due}}
	rec := httptest.NewRecorder()

	newTestHandler(svc, nil).Get(rec, httptest.NewRequest(http.MethodGet, "/tasks/1", nil).WithContext(ctxWithClaims()))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"priority":"urgent"`)
	require.Contains(t, rec.Body.String(), `"due_date":"2026-11-15"`)
}

func TestCreateRejectsMalformedDueDate(t *testing.T) {
	for _, due := range []string{`"15/11/2026"`, `"2026-11-15T00:00:00Z"`, `20261115`} {
		counter := &fakeCounter{}
		svc := &fakeService{task: &domain.Task{ID: 1}}
		body := bytes.NewBufferString(`{"title":"x","due_date":` + due + `}`)
		rec := httptest.NewRecorder()

		newTestHandlerWithCounter(svc, nil, counter).Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", body).WithContext(ctxWithClaims()))

		require.Equal(t, http.StatusBadRequest, rec.Code, due)
		require.Equal(t, [][]string{{"create", "invalid_payload"}}, counter.labels)
	}
}

func TestCreateInvalidInputMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(&fakeService{err: domain.ErrInvalidInput}, nil).Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"x","priority":"critical"}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid title, status, priority or due date")
}

func TestListPassesFiltersAndSort(t *testing.T) {
	svc := &fakeService{}
	rec := httptest.NewRecorder()

	newTestHandler(svc, nil).List(rec, httptest.NewRequest(http.MethodGet, "/tasks?status=doing&priority=high&overdue=true&sort=due_date&order=desc&limit=10&offset=20", nil).WithContext(ctxWithClaims()))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, domain.ListQuery{
		UserID: 1, Status: "doing", Priority: "high", Overdue: true, Sort: "due_date", Order: "desc", Limit: 10, Offset: 20,
	}, svc.query)
}

func TestListRejectsInvalidFilters(t *testing.T) {
	tests := []struct {
		name  string
		query string
		err   error
	}{
		{name: "overdue is not a boolean", query: "overdue=maybe"},
		{name: "service rejects filter", query: "sort=title", err: domain.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			rec := httptest.NewRecorder()

			newTestHandlerWithCounter(&fakeService{err: tt.err}, nil, counter).List(rec, httptest.NewRequest(http.MethodGet, "/tasks?"+tt.query, nil).WithContext(ctxWithClaims()))

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "invalid filter")
			require.Equal(t, [][]string{{"list", "invalid_filter"}}, counter.labels)
		})
	}
}

func historyRequest(id, query string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/tasks/"+id+"/history"+query, nil).WithContext(ctxWithClaims())
	req.SetPathValue("id", id)
	return req
}

func TestHistorySuccess(t *testing.T) {
	changedAt := time.Date(2026, 10, 30, 14, 2, 0, 0, time.UTC)
	svc := &fakeService{history: []domain.HistoryEntry{{
		ID: 3, TaskID: 5, Action: domain.ActionMoved, CorrelationID: "cid-secret",
		Changes: []domain.FieldChange{domain.StatusChange("todo", "doing")}, ChangedAt: changedAt,
	}}}
	counter := &fakeCounter{}
	rec := httptest.NewRecorder()

	newTestHandlerWithCounter(svc, nil, counter).History(rec, historyRequest("5", "?limit=10&offset=10"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `[{"id":3,"task_id":5,"action":"moved","changes":[{"field":"status","from":"todo","to":"doing"}],"changed_at":"2026-10-30T14:02:00Z"}]`, rec.Body.String())
	require.Equal(t, 10, svc.limit)
	require.Equal(t, 10, svc.offset)
	require.Equal(t, [][]string{{"history", "success"}}, counter.labels)
}

func TestHistoryNilBecomesEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(&fakeService{}, nil).History(rec, historyRequest("5", ""))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `[]`, rec.Body.String())
}

func TestHistoryParsesIDWithoutMux(t *testing.T) {
	svc := &fakeService{}
	rec := httptest.NewRecorder()
	newTestHandler(svc, nil).History(rec, httptest.NewRequest(http.MethodGet, "/tasks/7/history", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHistoryErrors(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		query    string
		err      error
		wantCode int
		wantLbl  string
	}{
		{name: "invalid id", id: "x", wantCode: http.StatusBadRequest, wantLbl: "invalid_id"},
		{name: "invalid pagination", id: "5", query: "?limit=0", wantCode: http.StatusBadRequest, wantLbl: "invalid_pagination"},
		{name: "not found", id: "5", err: domain.ErrNotFound, wantCode: http.StatusNotFound, wantLbl: "not_found"},
		{name: "internal error", id: "5", err: errors.New("db down"), wantCode: http.StatusInternalServerError, wantLbl: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			rec := httptest.NewRecorder()

			newTestHandlerWithCounter(&fakeService{err: tt.err}, nil, counter).History(rec, historyRequest(tt.id, tt.query))

			require.Equal(t, tt.wantCode, rec.Code)
			require.Equal(t, [][]string{{"history", tt.wantLbl}}, counter.labels)
		})
	}
}

func TestRoutesHistoryRequiresJWT(t *testing.T) {
	rr := newTestHandler(&fakeService{}, nil).Routes()

	rec := httptest.NewRecorder()
	rr.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/5/history", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	req := httptest.NewRequest(http.MethodGet, "/tasks/5/history", nil)
	req.Header.Set("Authorization", "Bearer "+token(t, "secret"))
	rec = httptest.NewRecorder()
	rr.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
