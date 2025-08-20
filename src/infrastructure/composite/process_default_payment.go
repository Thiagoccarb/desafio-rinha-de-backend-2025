package composite

import (
	"payment-processor/controllers"
	"payment-processor/infrastructure"
	"payment-processor/infrastructure/repositories"
	usecases "payment-processor/use_cases"
)

func ProcessDefaultPaymentComposer() *controllers.PaymentController {
	conn := infrastructure.NewPostgresConnection()
	repository := repositories.NewPaymentRepository(conn)

	redisClient := infrastructure.NewRedis()
	enqueueUseCase := usecases.NewQueuePaymentsUseCase(redisClient)
	getSummaryUseCase := usecases.NewGetPaymentsSummaryUseCase(repository)
	controller := controllers.NewPaymentController(enqueueUseCase, getSummaryUseCase)
	return controller
}
