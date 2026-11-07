package config

import (
	"strings"
	"testing"
	"time"
)

// setRequiredEnv sets every required variable plus explicit defaults so tests
// never depend on the developer machine.
func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("GK_ENV", EnvDevelopment)
	t.Setenv("GK_HTTP_ADDR", ":8091")
	t.Setenv("GK_LOG_LEVEL", LogLevelInfo)
	t.Setenv("GK_SECRET_KEY", strings.Repeat("a", 32))
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/grafikarsa")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_REGION", "us-east-1")
	t.Setenv("S3_ACCESS_KEY", "access-key")
	t.Setenv("S3_SECRET_KEY", "secret-key")
	t.Setenv("S3_BUCKET_PRIVATE", "gk-private")
	t.Setenv("S3_BUCKET_PUBLIC", "gk-public")
	t.Setenv("GK_SHUTDOWN_TIMEOUT", "25s")
}

func TestParseValid(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Env != EnvDevelopment {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvDevelopment)
	}
	if cfg.HTTPAddr != ":8091" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8091")
	}
	if cfg.LogLevel != LogLevelInfo {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, LogLevelInfo)
	}
	if cfg.SecretKey != strings.Repeat("a", 32) {
		t.Errorf("SecretKey = %q, want 32 a characters", cfg.SecretKey)
	}
	if cfg.DatabaseURL == "" || cfg.ValkeyURL == "" {
		t.Errorf("DatabaseURL or ValkeyURL is empty")
	}
	if cfg.S3Endpoint != "http://localhost:9000" {
		t.Errorf("S3Endpoint = %q", cfg.S3Endpoint)
	}
	if cfg.S3Region != "us-east-1" {
		t.Errorf("S3Region = %q", cfg.S3Region)
	}
	if cfg.S3BucketPrivate != "gk-private" || cfg.S3BucketPublic != "gk-public" {
		t.Errorf("buckets = %q, %q", cfg.S3BucketPrivate, cfg.S3BucketPublic)
	}
	if cfg.ShutdownTimeout != 25*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 25s", cfg.ShutdownTimeout)
	}
	if cfg.IsProduction() {
		t.Errorf("IsProduction() = true for development")
	}
}

func TestParseMissingSecret(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GK_SECRET_KEY", "")

	_, err := Parse()
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "GK_SECRET_KEY") {
		t.Errorf("error = %q, want mention of GK_SECRET_KEY", err)
	}
}

func TestParseShortSecret(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GK_SECRET_KEY", "too-short")

	_, err := Parse()
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "at least 32") {
		t.Errorf("error = %q, want minimum length message", err)
	}
}

func TestParseInvalidEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GK_ENV", "test")

	_, err := Parse()
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "GK_ENV") {
		t.Errorf("error = %q, want mention of GK_ENV", err)
	}
}

func TestParseDefaults(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GK_ENV", "")
	t.Setenv("GK_HTTP_ADDR", "")
	t.Setenv("GK_LOG_LEVEL", "")
	t.Setenv("S3_REGION", "")
	t.Setenv("S3_BUCKET_PRIVATE", "")
	t.Setenv("S3_BUCKET_PUBLIC", "")
	t.Setenv("GK_SHUTDOWN_TIMEOUT", "")

	cfg, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Env != EnvDevelopment {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvDevelopment)
	}
	if cfg.HTTPAddr != ":8091" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8091")
	}
	if cfg.LogLevel != LogLevelInfo {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, LogLevelInfo)
	}
	if cfg.S3Region != "us-east-1" {
		t.Errorf("S3Region = %q, want %q", cfg.S3Region, "us-east-1")
	}
	if cfg.S3BucketPrivate != "gk-private" {
		t.Errorf("S3BucketPrivate = %q, want %q", cfg.S3BucketPrivate, "gk-private")
	}
	if cfg.S3BucketPublic != "gk-public" {
		t.Errorf("S3BucketPublic = %q, want %q", cfg.S3BucketPublic, "gk-public")
	}
	if cfg.ShutdownTimeout != 25*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 25s", cfg.ShutdownTimeout)
	}
}

func TestValidateJoinsErrors(t *testing.T) {
	cfg := Config{Env: "invalid", LogLevel: "verbose"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}

	for _, field := range []string{
		"GK_ENV",
		"GK_LOG_LEVEL",
		"GK_SECRET_KEY",
		"DATABASE_URL",
		"VALKEY_URL",
		"S3_ENDPOINT",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
	} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error = %q, want mention of %s", err, field)
		}
	}
}
