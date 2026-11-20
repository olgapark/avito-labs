package positions

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/olgapark/avito-labs/internal/model"
)

type Config struct {
	StaleAfter  time.Duration
	PushTimeout time.Duration
	Workers     int
	QueueSize   int
}

type job struct {
	trip    model.StaleTrip
	scan    trace.SpanContext
	started bool
}

type Pool struct {
	cfg     Config
	repo    repository
	push    pushService
	metrics metrics
	logger  *slog.Logger
	tracer  trace.Tracer
	queue   chan *job
	active  atomic.Int64

	mu              sync.Mutex
	pending         map[uuid.UUID]*job
	savedDuringScan map[uuid.UUID]struct{}

	stopScan    context.CancelFunc
	stopWorkers context.CancelFunc
	done        chan struct{}
}

func New(cfg Config, repo repository, push pushService, metrics metrics, logger *slog.Logger, tracer trace.Tracer) *Pool {
	return &Pool{
		cfg: cfg, repo: repo, push: push, metrics: metrics, logger: logger, tracer: tracer,
		queue: make(chan *job, cfg.QueueSize), pending: make(map[uuid.UUID]*job), done: make(chan struct{}),
	}
}

type systemTicker struct{ *time.Ticker }

func (t systemTicker) Ticks() <-chan time.Time { return t.C }

func NewTicker(interval time.Duration) Ticker { return systemTicker{time.NewTicker(interval)} }

func (p *Pool) Start(ctx context.Context, ticker Ticker) {
	scanCtx, stopScan := context.WithCancel(ctx)
	workerCtx, stopWorkers := context.WithCancel(context.WithoutCancel(ctx))
	p.stopScan, p.stopWorkers = stopScan, stopWorkers
	var workers sync.WaitGroup
	for range p.cfg.Workers {
		workers.Go(func() { p.work(workerCtx) })
	}
	go func() {
		defer close(p.done)
		defer stopWorkers()
		defer stopScan()
		p.scan(scanCtx, ticker)
		close(p.queue)
		workers.Wait()
		for range p.queue {
		}
		p.mu.Lock()
		clear(p.pending)
		p.mu.Unlock()
	}()
}

func (p *Pool) Shutdown(ctx context.Context) error {
	p.stopScan()
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		p.stopWorkers()
		return ctx.Err()
	}
}

func (p *Pool) QueueSize() int64     { return int64(len(p.queue)) }
func (p *Pool) ActiveWorkers() int64 { return p.active.Load() }

func (p *Pool) Cancel(ctx context.Context, id uuid.UUID) {
	p.mu.Lock()
	if p.savedDuringScan != nil {
		p.savedDuringScan[id] = struct{}{}
	}
	task, exists := p.pending[id]
	cancelled := exists && !task.started
	if cancelled {
		delete(p.pending, id)
	}
	p.mu.Unlock()
	if cancelled {
		p.metrics.RequestDone(ctx, "cancelled")
	}
}
