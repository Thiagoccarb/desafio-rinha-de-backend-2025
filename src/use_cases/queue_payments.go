package usecases

import (
	"encoding/json"
	"fmt"
	"payment-processor/core/models"
	"payment-processor/infrastructure"

	"github.com/redis/go-redis/v9"
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

func (u *QueuePaymentsUseCase) StoreAsScore(ctx context.Context, queueName string, requestedAtFloat float64, paymentData models.Payment) error {
	var err error
	if paymentString, err := json.Marshal(paymentData); err == nil {
		err := u.Redis.ZAdd(ctx, queueName, redis.Z{Score: requestedAtFloat, Member: string(paymentString)})
		if err != nil {
			fmt.Println("Error adding payment to sorted set:", err)
		}
		return nil
	}
	return err

}
