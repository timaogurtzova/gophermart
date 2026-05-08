package httpserver_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func TestLoggingMiddlewareLogsRequestAndResponseData(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.InfoLevel)
	t.Cleanup(func() {
		log.Logger = oldLogger
	})

	router := httpserver.NewRouter(httpserver.RouterHandlers{
		Register: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("register"))
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/register?trace=1", strings.NewReader("body"))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"level":"info"`)
	assert.Contains(t, logOutput, `"uri":"/api/user/register?trace=1"`)
	assert.Contains(t, logOutput, `"method":"POST"`)
	assert.Contains(t, logOutput, `"duration":"`)
	assert.Contains(t, logOutput, `"status":200`)
	assert.Contains(t, logOutput, `"size":8`)
	assert.Contains(t, logOutput, `"message":"HTTP request completed"`)
}

func TestLoggingMiddlewareLogsImplicitStatusCode(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.InfoLevel)
	t.Cleanup(func() {
		log.Logger = oldLogger
	})

	router := httpserver.NewRouter(httpserver.RouterHandlers{
		Register: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("register"))
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader("body"))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, buf.String(), `"status":200`)
}

func TestGzipRequestMiddlewareDecompressesRequestBody(t *testing.T) {
	var requestBody string
	router := httpserver.NewRouter(httpserver.RouterHandlers{
		UploadOrder: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			requestBody = string(body)

			w.WriteHeader(http.StatusAccepted)
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(gzipData(t, "12345678903")))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusAccepted, recorder.Code)
	assert.Equal(t, "12345678903", requestBody)
}

func TestGzipRequestMiddlewareReturnsBadRequestForUnsupportedEncoding(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouterHandlers{
		UploadOrder: namedHandler("upload-order", http.StatusAccepted, new(string)),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "deflate")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "bad request", strings.TrimSpace(recorder.Body.String()))
}

func TestGzipRequestMiddlewareReturnsBadRequestForBrokenGzip(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouterHandlers{
		UploadOrder: namedHandler("upload-order", http.StatusAccepted, new(string)),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("not-a-gzip-stream"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "bad request", strings.TrimSpace(recorder.Body.String()))
}

func TestGzipRequestMiddlewareReturnsBadRequestForMultipleEncodings(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouterHandlers{
		UploadOrder: namedHandler("upload-order", http.StatusAccepted, new(string)),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "gzip, deflate")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "bad request", strings.TrimSpace(recorder.Body.String()))
}

func TestGzipResponseMiddlewareCompressesJSONResponse(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouterHandlers{
		GetBalance: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"current":500.5,"withdrawn":42}`))
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	res := recorder.Result()
	defer res.Body.Close()

	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	assert.Contains(t, res.Header.Values("Vary"), "Accept-Encoding")
	assert.Equal(t, `{"current":500.5,"withdrawn":42}`, ungzipBody(t, res.Body))
}

func TestGzipResponseMiddlewareSkipsUnsupportedContentType(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouterHandlers{
		UploadOrder: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("accepted"))
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	res := recorder.Result()
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	assert.NoError(t, err)

	assert.Empty(t, res.Header.Get("Content-Encoding"))
	assert.Equal(t, "accepted", string(body))
}

func namedHandler(name string, status int, called *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*called = name
		w.WriteHeader(status)
		_, _ = w.Write([]byte(name))
	}
}

func gzipData(t *testing.T, data string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write([]byte(data))
	assert.NoError(t, err)
	assert.NoError(t, writer.Close())

	return buf.Bytes()
}

func ungzipBody(t *testing.T, body io.Reader) string {
	t.Helper()

	reader, err := gzip.NewReader(body)
	assert.NoError(t, err)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	assert.NoError(t, err)

	return string(data)
}
