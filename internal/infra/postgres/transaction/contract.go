package transaction

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type beginner interface {
	BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error)
}
