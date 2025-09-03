package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	Host            string
	Port            string
	Database        string
	Username        string
	Password        string
	MinPoolSize     int
	MaxPoolSize     int
	PruningInterval int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

type ServiceConfig struct {
	DefaultHealthCheckURL     string
	FallbackHealthCheckURL    string
	DefaultProcessPaymentURL  string
	FallbackProcessPaymentURL string
}

type Config struct {
	Database                      DatabaseConfig
	Services                      ServiceConfig
	Redis                         RedisConfig
	Queue                         string
	SetQueue                      string
	DQLQueue                      string
	RedisDefaultServiceStatuskey  string
	RedisFallbackServiceStatuskey string
}

var (
	config *Config
	once   sync.Once
)

func LoadConfig() *Config {
	once.Do(func() {
		// Load .env file
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found or error loading .env file:", err)
		}

		config = &Config{
			Database: DatabaseConfig{
				Host:     getEnv("DB_HOST", "localhost"),
				Port:     getEnv("DB_PORT", "5432"),
				Database: getEnv("DB_NAME", "rinha"),
				Username: getEnv("DB_USER", "postgres"),
				Password: getEnv("DB_PASSWORD", "postgres"),
			},
			Services: ServiceConfig{
				DefaultHealthCheckURL:     getEnv("DEFAULT_HEALTH_CHECK_URL", "http://localhost:8001/payments/service-health"),
				FallbackHealthCheckURL:    getEnv("FALLBACK_HEALTH_CHECK_URL", "http://localhost:8002/payments/service-health"),
				DefaultProcessPaymentURL:  getEnv("DEFAULT_PROCESS_PAYMENT_URL", "http://localhost:8001/payments"),
				FallbackProcessPaymentURL: getEnv("FALLBACK_PROCESS_PAYMENT_URL", "http://localhost:8002/payments"),
			},
			Redis: RedisConfig{
				Host:     getEnv("REDIS_HOST", "localhost"),
				Port:     getEnv("REDIS_PORT", "6379"),
				Password: getEnv("REDIS_PASSWORD", ""),
			},
			Queue:                         getEnv("QUEUE_NAME", "payments"),
			DQLQueue:                      getEnv("DQL_QUEUE_NAME", "dql_payments"),
			SetQueue:                      getEnv("SET_QUEUE_NAME", "processed_payments"),
			RedisDefaultServiceStatuskey:  getEnv("REDIS_DEFAULT_SERVICE_STATUS_KEY", "default_service_status"),
			RedisFallbackServiceStatuskey: getEnv("REDIS_FALLBACK_SERVICE_STATUS_KEY", "fallback_service_status"),
		}
	})
	return config
}

func (db *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.Username, db.Password, db.Database)
}
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
