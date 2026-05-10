package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/service"
)

func TestBalanceReadServiceGetBalance(t *testing.T) {
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
	svc := service.NewBalanceReadService(balances)

	gotBalance, err := svc.GetBalance(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, wantBalance, gotBalance)
}

func newTestPoints(t *testing.T, value string) model.Points {
	t.Helper()

	points, err := model.NewPoints(value)
	require.NoError(t, err)
	return points
}

type fakeBalanceRepository struct {
	getByUserID func(ctx context.Context, userID int64) (model.Balance, error)
}

func (r *fakeBalanceRepository) GetByUserID(ctx context.Context, userID int64) (model.Balance, error) {
	if r.getByUserID == nil {
		return model.Balance{}, nil
	}

	return r.getByUserID(ctx, userID)
}
