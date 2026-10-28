package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// WriteJSON escreve payload como JSON com o status informado.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError escreve um erro JSON com o status informado.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// DecodeJSON lê um único objeto JSON, rejeita campos desconhecidos e limita o corpo a maxBytes.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("payload must contain exactly one JSON object")
	}
	return nil
}

// Health executa as checagens de dependência e responde 200 ou 503.
func Health(
	w http.ResponseWriter,
	r *http.Request,
	deps map[string]func(context.Context) error,
	onError func(name string, err error),
) {
	dependencies := make(map[string]string, len(deps))
	status := http.StatusOK
	overall := "ok"
	for name, check := range deps {
		if err := check(r.Context()); err != nil {
			if onError != nil {
				onError(name, err)
			}
			dependencies[name] = "error"
			status = http.StatusServiceUnavailable
			overall = "degraded"
			continue
		}
		dependencies[name] = "ok"
	}
	WriteJSON(w, status, map[string]any{"status": overall, "dependencies": dependencies})
}
