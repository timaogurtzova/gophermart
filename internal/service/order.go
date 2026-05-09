package service

import (
	"context"
	"errors"
	"strings"

	"github.com/timaogurtzova/gophermart/internal/repository"
)

var (
	// ErrInvalidOrderFormat возвращается, когда номер заказа не является последовательностью цифр.
	ErrInvalidOrderFormat = errors.New("invalid order format")
	// ErrInvalidOrderNumber возвращается, когда номер заказа не проходит проверку алгоритмом Луна.
	ErrInvalidOrderNumber = errors.New("invalid order number")
	// ErrOrderAlreadyUploadedByUser возвращается, когда заказ уже загружен этим пользователем.
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by user")
	// ErrOrderAlreadyUploadedByAnotherUser возвращается, когда заказ уже загружен другим пользователем.
	ErrOrderAlreadyUploadedByAnotherUser = errors.New("order already uploaded by another user")
)

// OrderUploadService реализует загрузку номеров заказов.
type OrderUploadService struct {
	orders repository.OrderRepository
}

// NewOrderUploadService создаёт сервис загрузки номеров заказов.
func NewOrderUploadService(orders repository.OrderRepository) *OrderUploadService {
	return &OrderUploadService{orders: orders}
}

// UploadOrder проверяет номер заказа и сохраняет его для пользователя.
func (s *OrderUploadService) UploadOrder(ctx context.Context, userID int64, number string) error {
	number = strings.TrimSpace(number)
	if !isDigitsOnly(number) {
		return ErrInvalidOrderFormat
	}

	if !isValidLuhn(number) {
		return ErrInvalidOrderNumber
	}

	err := s.orders.Upload(ctx, userID, number)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrderAlreadyUploadedByUser):
			return ErrOrderAlreadyUploadedByUser
		case errors.Is(err, repository.ErrOrderAlreadyUploadedByAnotherUser):
			return ErrOrderAlreadyUploadedByAnotherUser
		default:
			return err
		}
	}

	return nil
}

func isDigitsOnly(value string) bool {
	if value == "" {
		return false
	}

	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
}

func isValidLuhn(number string) bool {
	sum := 0
	doubleDigit := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if doubleDigit {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		doubleDigit = !doubleDigit
	}

	return sum%10 == 0
}
