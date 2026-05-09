package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type fakeOrderRepository struct {
	upload func(ctx context.Context, userID int64, number string) error
}

func (r *fakeOrderRepository) Upload(ctx context.Context, userID int64, number string) error {
	if r.upload == nil {
		return nil
	}

	return r.upload(ctx, userID, number)
}
