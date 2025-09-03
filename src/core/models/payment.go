package models

type PaymentsSummary struct {
	TotalRequests int     `json:"totalRequests" required:"true"`
	TotalAmount   float64 `json:"totalAmount" required:"true"`
	Type          string  `json:"type" required:"true"`
}

type Payment struct {
	CorrelationID string  `json:"correlationId" required:"true"`
	Amount        float64 `json:"amount" required:"true"`
	RequestedAt   string  `json:"requestedAt" required:"true"`
	Type          string  `json:"type" required:"false"`
}
