package worker

import (
	"context"
	"go-backend/internal/model"
	"log/slog"
	"sync"
	"time"
)

const jobTimeout = 5 * time.Minute

type HandlerFunc func(ctx context.Context, job *model.Job) error

type Pool struct {
	queue    chan model.Job
	workers  int
	wg       sync.WaitGroup
	handlers map[string]HandlerFunc
}

func NewPool(queue chan model.Job, workers int) *Pool {
	return &Pool{
		queue:    queue,
		workers:  workers,
		handlers: make(map[string]HandlerFunc),
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

func (p *Pool) Stop() {
	p.wg.Wait()
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

	slog.Info("processing job",
		"worker_id", workerID,
		"job_id", job.ID,
		"type", job.Type,
		"attempt", job.RetryCount+1,
	)

	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	if err := h(ctx, job); err != nil {
		if job.RetryCount < model.MaxRetries {
			job.RetryCount++
			delay := time.Duration(1<<job.RetryCount) * time.Second
			slog.Warn("job failed, scheduling retry",
				"worker_id", workerID,
				"job_id", job.ID,
				"retry", job.RetryCount,
				"delay", delay,
				"error", err,
			)
			go func() {
				time.Sleep(delay)
				p.queue <- *job
			}()
			return
		}
		slog.Error("job failed, retries exhausted",
			"worker_id", workerID,
			"job_id", job.ID,
			"retries", job.RetryCount,
			"error", err,
		)
		return
	}

	slog.Info("job completed",
		"worker_id", workerID,
		"job_id", job.ID,
	)
}
