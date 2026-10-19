package http

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txRunner struct {
	pool *pgxpool.Pool
}

func newTxRunner(pool *pgxpool.Pool) *txRunner { return &txRunner{pool: pool} }

func NewTxRunner(pool *pgxpool.Pool) *txRunner { return newTxRunner(pool) }

func (t *txRunner) Begin(ctx context.Context) (pgx.Tx, error) {
	return t.pool.Begin(ctx)
}
