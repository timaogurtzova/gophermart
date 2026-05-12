package accrual_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/accrual"
)

func TestClientGetOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/orders/9278923470", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"order":"9278923470","status":"PROCESSED","accrual":500.5}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client, err := accrual.NewClient(server.URL)
	require.NoError(t, err)

	order, err := client.GetOrder(context.Background(), "9278923470")
	require.NoError(t, err)

	assert.Equal(t, "9278923470", order.Number)
	assert.Equal(t, accrual.StatusProcessed, order.Status)
	require.NotNil(t, order.Accrual)
	assert.Equal(t, "500.5", order.Accrual.String())
}

func TestClientGetOrderNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := accrual.NewClient(server.URL)
	require.NoError(t, err)

	_, err = client.GetOrder(context.Background(), "12345678903")

	assert.ErrorIs(t, err, accrual.ErrOrderNotRegistered)
}

func TestClientGetOrderRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := accrual.NewClient(server.URL)
	require.NoError(t, err)

	_, err = client.GetOrder(context.Background(), "12345678903")
	require.Error(t, err)

	var rateLimitErr *accrual.RateLimitError
	require.True(t, errors.As(err, &rateLimitErr))
	assert.Equal(t, time.Minute, rateLimitErr.RetryAfter)
}

func TestClientGetOrderUnknownStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"order":"9278923470","status":"UNKNOWN"}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client, err := accrual.NewClient(server.URL)
	require.NoError(t, err)

	_, err = client.GetOrder(context.Background(), "9278923470")

	assert.ErrorIs(t, err, accrual.ErrUnknownStatus)
}

func TestNewClientInvalidAddress(t *testing.T) {
	client, err := accrual.NewClient("localhost:8081")

	require.Error(t, err)
	assert.Nil(t, client)
}
