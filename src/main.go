package main

import (
	"context"
	"log"
	"net/http"
	"payment-processor/config"
	"payment-processor/core/services"
	"payment-processor/infrastructure"
	"payment-processor/infrastructure/migrations"
	"payment-processor/infrastructure/repositories"
	"payment-processor/routes"
	usecases "payment-processor/use_cases"
	"payment-processor/workers"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := config.LoadConfig()
	redis := infrastructure.NewRedis()
	conn := infrastructure.NewPostgresConnection()
	queueUseCase := usecases.NewQueuePaymentsUseCase(redis)
	paymentRepository := repositories.NewPaymentRepository(conn)
	processPaymentService := services.NewProcessPaymentService(queueUseCase)
	queuePaymentUseCase := usecases.NewQueuePaymentsUseCase(redis)
	getPaymentUseCase := usecases.NewGetPaymentsSummaryUseCase(redis)

	streamWorkerPool := workers.NewStreamWorkerPool(
		*redis,
		config.Queue,
		"payment-group",
		10,
		*processPaymentService,
		*queuePaymentUseCase,
	)
	if err := streamWorkerPool.Start(ctx); err != nil {
		log.Fatal("Failed to start stream worker pool:", err)
	}
	defer streamWorkerPool.Stop()

	log.Println("Starting Rinha de Backend 2025...")
	migrations.CreateRinhaTable()

	defer redis.Close()

	processPaymentUseCase := usecases.NewProcessPaymentUseCase(
		*paymentRepository,
		*getPaymentUseCase,
		*redis,
	)

	go processPaymentUseCase.Execute(ctx)

	router := gin.Default()
	router.Use(corsMiddleware())
	routes.RegisterprocessPaymentRoutes(router)

	router.Run(":8080")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
