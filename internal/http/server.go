package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"github.com/timaogurtzova/gophermart/internal/config"
	httpmiddleware "github.com/timaogurtzova/gophermart/internal/http/middleware"
)

// Server инкапсулирует HTTP-сервер накопительной системы лояльности.
type Server struct {
	httpServer *http.Server
}

// RouterHandlers объединяет HTTP-обработчики роутера по именованным полям.
type RouterHandlers struct {
	Register       http.HandlerFunc
	Login          http.HandlerFunc
	UploadOrder    http.HandlerFunc
	GetOrders      http.HandlerFunc
	GetBalance     http.HandlerFunc
	Withdraw       http.HandlerFunc
	GetWithdrawals http.HandlerFunc
}

// NewServer создаёт HTTP-сервер с адресом из конфигурации.
func NewServer(cfg *config.Configuration, router http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    cfg.Server.Address,
			Handler: router,
		},
	}
}

// NewRouter регистрирует HTTP-маршруты накопительной системы лояльности.
func NewRouter(handlers RouterHandlers) http.Handler {
	r := chi.NewRouter()

	r.Use(httpmiddleware.GunzipRequest)
	r.Use(chimiddleware.Compress(5, "application/json", "text/html"))

	r.Post("/api/user/register", handlerOrNotImplemented(handlers.Register))
	r.Post("/api/user/login", handlerOrNotImplemented(handlers.Login))
	r.Post("/api/user/orders", handlerOrNotImplemented(handlers.UploadOrder))
	r.Get("/api/user/orders", handlerOrNotImplemented(handlers.GetOrders))
	r.Get("/api/user/balance", handlerOrNotImplemented(handlers.GetBalance))
	r.Post("/api/user/balance/withdraw", handlerOrNotImplemented(handlers.Withdraw))
	r.Get("/api/user/withdrawals", handlerOrNotImplemented(handlers.GetWithdrawals))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusBadRequest)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "method not allowed", http.StatusBadRequest)
	})

	return r
}

// Run запускает HTTP-сервер и выполняет graceful shutdown по сигналам ОС.
func (s *Server) Run() error {
	errChan := make(chan error, 1)

	go func() {
		log.Info().
			Str("addr", s.httpServer.Addr).
			Msg("HTTP server started")

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}

		log.Info().Msg("Stopped serving new connections")
		close(errChan)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	shutdownTimeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	log.Info().Msg("shutting down http server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("could not gracefully shutdown: %w", err)
	}

	log.Info().Msg("Server shutdown gracefully")
	return nil
}

func handlerOrNotImplemented(handler http.HandlerFunc) http.HandlerFunc {
	if handler != nil {
		return handler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}
}
