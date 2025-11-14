package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Kafka
	KafkaBrokers []string

	// Service-specific
	ServiceName string
	LogLevel    string
}

func Load() *Config {
	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://indexer:password@localhost:5432/indexer?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		ServiceName:  getEnv("SERVICE_NAME", "indexer"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
