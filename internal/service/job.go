package service

import (
	"fmt"
	"go-backend/internal/model"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type JobService interface {
	AddJob(jobType string, payload model.JobPayload) (model.JobRespond, error)
}

type jobService struct {
	queue chan model.Job
}

func NewJobService(queue chan model.Job) *jobService {
	return &jobService{queue: queue}
}

func (j *jobService) AddJob(jobType string, payload model.JobPayload) (model.JobRespond, error) {
	now := time.Now()
	job := model.Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Payload:   payload,
		Status:    model.Pending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	select {
	case j.queue <- job:
		slog.Info("job created",
			"job_id", job.ID,
			"type", job.Type,
		)
	default:
		return model.JobRespond{}, fmt.Errorf("%w: job queue is full", ErrUnavailable)
	}

	return model.JobRespond{
		ID:     job.ID,
		Status: job.Status,
	}, nil

}
