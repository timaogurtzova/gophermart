package service

import (
	"context"

	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/repository"
)

// BalanceReadService реализует получение накопительного счёта пользователя.
type BalanceReadService struct {
	balances repository.BalanceRepository
}

// NewBalanceReadService создаёт сервис получения накопительного счёта.
func NewBalanceReadService(balances repository.BalanceRepository) *BalanceReadService {
	return &BalanceReadService{balances: balances}
}

// GetBalance возвращает накопительный счёт пользователя.
func (s *BalanceReadService) GetBalance(ctx context.Context, userID int64) (model.Balance, error) {
	return s.balances.GetByUserID(ctx, userID)
}
