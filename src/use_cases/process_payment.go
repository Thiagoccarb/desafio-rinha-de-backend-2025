package usecases

import (
	"fmt"
	"payment-processor/core/models"
	"payment-processor/infrastructure/repositories"

	"golang.org/x/net/context"
)

type ProcessPaymentUseCase struct {
	Repo repositories.PaymentRepository
}

func NewProcessPaymentUseCase(repo repositories.PaymentRepository) *ProcessPaymentUseCase {
	return &ProcessPaymentUseCase{
		Repo: repo,
	}
}

func (p *ProcessPaymentUseCase) Execute(ctx context.Context, data models.Payment) bool {
	err := p.Repo.CreatePayment(ctx, data)
	if err != nil {
		fmt.Printf("Error creating payment: %v", err)
		return false
	}
	return true
}
