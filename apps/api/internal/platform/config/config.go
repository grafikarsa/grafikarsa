// Package config loads and validates the API configuration from the
// environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Valid values for the deployment environment.
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Valid log levels.
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Config contains every setting required by the API.
type Config struct {
	Env             string        `env:"GK_ENV" envDefault:"development"`
	HTTPAddr        string        `env:"GK_HTTP_ADDR" envDefault:":8091"`
	LogLevel        string        `env:"GK_LOG_LEVEL" envDefault:"info"`
	SecretKey       string        `env:"GK_SECRET_KEY"`
	DatabaseURL     string        `env:"DATABASE_URL"`
	ValkeyURL       string        `env:"VALKEY_URL"`
	S3Endpoint      string        `env:"S3_ENDPOINT"`
	S3Region        string        `env:"S3_REGION" envDefault:"us-east-1"`
	S3AccessKey     string        `env:"S3_ACCESS_KEY"`
	S3SecretKey     string        `env:"S3_SECRET_KEY"`
	S3BucketPrivate string        `env:"S3_BUCKET_PRIVATE" envDefault:"gk-private"`
	S3BucketPublic  string        `env:"S3_BUCKET_PUBLIC" envDefault:"gk-public"`
	ShutdownTimeout time.Duration `env:"GK_SHUTDOWN_TIMEOUT" envDefault:"25s"`
}

// Load reads configuration from the environment. Outside production it first
// looks for a .env file in the current directory or one of its three parents.
// Values that are already set in the environment take precedence because
// godotenv does not override them.
func Load() (Config, error) {
	if err := LoadDotEnv(); err != nil {
		return Config{}, err
	}

	return Parse()
}

// LoadDotEnv loads the closest .env file in the current directory or one of
// its three parents. It does nothing in production or when no .env file is
// found. Values that are already set in the environment take precedence
// because godotenv does not override them.
func LoadDotEnv() error {
	if os.Getenv("GK_ENV") == EnvProduction {
		return nil
	}

	path := findEnvFile()
	if path == "" {
		return nil
	}
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}

// Parse reads configuration from environment variables only. It does not
// look for a .env file, which keeps tests deterministic.
func Parse() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate reports every invalid field joined into a single error.
func (c Config) Validate() error {
	var errs []error

	switch c.Env {
	case EnvDevelopment, EnvStaging, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("GK_ENV must be one of %s, %s, %s", EnvDevelopment, EnvStaging, EnvProduction))
	}

	switch c.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		errs = append(errs, fmt.Errorf("GK_LOG_LEVEL must be one of %s, %s, %s, %s", LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError))
	}

	if c.SecretKey == "" {
		errs = append(errs, errors.New("GK_SECRET_KEY is required"))
	} else if len(c.SecretKey) < 32 {
		errs = append(errs, errors.New("GK_SECRET_KEY must be at least 32 characters"))
	}

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.ValkeyURL == "" {
		errs = append(errs, errors.New("VALKEY_URL is required"))
	}
	if c.S3Endpoint == "" {
		errs = append(errs, errors.New("S3_ENDPOINT is required"))
	}
	if c.S3AccessKey == "" {
		errs = append(errs, errors.New("S3_ACCESS_KEY is required"))
	}
	if c.S3SecretKey == "" {
		errs = append(errs, errors.New("S3_SECRET_KEY is required"))
	}

	return errors.Join(errs...)
}

// IsProduction reports whether the API runs in the production environment.
func (c Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// findEnvFile returns the path of the closest .env file within the current
// directory and its three parents, or an empty string when none exists.
func findEnvFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for range 4 {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}
