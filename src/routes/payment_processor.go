package routes

import (
	"payment-processor/infrastructure/composite"

	"github.com/gin-gonic/gin"
)

func RegisterprocessPaymentRoutes(router *gin.Engine) {
	group := router.Group("/")
	defaultPaymentController := composite.ProcessDefaultPaymentComposer()

	group.POST("/payments", defaultPaymentController.EnqueuePayment)
	group.GET("/payments-summary", defaultPaymentController.GetPaymentsSummary)
}
