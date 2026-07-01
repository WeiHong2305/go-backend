package model

import "time"

type JobStatus string

const (
	Pending   JobStatus = "PENDING"
	Running   JobStatus = "RUNNING"
	Completed JobStatus = "COMPLETED"
	Failed    JobStatus = "FAILED"
)

type Job struct {
	ID         string
	Type       string
	Payload    any
	Status     JobStatus
	RetryCount int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const JobTypeMovieImport = "movie_import"

type MovieImportPayload struct {
	Movies []Movie
}

type JobRespond struct {
	ID     string    `json:"job_id"`
	Status JobStatus `json:"status"`
}
