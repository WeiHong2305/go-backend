package main

import (
	"log/slog"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func must[T any](val T, err error) T {
	if err != nil {
		slog.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
	return val
}

type RabbitMQ struct {
	Conn     *amqp.Connection
	JobPubCh *amqp.Channel
	JobConCh *amqp.Channel
	JobQ     amqp.Queue
}

func newRabbitMQ() *RabbitMQ {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
		slog.Warn("RABBITMQ_URL not set, using default local connection string")
	}

	conn := must(amqp.Dial(url))
	jobPubCh := must(conn.Channel())
	jobConCh := must(conn.Channel())
	jobQ := must(jobPubCh.QueueDeclare(
		"jobs",
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	))

	return &RabbitMQ{
		Conn:     conn,
		JobPubCh: jobPubCh,
		JobConCh: jobConCh,
		JobQ:     jobQ,
	}
}

func (r *RabbitMQ) Close() {
	r.JobPubCh.Close()
	r.JobConCh.Close()
	r.Conn.Close()
}
