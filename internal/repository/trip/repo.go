package trip

import (
	"context"
	"time"

	"github.com/olgapark/avito-labs/internal/infra/postgres/transaction"
)

type Repository struct {
	db      database
	timeout time.Duration
}

func New(db database, timeout time.Duration) *Repository {
	return &Repository{db: db, timeout: timeout}
}

func (r *Repository) executor(ctx context.Context) database {
	if tx := transaction.FromContext(ctx); tx != nil {
		return tx
	}
	return r.db
}
