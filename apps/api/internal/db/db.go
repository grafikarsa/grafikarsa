package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/config"
)

func NewPgxPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, err
	}
	pc.MinConns = 4
	pc.MaxConns = 25
	pc.MaxConnLifetime = 30 * time.Minute
	pc.MaxConnIdleTime = 5 * time.Minute
	return pgxpool.NewWithConfig(ctx, pc)
}

func Health(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return pool.Ping(ctx)
}
