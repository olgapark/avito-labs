package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/olgapark/avito-labs/internal/config"
	"github.com/olgapark/avito-labs/internal/http/health"
	"github.com/olgapark/avito-labs/internal/http/middleware"
	createhandler "github.com/olgapark/avito-labs/internal/http/trip/create"
	finishhandler "github.com/olgapark/avito-labs/internal/http/trip/finish"
	gethandler "github.com/olgapark/avito-labs/internal/http/trip/get"
	"github.com/olgapark/avito-labs/internal/infra/postgres"
	"github.com/olgapark/avito-labs/internal/infra/postgres/transaction"
	"github.com/olgapark/avito-labs/internal/infra/telemetry"
	"github.com/olgapark/avito-labs/internal/repository/trip"
	"github.com/olgapark/avito-labs/internal/usecase/trip/create"
	"github.com/olgapark/avito-labs/internal/usecase/trip/finish"
	"github.com/olgapark/avito-labs/internal/usecase/trip/get"
)

type Container struct {
	Server    *http.Server
	Logger    *slog.Logger
	pool      *pgxpool.Pool
	telemetry *telemetry.Providers
}

func New(ctx context.Context, cfg *config.Config) (_ *Container, resultErr error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	logger := telemetry.NewLogger(os.Stdout, level, cfg.OTELServiceName)
	providers, err := telemetry.New(ctx, cfg, "lab2")
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
			defer cancel()
			resultErr = errors.Join(resultErr, providers.Shutdown(cleanupCtx))
		}
	}()
	poolConfig, err := postgres.PoolConfig(cfg)
	if err != nil {
		return nil, err
	}
	poolConfig.ConnConfig.Tracer, err = postgres.NewTracer(providers.Traces, providers.Meters)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	defer func() {
		if resultErr != nil {
			pool.Close()
		}
	}()
	pingCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	repo := trip.New(pool, cfg.DatabaseQueryTimeout)
	tx := transaction.New(pool, cfg.DatabaseQueryTimeout)
	meter := providers.Meters.Meter("trip-service")
	metrics, err := telemetry.NewTripMetrics(meter, repo)
	if err != nil {
		return nil, err
	}
	createTrip := create.New(repo, tx, logger, metrics, time.Now, uuid.New)
	getTrip := get.New(repo)
	finishTrip := finish.New(repo, tx, logger, metrics, time.Now)
	healthHandler := health.New(pool, cfg.DatabaseQueryTimeout)
	router := chi.NewRouter()
	router.Post("/api/v1/trips", createhandler.New(createTrip).ServeHTTP)
	router.Get("/api/v1/trips/{tripId}", gethandler.New(getTrip).ServeHTTP)
	router.Post("/api/v1/trips/{tripId}/finish", finishhandler.New(finishTrip).ServeHTTP)
	router.Get("/health", healthHandler.Health)
	router.Get("/ready", healthHandler.Ready)
	handler, err := middleware.Metrics(meter, router)
	if err != nil {
		return nil, err
	}
	handler = middleware.Logging(logger, handler)
	handler = middleware.Tracing(router, providers.Traces, handler)
	handler = middleware.Recover(logger, handler)
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: handler,
		ReadTimeout: cfg.HTTPReadTimeout, ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout, IdleTimeout: cfg.HTTPIdleTimeout,
	}
	return &Container{Server: server, Logger: logger, pool: pool, telemetry: providers}, nil
}

func (c *Container) Shutdown(ctx context.Context) error {
	serverErr := c.Server.Shutdown(ctx)
	if serverErr != nil {
		serverErr = errors.Join(serverErr, c.Server.Close())
	}
	if err := c.telemetry.Shutdown(ctx); err != nil {
		c.Logger.WarnContext(ctx, "telemetry shutdown failed", "error", err)
	}
	c.pool.Close()
	return serverErr
}
