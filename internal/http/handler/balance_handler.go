package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/timaogurtzova/gophermart/internal/auth"
	"github.com/timaogurtzova/gophermart/internal/service"
)

// BalanceHandler обслуживает накопительный счёт и списания пользователя.
type BalanceHandler struct {
	service service.BalanceService
	auth    userAuthenticator
}

// NewBalanceHandler создаёт обработчик накопительного счёта и списаний.
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

// Withdraw списывает баллы с накопительного счёта в счёт оплаты нового заказа.
func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := h.auth.UserID(r)
	if err != nil {
		if errors.Is(err, auth.ErrUserIDMissing) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	request, ok := readWithdrawRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.Withdraw(r.Context(), userID, request.Order, request.Sum); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOrderNumber):
			writeError(w, http.StatusUnprocessableEntity, "invalid order number")
		case errors.Is(err, service.ErrInsufficientBalance):
			writeError(w, http.StatusPaymentRequired, "insufficient balance")
		case errors.Is(err, service.ErrInvalidWithdrawalAmount):
			writeError(w, http.StatusBadRequest, "bad request")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetWithdrawals возвращает историю списаний пользователя.
func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, err := h.auth.UserID(r)
	if err != nil {
		if errors.Is(err, auth.ErrUserIDMissing) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	withdrawals, err := h.service.GetWithdrawals(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]withdrawalResponse, len(withdrawals))
	for i, withdrawal := range withdrawals {
		response[i] = newWithdrawalResponse(withdrawal)
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	writeResponse(w, responseBody)
}

func readWithdrawRequest(w http.ResponseWriter, r *http.Request) (withdrawRequest, bool) {
	if !hasContentType(r, contentTypeJSON) {
		writeError(w, http.StatusBadRequest, "bad request")
		return withdrawRequest{}, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer r.Body.Close()

	var request withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return withdrawRequest{}, false
	}

	request.normalize()
	return request, true
}
