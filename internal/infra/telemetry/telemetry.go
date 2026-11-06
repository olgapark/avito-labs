package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/olgapark/avito-labs/internal/config"
)

type Providers struct {
	Traces *sdktrace.TracerProvider
	Meters *sdkmetric.MeterProvider
}

func New(ctx context.Context, cfg *config.Config, version string) (*Providers, error) {
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(
		attribute.String("service.name", cfg.OTELServiceName), attribute.String("service.version", version),
	))
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}
	traceExporter, metricExporter, err := exporters(ctx, cfg.OTELProtocol)
	if err != nil {
		return nil, err
	}
	return &Providers{
		Traces: sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExporter)),
		Meters: sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter))),
	}, nil
}

func exporters(ctx context.Context, protocol string) (sdktrace.SpanExporter, sdkmetric.Exporter, error) {
	var traces sdktrace.SpanExporter
	var meters sdkmetric.Exporter
	var err error
	if protocol == "grpc" {
		traces, err = otlptracegrpc.New(ctx)
	} else {
		traces, err = otlptracehttp.New(ctx)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("create trace exporter: %w", err)
	}
	if protocol == "grpc" {
		meters, err = otlpmetricgrpc.New(ctx)
	} else {
		meters, err = otlpmetrichttp.New(ctx)
	}
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("create metric exporter: %w", err), traces.Shutdown(ctx))
	}
	return traces, meters, nil
}

func (p *Providers) Shutdown(ctx context.Context) error {
	return errors.Join(p.Meters.Shutdown(ctx), p.Traces.Shutdown(ctx))
}
