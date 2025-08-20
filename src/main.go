package main

import (
	"context"
	"log"
	"net/http"
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

	redis := infrastructure.NewRedis()
	conn := infrastructure.NewPostgresConnection()
	queueUseCase := usecases.NewQueuePaymentsUseCase(redis)
	paymentRepository := repositories.NewPaymentRepository(conn)
	processPaymentService := services.NewProcessPaymentService(queueUseCase)
	processPaymentUseCase := usecases.NewProcessPaymentUseCase(*paymentRepository)

	streamWorkerPool := workers.NewStreamWorkerPool(
		*redis,
		"payments",
		"payment-group",
		5,
		*processPaymentService,
		*processPaymentUseCase,
	)
	if err := streamWorkerPool.Start(ctx); err != nil {
		log.Fatal("Failed to start stream worker pool:", err)
	}
	defer streamWorkerPool.Stop()

	log.Println("Starting Rinha de Backend 2025...")
	migrations.CreateRinhaTable()

	defer redis.Close()

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
