package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                string
	LogLevel                string
	ShutdownTimeout         time.Duration
	DatabaseURL             string
	DatabasePassword        *string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	HTTPReadTimeout         time.Duration
	HTTPReadHeaderTimeout   time.Duration
	HTTPWriteTimeout        time.Duration
	HTTPIdleTimeout         time.Duration
}

func Load() (*Config, error) {
	config := &Config{}
	var err error

	config.HTTPAddr, err = requiredEnv("HTTP_ADDR")
	if err != nil {
		return nil, err
	}

	config.LogLevel, err = requiredEnv("LOG_LEVEL")
	if err != nil {
		return nil, err
	}

	config.ShutdownTimeout, err = durationEnv("SHUTDOWN_TIMEOUT")
	if err != nil {
		return nil, err
	}

	config.DatabaseURL, err = requiredEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	if path := os.Getenv("DATABASE_PASSWORD_FILE"); path != "" {
		password, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read DATABASE_PASSWORD_FILE: %w", err)
		}
		value := string(password)
		config.DatabasePassword = &value
	}

	config.DatabaseMaxConns, err = int32Env("DATABASE_MAX_CONNS")
	if err != nil {
		return nil, err
	}

	config.DatabaseMinConns, err = int32Env("DATABASE_MIN_CONNS")
	if err != nil {
		return nil, err
	}

	config.DatabaseMaxConnLifetime, err = durationEnv("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return nil, err
	}

	config.DatabaseConnectTimeout, err = durationEnv("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return nil, err
	}

	config.DatabaseQueryTimeout, err = durationEnv("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return nil, err
	}

	config.HTTPReadTimeout, err = durationEnv("HTTP_READ_TIMEOUT")
	if err != nil {
		return nil, err
	}

	config.HTTPReadHeaderTimeout, err = durationEnv("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return nil, err
	}

	config.HTTPWriteTimeout, err = durationEnv("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return nil, err
	}

	config.HTTPIdleTimeout, err = durationEnv("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return nil, err
	}

	if config.DatabaseMaxConns <= 0 || config.DatabaseMinConns < 0 || config.DatabaseMinConns > config.DatabaseMaxConns {
		return nil, fmt.Errorf("require DATABASE_MAX_CONNS > 0 and 0 <= DATABASE_MIN_CONNS <= DATABASE_MAX_CONNS")
	}

	return config, nil

}

func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is required", name)
	}

	return value, nil
}

func durationEnv(name string) (time.Duration, error) {
	value, err := requiredEnv(name)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration == 0 {
		return 0, fmt.Errorf("%s must not be zero", name)
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
