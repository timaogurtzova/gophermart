package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/timaogurtzova/gophermart/internal/service"
)

// AuthHandler обслуживает регистрацию и аутентификацию пользователей.
type AuthHandler struct {
	service service.UserService
	auth    userAuthenticator
}

// NewAuthHandler создаёт обработчик регистрации и аутентификации пользователей.
func NewAuthHandler(service service.UserService, authenticator userAuthenticator) *AuthHandler {
	return &AuthHandler{service: service, auth: authenticator}
}

// Register регистрирует пользователя и сразу устанавливает аутентификационную cookie.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	request, ok := readCredentials(w, r)
	if !ok {
		return
	}

	user, err := h.service.Register(r.Context(), request.Login, request.Password)
	if err != nil {
		if errors.Is(err, service.ErrLoginAlreadyTaken) {
			writeError(w, http.StatusConflict, "login already taken")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.auth.SetUserID(w, r, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Login аутентифицирует пользователя и устанавливает аутентификационную cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	request, ok := readCredentials(w, r)
	if !ok {
		return
	}

	user, err := h.service.Login(r.Context(), request.Login, request.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.auth.SetUserID(w, r, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func readCredentials(w http.ResponseWriter, r *http.Request) (credentialsRequest, bool) {
	if !hasContentType(r, contentTypeJSON) {
		writeError(w, http.StatusBadRequest, "bad request")
		return credentialsRequest{}, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer r.Body.Close()

	var request credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return credentialsRequest{}, false
	}

	request.Login = strings.TrimSpace(request.Login)
	if request.Login == "" || strings.TrimSpace(request.Password) == "" {
		writeError(w, http.StatusBadRequest, "bad request")
		return credentialsRequest{}, false
	}

	return request, true
}
