package usecases

import (
	"payment-processor/core/models"
	"payment-processor/infrastructure/repositories"
	"time"

	"golang.org/x/net/context"
)

type GetPaymentsSummaryUseCase struct {
	Repo *repositories.PaymentRepository
}

func NewGetPaymentsSummaryUseCase(repo *repositories.PaymentRepository) *GetPaymentsSummaryUseCase {
	return &GetPaymentsSummaryUseCase{
		Repo: repo,
	}
}

func (g *GetPaymentsSummaryUseCase) Execute(ctx context.Context, from, to time.Time) ([]models.PaymentsSummary, error) {
	return g.Repo.GetPaymentSummary(ctx, from, to)
}
