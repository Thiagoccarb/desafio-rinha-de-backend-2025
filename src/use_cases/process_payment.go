package usecases

import (
	"encoding/json"
	"fmt"
	"payment-processor/config"
	"payment-processor/core/models"
	"payment-processor/infrastructure"
	"payment-processor/infrastructure/repositories"
	"strconv"
	"time"

	"golang.org/x/net/context"
)

type ProcessPaymentUseCase struct {
	Repo    repositories.PaymentRepository
	UseCase GetPaymentsSummaryUseCase
	Redis   infrastructure.Redis
}

func NewProcessPaymentUseCase(repo repositories.PaymentRepository, useCase GetPaymentsSummaryUseCase, redis infrastructure.Redis) *ProcessPaymentUseCase {
	return &ProcessPaymentUseCase{
		Repo:    repo,
		UseCase: useCase,
		Redis:   redis,
	}
}

func (p *ProcessPaymentUseCase) Execute(ctx context.Context) {

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("ProcessPaymentUseCase: Context canceled, stopping execution")
			return
		case t := <-ticker.C:
			p.processBatch(ctx, t)
		}
	}
}

func (p *ProcessPaymentUseCase) processBatch(ctx context.Context, t time.Time) {
	config := config.LoadConfig()
	var payments []models.Payment
	var (
		minScore time.Time
		maxScore time.Time
	)

	val, _ := p.Redis.Get(ctx, "score")
	if val == "" {
		minScore = time.Now().UTC().Add(-50 * time.Second) // Look back 10 seconds
	} else {
		tsFloat, _ := strconv.ParseFloat(val, 64)
		sec := int64(tsFloat)
		nsec := int64((tsFloat - float64(sec)) * 1e9)
		minScore = time.Unix(sec, nsec).UTC()
	}
	maxScore = t.UTC()

	data, err := p.Redis.ZRangeByScore(ctx, config.SetQueue, minScore, maxScore)
	if err != nil {
		fmt.Println("Error fetching from sorted set for batch creating payments in db:", err)
		return
	}

	for _, item := range data {
		var payment models.Payment
		err := json.Unmarshal([]byte(item), &payment)
		if err != nil {
			fmt.Println("Error unmarshaling payment from sorted set:", err)
			continue
		}
		payments = append(payments, payment)
	}

	if len(payments) > 0 {
		err = p.Repo.BatchCreatePayments(ctx, payments)
		if err != nil {
			fmt.Println("Error batch inserting payments into the database:", err)
			return
		}
	}
	maxScore = maxScore.Add(-1 * time.Second)

	err = p.Redis.Set(ctx, "score", fmt.Sprintf("%.6f", float64(maxScore.Unix())+float64(maxScore.Nanosecond())/1e9), 3600)
	if err != nil {
		fmt.Println("Error updating score in Redis:", err)
	}
}
