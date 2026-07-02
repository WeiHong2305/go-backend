package model

import "time"

type JobStatus string

const (
	Pending   JobStatus = "PENDING"
	Running   JobStatus = "RUNNING"
	Completed JobStatus = "COMPLETED"
	Failed    JobStatus = "FAILED"
)

type JobPayload interface {
	jobPayload()
}

type Job struct {
	ID         string
	Type       string
	Payload    JobPayload
	Status     JobStatus
	RetryCount int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const JobTypeMovieImport = "movie_import"

const MaxRetries = 3

type MovieImportPayload struct {
	Movies []Movie
	Done   map[int]bool
}

func (*MovieImportPayload) jobPayload() {}

type JobRespond struct {
	ID     string    `json:"job_id"`
	Status JobStatus `json:"status"`
}
