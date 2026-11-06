package trip

import (
	"math"
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

type Data struct {
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
}

type Trip struct {
	Data
	ID         uuid.UUID
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
}

func (d Data) Validate() error {
	if d.UserID == uuid.Nil {
		return &ValidationError{Field: "user_id", Message: "must be a non-zero UUID"}
	}
	if d.DriverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Message: "must be a non-zero UUID"}
	}
	if d.Price < 0 {
		return &ValidationError{Field: "price", Message: "must be non-negative"}
	}
	coordinates := []struct {
		field string
		value float64
		limit float64
	}{
		{"start_point.latitude", d.StartLatitude, 90},
		{"start_point.longitude", d.StartLongitude, 180},
		{"end_point.latitude", d.EndLatitude, 90},
		{"end_point.longitude", d.EndLongitude, 180},
	}
	for _, coordinate := range coordinates {
		if math.IsNaN(coordinate.value) || math.IsInf(coordinate.value, 0) || math.Abs(coordinate.value) > coordinate.limit {
			return &ValidationError{Field: coordinate.field, Message: "coordinate is out of range"}
		}
	}
	return nil
}

func New(data Data, id uuid.UUID, at time.Time) (Trip, error) {
	if err := data.Validate(); err != nil {
		return Trip{}, err
	}
	return Trip{Data: data, ID: id, Status: StatusActive, StartedAt: at.UTC().Truncate(time.Microsecond)}, nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	finishedAt := at.UTC().Truncate(time.Microsecond)
	t.Status = StatusCompleted
	t.FinishedAt = &finishedAt
	return nil
}
