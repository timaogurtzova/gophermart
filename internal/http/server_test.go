package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	httpserver "github.com/timaogurtzova/gophermart/internal/http"
)

func TestServerRouting(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantBody   string
		wantCode   int
		wantCalled string
	}{
		{
			name:       "POST /api/user/register -> register handler ok",
			method:     http.MethodPost,
			path:       "/api/user/register",
			wantBody:   "register",
			wantCode:   http.StatusOK,
			wantCalled: "register",
		},
		{
			name:       "POST /api/user/login -> login handler ok",
			method:     http.MethodPost,
			path:       "/api/user/login",
			wantBody:   "login",
			wantCode:   http.StatusOK,
			wantCalled: "login",
		},
		{
			name:       "POST /api/user/orders -> upload order handler ok",
			method:     http.MethodPost,
			path:       "/api/user/orders",
			wantBody:   "upload-order",
			wantCode:   http.StatusAccepted,
			wantCalled: "upload-order",
		},
		{
			name:       "GET /api/user/orders -> get orders handler ok",
			method:     http.MethodGet,
			path:       "/api/user/orders",
			wantBody:   "get-orders",
			wantCode:   http.StatusOK,
			wantCalled: "get-orders",
		},
		{
			name:       "GET /api/user/balance -> get balance handler ok",
			method:     http.MethodGet,
			path:       "/api/user/balance",
			wantBody:   "get-balance",
			wantCode:   http.StatusOK,
			wantCalled: "get-balance",
		},
		{
			name:       "POST /api/user/balance/withdraw -> withdraw handler ok",
			method:     http.MethodPost,
			path:       "/api/user/balance/withdraw",
			wantBody:   "withdraw",
			wantCode:   http.StatusOK,
			wantCalled: "withdraw",
		},
		{
			name:       "GET /api/user/withdrawals -> get withdrawals handler ok",
			method:     http.MethodGet,
			path:       "/api/user/withdrawals",
			wantBody:   "get-withdrawals",
			wantCode:   http.StatusOK,
			wantCalled: "get-withdrawals",
		},
		{
			name:     "GET /api/user/register -> method not allowed",
			method:   http.MethodGet,
			path:     "/api/user/register",
			wantBody: "method not allowed",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "GET /unknown -> not found",
			method:   http.MethodGet,
			path:     "/unknown",
			wantBody: "not found",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := ""
			router := httpserver.NewRouter(httpserver.RouterHandlers{
				Register:       namedHandler("register", http.StatusOK, &called),
				Login:          namedHandler("login", http.StatusOK, &called),
				UploadOrder:    namedHandler("upload-order", http.StatusAccepted, &called),
				GetOrders:      namedHandler("get-orders", http.StatusOK, &called),
				GetBalance:     namedHandler("get-balance", http.StatusOK, &called),
				Withdraw:       namedHandler("withdraw", http.StatusOK, &called),
				GetWithdrawals: namedHandler("get-withdrawals", http.StatusOK, &called),
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			assert.Equal(t, tt.wantCode, recorder.Code)
			assert.Equal(t, tt.wantBody, strings.TrimSpace(recorder.Body.String()))
			assert.Equal(t, tt.wantCalled, called)
		})
	}
}

func TestServerRoutingUsesNotImplementedFallbackForEmptyHandlers(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouterHandlers{})
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotImplemented, recorder.Code)
	assert.Equal(t, "not implemented", strings.TrimSpace(recorder.Body.String()))
}

func namedHandler(name string, status int, called *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*called = name
		w.WriteHeader(status)
		_, _ = w.Write([]byte(name))
	}
}
