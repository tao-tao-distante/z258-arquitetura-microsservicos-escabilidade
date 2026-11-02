package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
)

func moveRequest(id, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPatch, "/tasks/"+id+"/move", bytes.NewBufferString(body)).WithContext(ctxWithClaims())
	req.SetPathValue("id", id)
	return req
}

func TestMovePassesPayloadToService(t *testing.T) {
	svc := &fakeService{task: &domain.Task{ID: 5, Status: "doing", Position: 1536}}
	rec := httptest.NewRecorder()

	newTestHandler(svc, nil).Move(rec, moveRequest("5", `{"status":"doing","before_id":2,"after_id":3}`))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"position":1536`)
	require.Equal(t, int64(5), svc.move.ID)
	require.Equal(t, int64(1), svc.move.UserID)
	require.Equal(t, "doing", svc.move.Status)
	require.Equal(t, int64(2), *svc.move.BeforeID)
	require.Equal(t, int64(3), *svc.move.AfterID)
}

func TestMoveAcceptsNullNeighbors(t *testing.T) {
	svc := &fakeService{task: &domain.Task{ID: 5}}
	rec := httptest.NewRecorder()

	newTestHandler(svc, nil).Move(rec, moveRequest("5", `{"status":"todo","before_id":null}`))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Nil(t, svc.move.BeforeID)
	require.Nil(t, svc.move.AfterID)
}

func TestMoveErrors(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		body     string
		err      error
		wantCode int
		wantLbl  string
	}{
		{name: "invalid id", id: "x", body: `{}`, wantCode: http.StatusBadRequest, wantLbl: "invalid_id"},
		{name: "invalid json", id: "5", body: `{`, wantCode: http.StatusBadRequest, wantLbl: "invalid_payload"},
		{name: "payload too large", id: "5", body: `{"status":"` + string(bytes.Repeat([]byte("x"), maxPayloadBytes)) + `"}`, wantCode: http.StatusRequestEntityTooLarge, wantLbl: "invalid_payload"},
		{name: "invalid input", id: "5", body: `{"status":"x"}`, err: domain.ErrInvalidInput, wantCode: http.StatusBadRequest, wantLbl: "invalid_input"},
		{name: "not found", id: "5", body: `{"status":"todo"}`, err: domain.ErrNotFound, wantCode: http.StatusNotFound, wantLbl: "not_found"},
		{name: "conflict", id: "5", body: `{"status":"todo","before_id":9}`, err: domain.ErrConflict, wantCode: http.StatusConflict, wantLbl: "conflict"},
		{name: "internal error", id: "5", body: `{"status":"todo"}`, err: errors.New("db down"), wantCode: http.StatusInternalServerError, wantLbl: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			rec := httptest.NewRecorder()

			newTestHandlerWithCounter(&fakeService{err: tt.err}, nil, counter).Move(rec, moveRequest(tt.id, tt.body))

			require.Equal(t, tt.wantCode, rec.Code)
			require.Equal(t, [][]string{{"move", tt.wantLbl}}, counter.labels)
		})
	}
}

func TestRoutesMoveRequiresJWT(t *testing.T) {
	rr := newTestHandler(&fakeService{task: &domain.Task{ID: 5}}, nil).Routes()

	rec := httptest.NewRecorder()
	rr.ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/tasks/5/move", bytes.NewBufferString(`{"status":"todo"}`)))
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/5/move", bytes.NewBufferString(`{"status":"todo"}`))
	req.Header.Set("Authorization", "Bearer "+token(t, "secret"))
	rec = httptest.NewRecorder()
	rr.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
