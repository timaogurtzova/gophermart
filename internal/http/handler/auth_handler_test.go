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
	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/service"
)

func TestAuthHandlerRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		content    string
		register   func(ctx context.Context, login, password string) (model.User, error)
		wantStatus int
		wantCookie bool
	}{
		{
			name:    "success",
			body:    `{"login":"user","password":"password"}`,
			content: "application/json",
			register: func(_ context.Context, login, password string) (model.User, error) {
				assert.Equal(t, "user", login)
				assert.Equal(t, "password", password)
				return model.User{ID: 42, Login: login}, nil
			},
			wantStatus: http.StatusOK,
			wantCookie: true,
		},
		{
			name:       "bad content type",
			body:       `{"login":"user","password":"password"}`,
			content:    "text/plain",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad json",
			body:       `broken`,
			content:    "application/json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty login",
			body:       `{"login":"","password":"password"}`,
			content:    "application/json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "login already taken",
			body:    `{"login":"user","password":"password"}`,
			content: "application/json",
			register: func(_ context.Context, _, _ string) (model.User, error) {
				return model.User{}, service.ErrLoginAlreadyTaken
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:    "internal error",
			body:    `{"login":"user","password":"password"}`,
			content: "application/json",
			register: func(_ context.Context, _, _ string) (model.User, error) {
				return model.User{}, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestAuthHandler(t, &fakeUserService{register: tt.register})
			req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.content)
			recorder := httptest.NewRecorder()

			h.Register(recorder, req)

			res := recorder.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantCookie, len(res.Cookies()) > 0)
		})
	}
}

func TestAuthHandlerLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		login      func(ctx context.Context, login, password string) (model.User, error)
		wantStatus int
		wantCookie bool
	}{
		{
			name: "success",
			body: `{"login":"user","password":"password"}`,
			login: func(_ context.Context, login, password string) (model.User, error) {
				assert.Equal(t, "user", login)
				assert.Equal(t, "password", password)
				return model.User{ID: 42, Login: login}, nil
			},
			wantStatus: http.StatusOK,
			wantCookie: true,
		},
		{
			name: "invalid credentials",
			body: `{"login":"user","password":"wrong"}`,
			login: func(_ context.Context, _, _ string) (model.User, error) {
				return model.User{}, service.ErrInvalidCredentials
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "internal error",
			body: `{"login":"user","password":"password"}`,
			login: func(_ context.Context, _, _ string) (model.User, error) {
				return model.User{}, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestAuthHandler(t, &fakeUserService{login: tt.login})
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			recorder := httptest.NewRecorder()

			h.Login(recorder, req)

			res := recorder.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantCookie, len(res.Cookies()) > 0)
		})
	}
}

func newTestAuthHandler(t *testing.T, userService service.UserService) *handler.AuthHandler {
	t.Helper()

	authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)

	return handler.NewAuthHandler(userService, authenticator)
}

func newTestPoints(t *testing.T, value string) model.Points {
	t.Helper()

	points, err := model.NewPoints(value)
	require.NoError(t, err)
	return points
}

type fakeUserService struct {
	register func(ctx context.Context, login, password string) (model.User, error)
	login    func(ctx context.Context, login, password string) (model.User, error)
}

func (s *fakeUserService) Register(ctx context.Context, login, password string) (model.User, error) {
	if s.register == nil {
		return model.User{}, nil
	}

	return s.register(ctx, login, password)
}

func (s *fakeUserService) Login(ctx context.Context, login, password string) (model.User, error) {
	if s.login == nil {
		return model.User{}, nil
	}

	return s.login(ctx, login, password)
}
