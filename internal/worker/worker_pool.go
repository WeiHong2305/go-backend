package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go-backend/internal/metrics"
	"go-backend/internal/model"
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

	start := time.Now()
	if err := h(ctx, job); err != nil {
		if job.RetryCount < model.MaxRetries {
			job.RetryCount++
			job.Status = model.Pending
			if p.metrics != nil {
				p.metrics.RecordJobRetry(ctx, job.Type, job.RetryCount)
			}
			delay := time.Duration(1<<job.RetryCount) * time.Second
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
