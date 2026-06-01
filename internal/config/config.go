package config

import "os"

const (
	DefaultPort    = "8080"
	DefaultHost    = "0.0.0.0"
	DefaultAppMode = "mock"
)

type Config struct {
	Host        string
	Port        string
	AppMode     string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Host:        getEnv("HOST", DefaultHost),
		Port:        getEnv("PORT", DefaultPort),
		AppMode:     getEnv("APP_MODE", DefaultAppMode),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
