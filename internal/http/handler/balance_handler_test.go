package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/auth"
	"github.com/timaogurtzova/gophermart/internal/http/handler"
	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/service"
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

func TestBalanceHandlerWithdraw(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		content    string
		withCookie bool
		withdraw   func(ctx context.Context, userID int64, orderNumber string, sum model.Points) error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"order":"2377225624","sum":751}`,
			content:    "application/json",
			withCookie: true,
			withdraw: func(_ context.Context, userID int64, orderNumber string, sum model.Points) error {
				assert.Equal(t, int64(42), userID)
				assert.Equal(t, "2377225624", orderNumber)
				assert.Equal(t, "751", sum.String())
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			body:       `{"order":"2377225624","sum":751}`,
			content:    "application/json",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bad content type",
			body:       `{"order":"2377225624","sum":751}`,
			content:    "text/plain",
			withCookie: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad json",
			body:       `broken`,
			content:    "application/json",
			withCookie: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid order number",
			body:       `{"order":"1234567890","sum":751}`,
			content:    "application/json",
			withCookie: true,
			withdraw: func(_ context.Context, _ int64, _ string, _ model.Points) error {
				return service.ErrInvalidOrderNumber
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "insufficient balance",
			body:       `{"order":"2377225624","sum":751}`,
			content:    "application/json",
			withCookie: true,
			withdraw: func(_ context.Context, _ int64, _ string, _ model.Points) error {
				return service.ErrInsufficientBalance
			},
			wantStatus: http.StatusPaymentRequired,
		},
		{
			name:       "invalid amount",
			body:       `{"order":"2377225624","sum":0}`,
			content:    "application/json",
			withCookie: true,
			withdraw: func(_ context.Context, _ int64, _ string, _ model.Points) error {
				return service.ErrInvalidWithdrawalAmount
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal error",
			body:       `{"order":"2377225624","sum":751}`,
			content:    "application/json",
			withCookie: true,
			withdraw: func(_ context.Context, _ int64, _ string, _ model.Points) error {
				return assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
			require.NoError(t, err)

			h := handler.NewBalanceHandler(&fakeBalanceService{withdraw: tt.withdraw}, authenticator)
			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.content)
			if tt.withCookie {
				cookie, err := authenticator.NewCookie(42)
				require.NoError(t, err)
				req.AddCookie(cookie)
			}
			recorder := httptest.NewRecorder()

			h.Withdraw(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func TestBalanceHandlerGetWithdrawals(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*60*60)

	tests := []struct {
		name           string
		withCookie     bool
		getWithdrawals func(ctx context.Context, userID int64) ([]model.Withdrawal, error)
		wantStatus     int
		wantBody       string
	}{
		{
			name:       "success",
			withCookie: true,
			getWithdrawals: func(_ context.Context, userID int64) ([]model.Withdrawal, error) {
				assert.Equal(t, int64(42), userID)
				return []model.Withdrawal{
					{
						OrderNumber: "2377225624",
						Sum:         newTestPoints(t, "500.00"),
						ProcessedAt: time.Date(2020, 12, 9, 16, 9, 57, 0, moscow),
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `[{"order":"2377225624","sum":500,"processed_at":"2020-12-09T16:09:57+03:00"}]`,
		},
		{
			name:       "no content",
			withCookie: true,
			getWithdrawals: func(_ context.Context, _ int64) ([]model.Withdrawal, error) {
				return nil, nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "unauthorized",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "internal error",
			withCookie: true,
			getWithdrawals: func(_ context.Context, _ int64) ([]model.Withdrawal, error) {
				return nil, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
			require.NoError(t, err)

			h := handler.NewBalanceHandler(&fakeBalanceService{getWithdrawals: tt.getWithdrawals}, authenticator)
			req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			if tt.withCookie {
				cookie, err := authenticator.NewCookie(42)
				require.NoError(t, err)
				req.AddCookie(cookie)
			}
			recorder := httptest.NewRecorder()

			h.GetWithdrawals(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			if tt.wantBody != "" {
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
				assert.JSONEq(t, tt.wantBody, recorder.Body.String())
			}
		})
	}
}

type fakeBalanceService struct {
	getBalance     func(ctx context.Context, userID int64) (model.Balance, error)
	withdraw       func(ctx context.Context, userID int64, orderNumber string, sum model.Points) error
	getWithdrawals func(ctx context.Context, userID int64) ([]model.Withdrawal, error)
}

func (s *fakeBalanceService) GetBalance(ctx context.Context, userID int64) (model.Balance, error) {
	if s.getBalance == nil {
		return model.Balance{}, nil
	}

	return s.getBalance(ctx, userID)
}

func (s *fakeBalanceService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum model.Points) error {
	if s.withdraw == nil {
		return nil
	}

	return s.withdraw(ctx, userID, orderNumber, sum)
}

func (s *fakeBalanceService) GetWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	if s.getWithdrawals == nil {
		return nil, nil
	}

	return s.getWithdrawals(ctx, userID)
}
