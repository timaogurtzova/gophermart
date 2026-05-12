package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/timaogurtzova/gophermart/internal/model"
)

const (
	defaultHTTPTimeout = 5 * time.Second
	defaultRetryAfter  = time.Minute
)

// Status описывает статус расчёта начисления во внешней системе.
type Status string

const (
	// StatusRegistered означает, что заказ зарегистрирован, но вознаграждение не рассчитано.
	StatusRegistered Status = "REGISTERED"
	// StatusInvalid означает, что заказ не принят к расчёту.
	StatusInvalid Status = "INVALID"
	// StatusProcessing означает, что расчёт начисления в процессе.
	StatusProcessing Status = "PROCESSING"
	// StatusProcessed означает, что расчёт начисления окончен.
	StatusProcessed Status = "PROCESSED"
)

var (
	// ErrOrderNotRegistered возвращается, когда заказ не зарегистрирован в системе расчёта.
	ErrOrderNotRegistered = errors.New("order is not registered in accrual system")
	// ErrUnknownStatus возвращается, когда система расчёта вернула неизвестный статус.
	ErrUnknownStatus = errors.New("unknown accrual status")
)

// RateLimitError описывает ограничение количества запросов к системе расчёта.
type RateLimitError struct {
	RetryAfter time.Duration
}

// Error возвращает текст ошибки ограничения запросов.
func (e *RateLimitError) Error() string {
	return "accrual rate limit exceeded"
}

// UnexpectedStatusError описывает неожиданный HTTP-статус системы расчёта.
type UnexpectedStatusError struct {
	StatusCode int
}

// Error возвращает текст ошибки неожиданного HTTP-статуса.
func (e *UnexpectedStatusError) Error() string {
	return fmt.Sprintf("unexpected accrual status code: %d", e.StatusCode)
}

// Order содержит результат расчёта начислений по заказу.
type Order struct {
	Number  string
	Status  Status
	Accrual *model.Points
}

// Client обращается к внешней системе расчёта начислений.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewClient создаёт HTTP-клиент внешней системы расчёта начислений.
func NewClient(address string) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimSpace(address))
	if err != nil {
		return nil, err
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("invalid accrual system address")
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}, nil
}

// GetOrder запрашивает информацию о расчёте начислений по номеру заказа.
func (c *Client) GetOrder(ctx context.Context, number string) (Order, error) {
	requestURL := c.orderURL(number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Order{}, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return Order{}, err
	}
	defer closeBody(res.Body)

	switch res.StatusCode {
	case http.StatusOK:
		return decodeOrder(res.Body)
	case http.StatusNoContent:
		return Order{}, ErrOrderNotRegistered
	case http.StatusTooManyRequests:
		return Order{}, &RateLimitError{RetryAfter: retryAfter(res.Header.Get("Retry-After"))}
	default:
		return Order{}, &UnexpectedStatusError{StatusCode: res.StatusCode}
	}
}

func (c *Client) orderURL(number string) string {
	requestURL := *c.baseURL
	basePath := strings.TrimRight(requestURL.Path, "/")
	requestURL.Path = basePath + "/api/orders/" + url.PathEscape(number)
	return requestURL.String()
}

type orderResponse struct {
	Order   string        `json:"order"`
	Status  Status        `json:"status"`
	Accrual *model.Points `json:"accrual,omitempty"`
}

func decodeOrder(body io.Reader) (Order, error) {
	var response orderResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return Order{}, err
	}

	if !isKnownStatus(response.Status) {
		return Order{}, ErrUnknownStatus
	}

	return Order{
		Number:  response.Order,
		Status:  response.Status,
		Accrual: response.Accrual,
	}, nil
}

func isKnownStatus(status Status) bool {
	switch status {
	case StatusRegistered, StatusInvalid, StatusProcessing, StatusProcessed:
		return true
	default:
		return false
	}
}

func retryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultRetryAfter
	}

	seconds, err := strconv.Atoi(value)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	retryAt, err := http.ParseTime(value)
	if err == nil {
		duration := time.Until(retryAt)
		if duration > 0 {
			return duration
		}
	}

	return defaultRetryAfter
}

func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close accrual response body")
	}
}
