package config

import "os"

const (
	DefaultPort             = "8080"
	DefaultHost             = "0.0.0.0"
	DefaultAppMode          = "mock"
	DefaultAIServiceBaseURL = "http://121.40.241.161:8081"
)

type Config struct {
	Host             string
	Port             string
	AppMode          string
	DatabaseURL      string
	AIServiceBaseURL string
}

func Load() Config {
	return Config{
		Host:             getEnv("HOST", DefaultHost),
		Port:             getEnv("PORT", DefaultPort),
		AppMode:          getEnv("APP_MODE", DefaultAppMode),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		AIServiceBaseURL: getEnv("AI_SERVICE_BASE_URL", DefaultAIServiceBaseURL),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
