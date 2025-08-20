package repositories

import (
	"context"
	"payment-processor/core/models"
	"payment-processor/interfaces"
	"time"
)

type PaymentRepository struct {
	conn interfaces.DatabaseConnection
}

func NewPaymentRepository(conn interfaces.DatabaseConnection) *PaymentRepository {
	return &PaymentRepository{
		conn: conn,
	}
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, data models.Payment) error {
	var Type int
	if data.Type == "default" {
		Type = 1
	} else {
		Type = 2
	}
	query := `INSERT INTO rinha (uuid, amount, type, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.conn.Execute(ctx, query, data.CorrelationID, data.Amount, Type, data.RequestedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *PaymentRepository) GetPaymentSummary(ctx context.Context, from, to time.Time) ([]models.PaymentsSummary, error) {
	query := `
		SELECT type, COUNT(*) as total_requests, COALESCE(SUM(amount), 0) as total_amount
		FROM rinha
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY type;
	`
	rows, err := r.conn.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}

	var summaries []models.PaymentsSummary

	for rows.Next() {
		var summary models.PaymentsSummary
		err := rows.Scan(&summary.Type, &summary.TotalRequests, &summary.TotalAmount)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return summaries, nil

}
