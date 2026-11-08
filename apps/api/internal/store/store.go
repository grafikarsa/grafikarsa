// Package store contains sqlc-generated queries plus a small wrapper.
package store

import (
	"context"

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
	return s.WithTxFunc(ctx, func(q *Queries, _ pgx.Tx) error {
		return fn(q)
	})
}

// WithTxFunc runs fn inside a transaction, passing both the transaction-bound
// queries and the transaction itself. Callers use it when a data change must
// also insert a River job atomically.
func (s *Store) WithTxFunc(ctx context.Context, fn func(q *Queries, tx pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	// Rollback after a successful Commit returns pgx.ErrTxClosed; ignoring it is safe.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(s.Queries.WithTx(tx), tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
