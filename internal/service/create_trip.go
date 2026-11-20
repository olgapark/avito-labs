package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/olgapark/avito-labs/internal/domain"
)

func (s *Service) CreateTrip(
	ctx context.Context,
	trip *domain.Trip,
) error {
	trip.ID = uuid.New()
	trip.Status = "active"
	trip.StartedAt = time.Now().UTC().Truncate(time.Microsecond)

	return s.txManager.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.CreateTrip(ctx, trip); err != nil {
			return err
		}

		if err := s.repo.AddTripStatusHistory(ctx, trip.ID, nil, "active", nil); err != nil {
			return err
		}

		return nil
	})
}
