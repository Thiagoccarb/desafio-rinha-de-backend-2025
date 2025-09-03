package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"payment-processor/config"
	"payment-processor/core/models"
	"payment-processor/core/services"
	"payment-processor/infrastructure"
	"payment-processor/structs"
	usecases "payment-processor/use_cases"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type StreamWorkerPool struct {
	redis                 infrastructure.Redis
	streamName            string
	groupName             string
	numWorkers            int
	stopCh                chan struct{}
	wg                    sync.WaitGroup
	processPaymentService services.ProcessPaymentService
	queuePaymentUseCase   usecases.QueuePaymentsUseCase
}

func NewStreamWorkerPool(
	redis infrastructure.Redis,
	streamName,
	groupName string,
	numWorkers int,
	processPaymentService services.ProcessPaymentService,
	queuePaymentUseCase usecases.QueuePaymentsUseCase,
) *StreamWorkerPool {
	return &StreamWorkerPool{
		redis:                 redis,
		streamName:            streamName,
		groupName:             groupName,
		numWorkers:            numWorkers,
		stopCh:                make(chan struct{}),
		processPaymentService: processPaymentService,
		queuePaymentUseCase:   queuePaymentUseCase,
	}
}

func (swp *StreamWorkerPool) Start(ctx context.Context) error {
	if err := swp.redis.XGroupCreate(ctx, swp.streamName, swp.groupName); err != nil {
		return err
	}

	for i := 0; i < swp.numWorkers; i++ {
		swp.wg.Add(1)
		go swp.worker(ctx, fmt.Sprintf("worker-%d", i))
	}
	go swp.getServiceStatusData(ctx)

	log.Printf("Started %d stream workers for %s", swp.numWorkers, swp.streamName)
	return nil
}

func (swp *StreamWorkerPool) Stop() {
	close(swp.stopCh)
	swp.wg.Wait()
	log.Println("All stream workers stopped")
}

func (swp *StreamWorkerPool) worker(ctx context.Context, consumerName string) {
	defer swp.wg.Done()

	for {
		select {
		case <-swp.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			streams, err := swp.redis.MessagesConsumer(ctx, swp.groupName, consumerName, swp.streamName, 100)
			if err != nil {
				log.Printf("Worker %s: Stream read error: %v", consumerName, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					defaultStatus := swp.getSerializedServiceStatus(ctx, "default")
					fallbackStatus := swp.getSerializedServiceStatus(ctx, "fallback")

					if defaultStatus.Failing && fallbackStatus.Failing {
						swp.redis.XAdd(ctx, swp.streamName, message.Values)
						continue
					}

					var serviceType string

					if defaultStatus.Failing && !fallbackStatus.Failing {
						serviceType = "fallback"
					} else if !defaultStatus.Failing && fallbackStatus.Failing {
						serviceType = "default"
					} else {
						if defaultStatus.MinResponseTime <= fallbackStatus.MinResponseTime {
							serviceType = "default"
						} else {
							serviceType = "fallback"
						}
					}
					success := swp.processPayment(serviceType, message, ctx)
					if !success {
						log.Printf("Worker %s: Failed to process payment for message %s", consumerName, message.ID)
						swp.redis.XAdd(ctx, swp.streamName, message.Values)
					}
				}
			}
		}
	}
}

func (swp *StreamWorkerPool) processPayment(serviceType string, message redis.XMessage, ctx context.Context) bool {

	// Parse payment data
	correlationID := message.Values["correlationId"].(string)
	amount := message.Values["amount"].(string)
	requestedAt := message.Values["requestedAt"].(string)
	amountFloat, _ := strconv.ParseFloat(amount, 64)

	paymentData := models.Payment{
		CorrelationID: correlationID,
		Amount:        amountFloat,
		RequestedAt:   requestedAt,
		Type:          serviceType,
	}

	success := swp.processPaymentService.ProcessPayment(serviceType, paymentData, ctx)
	if success {
		parsedTime, _ := time.Parse(time.RFC3339, requestedAt)
		tsFloat := float64(parsedTime.Unix())
		swp.queuePaymentUseCase.StoreAsScore(ctx, config.LoadConfig().SetQueue, tsFloat, paymentData)
	}
	return true
}

func (swp *StreamWorkerPool) getSerializedServiceStatus(ctx context.Context, serviceType string) structs.ServiceStatus {
	config := config.LoadConfig()
	var data string
	var status structs.ServiceStatus
	if serviceType == "default" {
		data, _ = swp.redis.Get(ctx, config.RedisDefaultServiceStatuskey)
	} else {
		data, _ = swp.redis.Get(ctx, config.RedisFallbackServiceStatuskey)
	}

	if data == "" {
		return structs.ServiceStatus{
			Failing:         true,
			MinResponseTime: 0,
		}
	}
	data = string(data)
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		log.Printf("Failed to unmarshal  service status: %v", err)
		return structs.ServiceStatus{
			Failing:         true,
			MinResponseTime: 0,
		}
	}
	return status
}

func (swp *StreamWorkerPool) getServiceStatusData(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-swp.stopCh:
			log.Println("Service status goroutine: Stop signal received, stopping execution")
			return
		case <-ctx.Done():
			log.Println("Service status goroutine: Context canceled, stopping execution")
			return
		case <-ticker.C:
			newCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			config := config.LoadConfig()
			defaultStatus, err := services.GetDefaultServiceStatusData(newCtx)
			if err != nil {
				log.Printf("Failed to get default service status: %v", err)
			}
			swp.redis.Set(ctx, config.RedisDefaultServiceStatuskey, string(defaultStatus))
			fallbackStatus, err := services.GetFallbackServiceStatusData(newCtx)
			if err != nil {
				log.Printf("Failed to get fallback service status: %v", err)
			}
			swp.redis.Set(ctx, config.RedisFallbackServiceStatuskey, string(fallbackStatus))
		}
	}

}
