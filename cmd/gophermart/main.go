package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/timaogurtzova/gophermart/internal/auth"
	"github.com/timaogurtzova/gophermart/internal/config"
	httpserver "github.com/timaogurtzova/gophermart/internal/http"
	"github.com/timaogurtzova/gophermart/internal/http/handler"
	"github.com/timaogurtzova/gophermart/internal/postgres"
	"github.com/timaogurtzova/gophermart/internal/repository"
	"github.com/timaogurtzova/gophermart/internal/service"
)

// main запускает HTTP API накопительной системы лояльности «Гофермарт».
//
// По ТЗ система должна предоставлять HTTP API для регистрации, аутентификации
// и авторизации пользователей; приёма номеров заказов от зарегистрированных
// пользователей; учёта списка переданных номеров заказов; учёта накопительного
// счёта; проверки заказов через систему расчёта баллов лояльности; начисления
// положенного вознаграждения на счёт пользователя.
func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("application stopped with error")
	}
}

// run выполняет запуск приложения.
func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	database, err := postgres.Open(context.Background(), cfg.Database)
	if err != nil {
		return fmt.Errorf("initialize database connection: %w", err)
	}
	if database != nil {
		defer func() {
			if err := database.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing database connection")
			}
		}()
	}

	userRepository, err := repository.NewUserRepository(database.SQLDB())
	if err != nil {
		return fmt.Errorf("initialize user repository: %w", err)
	}

	authService := service.NewAuthService(userRepository)
	authSecret, err := auth.NewRandomSecret(32)
	if err != nil {
		return fmt.Errorf("generate auth secret: %w", err)
	}

	authenticator, err := auth.NewAuthenticator(authSecret)
	if err != nil {
		return fmt.Errorf("initialize authenticator: %w", err)
	}

	authHandler := handler.NewAuthHandler(authService, authenticator)
	router := httpserver.NewRouter(httpserver.RouterHandlers{
		Register: authHandler.Register,
		Login:    authHandler.Login,
	})
	server := httpserver.NewServer(cfg, router)
	if err := server.Run(); err != nil {
		return fmt.Errorf("run http server: %w", err)
	}

	return nil
}
