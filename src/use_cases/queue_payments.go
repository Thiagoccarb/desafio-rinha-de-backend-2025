package usecases

import (
	"fmt"
	"payment-processor/core/models"
	"payment-processor/infrastructure"

	"golang.org/x/net/context"
)

type QueuePaymentsUseCase struct {
	Redis *infrastructure.Redis
}

func NewQueuePaymentsUseCase(redis *infrastructure.Redis) *QueuePaymentsUseCase {
	return &QueuePaymentsUseCase{
		Redis: redis,
	}
}

func (u *QueuePaymentsUseCase) EnqueuePayment(ctx context.Context, queueName string, paymentData models.Payment) error {
	var paymentMap = map[string]interface{}{
		"correlationId": paymentData.CorrelationID,
		"amount":        paymentData.Amount,
		"requestedAt":   paymentData.RequestedAt,
	}

	err := u.Redis.XAdd(ctx, queueName, paymentMap)
	if err != nil {
		fmt.Println("Error adding payment to queue:", err)
	}
	return nil
}
