package positions_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"

	"github.com/olgapark/avito-labs/internal/model"
	"github.com/olgapark/avito-labs/internal/worker/positions"
)

const staleAfter = time.Minute

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func TestPool_DeduplicatesQueuedTrip(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 2)
		release := f.occupyWorker(t)
		waiting := staleTrip()
		f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).Return([]model.StaleTrip{waiting}, nil).Times(2)
		f.metrics.EXPECT().StaleCheck(gomock.Any(), "queued")
		f.metrics.EXPECT().StaleCheck(gomock.Any(), "duplicate")
		f.push.EXPECT().RequestPosition(gomock.Any(), waiting.ID, waiting.DriverID).Return(nil)
		f.expectRequests("success", 1)

		f.tick()
		f.tick()

		assert.EqualValues(t, 1, f.pool.QueueSize())
		release()
		require.NoError(t, f.pool.Shutdown(t.Context()))
	})
}

func TestPool_CancelsQueuedRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		release := f.occupyWorker(t)
		waiting := staleTrip()
		f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).Return([]model.StaleTrip{waiting}, nil)
		f.metrics.EXPECT().StaleCheck(gomock.Any(), "queued")
		f.metrics.EXPECT().RequestDone(gomock.Any(), "cancelled")
		f.tick()
		require.EqualValues(t, 1, f.pool.QueueSize())

		f.pool.Cancel(t.Context(), waiting.ID)
		release()

		require.NoError(t, f.pool.Shutdown(t.Context()))
		assert.Zero(t, f.pool.QueueSize())
	})
}

func TestPool_DropsWhenQueueIsFull(t *testing.T) {
	tests := []struct {
		name        string
		queueSize   int
		wantDropped int
	}{
		{name: "одна задача сверх ёмкости", queueSize: 1, wantDropped: 1},
		{name: "несколько задач сверх ёмкости", queueSize: 2, wantDropped: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := startPool(t, 1, tt.queueSize)
				release := f.occupyWorker(t)
				tasks := make([]model.StaleTrip, tt.queueSize+tt.wantDropped)
				for i := range tasks {
					tasks[i] = staleTrip()
				}
				for _, task := range tasks[:tt.queueSize] {
					f.push.EXPECT().RequestPosition(gomock.Any(), task.ID, task.DriverID).Return(nil)
				}
				scannedAgain := make(chan struct{})
				gomock.InOrder(
					f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).Return(tasks, nil),
					f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).
						DoAndReturn(func(context.Context, time.Duration) ([]model.StaleTrip, error) {
							close(scannedAgain)
							return nil, nil
						}),
				)
				f.metrics.EXPECT().StaleCheck(gomock.Any(), "queued").Times(tt.queueSize)
				f.metrics.EXPECT().StaleCheck(gomock.Any(), "dropped").Times(tt.wantDropped)
				f.expectRequests("success", tt.queueSize)

				f.tick()
				f.tick()

				select {
				case <-scannedAgain:
				default:
					t.Fatal("сканер не обработал следующий тик при заполненной очереди")
				}
				assert.EqualValues(t, tt.queueSize, f.pool.QueueSize())
				release()
				require.NoError(t, f.pool.Shutdown(t.Context()))
			})
		})
	}
}

func TestPool_ProcessesTasksWithConfiguredWorkers(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		tasks   int
	}{
		{name: "один воркер обрабатывает три задачи", workers: 1, tasks: 3},
		{name: "три воркера обрабатывают семь задач", workers: 3, tasks: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := startPool(t, tt.workers, tt.tasks)
				hold, release := context.WithCancel(t.Context())
				t.Cleanup(release)
				tasks := make([]model.StaleTrip, tt.tasks)
				for i := range tasks {
					task := staleTrip()
					tasks[i] = task
					f.push.EXPECT().RequestPosition(gomock.Any(), task.ID, task.DriverID).
						DoAndReturn(waitForRelease(hold.Done()))
				}
				f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).Return(tasks, nil)
				f.metrics.EXPECT().StaleCheck(gomock.Any(), "queued").Times(tt.tasks)
				f.expectRequests("success", tt.tasks)

				f.tick()

				assert.EqualValues(t, tt.workers, f.pool.ActiveWorkers())
				assert.EqualValues(t, tt.tasks-tt.workers, f.pool.QueueSize())
				release()
				require.NoError(t, f.pool.Shutdown(t.Context()))
			})
		})
	}
}

func TestPool_StopsOnContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		release := f.occupyWorker(t)
		waiting := staleTrip()
		f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).Return([]model.StaleTrip{waiting}, nil)
		f.metrics.EXPECT().StaleCheck(gomock.Any(), "queued")
		f.push.EXPECT().RequestPosition(gomock.Any(), waiting.ID, waiting.DriverID).Return(nil)
		f.expectRequests("success", 1)
		f.tick()

		f.cancel()
		synctest.Wait()
		select {
		case <-f.ticker.stopped:
		default:
			t.Fatal("сканер не остановил тикер после отмены контекста")
		}
		release()

		require.NoError(t, f.pool.Shutdown(t.Context()))
		assert.Zero(t, f.pool.ActiveWorkers())
		assert.Zero(t, f.pool.QueueSize())
	})
}

func TestPool_ContinuesAfterPushError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "сервис пушей недоступен", err: errors.New("push unavailable")},
		{name: "истёк таймаут вызова", err: context.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := startPool(t, 1, 2)
				first, second := staleTrip(), staleTrip()
				f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).
					Return([]model.StaleTrip{first, second}, nil)
				gomock.InOrder(
					f.push.EXPECT().RequestPosition(gomock.Any(), first.ID, first.DriverID).Return(tt.err),
					f.push.EXPECT().RequestPosition(gomock.Any(), second.ID, second.DriverID).Return(nil),
				)
				f.metrics.EXPECT().StaleCheck(gomock.Any(), "queued").Times(2)
				f.expectRequests("error", 1)
				f.expectRequests("success", 1)

				f.tick()

				require.NoError(t, f.pool.Shutdown(t.Context()))
			})
		})
	}
}

type manualTicker struct {
	ticks   chan time.Time
	stopped chan struct{}
}

func (t *manualTicker) Ticks() <-chan time.Time { return t.ticks }
func (t *manualTicker) Stop()                   { close(t.stopped) }

type fixture struct {
	pool    *positions.Pool
	repo    *Mockrepository
	push    *MockpushService
	metrics *Mockmetrics
	ticker  *manualTicker
	cancel  context.CancelFunc
}

func startPool(t *testing.T, workers, size int) fixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	f := fixture{
		repo: NewMockrepository(ctrl), push: NewMockpushService(ctrl), metrics: NewMockmetrics(ctrl),
		ticker: &manualTicker{ticks: make(chan time.Time, 1), stopped: make(chan struct{})},
	}
	f.pool = positions.New(positions.Config{StaleAfter: staleAfter, PushTimeout: time.Second, Workers: workers, QueueSize: size},
		f.repo, f.push, f.metrics, slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewTracerProvider().Tracer("test"))
	ctx, cancel := context.WithCancel(t.Context())
	f.cancel = cancel
	f.pool.Start(ctx, f.ticker)
	t.Cleanup(func() {
		cancel()
		ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		assert.NoError(t, f.pool.Shutdown(ctx))
	})
	return f
}

func (f fixture) occupyWorker(t *testing.T) context.CancelFunc {
	t.Helper()
	busy := staleTrip()
	hold, release := context.WithCancel(t.Context())
	t.Cleanup(release)
	f.repo.EXPECT().ListStaleTrips(gomock.Any(), staleAfter).Return([]model.StaleTrip{busy}, nil)
	f.metrics.EXPECT().StaleCheck(gomock.Any(), "queued")
	f.push.EXPECT().RequestPosition(gomock.Any(), busy.ID, busy.DriverID).DoAndReturn(waitForRelease(hold.Done()))
	f.expectRequests("success", 1)
	f.tick()
	require.EqualValues(t, 1, f.pool.ActiveWorkers())
	return release
}

func (f fixture) tick() {
	f.ticker.ticks <- time.Now()
	synctest.Wait()
}

func (f fixture) expectRequests(result string, count int) {
	f.metrics.EXPECT().RequestDone(gomock.Any(), result).Times(count)
	f.metrics.EXPECT().PushDuration(gomock.Any(), result, gomock.Any()).Times(count)
}

func staleTrip() model.StaleTrip { return model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()} }

func waitForRelease(release <-chan struct{}) func(context.Context, uuid.UUID, uuid.UUID) error {
	return func(ctx context.Context, _, _ uuid.UUID) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
