package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/timaogurtzova/gophermart/internal/auth"
	"github.com/timaogurtzova/gophermart/internal/service"
)

// OrderHandler обслуживает загрузку номеров заказов.
type OrderHandler struct {
	service service.OrderService
	auth    userAuthenticator
}

// NewOrderHandler создаёт обработчик загрузки номеров заказов.
func NewOrderHandler(service service.OrderService, authenticator userAuthenticator) *OrderHandler {
	return &OrderHandler{service: service, auth: authenticator}
}

// Upload загружает номер заказа для расчёта начислений.
func (h *OrderHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, err := h.auth.UserID(r)
	if err != nil {
		if errors.Is(err, auth.ErrUserIDMissing) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !hasContentType(r, contentTypeText) {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	number := strings.TrimSpace(string(body))
	if err := h.service.UploadOrder(r.Context(), userID, number); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOrderFormat):
			writeError(w, http.StatusBadRequest, "bad request")
		case errors.Is(err, service.ErrInvalidOrderNumber):
			writeError(w, http.StatusUnprocessableEntity, "invalid order number")
		case errors.Is(err, service.ErrOrderAlreadyUploadedByUser):
			w.WriteHeader(http.StatusOK)
		case errors.Is(err, service.ErrOrderAlreadyUploadedByAnotherUser):
			writeError(w, http.StatusConflict, "order already uploaded by another user")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// GetOrders возвращает загруженные пользователем номера заказов.
func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := h.auth.UserID(r)
	if err != nil {
		if errors.Is(err, auth.ErrUserIDMissing) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	orders, err := h.service.GetOrders(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]orderResponse, len(orders))
	for i, order := range orders {
		response[i] = newOrderResponse(order)
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
