package usecases

import (
	"encoding/json"
	"payment-processor/config"
	"payment-processor/core/models"
	"payment-processor/infrastructure"
	"time"

	"golang.org/x/net/context"
)

type GetPaymentsSummaryUseCase struct {
	Redis *infrastructure.Redis
}

type PaymentsSummary struct {
	Default  *SummaryItem `json:"default"`
	Fallback *SummaryItem `json:"fallback"`
}

type SummaryItem struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

func NewGetPaymentsSummaryUseCase(redis *infrastructure.Redis) *GetPaymentsSummaryUseCase {
	return &GetPaymentsSummaryUseCase{
		Redis: redis,
	}
}

func (g *GetPaymentsSummaryUseCase) Execute(ctx context.Context, from, to time.Time) (*PaymentsSummary, error) {
	config := config.LoadConfig()
	summary := &PaymentsSummary{
		Default:  &SummaryItem{TotalRequests: 0, TotalAmount: 0},
		Fallback: &SummaryItem{TotalRequests: 0, TotalAmount: 0},
	}
	data, err := g.Redis.ZRangeByScore(ctx, config.SetQueue, from, to)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return summary, nil
	}

	for _, item := range data {
		var payment models.Payment
		json.Unmarshal([]byte(item), &payment)
		if payment.RequestedAt < from.Format(time.RFC3339) || payment.RequestedAt > to.Format(time.RFC3339) {
			continue
		}
		switch payment.Type {
		case "default":
			summary.Default.TotalRequests += 1
			summary.Default.TotalAmount += payment.Amount
		case "fallback":
			summary.Fallback.TotalRequests += 1
			summary.Fallback.TotalAmount += payment.Amount
		}
	}
	return summary, nil
}
