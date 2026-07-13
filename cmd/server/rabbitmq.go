package main

import (
	"log/slog"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn     *amqp.Connection
	JobPubCh *amqp.Channel
	JobConCh *amqp.Channel
	JobQ     amqp.Queue
}

func newRabbitMQ() *RabbitMQ {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}

	jobPubCh, err := conn.Channel()
	if err != nil {
		slog.Error("failed to open job publish channel", "error", err)
		os.Exit(1)
	}

	jobConCh, err := conn.Channel()
	if err != nil {
		slog.Error("failed to open consume channel", "error", err)
	}

	jobQ, err := jobPubCh.QueueDeclare(
		"jobs",
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	if err != nil {
		slog.Error("failed to declare jobs queue", "error", err)
		os.Exit(1)
	}

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
