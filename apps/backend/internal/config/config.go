package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	MinIO    MinIOConfig
	JWT      JWTConfig
	CORS     CORSConfig
}

type AppConfig struct {
	Env       string
	Port      string
	URL       string
	AdminPath string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type MinIOConfig struct {
	Endpoint      string
	PresignHost   string // Host to use in presigned URLs (for browser access)
	PresignUseSSL bool   // Whether presigned URLs should use HTTPS (for Cloudflare proxy)
	AccessKey     string
	SecretKey     string
	Bucket        string
	UseSSL        bool
	PublicURL     string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type CORSConfig struct {
	Origins []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))

	cfg := &Config{
		App: AppConfig{
			Env:       getEnv("APP_ENV", "development"),
			Port:      getEnv("APP_PORT", "8080"),
			URL:       getEnv("APP_URL", "http://localhost:8080"),
			AdminPath: getEnv("ADMIN_LOGIN_PATH", "loginadmin"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "grafikarsa"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "grafikarsa"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		MinIO: MinIOConfig{
			Endpoint:      getEnv("MINIO_ENDPOINT", "localhost:9000"),
			PresignHost:   getEnv("MINIO_PRESIGN_HOST", "localhost:9000"),
			PresignUseSSL: getEnvBool("MINIO_PRESIGN_USE_SSL", false),
			AccessKey:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:     getEnv("MINIO_SECRET_KEY", ""),
			Bucket:        getEnv("MINIO_BUCKET", "grafikarsa"),
			UseSSL:        getEnvBool("MINIO_USE_SSL", false),
			PublicURL:     getEnv("STORAGE_PUBLIC_URL", "http://localhost:9000/grafikarsa"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", ""),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		CORS: CORSConfig{
			Origins: func() []string {
				raw := strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000"), ",")
				var normalized []string
				for _, o := range raw {
					o = strings.TrimSpace(o)
					o = strings.TrimSuffix(o, "/")
					if o != "" {
						normalized = append(normalized, o)
					}
				}
				return normalized
			}(),
		},
	}

	// Validate critical configuration
	if cfg.App.Env == "production" {
		if cfg.JWT.AccessSecret == "" || cfg.JWT.RefreshSecret == "" {
			return nil, errors.New("JWT secrets must be configured in production environment")
		}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defaultValue
}
