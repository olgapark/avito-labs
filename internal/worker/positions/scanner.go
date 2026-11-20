package positions

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/olgapark/avito-labs/internal/model"
)

func (p *Pool) scan(ctx context.Context, ticker Ticker) {
	defer ticker.Stop()
	ticks := ticker.Ticks()
	for {
		select {
		case <-ctx.Done():
			return
		case _, open := <-ticks:
			if !open || ctx.Err() != nil {
				return
			}
			p.scanOnce(ctx)
		}
	}
}

func (p *Pool) scanOnce(ctx context.Context) {
	ctx, span := p.tracer.Start(ctx, "positions.scan")
	defer span.End()
	p.mu.Lock()
	p.savedDuringScan = make(map[uuid.UUID]struct{})
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.savedDuringScan = nil
		p.mu.Unlock()
	}()

	trips, err := p.repo.ListStaleTrips(ctx, p.cfg.StaleAfter)
	if err != nil {
		if ctx.Err() == nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "list stale trips")
			p.logger.WarnContext(ctx, "scan stale trips failed", "error", err)
		}
		return
	}
	for _, trip := range trips {
		if ctx.Err() != nil {
			return
		}
		result := p.enqueue(trip, span.SpanContext())
		if result == "" {
			continue
		}
		p.metrics.StaleCheck(ctx, result)
		if result == "dropped" {
			p.logger.WarnContext(ctx, "position request queue full", "trip_id", trip.ID)
		}
	}
}

func (p *Pool) enqueue(trip model.StaleTrip, scan trace.SpanContext) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pending[trip.ID]; ok {
		return "duplicate"
	}
	if _, ok := p.savedDuringScan[trip.ID]; ok {
		return ""
	}
	task := &job{trip: trip, scan: scan}
	select {
	case p.queue <- task:
		p.pending[trip.ID] = task
		return "queued"
	default:
		return "dropped"
	}
}
