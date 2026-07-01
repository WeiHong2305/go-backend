package worker

import (
	"context"
	"go-backend/internal/model"
	"log/slog"
	"sync"
)

type HandlerFunc func(ctx context.Context, job model.Job) error

type Pool struct {
	queue    <-chan model.Job
	workers  int
	wg       sync.WaitGroup
	handlers map[string]HandlerFunc
}

func NewPool(queue <-chan model.Job, workers int) *Pool {
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
				p.dispatch(id, job)
			}
			slog.Info("worker stopped", "worker_id", id)
		}(i)
	}
}

func (p *Pool) Stop() {
	p.wg.Wait()
}

func (p *Pool) dispatch(workerID int, job model.Job) {
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
	)

	if err := h(context.Background(), job); err != nil {
		slog.Error("job failed",
			"worker_id", workerID,
			"job_id", job.ID,
			"error", err,
		)
		return
	}

	slog.Info("job completed",
		"worker_id", workerID,
		"job_id", job.ID,
	)
}
