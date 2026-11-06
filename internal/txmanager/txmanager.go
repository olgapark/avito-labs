package txmanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(
		ctx context.Context,
		fn func(ctx context.Context) error,
	) error
}

type ctxKey string

const txKey ctxKey = "tx"

type Manager struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Manager {
	return &Manager{
		pool: pool, timeout: timeout,
	}
}

func (m *Manager) Do(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	if tx := TxFromContext(ctx); tx != nil {
		return fn(ctx)
	}

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	rollback := func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.timeout)
		defer cancel()
		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("rollback transaction: %v", err)
		}
	}

	defer func() {
		if p := recover(); p != nil {
			rollback()
			panic(p)
		}
	}()

	txCtx := context.WithValue(ctx, txKey, tx)
	if err := fn(txCtx); err != nil {
		rollback()
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey).(pgx.Tx)
	return tx
}
