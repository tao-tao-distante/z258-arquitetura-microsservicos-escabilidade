package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
)

func TestTaskBusinessLogsUseInfoWithoutContent(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	svc := &fakeService{task: &domain.Task{ID: 9, UserID: 1, Title: "segredo", Description: "conteudo", Status: "todo", Priority: "high", Position: 1.5}}

	h := New(svc, nil, log, &fakeCounter{}, "secret")
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"segredo","description":"conteudo","priority":"high"}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/9", bytes.NewBufferString(`{"title":"segredo","description":"conteudo","status":"done"}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/tasks/9/move", bytes.NewBufferString(`{"status":"doing"}`))
	req.SetPathValue("id", "9")
	h.Move(rec, req.WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	h.Delete(rec, httptest.NewRequest(http.MethodDelete, "/tasks/9", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusNoContent, rec.Code)

	output := logs.String()
	for _, event := range []string{"task_created", "task_updated", "task_moved", "task_deleted"} {
		require.Contains(t, output, `"msg":"`+event+`"`)
	}
	require.Contains(t, output, `"level":"INFO"`)
	require.Contains(t, output, `"task_id":9`)
	require.Contains(t, output, `"user_id":1`)
	require.Contains(t, output, `"priority":"high"`)
	require.NotContains(t, output, "segredo")
	require.NotContains(t, output, "conteudo")
}
