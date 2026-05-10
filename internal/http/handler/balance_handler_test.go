package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/auth"
	"github.com/timaogurtzova/gophermart/internal/http/handler"
	"github.com/timaogurtzova/gophermart/internal/model"
)

func TestBalanceHandlerGetBalance(t *testing.T) {
	tests := []struct {
		name       string
		withCookie bool
		getBalance func(ctx context.Context, userID int64) (model.Balance, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			withCookie: true,
			getBalance: func(_ context.Context, userID int64) (model.Balance, error) {
				assert.Equal(t, int64(42), userID)
				return model.Balance{
					UserID:    userID,
					Current:   newTestPoints(t, "500.50"),
					Withdrawn: newTestPoints(t, "42.00"),
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"current":500.5,"withdrawn":42}`,
		},
		{
			name:       "unauthorized",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "internal error",
			withCookie: true,
			getBalance: func(_ context.Context, _ int64) (model.Balance, error) {
				return model.Balance{}, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
			require.NoError(t, err)

			h := handler.NewBalanceHandler(&fakeBalanceService{getBalance: tt.getBalance}, authenticator)
			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			if tt.withCookie {
				cookie, err := authenticator.NewCookie(42)
				require.NoError(t, err)
				req.AddCookie(cookie)
			}
			recorder := httptest.NewRecorder()

			h.GetBalance(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			if tt.wantBody != "" {
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
				assert.JSONEq(t, tt.wantBody, recorder.Body.String())
			}
		})
	}
}

type fakeBalanceService struct {
	getBalance func(ctx context.Context, userID int64) (model.Balance, error)
}

func (s *fakeBalanceService) GetBalance(ctx context.Context, userID int64) (model.Balance, error) {
	if s.getBalance == nil {
		return model.Balance{}, nil
	}

	return s.getBalance(ctx, userID)
}
