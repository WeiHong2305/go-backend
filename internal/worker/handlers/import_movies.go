package handlers

import (
	"context"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/service"
	"go-backend/internal/worker"
	"log/slog"
)

func ImporMovies(movieSvc service.MovieService) worker.HandlerFunc {
	return func(ctx context.Context, job model.Job) error {
		payload, ok := job.Payload.(model.MovieImportPayload)
		if !ok {
			return fmt.Errorf("unexpected payload type %T", job.Payload)
		}
		movies := payload.Movies
		failed := 0

		for i, m := range movies {
			if _, err := movieSvc.CreateMovie(ctx, m); err != nil {
				slog.Warn("failed to import movie",
					"job_id", job.ID,
					"row", i+1,
					"title", m.Title,
					"error", err,
				)
				failed++
				continue
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d of %d movies failed to import", failed, len(movies))
		}
		return nil
	}
}
