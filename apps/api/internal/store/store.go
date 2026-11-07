// Package store contains sqlc-generated queries plus a small wrapper.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store bundles the generated queries with the pgx pool for transactions.
type Store struct {
	*Queries
	pool *pgxpool.Pool
}

// NewStore builds a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		Queries: New(pool),
		pool:    pool,
	}
}

// WithTx runs fn inside a transaction; commits if fn returns nil, otherwise rolls back.
func (s *Store) WithTx(ctx context.Context, fn func(q *Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		// After a successful commit the transaction is closed and Rollback
		// returns pgx.ErrTxClosed, which is expected.
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			// Nothing actionable: fn's error (if any) is what callers see.
		}
	}()

	if err := fn(s.Queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
