package model

import (
	"encoding/json"
	"fmt"
	"time"
)

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
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	RawPayload json.RawMessage `json:"payload"`
	Payload    JobPayload      `json:"-"`
	Status     JobStatus       `json:"status"`
	RetryCount int             `json:"retry_count"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (j *Job) UnmarshalPayload() error {
	switch j.Type {
	case JobTypeMovieImport:
		var p MovieImportPayload
		if err := json.Unmarshal(j.RawPayload, &p); err != nil {
			return fmt.Errorf("unmarshal movie import payload: %w", err)
		}
		j.Payload = &p
	default:
		return fmt.Errorf("unknown job type: %s", j.Type)
	}
	return nil
}

func (j *Job) MarshalPayload() error {
	raw, err := json.Marshal(j.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	j.RawPayload = raw
	return nil
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
