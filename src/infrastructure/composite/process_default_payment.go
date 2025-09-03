package composite

import (
	"payment-processor/controllers"
	"payment-processor/infrastructure"
	usecases "payment-processor/use_cases"
)

func ProcessDefaultPaymentComposer() *controllers.PaymentController {

	redisClient := infrastructure.NewRedis()
	enqueueUseCase := usecases.NewQueuePaymentsUseCase(redisClient)
	getSummaryUseCase := usecases.NewGetPaymentsSummaryUseCase(redisClient)
	controller := controllers.NewPaymentController(enqueueUseCase, getSummaryUseCase)
	return controller
}
