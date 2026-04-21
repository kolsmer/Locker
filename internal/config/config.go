package config

import (
	"os"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Server
	Port string

	// JWT
	JWTSecret string

	// Payment provider
	YookassaShopID string
	YookassaAPIKey string

	// Environment
	Environment string
}

func NewConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "locker"),
		Port:       getEnv("PORT", "8080"),
		JWTSecret:  getEnv("JWT_SECRET", "dev-secret-key"),
		Environment: getEnv("ENVIRONMENT", "development"),
		YookassaShopID: getEnv("YOOKASSA_SHOP_ID", ""),
		YookassaAPIKey: getEnv("YOOKASSA_API_KEY", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}