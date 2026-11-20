package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/olgapark/avito-labs/internal/config"
	"github.com/olgapark/avito-labs/internal/container"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := container.New(ctx, cfg)
	if err != nil {
		return err
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- app.Server.ListenAndServe() }()
	app.Logger.InfoContext(ctx, "service started", "addr", cfg.HTTPAddr)
	select {
	case <-ctx.Done():
		stop()
	case err = <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- app.Shutdown(shutdownCtx) }()
	var shutdownErr error
	select {
	case shutdownErr = <-shutdownDone:
	case <-shutdownCtx.Done():
		shutdownErr = shutdownCtx.Err()
	}
	if shutdownErr != nil {
		app.Logger.ErrorContext(shutdownCtx, "service shutdown failed", "error", shutdownErr)
		return errors.Join(err, shutdownErr)
	}
	app.Logger.InfoContext(shutdownCtx, "service stopped")
	return err
}
