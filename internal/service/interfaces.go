package service

import (
	"context"

	"github.com/timaogurtzova/gophermart/internal/model"
)

// UserService описывает контракт сервиса регистрации и аутентификации.
type UserService interface {
	Register(ctx context.Context, login, password string) (model.User, error)
	Login(ctx context.Context, login, password string) (model.User, error)
}
