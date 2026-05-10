package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/timaogurtzova/gophermart/internal/auth"
	"github.com/timaogurtzova/gophermart/internal/service"
)

// BalanceHandler обслуживает получение накопительного счёта пользователя.
type BalanceHandler struct {
	service service.BalanceService
	auth    userAuthenticator
}

// NewBalanceHandler создаёт обработчик получения накопительного счёта.
func NewBalanceHandler(service service.BalanceService, authenticator userAuthenticator) *BalanceHandler {
	return &BalanceHandler{service: service, auth: authenticator}
}

// GetBalance возвращает текущий баланс пользователя и сумму списаний.
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := h.auth.UserID(r)
	if err != nil {
		if errors.Is(err, auth.ErrUserIDMissing) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	balance, err := h.service.GetBalance(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responseBody, err := json.Marshal(newBalanceResponse(balance))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	writeResponse(w, responseBody)
}
