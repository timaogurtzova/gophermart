package handler

import "net/http"

// userAuthenticator описывает минимальный контракт аутентификатора для HTTP-обработчиков.
type userAuthenticator interface {
	SetUserID(w http.ResponseWriter, r *http.Request, userID int64) error
	UserID(r *http.Request) (int64, error)
}
