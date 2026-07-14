package main

import (
	"context"
	"fmt"
	"go-backend/internal/retry"
	"log/slog"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	dlxExchange   = "x-dead-letter-exchange"
	dlxRoutingKey = "x-dead-letter-routing-key"
)

func must[T any](val T, err error) T {
	if err != nil {
		slog.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
	return val
}

type RabbitMQ struct {
	mu       sync.RWMutex
	Conn     *amqp.Connection
	JobPubCh *amqp.Channel
	JobConCh *amqp.Channel
	JobQ     amqp.Queue
	RetryQ   amqp.Queue
	DLQ      amqp.Queue
	url      string
	closed   chan struct{}
}

func newRabbitMQ() *RabbitMQ {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
		slog.Warn("RABBITMQ_URL not set, using default local connection string")
	}

	mq := &RabbitMQ{
		url:    url,
		closed: make(chan struct{}),
	}
	mq.connect()
	return mq
}

func (mq *RabbitMQ) connect() {
	mq.Conn = must(amqp.Dial(mq.url))

	mq.JobPubCh = must(mq.Conn.Channel())
	must(struct{}{}, mq.JobPubCh.Confirm(false))

	mq.JobConCh = must(mq.Conn.Channel())

	mq.DLQ = must(mq.JobPubCh.QueueDeclare(
		"jobs.dlq",
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	))

	mq.JobQ = must(mq.JobPubCh.QueueDeclare(
		"jobs",
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
			dlxExchange:       "",
			dlxRoutingKey:     "jobs.dlq",
		},
	))

	mq.RetryQ = must(mq.JobPubCh.QueueDeclare(
		"jobs.retry",
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
			dlxExchange:       "",
			dlxRoutingKey:     "jobs",
		},
	))
}

func (mq *RabbitMQ) reconnect() error {
	conn, err := amqp.Dial(mq.url)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	pubCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("open publish channel: %w", err)
	}
	if err := pubCh.Confirm(false); err != nil {
		conn.Close()
		return fmt.Errorf("enable confirms: %w", err)
	}

	conCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("open consume channel: %w", err)
	}

	dlq, err := pubCh.QueueDeclare(
		"jobs.qlq", true, false, false, false,
		amqp.Table{amqp.QueueTypeArg: amqp.QueueTypeQuorum},
	)
	if err != nil {
		conn.Close()
		return fmt.Errorf("declare dlq: %w", err)
	}

	jobQ, err := pubCh.QueueDeclare(
		"jobs", true, false, false, false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
			dlxExchange:       "",
			dlxRoutingKey:     "jobs.dlq",
		},
	)
	if err != nil {
		conn.Close()
		return fmt.Errorf("declare jobs queue: %w", err)
	}

	retryQ, err := pubCh.QueueDeclare(
		"jobs.retry", true, false, false, false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
			dlxExchange:       "",
			dlxRoutingKey:     "jobs",
		},
	)
	if err != nil {
		conn.Close()
		return fmt.Errorf("declare retry queue: %w", err)
	}

	mq.mu.Lock()
	mq.Conn = conn
	mq.JobPubCh = pubCh
	mq.JobConCh = conCh
	mq.DLQ = dlq
	mq.JobQ = jobQ
	mq.RetryQ = retryQ
	mq.mu.Unlock()

	return nil
}

func (mq *RabbitMQ) HandleReconnect() {
	go func() {
		for {
			connClose := mq.Conn.NotifyClose(make(chan *amqp.Error, 1))

			select {
			case <-mq.closed:
				return
			case amqpErr := <-connClose:
				if amqpErr == nil {
					return
				}
				slog.Error("RabbitMQ connection lost", "error", amqpErr)
			}

			backoff := retry.Config{
				BaseDelay:  time.Second,
				MaxDelay:   30 * time.Second,
				MaxRetries: 0,
			}
			for attempt := 0; ; attempt++ {
				delay := backoff.Delay(attempt)
				slog.Info("attempting RabbitMQ reconnect", "attempt", attempt+1, "delay", delay)
				time.Sleep(delay)

				if err := mq.reconnect(); err != nil {
					slog.Error("RabbitMQ reconnect failed", "attempt", attempt+1, "error", err)
					continue
				}

				slog.Info("RabbitMQ reconnected successfully")
				break
			}
		}
	}()
}

func (mq *RabbitMQ) Publish(ctx context.Context, queue string, body []byte, expiration string) error {
	mq.mu.RLock()
	ch := mq.JobPubCh
	mq.mu.RUnlock()

	confirm, err := ch.PublishWithDeferredConfirmWithContext(ctx,
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Expiration:   expiration,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish retry: %w", err)
	}

	confirmed, err := confirm.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("confirm timed out: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("broker did not confirm message")
	}

	return nil
}

func (mq *RabbitMQ) Close() {
	close(mq.closed)
	mq.JobPubCh.Close()
	mq.JobConCh.Close()
	mq.Conn.Close()
}
