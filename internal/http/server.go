package httpserver

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

type route struct {
	pattern string
	handler http.HandlerFunc
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
	mux := http.NewServeMux()

	for route := range handlers.routes() {
		mux.HandleFunc(route.pattern, handlerOrNotImplemented(route.handler))
	}

	return httpmiddleware.Logging(httpmiddleware.GunzipRequest(httpmiddleware.GzipResponse(mux)))
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

func (handlers RouterHandlers) routes() iter.Seq[route] {
	return func(yield func(route) bool) {
		for _, route := range []route{
			{pattern: "POST /api/user/register", handler: handlers.Register},
			{pattern: "POST /api/user/login", handler: handlers.Login},
			{pattern: "POST /api/user/orders", handler: handlers.UploadOrder},
			{pattern: "GET /api/user/orders", handler: handlers.GetOrders},
			{pattern: "GET /api/user/balance", handler: handlers.GetBalance},
			{pattern: "POST /api/user/balance/withdraw", handler: handlers.Withdraw},
			{pattern: "GET /api/user/withdrawals", handler: handlers.GetWithdrawals},
		} {
			if !yield(route) {
				return
			}
		}
	}
}
