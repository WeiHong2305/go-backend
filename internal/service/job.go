package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-backend/internal/model"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type JobService interface {
	AddJob(ctx context.Context, jobType string, payload model.JobPayload) (model.JobRespond, error)
}

type jobService struct {
	ch        *amqp.Channel
	queueName string
}

func NewJobService(ch *amqp.Channel, queueName string) *jobService {
	return &jobService{ch: ch, queueName: queueName}
}

func (j *jobService) AddJob(ctx context.Context, jobType string, payload model.JobPayload) (model.JobRespond, error) {
	now := time.Now()
	job := model.Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Payload:   payload,
		Status:    model.Pending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	body, err := json.Marshal(job)
	if err != nil {
		slog.Error("failed to marshal job to JSON", "error", err)
		return model.JobRespond{}, fmt.Errorf("marshal job: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = j.ch.PublishWithContext(ctx,
		"",
		j.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		slog.Error("failed to publish a job message", "error", err)
		return model.JobRespond{}, fmt.Errorf("publish job: %w", err)
	}

	return model.JobRespond{
		ID:     job.ID,
		Status: job.Status,
	}, nil

}
