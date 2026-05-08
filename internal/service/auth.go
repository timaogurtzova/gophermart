package service

import (
	"context"
	"errors"

	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrLoginAlreadyTaken возвращается, когда логин уже занят.
	ErrLoginAlreadyTaken = errors.New("login already taken")
	// ErrInvalidCredentials возвращается, когда логин или пароль неверные.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// AuthService реализует регистрацию и аутентификацию пользователей.
type AuthService struct {
	users repository.UserRepository
}

// NewAuthService создаёт сервис регистрации и аутентификации.
func NewAuthService(users repository.UserRepository) *AuthService {
	return &AuthService{users: users}
}

// Register регистрирует пользователя по паре логин/пароль.
func (s *AuthService) Register(ctx context.Context, login, password string) (model.User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}

	user, err := s.users.Create(ctx, login, string(passwordHash))
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return model.User{}, ErrLoginAlreadyTaken
		}

		return model.User{}, err
	}

	return user, nil
}

// Login аутентифицирует пользователя по паре логин/пароль.
func (s *AuthService) Login(ctx context.Context, login, password string) (model.User, error) {
	user, err := s.users.FindByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.User{}, ErrInvalidCredentials
		}

		return model.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return model.User{}, ErrInvalidCredentials
	}

	return user, nil
}
