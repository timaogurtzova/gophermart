package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/repository"
	"github.com/timaogurtzova/gophermart/internal/service"
)

func TestBalanceAccountServiceGetBalance(t *testing.T) {
	wantBalance := model.Balance{
		UserID:    42,
		Current:   newTestPoints(t, "500.50"),
		Withdrawn: newTestPoints(t, "42.00"),
	}
	balances := &fakeBalanceRepository{
		getByUserID: func(_ context.Context, userID int64) (model.Balance, error) {
			assert.Equal(t, int64(42), userID)
			return wantBalance, nil
		},
	}
	svc := service.NewBalanceAccountService(balances)

	gotBalance, err := svc.GetBalance(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, wantBalance, gotBalance)
}

func TestBalanceAccountServiceWithdraw(t *testing.T) {
	tests := []struct {
		name        string
		orderNumber string
		sum         model.Points
		withdraw    func(ctx context.Context, userID int64, orderNumber string, sum model.Points) error
		wantErr     error
		wantCalled  bool
	}{
		{
			name:        "success",
			orderNumber: " 2377225624 ",
			sum:         newTestPoints(t, "751"),
			withdraw: func(_ context.Context, userID int64, orderNumber string, sum model.Points) error {
				assert.Equal(t, int64(42), userID)
				assert.Equal(t, "2377225624", orderNumber)
				assert.Equal(t, "751", sum.String())
				return nil
			},
			wantCalled: true,
		},
		{
			name:        "not digits",
			orderNumber: "237abc",
			sum:         newTestPoints(t, "751"),
			wantErr:     service.ErrInvalidOrderNumber,
		},
		{
			name:        "invalid luhn",
			orderNumber: "1234567890",
			sum:         newTestPoints(t, "751"),
			wantErr:     service.ErrInvalidOrderNumber,
		},
		{
			name:        "invalid sum",
			orderNumber: "2377225624",
			sum:         newTestPoints(t, "0"),
			wantErr:     service.ErrInvalidWithdrawalAmount,
		},
		{
			name:        "insufficient balance",
			orderNumber: "2377225624",
			sum:         newTestPoints(t, "751"),
			withdraw: func(_ context.Context, _ int64, _ string, _ model.Points) error {
				return repository.ErrInsufficientBalance
			},
			wantErr:    service.ErrInsufficientBalance,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			balances := &fakeBalanceRepository{
				withdraw: func(ctx context.Context, userID int64, orderNumber string, sum model.Points) error {
					called = true
					if tt.withdraw == nil {
						return nil
					}

					return tt.withdraw(ctx, userID, orderNumber, sum)
				},
			}
			svc := service.NewBalanceAccountService(balances)

			err := svc.Withdraw(context.Background(), 42, tt.orderNumber, tt.sum)
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.wantCalled, called)
		})
	}
}

func TestBalanceAccountServiceGetWithdrawals(t *testing.T) {
	wantWithdrawals := []model.Withdrawal{
		{
			ID:          1,
			UserID:      42,
			OrderNumber: "2377225624",
			Sum:         newTestPoints(t, "751"),
		},
	}
	balances := &fakeBalanceRepository{
		findWithdrawalsByUserID: func(_ context.Context, userID int64) ([]model.Withdrawal, error) {
			assert.Equal(t, int64(42), userID)
			return wantWithdrawals, nil
		},
	}
	svc := service.NewBalanceAccountService(balances)

	gotWithdrawals, err := svc.GetWithdrawals(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, wantWithdrawals, gotWithdrawals)
}

func newTestPoints(t *testing.T, value string) model.Points {
	t.Helper()

	points, err := model.NewPoints(value)
	require.NoError(t, err)
	return points
}

type fakeBalanceRepository struct {
	getByUserID             func(ctx context.Context, userID int64) (model.Balance, error)
	withdraw                func(ctx context.Context, userID int64, orderNumber string, sum model.Points) error
	findWithdrawalsByUserID func(ctx context.Context, userID int64) ([]model.Withdrawal, error)
}

func (r *fakeBalanceRepository) GetByUserID(ctx context.Context, userID int64) (model.Balance, error) {
	if r.getByUserID == nil {
		return model.Balance{}, nil
	}

	return r.getByUserID(ctx, userID)
}

func (r *fakeBalanceRepository) Withdraw(ctx context.Context, userID int64, orderNumber string, sum model.Points) error {
	if r.withdraw == nil {
		return nil
	}

	return r.withdraw(ctx, userID, orderNumber, sum)
}

func (r *fakeBalanceRepository) FindWithdrawalsByUserID(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	if r.findWithdrawalsByUserID == nil {
		return nil, nil
	}

	return r.findWithdrawalsByUserID(ctx, userID)
}
