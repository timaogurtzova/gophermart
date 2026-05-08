package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/timaogurtzova/gophermart/internal/config"
	httpserver "github.com/timaogurtzova/gophermart/internal/http"
	"github.com/timaogurtzova/gophermart/internal/postgres"
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

	router := httpserver.NewRouter(httpserver.RouterHandlers{})
	server := httpserver.NewServer(cfg, router)
	if err := server.Run(); err != nil {
		return fmt.Errorf("run http server: %w", err)
	}

	return nil
}
