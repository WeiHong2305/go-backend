package service

import "go-backend/internal/model"

type JobService interface {
	AddJob(job model.Job) (model.JobRespond, error)
}

type jobService struct {
	queue chan model.Job
}

func NewJobService(queue chan model.Job) *jobService {
	return &jobService{queue: queue}
}

func (j *jobService) AddJob(job model.Job) (model.JobRespond, error) {
	j.queue <- job

}
