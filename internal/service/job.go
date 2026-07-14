package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-backend/internal/model"
	"time"

	"github.com/google/uuid"
)

type JobPublisher interface {
	Publish(ctx context.Context, queue string, body []byte, expiration string) error
}

type JobService interface {
	AddJob(ctx context.Context, jobType string, payload model.JobPayload) (model.JobRespond, error)
}

type jobService struct {
	publisher JobPublisher
	queueName string
}

func NewJobService(publisher JobPublisher, queueName string) *jobService {
	return &jobService{publisher: publisher, queueName: queueName}
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
		return model.JobRespond{}, fmt.Errorf("marshal job: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := j.publisher.Publish(ctx, j.queueName, body, ""); err != nil {
		return model.JobRespond{}, fmt.Errorf("publish job: %w", err)
	}

	return model.JobRespond{
		ID:     job.ID,
		Status: job.Status,
	}, nil
}
