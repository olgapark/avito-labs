package create_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
	"github.com/olgapark/avito-labs/internal/usecase/trip/create"
)

func TestExecute(t *testing.T) {
	dbErr := errors.New("database failed")
	for _, test := range []struct {
		name       string
		saveErr    error
		historyErr error
		txErr      error
	}{
		{name: "success"},
		{name: "driver busy", saveErr: trip.ErrDriverBusy},
		{name: "repository error", saveErr: dbErr},
		{name: "history error", historyErr: dbErr},
		{name: "commit error", txErr: dbErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo, tx, metrics := NewMocktripRepository(ctrl), NewMocktxManager(ctrl), NewMockmetrics(ctrl)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			txCtx, txCancel := context.WithCancel(ctx)
			defer txCancel()
			now, id := time.Now().UTC().Truncate(time.Microsecond), uuid.New()
			input := model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New(), Price: 100}
			want := model.Trip{Data: input, ID: id, Status: trip.StatusActive, StartedAt: now}
			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				if err := fn(txCtx); err != nil {
					return err
				}
				return test.txErr
			})
			repo.EXPECT().Save(txCtx, want).Return(test.saveErr)
			if test.saveErr == nil {
				repo.EXPECT().AddHistory(txCtx, want, nil).Return(test.historyErr)
			}
			wantErr := errors.Join(test.saveErr, test.historyErr, test.txErr)
			if wantErr == nil {
				metrics.EXPECT().Created(ctx)
			}
			u := create.New(repo, tx, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics, func() time.Time { return now }, func() uuid.UUID { return id })
			got, err := u.Execute(ctx, input)
			if wantErr == nil {
				if err != nil || got != want {
					t.Fatalf("got %+v, %v; want %+v", got, err, want)
				}
			} else {
				for _, expected := range []error{test.saveErr, test.historyErr, test.txErr} {
					if expected != nil && !errors.Is(err, expected) {
						t.Fatalf("want %v, got %v", expected, err)
					}
				}
				if got != (model.Trip{}) {
					t.Fatalf("returned unsaved trip: %+v", got)
				}
			}
		})
	}
}

func TestInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	u := create.New(NewMocktripRepository(ctrl), NewMocktxManager(ctrl), slog.New(slog.NewTextHandler(io.Discard, nil)), NewMockmetrics(ctrl), time.Now, uuid.New)
	_, err := u.Execute(context.Background(), model.CreateTrip{})
	var validation *trip.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("want validation error, got %v", err)
	}
}
