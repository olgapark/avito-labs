package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/olgapark/avito-labs/internal/domain"
)

func (s *Service) GetTrip(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	return s.repo.GetTrip(ctx, id)
}
