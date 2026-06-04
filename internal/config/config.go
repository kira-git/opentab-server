package config

import (
	"os"
	"strconv"
	"time"
)

const (
	DefaultPort             = "8080"
	DefaultHost             = "0.0.0.0"
	DefaultAppMode          = "mock"
	DefaultAIServiceBaseURL = "http://121.40.241.161:8081"
)

type Config struct {
	Host                    string
	Port                    string
	AppMode                 string
	DatabaseURL             string
	AIServiceBaseURL        string
	AIConcurrentLimit       int
	AIUserConcurrentLimit   int
	AIStreamSmoothInterval  time.Duration
	AIStreamSmoothChunkSize int
	DBMaxOpenConns          int
	DBMaxIdleConns          int
	DBConnMaxLifetimeMin    int
}

func Load() Config {
	return Config{
		Host:                    getEnv("HOST", DefaultHost),
		Port:                    getEnv("PORT", DefaultPort),
		AppMode:                 getEnv("APP_MODE", DefaultAppMode),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		AIServiceBaseURL:        getEnv("AI_SERVICE_BASE_URL", DefaultAIServiceBaseURL),
		AIConcurrentLimit:       getEnvInt("AI_CONCURRENT_LIMIT", 3),
		AIUserConcurrentLimit:   getEnvInt("AI_USER_CONCURRENT_LIMIT", 1),
		AIStreamSmoothInterval:  getEnvDurationMS("AI_STREAM_SMOOTH_INTERVAL_MS", 25*time.Millisecond),
		AIStreamSmoothChunkSize: getEnvInt("AI_STREAM_SMOOTH_CHUNK_SIZE", 2),
		DBMaxOpenConns:          getEnvInt("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:          getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetimeMin:    getEnvInt("DB_CONN_MAX_LIFETIME_MIN", 30),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDurationMS(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return time.Duration(parsed) * time.Millisecond
}
