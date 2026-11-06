package postgres_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olgapark/avito-labs/internal/config"
	"github.com/olgapark/avito-labs/internal/infra/postgres"
)

func TestPoolConfigPassword(t *testing.T) {
	const secret = "secret:@/?#"
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte(secret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		file string
		want string
	}{
		{name: "URL password", want: "url-password"},
		{name: "secret overrides URL", file: path, want: secret},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DATABASE_PASSWORD_FILE", test.file)
			pool, err := postgres.PoolConfig(&config.Config{DatabaseURL: "postgres://tripgo:url-password@localhost/tripgo"})
			if err != nil {
				t.Fatal(err)
			}
			if pool.ConnConfig.Password != test.want {
				t.Fatal("unexpected database password")
			}
		})
	}
}

func TestPoolConfigMissingPasswordFile(t *testing.T) {
	t.Setenv("DATABASE_PASSWORD_FILE", filepath.Join(t.TempDir(), "missing"))
	_, err := postgres.PoolConfig(&config.Config{DatabaseURL: "postgres://tripgo@localhost/tripgo"})
	if err == nil || !strings.Contains(err.Error(), "DATABASE_PASSWORD_FILE") {
		t.Fatalf("want password file error, got %v", err)
	}
}
