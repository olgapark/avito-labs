package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/olgapark/avito-labs/internal/config"
)

const (
	httpPortEnv         = "HTTP_PORT"
	databaseMaxConnsEnv = "DATABASE_MAX_CONNS"
	databaseMinConnsEnv = "DATABASE_MIN_CONNS"
	otlpEndpointEnv     = "OTEL_EXPORTER_OTLP_ENDPOINT"
)

func setup(t *testing.T) {
	t.Helper()
	for name, value := range map[string]string{
		httpPortEnv: "8080", "LOG_LEVEL": "info", "SHUTDOWN_TIMEOUT": "10s",
		"DATABASE_URL": "postgres://localhost/tripgo", databaseMaxConnsEnv: "10", databaseMinConnsEnv: "0",
		"DATABASE_MAX_CONN_LIFETIME": "30m", "DATABASE_CONNECT_TIMEOUT": "5s", "DATABASE_QUERY_TIMEOUT": "3s",
		"HTTP_READ_TIMEOUT": "5s", "HTTP_READ_HEADER_TIMEOUT": "2s", "HTTP_WRITE_TIMEOUT": "10s", "HTTP_IDLE_TIMEOUT": "60s",
		"OTEL_SERVICE_NAME": "trip-service", otlpEndpointEnv: "http://localhost:4318", "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
	} {
		t.Setenv(name, value)
	}
}

func TestLoad(t *testing.T) {
	setup(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ShutdownTimeout != 10*time.Second || cfg.DatabaseMaxConns != 10 || cfg.DatabaseMinConns != 0 || cfg.HTTPAddr != ":8080" {
		t.Fatalf("wrong config: %+v", cfg)
	}
}

func TestInvalidEnvironment(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
	}{
		{"DATABASE_URL", ""},
		{httpPortEnv, ""},
		{httpPortEnv, "bad-port"},
		{httpPortEnv, "0"},
		{httpPortEnv, "-1"},
		{httpPortEnv, "65536"},
		{"LOG_LEVEL", "nonsense"},
		{"SHUTDOWN_TIMEOUT", "-1s"},
		{"HTTP_READ_TIMEOUT", "0s"},
		{"HTTP_WRITE_TIMEOUT", "oops"},
		{databaseMaxConnsEnv, "0"},
		{databaseMinConnsEnv, "11"},
		{databaseMinConnsEnv, "-1"},
		{databaseMaxConnsEnv, "invalid"},
		{databaseMaxConnsEnv, "2147483648"},
		{otlpEndpointEnv, ""},
		{otlpEndpointEnv, "bad-url"},
		{"OTEL_EXPORTER_OTLP_PROTOCOL", "unknown"},
	} {
		t.Run(test.name+"="+test.value, func(t *testing.T) {
			setup(t)
			t.Setenv(test.name, test.value)
			_, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), test.name) {
				t.Fatalf("want error for %s, got %v", test.name, err)
			}
		})
	}
}
