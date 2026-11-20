package trip

import (
	"errors"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

type Data struct {
	UserID         uuid.UUID `validate:"required" field:"user_id"`
	DriverID       uuid.UUID `validate:"required" field:"driver_id"`
	Price          int64     `validate:"min=0" field:"price"`
	StartLatitude  float64   `validate:"gte=-90,lte=90" field:"start_point.latitude"`
	StartLongitude float64   `validate:"gte=-180,lte=180" field:"start_point.longitude"`
	EndLatitude    float64   `validate:"gte=-90,lte=90" field:"end_point.latitude"`
	EndLongitude   float64   `validate:"gte=-180,lte=180" field:"end_point.longitude"`
}

type Trip struct {
	Data
	ID         uuid.UUID
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
}

var dataValidator = func() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		return field.Tag.Get("field")
	})
	return validate
}()

func (d Data) Validate() error {
	err := dataValidator.Struct(d)
	failures, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}

	failure := failures[0]
	message := "coordinate is out of range"
	switch failure.Tag() {
	case "required":
		message = "must be a non-zero UUID"
	case "min":
		message = "must be non-negative"
	}
	return &ValidationError{Field: failure.Field(), Message: message}
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
