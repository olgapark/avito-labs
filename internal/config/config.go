package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                string
	LogLevel                string
	ShutdownTimeout         time.Duration
	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	HTTPReadTimeout         time.Duration
	HTTPReadHeaderTimeout   time.Duration
	HTTPWriteTimeout        time.Duration
	HTTPIdleTimeout         time.Duration
	OTELServiceName         string
	OTELProtocol            string
	PositionCheckInterval   time.Duration
	PositionStaleAfter      time.Duration
	PositionWorkers         int
	PositionQueueSize       int
	PushHTTPURL             *url.URL
	PushRequestTimeout      time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{}
	for _, variable := range []struct {
		name   string
		target *string
	}{
		{"LOG_LEVEL", &cfg.LogLevel},
		{"DATABASE_URL", &cfg.DatabaseURL},
		{"OTEL_SERVICE_NAME", &cfg.OTELServiceName},
		{"OTEL_EXPORTER_OTLP_PROTOCOL", &cfg.OTELProtocol},
	} {
		value, err := requiredEnv(variable.name)
		if err != nil {
			return nil, err
		}
		*variable.target = value
	}
	addr, err := httpAddr()
	if err != nil {
		return nil, err
	}
	cfg.HTTPAddr = addr
	for _, variable := range []struct {
		name     string
		fallback time.Duration
		target   *time.Duration
	}{
		{"SHUTDOWN_TIMEOUT", 0, &cfg.ShutdownTimeout},
		{"DATABASE_MAX_CONN_LIFETIME", 0, &cfg.DatabaseMaxConnLifetime},
		{"DATABASE_CONNECT_TIMEOUT", 0, &cfg.DatabaseConnectTimeout},
		{"DATABASE_QUERY_TIMEOUT", 0, &cfg.DatabaseQueryTimeout},
		{"HTTP_READ_TIMEOUT", 10 * time.Second, &cfg.HTTPReadTimeout},
		{"HTTP_READ_HEADER_TIMEOUT", 5 * time.Second, &cfg.HTTPReadHeaderTimeout},
		{"HTTP_WRITE_TIMEOUT", 10 * time.Second, &cfg.HTTPWriteTimeout},
		{"HTTP_IDLE_TIMEOUT", time.Minute, &cfg.HTTPIdleTimeout},
		{"POSITION_CHECK_INTERVAL", 30 * time.Second, &cfg.PositionCheckInterval},
		{"POSITION_STALE_AFTER", time.Minute, &cfg.PositionStaleAfter},
		{"PUSH_REQUEST_TIMEOUT", time.Second, &cfg.PushRequestTimeout},
	} {
		value, err := durationEnv(variable.name, variable.fallback)
		if err != nil {
			return nil, err
		}
		*variable.target = value
	}
	cfg.DatabaseMaxConns, err = int32Env("DATABASE_MAX_CONNS")
	if err != nil {
		return nil, err
	}
	cfg.DatabaseMinConns, err = int32Env("DATABASE_MIN_CONNS")
	if err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if err := cfg.loadPositions(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) loadPositions() error {
	for _, variable := range []struct {
		name   string
		value  int
		target *int
	}{
		{"POSITION_WORKERS", 4, &c.PositionWorkers},
		{"POSITION_QUEUE_SIZE", 100, &c.PositionQueueSize},
	} {
		*variable.target = variable.value
		if os.Getenv(variable.name) != "" {
			value, err := int32Env(variable.name)
			if err != nil {
				return err
			}
			if value <= 0 {
				return fmt.Errorf("%s must be positive", variable.name)
			}
			*variable.target = int(value)
		}
	}
	endpoint, err := requiredEnv("PUSH_HTTP_URL")
	if err != nil {
		return err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("PUSH_HTTP_URL must be an http(s) URL")
	}
	c.PushHTTPURL = parsed
	return nil
}

func (c *Config) validate() error {
	if c.DatabaseMaxConns <= 0 || c.DatabaseMinConns < 0 || c.DatabaseMinConns > c.DatabaseMaxConns {
		return fmt.Errorf("require DATABASE_MAX_CONNS > 0 and 0 <= DATABASE_MIN_CONNS <= DATABASE_MAX_CONNS")
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		return fmt.Errorf("parse LOG_LEVEL: %w", err)
	}
	endpoint, err := requiredEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if err != nil {
		return err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT must be an http(s) URL")
	}
	if c.OTELProtocol != "http/protobuf" && c.OTELProtocol != "grpc" {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be http/protobuf or grpc")
	}
	return nil
}

func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is required", name)
	}

	return value, nil
}

func httpAddr() (string, error) {
	if os.Getenv("HTTP_PORT") != "" {
		port, err := int32Env("HTTP_PORT")
		if err != nil {
			return "", err
		}
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("HTTP_PORT must be between 1 and 65535")
		}
		return net.JoinHostPort("", strconv.Itoa(int(port))), nil
	}
	addr, err := requiredEnv("HTTP_ADDR")
	if err != nil {
		return "", err
	}
	_, port, splitErr := net.SplitHostPort(addr)
	number, parseErr := strconv.ParseUint(port, 10, 16)
	if splitErr != nil || parseErr != nil || number == 0 {
		return "", fmt.Errorf("HTTP_ADDR must be a host:port address with a port between 1 and 65535")
	}
	return addr, nil
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		if fallback > 0 {
			return fallback, nil
		}
		return 0, fmt.Errorf("environment variable %s is required", name)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return duration, nil
}

func int32Env(name string) (int32, error) {
	value, err := requiredEnv(name)
	if err != nil {
		return 0, err
	}

	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s %w", name, err)
	}

	return int32(number), nil
}
