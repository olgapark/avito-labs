package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/olgapark/avito-labs/internal/http/domainerror"
)

func Tracing(router chi.Routes, provider trace.TracerProvider, next http.Handler) http.Handler {
	traced := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("http.route", r.Pattern))
		if state := domainerror.RequestState(r.Context()); state != nil {
			state.Context = r.Context()
		}
		completed := false
		defer func() {
			if !completed {
				span.SetStatus(codes.Error, "internal_error")
				span.SetAttributes(attribute.String("error.type", "panic"), attribute.Int("http.response.status_code", 500))
			}
		}()
		next.ServeHTTP(w, r)
		completed = true
	}), "http request", otelhttp.WithTracerProvider(provider),
		otelhttp.WithMeterProvider(noop.NewMeterProvider()),
		otelhttp.WithPropagators(propagation.TraceContext{}),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return r.Method + " " + r.Pattern }))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := chi.NewRouteContext()
		r.Pattern = "unmatched"
		if router.Match(route, r.Method, r.URL.Path) {
			r.Pattern = route.RoutePattern()
		}
		traced.ServeHTTP(w, r)
	})
}
