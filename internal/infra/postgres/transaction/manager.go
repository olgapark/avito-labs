package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type contextKey struct{}

type Manager struct {
	db      beginner
	timeout time.Duration
}

func New(db beginner, timeout time.Duration) *Manager {
	return &Manager{db: db, timeout: timeout}
}

func FromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(contextKey{}).(pgx.Tx)
	return tx
}

func (m *Manager) Do(ctx context.Context, fn func(context.Context) error) (resultErr error) {
	if FromContext(ctx) != nil {
		return fn(ctx)
	}
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	tx, err := m.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		rollbackCtx, rollbackCancel := context.WithTimeout(context.WithoutCancel(ctx), m.timeout)
		defer rollbackCancel()
		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback transaction: %w", err))
		}
	}()
	if err := fn(context.WithValue(ctx, contextKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
