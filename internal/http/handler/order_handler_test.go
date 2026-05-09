package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/auth"
	"github.com/timaogurtzova/gophermart/internal/http/handler"
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

type fakeOrderService struct {
	uploadOrder func(ctx context.Context, userID int64, number string) error
}

func (s *fakeOrderService) UploadOrder(ctx context.Context, userID int64, number string) error {
	if s.uploadOrder == nil {
		return nil
	}

	return s.uploadOrder(ctx, userID, number)
}
