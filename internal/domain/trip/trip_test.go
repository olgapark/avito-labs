package trip_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/domain/trip"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(*trip.Data)
		field  string
	}{
		{"valid", func(*trip.Data) {}, ""},
		{"zero user", func(d *trip.Data) { d.UserID = uuid.Nil }, "user_id"},
		{"zero driver", func(d *trip.Data) { d.DriverID = uuid.Nil }, "driver_id"},
		{"negative price", func(d *trip.Data) { d.Price = -1 }, "price"},
		{"start latitude", func(d *trip.Data) { d.StartLatitude = 91 }, "start_point.latitude"},
		{"start longitude", func(d *trip.Data) { d.StartLongitude = -181 }, "start_point.longitude"},
		{"end latitude", func(d *trip.Data) { d.EndLatitude = -91 }, "end_point.latitude"},
		{"end longitude", func(d *trip.Data) { d.EndLongitude = 181 }, "end_point.longitude"},
		{"NaN", func(d *trip.Data) { d.EndLatitude = math.NaN() }, "end_point.latitude"},
		{"infinity", func(d *trip.Data) { d.StartLatitude = math.Inf(1) }, "start_point.latitude"},
		{"boundaries", func(d *trip.Data) {
			d.StartLatitude = -90
			d.EndLatitude = 90
			d.StartLongitude = -180
			d.EndLongitude = 180
		}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := trip.Data{UserID: uuid.New(), DriverID: uuid.New()}
			test.change(&data)
			err := data.Validate()
			if test.field == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var validation *trip.ValidationError
			if !errors.As(err, &validation) || validation.Field != test.field || validation.Error() == "" {
				t.Fatalf("want validation error for %s, got %v", test.field, err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	id, now := uuid.New(), time.Now()
	data := trip.Data{UserID: uuid.New(), DriverID: uuid.New(), Price: 10}
	created, err := trip.New(data, id, now)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != id || created.Data != data || created.Status != trip.StatusActive || created.FinishedAt != nil || !created.StartedAt.Equal(now.Truncate(time.Microsecond)) {
		t.Fatalf("unexpected trip: %+v", created)
	}
	if _, err := trip.New(trip.Data{}, id, now); err == nil {
		t.Fatal("invalid trip accepted")
	}
}

func TestFinish(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	for _, status := range []string{trip.StatusActive, trip.StatusCompleted} {
		t.Run(status, func(t *testing.T) {
			current := trip.Trip{Status: status}
			err := current.Finish(now)
			if status == trip.StatusCompleted {
				if !errors.Is(err, trip.ErrTripCompleted) || current.FinishedAt != nil {
					t.Fatalf("completed trip changed: %+v, %v", current, err)
				}
				return
			}
			if err != nil || current.Status != trip.StatusCompleted || current.FinishedAt == nil || !current.FinishedAt.Equal(now) {
				t.Fatalf("finish result: %+v, %v", current, err)
			}
		})
	}
}
