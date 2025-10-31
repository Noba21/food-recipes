package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	Port               string
	HasuraAdminSecret  string
	HasuraEndpoint     string
	ChapaSecretKey     string
	UploadDir          string
}

func Load() *Config {
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/food_recipes"),
		JWTSecret:         getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		Port:              getEnv("PORT", "8080"),
		HasuraAdminSecret: getEnv("HASURA_GRAPHQL_ADMIN_SECRET", "myadminsecretkey"),
		HasuraEndpoint:    getEnv("HASURA_GRAPHQL_ENDPOINT", "http://localhost:8080/v1/graphql"),
		ChapaSecretKey:    getEnv("CHAPA_SECRET_KEY", "your-chapa-secret-key"),
		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}