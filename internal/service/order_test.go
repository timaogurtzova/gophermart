package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/accrual"
	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/repository"
	"github.com/timaogurtzova/gophermart/internal/service"
)

func TestOrderUploadServiceUploadOrder(t *testing.T) {
	tests := []struct {
		name       string
		number     string
		upload     func(ctx context.Context, userID int64, number string) error
		wantErr    error
		wantCalled bool
	}{
		{
			name:   "success",
			number: " 12345678903 ",
			upload: func(_ context.Context, userID int64, number string) error {
				assert.Equal(t, int64(42), userID)
				assert.Equal(t, "12345678903", number)
				return nil
			},
			wantCalled: true,
		},
		{
			name:    "empty order number",
			number:  "",
			wantErr: service.ErrInvalidOrderFormat,
		},
		{
			name:    "not digits",
			number:  "12345abc",
			wantErr: service.ErrInvalidOrderFormat,
		},
		{
			name:    "unicode digits are rejected",
			number:  "１２３４",
			wantErr: service.ErrInvalidOrderFormat,
		},
		{
			name:    "invalid luhn",
			number:  "1234567890",
			wantErr: service.ErrInvalidOrderNumber,
		},
		{
			name:   "already uploaded by user",
			number: "12345678903",
			upload: func(_ context.Context, _ int64, _ string) error {
				return repository.ErrOrderAlreadyUploadedByUser
			},
			wantErr:    service.ErrOrderAlreadyUploadedByUser,
			wantCalled: true,
		},
		{
			name:   "already uploaded by another user",
			number: "12345678903",
			upload: func(_ context.Context, _ int64, _ string) error {
				return repository.ErrOrderAlreadyUploadedByAnotherUser
			},
			wantErr:    service.ErrOrderAlreadyUploadedByAnotherUser,
			wantCalled: true,
		},
		{
			name:   "repository error",
			number: "12345678903",
			upload: func(_ context.Context, _ int64, _ string) error {
				return errors.New("repository error")
			},
			wantErr:    errors.New("repository error"),
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			orders := &fakeOrderRepository{
				upload: func(ctx context.Context, userID int64, number string) error {
					called = true
					if tt.upload == nil {
						return nil
					}

					return tt.upload(ctx, userID, number)
				},
			}
			svc := service.NewOrderUploadService(orders)

			err := svc.UploadOrder(context.Background(), 42, tt.number)
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.wantCalled, called)
		})
	}
}

func TestOrderUploadServiceGetOrders(t *testing.T) {
	wantOrders := []model.Order{
		{
			ID:     1,
			UserID: 42,
			Number: "9278923470",
			Status: model.OrderStatusProcessed,
		},
	}
	orders := &fakeOrderRepository{
		findByUserID: func(_ context.Context, userID int64) ([]model.Order, error) {
			assert.Equal(t, int64(42), userID)
			return wantOrders, nil
		},
	}
	svc := service.NewOrderUploadService(orders)

	gotOrders, err := svc.GetOrders(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, wantOrders, gotOrders)
}

func TestOrderProcessorProcessPendingOrders(t *testing.T) {
	points := newOrderTestPoints(t, "500.50")

	tests := []struct {
		name         string
		result       accrual.Order
		accrualError error
		wantStatus   model.OrderStatus
		wantAccrual  *model.Points
		wantNoUpdate bool
		wantErr      error
	}{
		{
			name: "registered maps to processing",
			result: accrual.Order{
				Number: "9278923470",
				Status: accrual.StatusRegistered,
			},
			wantStatus: model.OrderStatusProcessing,
		},
		{
			name: "processing maps to processing",
			result: accrual.Order{
				Number: "9278923470",
				Status: accrual.StatusProcessing,
			},
			wantStatus: model.OrderStatusProcessing,
		},
		{
			name: "invalid maps to invalid",
			result: accrual.Order{
				Number: "9278923470",
				Status: accrual.StatusInvalid,
			},
			wantStatus: model.OrderStatusInvalid,
		},
		{
			name: "processed applies accrual",
			result: accrual.Order{
				Number:  "9278923470",
				Status:  accrual.StatusProcessed,
				Accrual: &points,
			},
			wantAccrual: &points,
		},
		{
			name:         "not registered keeps new status",
			accrualError: accrual.ErrOrderNotRegistered,
			wantNoUpdate: true,
		},
		{
			name:         "rate limit stops processing",
			accrualError: &accrual.RateLimitError{},
			wantNoUpdate: true,
			wantErr:      &accrual.RateLimitError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := &fakeOrderRepository{
				findForProcessing: func(_ context.Context, limit int) ([]model.Order, error) {
					assert.Equal(t, 10, limit)
					return []model.Order{
						{
							ID:     1,
							UserID: 42,
							Number: "9278923470",
							Status: model.OrderStatusNew,
						},
					}, nil
				},
				updateStatus: func(_ context.Context, orderID int64, status model.OrderStatus) error {
					assert.Equal(t, int64(1), orderID)
					assert.Equal(t, tt.wantStatus, status)
					return nil
				},
				applyAccrual: func(_ context.Context, orderID int64, accrual *model.Points) error {
					assert.Equal(t, int64(1), orderID)
					assert.Equal(t, tt.wantAccrual, accrual)
					return nil
				},
			}
			client := &fakeAccrualClient{
				getOrder: func(_ context.Context, number string) (accrual.Order, error) {
					assert.Equal(t, "9278923470", number)
					return tt.result, tt.accrualError
				},
			}
			processor := service.NewOrderProcessor(orders, client)

			err := processor.ProcessPendingOrders(context.Background())
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.IsType(t, tt.wantErr, err)
			}

			if tt.wantNoUpdate {
				assert.False(t, orders.updateStatusCalled)
				assert.False(t, orders.applyAccrualCalled)
			}
		})
	}
}

type fakeOrderRepository struct {
	upload             func(ctx context.Context, userID int64, number string) error
	findByUserID       func(ctx context.Context, userID int64) ([]model.Order, error)
	findForProcessing  func(ctx context.Context, limit int) ([]model.Order, error)
	updateStatus       func(ctx context.Context, orderID int64, status model.OrderStatus) error
	applyAccrual       func(ctx context.Context, orderID int64, accrual *model.Points) error
	updateStatusCalled bool
	applyAccrualCalled bool
}

func (r *fakeOrderRepository) Upload(ctx context.Context, userID int64, number string) error {
	if r.upload == nil {
		return nil
	}

	return r.upload(ctx, userID, number)
}

func (r *fakeOrderRepository) FindByUserID(ctx context.Context, userID int64) ([]model.Order, error) {
	if r.findByUserID == nil {
		return nil, nil
	}

	return r.findByUserID(ctx, userID)
}

func (r *fakeOrderRepository) FindForProcessing(ctx context.Context, limit int) ([]model.Order, error) {
	if r.findForProcessing == nil {
		return nil, nil
	}

	return r.findForProcessing(ctx, limit)
}

func (r *fakeOrderRepository) UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus) error {
	r.updateStatusCalled = true
	if r.updateStatus == nil {
		return nil
	}

	return r.updateStatus(ctx, orderID, status)
}

func (r *fakeOrderRepository) ApplyAccrual(ctx context.Context, orderID int64, accrual *model.Points) error {
	r.applyAccrualCalled = true
	if r.applyAccrual == nil {
		return nil
	}

	return r.applyAccrual(ctx, orderID, accrual)
}

type fakeAccrualClient struct {
	getOrder func(ctx context.Context, number string) (accrual.Order, error)
}

func (c *fakeAccrualClient) GetOrder(ctx context.Context, number string) (accrual.Order, error) {
	if c.getOrder == nil {
		return accrual.Order{}, nil
	}

	return c.getOrder(ctx, number)
}

func newOrderTestPoints(t *testing.T, value string) model.Points {
	t.Helper()

	points, err := model.NewPoints(value)
	require.NoError(t, err)
	return points
}
