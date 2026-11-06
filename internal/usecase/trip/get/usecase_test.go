package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	"github.com/olgapark/avito-labs/internal/model"
	"github.com/olgapark/avito-labs/internal/usecase/trip/get"
)

func TestExecute(t *testing.T) {
	for _, wantErr := range []error{nil, trip.ErrTripNotFound, errors.New("database failed")} {
		name := "success"
		if wantErr != nil {
			name = wantErr.Error()
		}
		t.Run(name, func(t *testing.T) {
			repo := NewMocktripRepository(gomock.NewController(t))
			ctx, id := context.Background(), uuid.New()
			want := model.Trip{ID: id, Status: trip.StatusActive}
			repo.EXPECT().Fetch(ctx, id).Return(want, wantErr)
			got, err := get.New(repo).Execute(ctx, id)
			if !errors.Is(err, wantErr) {
				t.Fatalf("got %v, want %v", err, wantErr)
			}
			if wantErr == nil && got != want {
				t.Fatalf("got %+v, want %+v", got, want)
			}
		})
	}
}
