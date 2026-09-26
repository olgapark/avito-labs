package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/olgapark/avito-labs/internal/config"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/handler"
	"github.com/olgapark/avito-labs/internal/postgres"
	"github.com/olgapark/avito-labs/internal/repository"
	"github.com/olgapark/avito-labs/internal/service"
	"github.com/olgapark/avito-labs/internal/txmanager"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPosgresPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.New(pool, config.DatabaseQueryTimeout)
	txManager := txmanager.New(pool, config.DatabaseQueryTimeout)
	svc := service.New(repo, txManager)
	h := handler.New(pool, svc, config.DatabaseQueryTimeout)

	router := chi.NewRouter()
	api.HandlerWithOptions(
		h,
		api.ChiServerOptions{
			BaseRouter:       router,
			ErrorHandlerFunc: handler.HandleAPIError,
		},
	)

	server := &http.Server{
		Addr:              config.HTTPAddr,
		Handler:           router,
		ReadTimeout:       config.HTTPReadTimeout,
		ReadHeaderTimeout: config.HTTPReadHeaderTimeout,
		WriteTimeout:      config.HTTPWriteTimeout,
		IdleTimeout:       config.HTTPIdleTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	pool.Close()
}
