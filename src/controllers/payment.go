package controllers

import (
	"log"
	"net/http"
	"payment-processor/core/models"
	usecases "payment-processor/use_cases"
	"time"

	"github.com/gin-gonic/gin"
)

type PaymentController struct {
	EnqueuePaymentuseCase     *usecases.QueuePaymentsUseCase
	GetPaymentsSummaryUseCase *usecases.GetPaymentsSummaryUseCase
}

type PaymentsSummaryResponse struct {
	Default  *SummaryItem `json:"default"`
	Fallback *SummaryItem `json:"fallback"`
}

type SummaryItem struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

func NewPaymentController(
	enqueuePaymentuseCase *usecases.QueuePaymentsUseCase,
	getPaymentSummaryUseCase *usecases.GetPaymentsSummaryUseCase,
) *PaymentController {
	return &PaymentController{
		EnqueuePaymentuseCase:     enqueuePaymentuseCase,
		GetPaymentsSummaryUseCase: getPaymentSummaryUseCase,
	}
}

type CreatePaymentRequest struct {
	CorrelationID string  `json:"correlationId" validate:"required,uuid"`
	Amount        float64 `json:"amount" validate:"required,gt=0"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (pc *PaymentController) EnqueuePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ErrorResponse{
			Error:   "Invalid Request",
			Message: "Invalid request data: correlationId and amount are required and must be valid",
		})
		return
	}

	err := pc.EnqueuePaymentuseCase.EnqueuePayment(
		c.Request.Context(),
		"payments",
		models.Payment{
			CorrelationID: req.CorrelationID,
			Amount:        req.Amount,
			RequestedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	)
	if err != nil {
		log.Printf("Failed to queue payment %s: %v", req.CorrelationID, err)
		c.JSON(500, ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to process payment",
		})
		return
	}

	c.Status(204)
}

func (pc *PaymentController) GetPaymentsSummary(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' date format"})
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' date format"})
		return
	}

	summary, err := pc.GetPaymentsSummaryUseCase.Execute(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payments summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}
