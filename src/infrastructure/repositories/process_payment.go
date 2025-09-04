package repositories

import (
	"context"
	"fmt"
	"payment-processor/core/models"
	"payment-processor/interfaces"
	"strings"
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

func (r *PaymentRepository) BatchCreatePayments(ctx context.Context, payments []models.Payment) error {
	if len(payments) == 0 {
		return nil
	}

	query := `INSERT INTO rinha (uuid, amount, type, created_at) VALUES `
	values := []interface{}{}
	placeholders := []string{}

	for i, payment := range payments {
		var Type int
		if payment.Type == "default" {
			Type = 1
		} else {
			Type = 2
		}

		createdAt, err := time.Parse(time.RFC3339, payment.RequestedAt)
		if err != nil {
			createdAt = time.Now().UTC()
		}

		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4))
		values = append(values, payment.CorrelationID, payment.Amount, Type, createdAt)
	}

	query += strings.Join(placeholders, ", ")
	query += ` ON CONFLICT (uuid) DO NOTHING`

	_, err := r.conn.Execute(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to batch insert payments: %w", err)
	}

	return nil
}
