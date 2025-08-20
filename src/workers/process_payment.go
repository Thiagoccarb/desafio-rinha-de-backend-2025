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
	processPaymentUseCase usecases.ProcessPaymentUseCase
}

func NewStreamWorkerPool(
	redis infrastructure.Redis,
	streamName,
	groupName string,
	numWorkers int,
	processPaymentService services.ProcessPaymentService,
	processPaymentUseCase usecases.ProcessPaymentUseCase,
) *StreamWorkerPool {
	return &StreamWorkerPool{
		redis:                 redis,
		streamName:            streamName,
		groupName:             groupName,
		numWorkers:            numWorkers,
		stopCh:                make(chan struct{}),
		processPaymentService: processPaymentService,
		processPaymentUseCase: processPaymentUseCase,
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
					defaultStatus := swp.getDefaultServiceStatusData(ctx)
					fallbackStatus := swp.getFallbackStatusData(ctx)
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
	var success bool
	success = swp.processPaymentService.ProcessPayment(serviceType, paymentData, ctx)
	if !success {
		return false
	}
	success = swp.processPaymentUseCase.Execute(ctx, paymentData)
	return success
}

func (swp *StreamWorkerPool) getDefaultServiceStatusData(ctx context.Context) structs.ServiceStatus {
	config := config.LoadConfig()
	var data string
	var status structs.ServiceStatus
	data, _ = swp.redis.Get(ctx, config.RedisDefaultServiceStatuskey)
	if data == "" {
		defaultServiceStatus, err := services.GetDefaultServiceStatusData()
		if err != nil {
			log.Printf("Failed to get default service status: %v", err)
			return structs.ServiceStatus{
				Failing:         true,
				MinResponseTime: 0,
			}
		}
		data = string(defaultServiceStatus)
		swp.redis.Set(ctx, config.RedisDefaultServiceStatuskey, data)
	}
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		log.Printf("Failed to unmarshal  service status: %v", err)
		return structs.ServiceStatus{
			Failing:         true,
			MinResponseTime: 0,
		}
	}
	return status
}

func (swp *StreamWorkerPool) getFallbackStatusData(ctx context.Context) structs.ServiceStatus {
	config := config.LoadConfig()
	var data string
	var status structs.ServiceStatus

	data, _ = swp.redis.Get(ctx, config.RedisFallbackServiceStatuskey)
	if data == "" {
		fallbackStatus, err2 := services.GetFallbackServiceStatusData()
		if err2 != nil {
			log.Printf("Failed to get fallback service status: %v", err2)
			return structs.ServiceStatus{
				Failing:         true,
				MinResponseTime: 0,
			}
		}
		data = string(fallbackStatus)
		swp.redis.Set(ctx, config.RedisFallbackServiceStatuskey, data)
	}
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		log.Printf("Failed to unmarshal  service status: %v", err)
		return structs.ServiceStatus{
			Failing:         true,
			MinResponseTime: 0,
		}
	}
	return status
}
