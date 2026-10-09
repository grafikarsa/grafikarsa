package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type DBConfig struct {
	Host, User, Password, Name string
	Port                       int
}

type S3Config struct {
	Endpoint, Region, Bucket, AccessKey, SecretKey, PublicURL string
	UsePathStyle                                              bool
}

type DragonflyConfig struct {
	Host, Password string
	Port, DB       int
}

type JWTConfig struct {
	AccessSecret, RefreshSecret, AccessTTL string
	CSRFSalt                               string
}

type AdminSeed struct {
	Email, Password string
}

type Config struct {
	AppEnv string
	Domain string
	DB     DBConfig
	S3     S3Config
	Cache  DragonflyConfig
	JWT    JWTConfig
	Admin  AdminSeed
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}

func Load() (*Config, error) {
	for _, p := range []string{".env", "../../.env", "../../../.env"} {
		if abs, err := filepath.Abs(p); err == nil {
			_ = godotenv.Load(abs)
		}
	}

	cfg := &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Domain: getEnv("DOMAIN", "localhost"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "grafikarsa"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "grafikarsa"),
		},
		S3: S3Config{
			Endpoint:     getEnv("S3_ENDPOINT", "http://localhost:9000"),
			Region:       getEnv("S3_REGION", "us-east-1"),
			Bucket:       getEnv("S3_BUCKET", "grafikarsa"),
			AccessKey:    getEnv("S3_ACCESS_KEY", ""),
			SecretKey:    getEnv("S3_SECRET_KEY", ""),
			UsePathStyle: strings.ToLower(getEnv("S3_USE_PATH_STYLE", "true")) == "true",
			PublicURL:    getEnv("S3_PUBLIC_URL", "http://localhost:3000/api/media"),
		},
		Cache: DragonflyConfig{
			Host:     getEnv("DRAGONFLY_HOST", "localhost"),
			Port:     getEnvInt("DRAGONFLY_PORT", 6379),
			Password: getEnv("DRAGONFLY_PASSWORD", ""),
			DB:       getEnvInt("DRAGONFLY_DB", 0),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", ""),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
			AccessTTL:     getEnv("JWT_ACCESS_TTL", "15m"),
			CSRFSalt:      getEnv("CSRF_SECRET", ""),
		},
		Admin: AdminSeed{
			Email:    getEnv("ADMIN_SEED_EMAIL", "admin@grafikarsa.com"),
			Password: getEnv("ADMIN_SEED_PASSWORD", ""),
		},
	}

	if cfg.AppEnv == "production" {
		for _, kv := range [][2]string{
			{"DB_PASSWORD", cfg.DB.Password},
			{"JWT_ACCESS_SECRET", cfg.JWT.AccessSecret},
			{"JWT_REFRESH_SECRET", cfg.JWT.RefreshSecret},
			{"S3_ACCESS_KEY", cfg.S3.AccessKey},
			{"S3_SECRET_KEY", cfg.S3.SecretKey},
		} {
			if kv[1] == "" || strings.Contains(kv[1], "<rand") || strings.Contains(kv[1], "change-me") {
				return nil, fmt.Errorf("config: %s wajib diisi di production", kv[0])
			}
		}
	}
	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DB.User, c.DB.Password, c.DB.Host, c.DB.Port, c.DB.Name)
}

func (c *Config) CacheAddr() string {
	return fmt.Sprintf("%s:%d", c.Cache.Host, c.Cache.Port)
}

func (c *JWTConfig) CSRFSecret() string { return c.CSRFSalt }
