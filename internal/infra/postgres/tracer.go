package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type (
	queryKey  struct{}
	queryInfo struct {
		started    time.Time
		operation  string
		collection string
	}
)

type QueryTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

func NewTracer(traces trace.TracerProvider, meters metric.MeterProvider) (*QueryTracer, error) {
	duration, err := meters.Meter("trip-service/postgres").Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("db duration histogram: %w", err)
	}
	return &QueryTracer{
		Tracer: otelpgx.NewTracer(otelpgx.WithTracerProvider(traces),
			otelpgx.WithMeterProvider(noop.NewMeterProvider()),
			otelpgx.WithDisableSQLStatementInAttributes(), otelpgx.WithDisableConnectionDetailsInAttributes()),
		duration: duration,
	}, nil
}

func (t *QueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	collection := ""
	for _, table := range []string{"trip_status_history", "trips"} {
		if strings.Contains(data.SQL, table) {
			collection = table
			break
		}
	}
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.collection.name", collection))
	return context.WithValue(ctx, queryKey{}, queryInfo{started: time.Now(), operation: otelpgx.SQLOperationName(ctx, data.SQL), collection: collection})
}

func (t *QueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	info, _ := ctx.Value(queryKey{}).(queryInfo)
	errorType := ""
	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		errorType = "database_error"
		var pgErr *pgconn.PgError
		if errors.As(data.Err, &pgErr) {
			errorType = pgErr.Code
		}
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", errorType))
	}
	t.duration.Record(ctx, time.Since(info.started).Seconds(), metric.WithAttributes(
		attribute.String("db.operation.name", info.operation), attribute.String("db.collection.name", info.collection), attribute.String("error.type", errorType),
	))
	t.Tracer.TraceQueryEnd(ctx, conn, data)
}
