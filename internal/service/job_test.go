package service

import (
	"context"
	"errors"
	"go-backend/internal/model"
	"testing"
)

type mockPublisher struct {
	publishFunc func(ctx context.Context, queue string, body []byte, expiration string) error
}

func (m *mockPublisher) Publish(ctx context.Context, queue string, body []byte, expiration string) error {
	return m.publishFunc(ctx, queue, body, expiration)
}

func TestAddJob(t *testing.T) {
	tests := []struct {
		name       string
		publishErr error
		wantErr    bool
	}{
		{
			name:       "success returns JobRespond with ID and Pending status",
			publishErr: nil,
		},
		{
			name:       "publish failure returns error",
			publishErr: errors.New("connection closed"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := &mockPublisher{
				publishFunc: func(_ context.Context, _ string, _ []byte, _ string) error {
					return tt.publishErr
				},
			}
			svc := NewJobService(pub, "jobs")

			payload := &model.MovieImportPayload{Movies: []model.Movie{{ID: 1, Title: "Test"}}}
			got, err := svc.AddJob(context.Background(), model.JobTypeMovieImport, payload)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID == "" {
				t.Errorf("expected non-empty job ID")
			}
			if got.Status != model.Pending {
				t.Errorf("got status %q, want %q", got.Status, model.Pending)
			}
		})
	}
}
