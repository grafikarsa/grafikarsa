package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server
	ServerPort string
	ServerHost string

	// Database
	DatabaseURL string

	// JWT
	JWTAccessSecret      string
	JWTRefreshSecret     string
	JWTAccessExpiration  time.Duration
	JWTRefreshExpiration time.Duration

	// MinIO
	MinIOEndpoint        string
	MinIOAccessKeyID     string
	MinIOSecretAccessKey string
	MinIOBucketName      string
	MinIOUseSSL          bool
	MinioCDNURL          string

	// App
	AppEnv     string
	AppBaseURL string
}

// Load reads configuration from environment variables
func Load() *Config {
	// Database components
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPass := getEnv("POSTGRES_PASSWORD", "postgres")
	dbName := getEnv("POSTGRES_DB", "grafikarsa_db")

	defaultDBURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPass, dbHost, dbPort, dbName)

	return &Config{
		// Server
		ServerPort: getEnv("PORT", "8080"),
		ServerHost: getEnv("HOST", "0.0.0.0"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", defaultDBURL),

		// JWT
		JWTAccessSecret:      getEnv("JWT_ACCESS_SECRET", "your-access-secret-key-change-in-production"),
		JWTRefreshSecret:     getEnv("JWT_REFRESH_SECRET", "your-refresh-secret-key-change-in-production"),
		JWTAccessExpiration:  getDurationEnv("JWT_ACCESS_EXPIRATION", 15*time.Minute),
		JWTRefreshExpiration: getDurationEnv("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),

		// MinIO
		MinIOEndpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKeyID:     getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinIOSecretAccessKey: getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
		MinIOBucketName:      getEnv("MINIO_BUCKET_NAME", "grafikarsa"),
		MinIOUseSSL:          getBoolEnv("MINIO_USE_SSL", false),
		MinioCDNURL:          getEnv("MINIO_CDN_URL", "http://localhost:9000/grafikarsa"),

		// App
		AppEnv:     getEnv("APP_ENV", "development"),
		AppBaseURL: getEnv("APP_BASE_URL", "http://localhost:8080"),
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return defaultValue
		}
		return parsed
	}
	return defaultValue
}
