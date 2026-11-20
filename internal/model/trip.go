package model

import (
	"github.com/google/uuid"

	"github.com/olgapark/avito-labs/internal/domain/trip"
)

type (
	Trip         = trip.Trip
	CreateTrip   = trip.Data
	Position     = trip.Position
	PositionData = trip.PositionData
)

type StaleTrip struct {
	ID       uuid.UUID
	DriverID uuid.UUID
}
