package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/timaogurtzova/gophermart/internal/accrual"
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
	defer func() {
		if err := database.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close database connection")
		}
	}()

	userRepository, err := repository.NewUserRepository(database.SQLDB())
	if err != nil {
		return fmt.Errorf("initialize user repository: %w", err)
	}
	orderRepository, err := repository.NewOrderRepository(database.SQLDB())
	if err != nil {
		return fmt.Errorf("initialize order repository: %w", err)
	}
	balanceRepository, err := repository.NewBalanceRepository(database.SQLDB())
	if err != nil {
		return fmt.Errorf("initialize balance repository: %w", err)
	}

	authService := service.NewAuthService(userRepository)
	orderService := service.NewOrderUploadService(orderRepository)
	balanceService := service.NewBalanceAccountService(balanceRepository)
	var authSecret []byte
	if !cfg.Auth.IsConfigured() {
		log.Warn().Msg("Auth secret is not configured; generated secret will invalidate sessions after restart")
		authSecret, err = auth.NewRandomSecret(32)
		if err != nil {
			return fmt.Errorf("generate auth secret: %w", err)
		}
	} else {
		authSecret = []byte(*cfg.Auth.Secret)
	}

	authenticator, err := auth.NewAuthenticator(authSecret)
	if err != nil {
		return fmt.Errorf("initialize authenticator: %w", err)
	}

	authHandler := handler.NewAuthHandler(authService, authenticator)
	orderHandler := handler.NewOrderHandler(orderService, authenticator)
	balanceHandler := handler.NewBalanceHandler(balanceService, authenticator)

	processorCtx, cancelProcessor := context.WithCancel(context.Background())
	defer cancelProcessor()
	if cfg.Accrual.IsConfigured() {
		accrualClient, err := accrual.NewClient(*cfg.Accrual.Address)
		if err != nil {
			return fmt.Errorf("initialize accrual client: %w", err)
		}

		orderProcessor := service.NewOrderProcessor(orderRepository, accrualClient)
		go orderProcessor.Run(processorCtx)
	} else {
		log.Warn().Msg("Accrual system address is not configured; order processing disabled")
	}

	router := httpserver.NewRouter(httpserver.RouterHandlers{
		Register:       authHandler.Register,
		Login:          authHandler.Login,
		UploadOrder:    orderHandler.Upload,
		GetOrders:      orderHandler.GetOrders,
		GetBalance:     balanceHandler.GetBalance,
		Withdraw:       balanceHandler.Withdraw,
		GetWithdrawals: balanceHandler.GetWithdrawals,
	})
	server := httpserver.NewServer(cfg, router)
	if err := server.Run(); err != nil {
		return fmt.Errorf("run http server: %w", err)
	}

	return nil
}
