package finish_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
	"github.com/olgapark/avito-labs/internal/usecase/trip/finish"
)

func TestExecute(t *testing.T) {
	dbErr := errors.New("database failed")
	for _, test := range []struct {
		name       string
		completed  bool
		fetchErr   error
		updateErr  error
		historyErr error
		txErr      error
		wantErr    error
	}{
		{name: "success"},
		{name: "completed", completed: true, wantErr: trip.ErrTripCompleted},
		{name: "not found", fetchErr: trip.ErrTripNotFound, wantErr: trip.ErrTripNotFound},
		{name: "fetch error", fetchErr: dbErr, wantErr: dbErr},
		{name: "update error", updateErr: dbErr, wantErr: dbErr},
		{name: "history error", historyErr: dbErr, wantErr: dbErr},
		{name: "commit error", txErr: dbErr, wantErr: dbErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo, tx, metrics := NewMocktripRepository(ctrl), NewMocktxManager(ctrl), NewMockmetrics(ctrl)
			ctx := context.Background()
			txCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			now, id := time.Now().UTC().Truncate(time.Microsecond), uuid.New()
			current := model.Trip{ID: id, Status: trip.StatusActive}
			if test.completed {
				current.Status = trip.StatusCompleted
			}
			want := model.Trip{ID: id, Status: trip.StatusCompleted, FinishedAt: &now}
			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				if err := fn(txCtx); err != nil {
					return err
				}
				return test.txErr
			})
			repo.EXPECT().FetchForUpdate(txCtx, id).Return(current, test.fetchErr)
			if test.fetchErr == nil && !test.completed {
				repo.EXPECT().Update(txCtx, want).Return(test.updateErr)
				if test.updateErr == nil {
					from := trip.StatusActive
					repo.EXPECT().AddHistory(txCtx, want, &from).Return(test.historyErr)
				}
			}
			if test.wantErr == nil {
				metrics.EXPECT().Completed(ctx)
			}
			u := finish.New(repo, tx, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics, func() time.Time { return now })
			got, err := u.Execute(ctx, id)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v, want %+v", got, want)
			}
			if test.wantErr != nil && got != (model.Trip{}) {
				t.Fatalf("returned uncommitted trip: %+v", got)
			}
		})
	}
}
