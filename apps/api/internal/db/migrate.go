package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"grafikarsa/api/internal/config"
)

func MigrateUp(ctx context.Context, cfg *config.Config) error {
	pc, err := parseConfig(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}
	db := stdlib.OpenDB(*pc)
	defer db.Close()
	goose.SetBaseFS(MigrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

func MigrateDown(ctx context.Context, cfg *config.Config) error {
	pc, err := parseConfig(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}
	db := stdlib.OpenDB(*pc)
	defer db.Close()
	goose.SetBaseFS(MigrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Down(db, "migrations")
}

func parseConfig(url string) (*pgx.ConnConfig, error) {
	return pgx.ParseConfig(url)
}
