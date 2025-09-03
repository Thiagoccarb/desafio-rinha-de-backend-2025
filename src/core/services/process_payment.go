package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"payment-processor/config"
	"payment-processor/core/models"
	usecases "payment-processor/use_cases"
	"time"
)

type ProcessPaymentService struct {
	QueueUseCase          *usecases.QueuePaymentsUseCase
	ProcessPaymentUseCase *usecases.ProcessPaymentUseCase
}

func NewProcessPaymentService(
	queueUseCase *usecases.QueuePaymentsUseCase,
) *ProcessPaymentService {
	return &ProcessPaymentService{
		QueueUseCase: queueUseCase,
	}
}

func (ps *ProcessPaymentService) ProcessPayment(
	paymentProcessorType string,
	payload models.Payment,
	ctx context.Context,
) bool {
	config := config.LoadConfig()
	var url string
	switch paymentProcessorType {
	case "default":
		url = config.Services.DefaultProcessPaymentURL
		payload.Type = "default"
	case "fallback":
		url = config.Services.FallbackProcessPaymentURL
		payload.Type = "fallback"
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("failed to marshal payment payload: %v", err)
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("failed to create HTTP request: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 1 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: false,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("HTTP request failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Payment processing failed with status: %s and correlationId: %s", resp.Status, payload.CorrelationID)
		return false
	}
	return true
}
