package main

import (
	"log/slog"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func newRabbitMQ() (*amqp.Connection, *amqp.Channel, amqp.Queue) {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		slog.Error("failed to connect to RabbitMQ")
		os.Exit(1)
	}

	jobCh, err := conn.Channel()
	if err != nil {
		slog.Error("failed to open a channel")
	}

	jobQ, err := jobCh.QueueDeclare(
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
		slog.Error("failed to declare a queue")
	}

	return conn, jobCh, jobQ
}
