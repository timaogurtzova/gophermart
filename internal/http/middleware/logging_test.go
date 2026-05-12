package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/http/middleware"
)

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.InfoLevel)
	t.Cleanup(func() {
		log.Logger = oldLogger
	})

	handler := middleware.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte("accepted"))
		require.NoError(t, err)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders?trace=1", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"uri":"/api/user/orders?trace=1"`)
	assert.Contains(t, logOutput, `"method":"POST"`)
	assert.Contains(t, logOutput, `"status":202`)
	assert.Contains(t, logOutput, `"size":8`)
	assert.Contains(t, logOutput, `"message":"HTTP request completed"`)
}

func TestLoggingImplicitStatus(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.InfoLevel)
	t.Cleanup(func() {
		log.Logger = oldLogger
	})

	handler := middleware.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	assert.Contains(t, buf.String(), `"status":200`)
}
