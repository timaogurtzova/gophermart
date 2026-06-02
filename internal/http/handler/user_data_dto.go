package handler

import (
	"strings"
	"time"

	"github.com/timaogurtzova/gophermart/internal/model"
)

type withdrawRequest struct {
	Order string       `json:"order"`
	Sum   model.Points `json:"sum"`
}

// DTO используют Points напрямую: тип сериализуется в JSON-число без float64.
type orderResponse struct {
	Number     string            `json:"number"`
	Status     model.OrderStatus `json:"status"`
	Accrual    *model.Points     `json:"accrual,omitempty"`
	UploadedAt string            `json:"uploaded_at"`
}

type balanceResponse struct {
	Current   model.Points `json:"current"`
	Withdrawn model.Points `json:"withdrawn"`
}

type withdrawalResponse struct {
	Order       string       `json:"order"`
	Sum         model.Points `json:"sum"`
	ProcessedAt string       `json:"processed_at"`
}

func (r *withdrawRequest) normalize() {
	r.Order = strings.TrimSpace(r.Order)
}

func newOrderResponse(order model.Order) orderResponse {
	response := orderResponse{
		Number:     order.Number,
		Status:     order.Status,
		UploadedAt: order.UploadedAt.Format(time.RFC3339),
	}

	if order.Accrual != nil {
		response.Accrual = order.Accrual
	}

	return response
}

func newBalanceResponse(balance model.Balance) balanceResponse {
	return balanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}
}

func newWithdrawalResponse(withdrawal model.Withdrawal) withdrawalResponse {
	return withdrawalResponse{
		Order:       withdrawal.OrderNumber,
		Sum:         withdrawal.Sum,
		ProcessedAt: withdrawal.ProcessedAt.Format(time.RFC3339),
	}
}
