package handler

import (
	"errors"
	"net/http"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/httpx"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
)

type profileResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type profilePayload struct {
	Name string `json:"name"`
}

type passwordPayload struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// protected exige Bearer token válido; sem segredo configurado, rejeita todas as requisições.
func (h *Handler) protected(operation string, next http.HandlerFunc) http.Handler {
	if h.secret == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			h.metrics.Inc(operation, "unauthorized")
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		})
	}
	return auth.Middleware(h.secret, func(*http.Request) {
		h.metrics.Inc(operation, "unauthorized")
	})(next)
}

// GetProfile devolve id, e-mail e nome do usuário autenticado.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(w, r, "get_profile")
	if !ok {
		return
	}
	user, err := h.svc.Profile(r.Context(), uid)
	if err != nil {
		h.writeProfileError(w, r, "get_profile", err)
		return
	}
	h.metrics.Inc("get_profile", "success")
	httpx.WriteJSON(w, http.StatusOK, toProfile(user))
}

// UpdateProfile altera o nome de exibição do usuário autenticado.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(w, r, "update_profile")
	if !ok {
		return
	}
	var p profilePayload
	if status, err := decodePayload(w, r, &p); err != nil {
		h.metrics.Inc("update_profile", "invalid_payload")
		httpx.WriteError(w, status, "invalid payload")
		return
	}
	user, err := h.svc.UpdateName(r.Context(), uid, p.Name)
	if errors.Is(err, domain.ErrInvalidInput) {
		h.metrics.Inc("update_profile", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid name")
		return
	}
	if err != nil {
		h.writeProfileError(w, r, "update_profile", err)
		return
	}
	h.metrics.Inc("update_profile", "success")
	logger.WithContext(r.Context(), h.log).InfoContext(r.Context(), "profile_updated", "user_id", uid)
	httpx.WriteJSON(w, http.StatusOK, toProfile(user))
}

// ChangePassword troca a senha do usuário autenticado após conferir a senha atual.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(w, r, "change_password")
	if !ok {
		return
	}
	var p passwordPayload
	status, err := decodePayload(w, r, &p)
	if err == nil && (p.CurrentPassword == "" || p.NewPassword == "") {
		status, err = http.StatusBadRequest, errors.New("passwords required")
	}
	if err != nil {
		h.metrics.Inc("change_password", "invalid_payload")
		httpx.WriteError(w, status, "invalid payload")
		return
	}
	err = h.svc.ChangePassword(r.Context(), uid, p.CurrentPassword, p.NewPassword)
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.metrics.Inc("change_password", "invalid_input")
		httpx.WriteError(w, http.StatusBadRequest, "invalid password")
	case errors.Is(err, domain.ErrWrongPassword):
		h.metrics.Inc("change_password", "wrong_password")
		httpx.WriteError(w, http.StatusForbidden, "current password is incorrect")
	case err != nil:
		h.writeProfileError(w, r, "change_password", err)
	default:
		h.metrics.Inc("change_password", "success")
		logger.WithContext(r.Context(), h.log).InfoContext(r.Context(), "password_changed", "user_id", uid)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *Handler) userID(w http.ResponseWriter, r *http.Request, operation string) (int64, bool) {
	claims := auth.FromContext(r.Context())
	if claims == nil {
		h.metrics.Inc(operation, "unauthorized")
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	return claims.UserID, true
}

func (h *Handler) writeProfileError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		h.metrics.Inc(operation, "not_found")
		httpx.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	h.logInternal(r.Context(), err)
	h.metrics.Inc(operation, "error")
	httpx.WriteError(w, http.StatusInternalServerError, "internal error")
}

func decodePayload(w http.ResponseWriter, r *http.Request, dst any) (int, error) {
	if err := httpx.DecodeJSON(w, r, dst, maxPayloadBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return http.StatusRequestEntityTooLarge, err
		}
		return http.StatusBadRequest, err
	}
	return 0, nil
}

func toProfile(user *domain.User) profileResponse {
	return profileResponse{ID: user.ID, Email: user.Email, Name: user.Name}
}
