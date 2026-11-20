package handler

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/service"
)

type Handler struct {
	api.Unimplemented

	pool         *pgxpool.Pool
	service      *service.Service
	queryTimeout time.Duration
}

func New(
	pool *pgxpool.Pool,
	service *service.Service,
	queryTimeout time.Duration,
) *Handler {
	return &Handler{
		pool:         pool,
		service:      service,
		queryTimeout: queryTimeout,
	}
}
