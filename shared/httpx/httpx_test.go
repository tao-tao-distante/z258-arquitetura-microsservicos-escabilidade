package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteJSONSetsContentTypeAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]string{"ok": "true"})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.JSONEq(t, `{"ok":"true"}`, rec.Body.String())
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "invalid")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.JSONEq(t, `{"error":"invalid"}`, rec.Body.String())
}

func TestDecodeJSONSuccess(t *testing.T) {
	var dst struct {
		Title string `json:"title"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"x"}`))
	rec := httptest.NewRecorder()
	require.NoError(t, DecodeJSON(rec, req, &dst, 1024))
	require.Equal(t, "x", dst.Title)
}

func TestDecodeJSONRejectsUnknownField(t *testing.T) {
	var dst struct {
		Title string `json:"title"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"other":"x"}`))
	rec := httptest.NewRecorder()
	require.Error(t, DecodeJSON(rec, req, &dst, 1024))
}

func TestDecodeJSONRejectsMultipleObjects(t *testing.T) {
	var dst struct {
		Title string `json:"title"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"a"}{"title":"b"}`))
	rec := httptest.NewRecorder()
	require.Error(t, DecodeJSON(rec, req, &dst, 1024))
}

func TestDecodeJSONRejectsOversizedPayload(t *testing.T) {
	var dst struct {
		Title string `json:"title"`
	}
	body := `{"title":"` + strings.Repeat("a", 2048) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	require.Error(t, DecodeJSON(rec, req, &dst, 16))
}
