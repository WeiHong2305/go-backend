package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"go-backend/internal/metrics"
	"go-backend/internal/model"
	"go-backend/internal/retry"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const jobTimeout = 5 * time.Minute

type HandlerFunc func(ctx context.Context, job *model.Job) error

type Pool struct {
	conCh          *amqp.Channel
	pubCh          *amqp.Channel
	queueName      string
	retryQueueName string
	workers        int
	wg             sync.WaitGroup
	handlers       map[string]HandlerFunc
	baseCtx        context.Context
	cancel         context.CancelFunc
	metrics        *metrics.Metrics
}

func NewPool(conCh, pubCh *amqp.Channel, queueName string, retryQueueName string, workers int, m *metrics.Metrics) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		conCh:          conCh,
		pubCh:          pubCh,
		queueName:      queueName,
		retryQueueName: retryQueueName,
		workers:        workers,
		handlers:       make(map[string]HandlerFunc),
		baseCtx:        ctx,
		cancel:         cancel,
		metrics:        m,
	}
}

func (p *Pool) Register(jobType string, h HandlerFunc) {
	p.handlers[jobType] = h
}

func (p *Pool) Start() {
	if err := p.conCh.Qos(p.workers, 0, false); err != nil {
		slog.Error("failed to set QoS", "error", err)
		os.Exit(1)
	}

	jobMsgs, err := p.conCh.Consume(
		p.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		slog.Error("failed to register a consumer", "error", err)
		os.Exit(1)
	}

	for i := range p.workers {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			slog.Info("worker started", "worker_id", id)
			for jobMsg := range jobMsgs {
				job := model.Job{}
				if err := json.Unmarshal(jobMsg.Body, &job); err != nil {
					slog.Error("failed to unmarshal job message", "error", err)
					jobMsg.Nack(false, false)
					continue
				}
				p.dispatch(id, jobMsg, &job)
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

func (p *Pool) dispatch(workerID int, jobMsg amqp.Delivery, job *model.Job) {
	h, ok := p.handlers[job.Type]
	if !ok {
		slog.Warn("no handler registered for job type",
			"worker_id", workerID,
			"job_id", job.ID,
			"type", job.Type,
		)
		jobMsg.Nack(false, false)
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
		p.handleFailure(ctx, workerID, jobMsg, job, err)
		return
	}

	jobMsg.Ack(false)
	job.Status = model.Completed
	if p.metrics != nil {
		p.metrics.RecordJobCompleted(ctx, job.Type, time.Since(start))
	}
	slog.Info("job completed",
		"worker_id", workerID,
		"job_id", job.ID,
	)
}

func (p *Pool) handleFailure(ctx context.Context, workerID int, jobMsg amqp.Delivery, job *model.Job, err error) {
	if job.RetryCount < model.MaxRetries {
		p.scheduleRetry(ctx, workerID, jobMsg, job, err)
		return
	}

	jobMsg.Nack(false, false)
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

func (p *Pool) scheduleRetry(ctx context.Context, workerID int, jobMsg amqp.Delivery, job *model.Job, err error) {
	job.RetryCount++
	job.Status = model.Pending
	if p.metrics != nil {
		p.metrics.RecordJobRetry(ctx, job.Type, job.RetryCount)
	}

	delay := retry.Config{
		BaseDelay: time.Second,
		MaxDelay:  30 * time.Second,
	}.Delay(job.RetryCount - 1)

	body, marshallErr := json.Marshal(job)
	if marshallErr != nil {
		slog.Error("failed to marshal retry job", "error", marshallErr, "job_id", job.ID)
		jobMsg.Nack(false, false)
		return
	}

	pubErr := p.pubCh.PublishWithContext(ctx,
		"",
		p.retryQueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Expiration:   strconv.FormatInt(delay.Milliseconds(), 10),
			Body:         body,
		},
	)
	if pubErr != nil {
		slog.Error("failed to publish retry", "error", pubErr, "job_id", job.ID)
		jobMsg.Nack(false, true)
		return
	}

	jobMsg.Ack(false)
	slog.Warn("job failed, scheduled retry via DLX",
		"worker_id", workerID,
		"job_id", job.ID,
		"retry", job.RetryCount,
		"delay", delay,
		"error", err,
	)
}
