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

func TestOrderHandlerUpload(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		content    string
		withCookie bool
		upload     func(ctx context.Context, userID int64, number string) error
		wantStatus int
	}{
		{
			name:       "success",
			body:       "12345678903",
			content:    "text/plain",
			withCookie: true,
			upload: func(_ context.Context, userID int64, number string) error {
				assert.Equal(t, int64(42), userID)
				assert.Equal(t, "12345678903", number)
				return nil
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "already uploaded by this user",
			body:       "12345678903",
			content:    "text/plain",
			withCookie: true,
			upload: func(_ context.Context, _ int64, _ string) error {
				return service.ErrOrderAlreadyUploadedByUser
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			body:       "12345678903",
			content:    "text/plain",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bad content type",
			body:       "12345678903",
			content:    "application/json",
			withCookie: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad format",
			body:       "not-digits",
			content:    "text/plain",
			withCookie: true,
			upload: func(_ context.Context, _ int64, _ string) error {
				return service.ErrInvalidOrderFormat
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid order number",
			body:       "1234567890",
			content:    "text/plain",
			withCookie: true,
			upload: func(_ context.Context, _ int64, _ string) error {
				return service.ErrInvalidOrderNumber
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "already uploaded by another user",
			body:       "12345678903",
			content:    "text/plain",
			withCookie: true,
			upload: func(_ context.Context, _ int64, _ string) error {
				return service.ErrOrderAlreadyUploadedByAnotherUser
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "internal error",
			body:       "12345678903",
			content:    "text/plain",
			withCookie: true,
			upload: func(_ context.Context, _ int64, _ string) error {
				return assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
			require.NoError(t, err)

			h := handler.NewOrderHandler(&fakeOrderService{uploadOrder: tt.upload}, authenticator)
			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.content)
			if tt.withCookie {
				cookie, err := authenticator.NewCookie(42)
				require.NoError(t, err)
				req.AddCookie(cookie)
			}
			recorder := httptest.NewRecorder()

			h.Upload(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func TestOrderHandlerGetOrders(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*60*60)
	accrual := newTestPoints(t, "500.50")

	tests := []struct {
		name        string
		withCookie  bool
		getOrders   func(ctx context.Context, userID int64) ([]model.Order, error)
		wantStatus  int
		wantContent string
		wantBody    string
	}{
		{
			name:       "success",
			withCookie: true,
			getOrders: func(_ context.Context, userID int64) ([]model.Order, error) {
				assert.Equal(t, int64(42), userID)
				return []model.Order{
					{
						Number:     "9278923470",
						Status:     model.OrderStatusProcessed,
						Accrual:    &accrual,
						UploadedAt: time.Date(2020, 12, 10, 15, 15, 45, 0, moscow),
					},
					{
						Number:     "12345678903",
						Status:     model.OrderStatusProcessing,
						UploadedAt: time.Date(2020, 12, 10, 15, 12, 1, 0, moscow),
					},
				}, nil
			},
			wantStatus:  http.StatusOK,
			wantContent: "application/json",
			wantBody: `[
				{"number":"9278923470","status":"PROCESSED","accrual":500.5,"uploaded_at":"2020-12-10T15:15:45+03:00"},
				{"number":"12345678903","status":"PROCESSING","uploaded_at":"2020-12-10T15:12:01+03:00"}
			]`,
		},
		{
			name:       "no content",
			withCookie: true,
			getOrders: func(_ context.Context, _ int64) ([]model.Order, error) {
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
			getOrders: func(_ context.Context, _ int64) ([]model.Order, error) {
				return nil, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
			require.NoError(t, err)

			h := handler.NewOrderHandler(&fakeOrderService{getOrders: tt.getOrders}, authenticator)
			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			if tt.withCookie {
				cookie, err := authenticator.NewCookie(42)
				require.NoError(t, err)
				req.AddCookie(cookie)
			}
			recorder := httptest.NewRecorder()

			h.GetOrders(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			if tt.wantContent != "" {
				assert.Equal(t, tt.wantContent, recorder.Header().Get("Content-Type"))
			}
			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, recorder.Body.String())
			}
		})
	}
}

type fakeOrderService struct {
	uploadOrder func(ctx context.Context, userID int64, number string) error
	getOrders   func(ctx context.Context, userID int64) ([]model.Order, error)
}

func (s *fakeOrderService) UploadOrder(ctx context.Context, userID int64, number string) error {
	if s.uploadOrder == nil {
		return nil
	}

	return s.uploadOrder(ctx, userID, number)
}

func (s *fakeOrderService) GetOrders(ctx context.Context, userID int64) ([]model.Order, error) {
	if s.getOrders == nil {
		return nil, nil
	}

	return s.getOrders(ctx, userID)
}
