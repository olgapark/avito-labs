package positions

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (p *Pool) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.queue:
			if !ok || ctx.Err() != nil {
				return
			}
			if p.begin(task) {
				p.request(ctx, task)
			}
		}
	}
}

func (p *Pool) begin(task *job) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pending[task.trip.ID] != task {
		return false
	}
	task.started = true
	p.active.Add(1)
	return true
}

func (p *Pool) request(ctx context.Context, task *job) {
	defer func() {
		p.mu.Lock()
		delete(p.pending, task.trip.ID)
		p.active.Add(-1)
		p.mu.Unlock()
	}()
	ctx, span := p.tracer.Start(ctx, "positions.request", trace.WithNewRoot(),
		trace.WithLinks(trace.Link{SpanContext: task.scan}))
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, p.cfg.PushTimeout)
	defer cancel()
	started := time.Now()
	err := p.push.RequestPosition(ctx, task.trip.ID, task.trip.DriverID)
	elapsed := time.Since(started)
	result := "success"
	if err != nil {
		result = "error"
		span.RecordError(err)
		span.SetStatus(codes.Error, "request position")
		p.logger.WarnContext(ctx, "request position failed", "trip_id", task.trip.ID, "error", err)
	}
	p.metrics.RequestDone(ctx, result)
	p.metrics.PushDuration(ctx, result, elapsed)
}
