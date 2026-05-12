package service

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/timaogurtzova/gophermart/internal/accrual"
	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/repository"
)

const (
	defaultProcessingInterval = time.Second
	defaultProcessingLimit    = 10
)

// AccrualClient описывает клиент внешней системы расчёта начислений.
type AccrualClient interface {
	GetOrder(ctx context.Context, number string) (accrual.Order, error)
}

// OrderProcessor обрабатывает заказы через внешнюю систему начислений.
type OrderProcessor struct {
	orders   repository.OrderRepository
	accrual  AccrualClient
	interval time.Duration
	limit    int
}

// NewOrderProcessor создаёт фоновый обработчик заказов.
func NewOrderProcessor(orders repository.OrderRepository, accrualClient AccrualClient) *OrderProcessor {
	return &OrderProcessor{
		orders:   orders,
		accrual:  accrualClient,
		interval: defaultProcessingInterval,
		limit:    defaultProcessingLimit,
	}
}

// Run запускает обработку заказов до отмены контекста.
func (p *OrderProcessor) Run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("order processor stopped")
			return
		case <-timer.C:
		}

		delay := p.interval
		if err := p.ProcessPendingOrders(ctx); err != nil {
			var rateLimitErr *accrual.RateLimitError
			if errors.As(err, &rateLimitErr) {
				delay = rateLimitErr.RetryAfter
				log.Warn().
					Dur("retry_after", rateLimitErr.RetryAfter).
					Msg("accrual rate limit exceeded")
			} else if !errors.Is(err, context.Canceled) {
				log.Error().Err(err).Msg("failed to process orders")
			}
		}

		timer.Reset(delay)
	}
}

// ProcessPendingOrders запрашивает начисления для заказов в обработке.
func (p *OrderProcessor) ProcessPendingOrders(ctx context.Context) error {
	orders, err := p.orders.FindForProcessing(ctx, p.limit)
	if err != nil {
		return err
	}

	for _, order := range orders {
		if err := p.processOrder(ctx, order); err != nil {
			return err
		}
	}

	return nil
}

func (p *OrderProcessor) processOrder(ctx context.Context, order model.Order) error {
	result, err := p.accrual.GetOrder(ctx, order.Number)
	if err != nil {
		if errors.Is(err, accrual.ErrOrderNotRegistered) {
			return nil
		}

		return err
	}

	switch result.Status {
	case accrual.StatusRegistered, accrual.StatusProcessing:
		return p.orders.UpdateStatus(ctx, order.ID, model.OrderStatusProcessing)
	case accrual.StatusInvalid:
		return p.orders.UpdateStatus(ctx, order.ID, model.OrderStatusInvalid)
	case accrual.StatusProcessed:
		return p.orders.ApplyAccrual(ctx, order.ID, result.Accrual)
	default:
		return accrual.ErrUnknownStatus
	}
}
