package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go-backend/internal/metrics"
	"go-backend/internal/model"
	"go-backend/internal/retry"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const jobTimeout = 5 * time.Minute

type HandlerFunc func(ctx context.Context, job *model.Job) error

type Pool struct {
	queue    chan model.Job
	workers  int
	wg       sync.WaitGroup
	handlers map[string]HandlerFunc
	stopCh   <-chan struct{}
	baseCtx  context.Context
	cancel   context.CancelFunc
	metrics  *metrics.Metrics
}

func NewPool(queue chan model.Job, workers int, stopCh <-chan struct{}, m *metrics.Metrics) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		queue:    queue,
		workers:  workers,
		handlers: make(map[string]HandlerFunc),
		stopCh:   stopCh,
		baseCtx:  ctx,
		cancel:   cancel,
		metrics:  m,
	}
}

func (p *Pool) Register(jobType string, h HandlerFunc) {
	p.handlers[jobType] = h
}

func (p *Pool) Start() {
	for i := range p.workers {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			slog.Info("worker started", "worker_id", id)
			for job := range p.queue {
				p.dispatch(id, &job)
			}
			slog.Info("worker stopped", "worker_id", id)
		}(i)
	}
}

func (p *Pool) Stop(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		p.cancel()
		return fmt.Errorf("worker pool did not stop within %s", timeout)
	}
}

func (p *Pool) dispatch(workerID int, job *model.Job) {
	h, ok := p.handlers[job.Type]
	if !ok {
		slog.Warn("no handler registered for job type",
			"worker_id", workerID,
			"job_id", job.ID,
			"type", job.Type,
		)
		return
	}

	job.Status = model.Running
	slog.Info("processing job",
		"worker_id", workerID,
		"job_id", job.ID,
		"type", job.Type,
		"attempt", job.RetryCount+1,
	)

	ctx, cancel := context.WithTimeout(p.baseCtx, jobTimeout)
	defer cancel()

	tracer := otel.Tracer("go-backend")
	ctx, span := tracer.Start(ctx, fmt.Sprintf("job %s", job.Type),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("job.type", job.Type),
			attribute.String("job.id", job.ID),
			attribute.Int("job.attempt", job.RetryCount+1),
		),
	)
	defer span.End()

	start := time.Now()
	if err := h(ctx, job); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		p.handleFailure(ctx, workerID, job, err)
		return
	}

	job.Status = model.Completed
	if p.metrics != nil {
		p.metrics.RecordJobCompleted(ctx, job.Type, time.Since(start))
	}
	slog.Info("job completed",
		"worker_id", workerID,
		"job_id", job.ID,
	)
}

func (p *Pool) handleFailure(ctx context.Context, workerID int, job *model.Job, err error) {
	if job.RetryCount < model.MaxRetries {
		p.scheduleRetry(ctx, workerID, job, err)
		return
	}

	job.Status = model.Failed
	if p.metrics != nil {
		p.metrics.RecordJobFailed(ctx, job.Type)
	}
	slog.Error("job failed, retries exhausted",
		"worker_id", workerID,
		"job_id", job.ID,
		"retries", job.RetryCount,
		"error", err,
	)
}

func (p *Pool) scheduleRetry(ctx context.Context, workerID int, job *model.Job, err error) {
	job.RetryCount++
	job.Status = model.Pending
	if p.metrics != nil {
		p.metrics.RecordJobRetry(ctx, job.Type, job.RetryCount)
	}

	delay := retry.Config{
		BaseDelay: time.Second,
		MaxDelay:  30 * time.Second,
	}.Delay(job.RetryCount - 1)
	slog.Warn("job failed, scheduling retry",
		"worker_id", workerID,
		"job_id", job.ID,
		"retry", job.RetryCount,
		"delay", delay,
		"error", err,
	)

	go func() {
		select {
		case <-p.stopCh:
			slog.Warn("shutdown: dropping retry",
				"job_id", job.ID,
				"retry", job.RetryCount,
			)
			return
		case <-time.After(delay):
		}
		select {
		case p.queue <- *job:
		case <-p.stopCh:
			slog.Warn("shutdown: could not requeue job",
				"job_id", job.ID,
			)
		}
	}()
}
