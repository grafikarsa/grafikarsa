package config_test

import (
	"os"
	"testing"

	"grafikarsa/api/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Port != 5432 || cfg.S3.Region != "us-east-1" {
		t.Fatalf("default salah: %+v", cfg)
	}
}

func TestProductionFailFast(t *testing.T) {
	os.Clearenv()
	os.Setenv("APP_ENV", "production")
	if _, err := config.Load(); err == nil {
		t.Fatal("production tanpa secret harus gagal")
	}
}
