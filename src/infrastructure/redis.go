package infrastructure

import (
	"context"
	"fmt"
	"payment-processor/config"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

func NewRedis() *Redis {
	configs := config.LoadConfig()
	r := redis.NewClient(&redis.Options{
		Addr:         configs.Redis.Host + ":" + configs.Redis.Port,
		Password:     configs.Redis.Password,
		DB:           0,
		PoolSize:     100,                    // Increased from 50
		MinIdleConns: 50,                     // Increased from 20
		MaxRetries:   1,                      // Reduced from 2 (fail faster)
		PoolTimeout:  500 * time.Millisecond, // Reduced from 1s
		DialTimeout:  500 * time.Millisecond, // Reduced from 1s
		ReadTimeout:  500 * time.Millisecond, // Reduced from 1s
		WriteTimeout: 500 * time.Millisecond, // Reduced from 1s

	})
	_, err := r.Ping(context.Background()).Result()
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}

	return &Redis{
		client: r,
	}
}

func (r *Redis) Set(ctx context.Context, key string, value string, expirationInSeconds ...int) error {
	var ttl time.Duration = 5 * time.Second

	if len(expirationInSeconds) > 0 {
		ttl = time.Duration(expirationInSeconds[0]) * time.Second
	}
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}
	return nil
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("failed to get key: %w", err)
	}
	return value, nil
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) XAdd(ctx context.Context, stream string, values map[string]interface{}) error {
	_, err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to add to stream: %w", err)
	}
	return nil
}

func (r *Redis) MessagesConsumer(ctx context.Context, group, consumer, stream string, count int64) ([]redis.XStream, error) {

	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Block:    4 * time.Second,
		Count:    100,
		NoAck:    true,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read new messages: %w", err)
	}

	return streams, nil
}

func (r *Redis) XGroupCreate(ctx context.Context, stream, group string) error {
	err := r.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

func (r *Redis) XAck(ctx context.Context, stream, group string, ids ...string) error {
	_, err := r.client.XAck(ctx, stream, group, ids...).Result()
	return err
}

func (r *Redis) ZAdd(ctx context.Context, key string, data redis.Z) error {
	_, err := r.client.ZAdd(ctx, key, data).Result()
	if err != nil {
		return fmt.Errorf("failed to add to sorted set: %w", err)
	}
	return nil
}

func (r *Redis) ZRangeByScore(ctx context.Context, key string, min, max time.Time) ([]string, error) {
	minFloat := fmt.Sprintf("%.6f", float64(min.Unix())+float64(min.Nanosecond())/1e9)
	maxFloat := fmt.Sprintf("%.6f", float64(max.Unix())+float64(max.Nanosecond())/1e9)
	values, err := r.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: minFloat,
		Max: maxFloat,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to range by score: %w", err)
	}
	return values, nil
}
