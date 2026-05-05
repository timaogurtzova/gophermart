package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/timaogurtzova/gophermart/internal/config"
	httpserver "github.com/timaogurtzova/gophermart/internal/http"
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
		return fmt.Errorf("загрузка конфигурации: %w", err)
	}

	router := httpserver.NewRouter(httpserver.RouterHandlers{})
	server := httpserver.NewServer(cfg, router)
	if err := server.Run(); err != nil {
		return fmt.Errorf("запуск HTTP-сервера: %w", err)
	}

	return nil
}
