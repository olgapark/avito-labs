package trip

import (
	"time"

	"github.com/google/uuid"
)

type PositionData struct {
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
}

type Position struct {
	PositionData
	ID     int64
	TripID uuid.UUID
}

func (p PositionData) Validate(now time.Time) error {
	if !(p.Latitude >= -90 && p.Latitude <= 90) {
		return &ValidationError{Field: "coordinates.latitude", Message: coordinateOutOfRange}
	}
	if !(p.Longitude >= -180 && p.Longitude <= 180) {
		return &ValidationError{Field: "coordinates.longitude", Message: coordinateOutOfRange}
	}
	if p.RecordedAt.After(now.Add(time.Minute)) {
		return &ValidationError{Field: "recorded_at", Message: "must be at most one minute in the future"}
	}
	return nil
}
