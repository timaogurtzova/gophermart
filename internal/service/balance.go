package service

import (
	"context"
	"errors"
	"strings"

	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/repository"
)

var (
	// ErrInsufficientBalance возвращается, когда баллов недостаточно для списания.
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrInvalidWithdrawalAmount возвращается, когда сумма списания не больше нуля.
	ErrInvalidWithdrawalAmount = errors.New("invalid withdrawal amount")
)

// BalanceAccountService реализует получение баланса и списание баллов.
type BalanceAccountService struct {
	balances repository.BalanceRepository
}

// NewBalanceAccountService создаёт сервис накопительного счёта.
func NewBalanceAccountService(balances repository.BalanceRepository) *BalanceAccountService {
	return &BalanceAccountService{balances: balances}
}

// GetBalance возвращает накопительный счёт пользователя.
func (s *BalanceAccountService) GetBalance(ctx context.Context, userID int64) (model.Balance, error) {
	return s.balances.GetByUserID(ctx, userID)
}

// Withdraw проверяет номер заказа и списывает баллы с накопительного счёта.
func (s *BalanceAccountService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum model.Points) error {
	orderNumber = strings.TrimSpace(orderNumber)
	if !isDigitsOnly(orderNumber) || !isValidLuhn(orderNumber) {
		return ErrInvalidOrderNumber
	}

	if !sum.IsPositive() {
		return ErrInvalidWithdrawalAmount
	}

	if err := s.balances.Withdraw(ctx, userID, orderNumber, sum); err != nil {
		if errors.Is(err, repository.ErrInsufficientBalance) {
			return ErrInsufficientBalance
		}

		return err
	}

	return nil
}

// GetWithdrawals возвращает историю списаний пользователя от новых к старым.
func (s *BalanceAccountService) GetWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	return s.balances.FindWithdrawalsByUserID(ctx, userID)
}
