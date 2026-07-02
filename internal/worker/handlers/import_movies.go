package handlers

import (
	"context"
	"errors"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/service"
	"go-backend/internal/worker"
	"log/slog"
)

func ImportMovies(movieSvc service.MovieService) worker.HandlerFunc {
	return func(ctx context.Context, job *model.Job) error {
		payload, ok := job.Payload.(*model.MovieImportPayload)
		if !ok {
			return fmt.Errorf("unexpected payload type %T", job.Payload)
		}

		if payload.Done == nil {
			payload.Done = make(map[int]bool)
		}

		hasTransient := false
		for i, m := range payload.Movies {
			if payload.Done[i] {
				continue
			}
			if err := processRow(ctx, movieSvc, job.ID, i, m); err != nil {
				hasTransient = true
				continue
			}
			payload.Done[i] = true
		}

		if hasTransient {
			return fmt.Errorf("some rows failed with transient errors")
		}
		return nil
	}
}

func processRow(ctx context.Context, svc service.MovieService, jobID string, i int, m model.Movie) error {
	_, err := svc.CreateMovie(ctx, m)
	if err == nil {
		return nil
	}

	if errors.Is(err, service.ErrValidation) || errors.Is(err, service.ErrConflict) {
		slog.Warn("skipping row (permanent failure)",
			"job_id", jobID,
			"row", i+1,
			"title", m.Title,
			"error", err,
		)
		return nil
	}

	slog.Warn("transient failure, will retry",
		"job_id", jobID,
		"row", i+1,
		"title", m.Title,
		"error", err,
	)
	return err
}
