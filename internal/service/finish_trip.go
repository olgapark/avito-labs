package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/olgapark/avito-labs/internal/domain"
)

func (s *Service) FinishTrip(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Trip, error) {
	finishedAt := time.Now().UTC()

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.FinishTrip(ctx, id, finishedAt); err != nil {
			return err
		}

		fromStatus := "active"
		reason := "trip finished"

		if err := s.repo.AddTripStatusHistory(ctx, id, &fromStatus, "completed", &reason); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.repo.GetTrip(ctx, id)
}
