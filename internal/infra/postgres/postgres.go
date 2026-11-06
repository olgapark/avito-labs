package postgres

import (
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/olgapark/avito-labs/internal/config"
)

func PoolConfig(cfg *config.Config) (*pgxpool.Config, error) {
	pool, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if path := os.Getenv("DATABASE_PASSWORD_FILE"); path != "" {
		password, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read DATABASE_PASSWORD_FILE: %w", err)
		}
		pool.ConnConfig.Password = strings.TrimSpace(string(password))
	}
	pool.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout
	pool.MaxConns = cfg.DatabaseMaxConns
	pool.MinConns = cfg.DatabaseMinConns
	pool.MaxConnLifetime = cfg.DatabaseMaxConnLifetime
	return pool, nil
}
