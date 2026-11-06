//go:build integration

package trip_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/infra/postgres/transaction"
	"github.com/olgapark/avito-labs/internal/model"
	repository "github.com/olgapark/avito-labs/internal/repository/trip"
)

func database(t *testing.T) (context.Context, *pgxpool.Pool, *repository.Repository, *transaction.Manager) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required; run make migrate before integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, repository.New(pool, 3*time.Second), transaction.New(pool, 5*time.Second)
}

func newTrip(t *testing.T, pool *pgxpool.Pool) model.Trip {
	t.Helper()
	id := uuid.New()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", id); err != nil {
			t.Error(err)
		}
	})
	created, err := trip.New(trip.Data{UserID: uuid.New(), DriverID: uuid.New(), Price: 150}, id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func historyCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id=$1", id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestRepositoryLifecycle(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	created := newTrip(t, pool)
	if err := tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Save(ctx, created); err != nil {
			return err
		}
		return repo.AddHistory(ctx, created, nil)
	}); err != nil {
		t.Fatal(err)
	}
	fetched, err := repo.Fetch(ctx, created.ID)
	if err != nil || fetched != created {
		t.Fatalf("fetch: %+v, %v; want %+v", fetched, err, created)
	}
	if count := historyCount(t, ctx, pool, created.ID); count != 1 {
		t.Fatalf("want one initial status, got %d", count)
	}
	busy := newTrip(t, pool)
	busy.DriverID = created.DriverID
	if err := repo.Save(ctx, busy); !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("want driver busy, got %v", err)
	}
	if err := tx.Do(ctx, func(ctx context.Context) error {
		current, err := repo.FetchForUpdate(ctx, created.ID)
		if err != nil {
			return err
		}
		from := current.Status
		if err := current.Finish(time.Now()); err != nil {
			return err
		}
		if err := repo.Update(ctx, current); err != nil {
			return err
		}
		return repo.AddHistory(ctx, current, &from)
	}); err != nil {
		t.Fatal(err)
	}
	fetched, err = repo.Fetch(ctx, created.ID)
	if err != nil || fetched.Status != trip.StatusCompleted || fetched.FinishedAt == nil {
		t.Fatalf("finish: %+v, %v", fetched, err)
	}
	if count := historyCount(t, ctx, pool, created.ID); count != 2 {
		t.Fatalf("want two statuses, got %d", count)
	}
	if err := repo.Save(ctx, busy); err != nil {
		t.Fatalf("driver must be available after finish: %v", err)
	}
	if _, err := repo.Fetch(ctx, uuid.New()); !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestTransactionRollback(t *testing.T) {
	for _, test := range []struct {
		name        string
		failHistory bool
	}{{"error after two inserts", false}, {"second insert fails", true}} {
		t.Run(test.name, func(t *testing.T) {
			ctx, pool, repo, tx := database(t)
			created := newTrip(t, pool)
			forced := errors.New("abort transaction")
			err := tx.Do(ctx, func(ctx context.Context) error {
				if err := repo.Save(ctx, created); err != nil {
					return err
				}
				if test.failHistory {
					created.Status = "invalid"
				}
				if err := repo.AddHistory(ctx, created, nil); err != nil {
					return err
				}
				return forced
			})
			if err == nil {
				t.Fatal("transaction unexpectedly committed")
			}
			if !test.failHistory && !errors.Is(err, forced) {
				t.Fatalf("lost original error: %v", err)
			}
			if _, err := repo.Fetch(ctx, created.ID); !errors.Is(err, trip.ErrTripNotFound) {
				t.Fatalf("trip survived rollback: %v", err)
			}
			if count := historyCount(t, ctx, pool, created.ID); count != 0 {
				t.Fatalf("history survived rollback: %d", count)
			}
		})
	}
}

func TestRollbackOnPanic(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	created := newTrip(t, pool)
	func() {
		defer func() {
			if recover() != "test panic" {
				t.Error("transaction swallowed panic")
			}
		}()
		if err := tx.Do(ctx, func(ctx context.Context) error {
			if err := repo.Save(ctx, created); err != nil {
				return err
			}
			if err := repo.AddHistory(ctx, created, nil); err != nil {
				return err
			}
			panic("test panic")
		}); err != nil {
			t.Fatal(err)
		}
	}()
	if _, err := repo.Fetch(ctx, created.ID); !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("trip survived panic: %v", err)
	}
	if count := historyCount(t, ctx, pool, created.ID); count != 0 {
		t.Fatalf("history survived panic: %d", count)
	}
}

func TestConcurrentCreate(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	first, second := newTrip(t, pool), newTrip(t, pool)
	second.DriverID = first.DriverID
	start, results := make(chan struct{}), make(chan error, 2)
	var workers sync.WaitGroup
	for _, created := range []model.Trip{first, second} {
		workers.Go(func() {
			<-start
			results <- tx.Do(ctx, func(ctx context.Context) error {
				if err := repo.Save(ctx, created); err != nil {
					return err
				}
				return repo.AddHistory(ctx, created, nil)
			})
		})
	}
	close(start)
	workers.Wait()
	close(results)
	success, busy := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, trip.ErrDriverBusy):
			busy++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || busy != 1 {
		t.Fatalf("success=%d, busy=%d", success, busy)
	}
}

func TestNestedTransaction(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	created := newTrip(t, pool)
	forced := errors.New("outer transaction failed")
	err := tx.Do(ctx, func(ctx context.Context) error {
		if err := tx.Do(ctx, func(ctx context.Context) error { return repo.Save(ctx, created) }); err != nil {
			return err
		}
		return fmt.Errorf("outer: %w", forced)
	})
	if !errors.Is(err, forced) {
		t.Fatalf("want outer error, got %v", err)
	}
	if _, err := repo.Fetch(ctx, created.ID); !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("nested transaction committed independently: %v", err)
	}
}

func TestConcurrentFinish(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	created := newTrip(t, pool)
	if err := tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Save(ctx, created); err != nil {
			return err
		}
		return repo.AddHistory(ctx, created, nil)
	}); err != nil {
		t.Fatal(err)
	}
	start, results := make(chan struct{}), make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() {
			<-start
			results <- tx.Do(ctx, func(ctx context.Context) error {
				current, err := repo.FetchForUpdate(ctx, created.ID)
				if err != nil {
					return err
				}
				from := current.Status
				if err := current.Finish(time.Now()); err != nil {
					return err
				}
				if err := repo.Update(ctx, current); err != nil {
					return err
				}
				return repo.AddHistory(ctx, current, &from)
			})
		})
	}
	close(start)
	workers.Wait()
	close(results)
	success, completed := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, trip.ErrTripCompleted):
			completed++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || completed != 1 {
		t.Fatalf("success=%d, completed=%d", success, completed)
	}
	if count := historyCount(t, ctx, pool, created.ID); count != 2 {
		t.Fatalf("duplicate completion history: %d", count)
	}
}
