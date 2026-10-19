package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	dbmigrations "grafikarsa/api/db/migrations"
	"grafikarsa/api/internal/config"
)

func MigrateUp(ctx context.Context, cfg *config.Config) error {
	pc, err := parseConfig(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}
	db := stdlib.OpenDB(*pc)
	defer db.Close()
	goose.SetBaseFS(dbmigrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, ".")
}

func MigrateDown(ctx context.Context, cfg *config.Config) error {
	pc, err := parseConfig(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}
	db := stdlib.OpenDB(*pc)
	defer db.Close()
	goose.SetBaseFS(dbmigrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Down(db, ".")
}

// MigrateStatus menampilkan status semua migrasi (dipakai `make migrate-status`
// dan saat verifikasi deploy).
func MigrateStatus(ctx context.Context, cfg *config.Config) error {
	pc, err := parseConfig(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}
	db := stdlib.OpenDB(*pc)
	defer db.Close()
	goose.SetBaseFS(dbmigrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Status(db, ".")
}

// MigrateVersion menampilkan versi migrasi saat ini.
func MigrateVersion(ctx context.Context, cfg *config.Config) error {
	pc, err := parseConfig(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}
	db := stdlib.OpenDB(*pc)
	defer db.Close()
	goose.SetBaseFS(dbmigrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	v, err := goose.GetDBVersion(db)
	if err != nil {
		return err
	}
	fmt.Printf("db version: %d\n", v)
	return nil
}

func parseConfig(url string) (*pgx.ConnConfig, error) {
	return pgx.ParseConfig(url)
}
