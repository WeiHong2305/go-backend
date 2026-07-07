package api

import (
	"context"
	"errors"
	"go-backend/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  int64
		wantErr bool
	}{
		{"valid", "123", 123, false},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
		{"non-nemuric", "abc", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, err := parseID(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if id != 0 {
					t.Errorf("expected 0 on error, got %d", id)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if id != tc.wantID {
				t.Errorf("got id %d, want %d", id, tc.wantID)
			}
		})
	}
}

func TestMapServiceError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", service.ErrNotFound, http.StatusNotFound},
		{"validation", service.ErrValidation, http.StatusBadRequest},
		{"conflict", service.ErrConflict, http.StatusConflict},
		{"unauthorized", service.ErrUnauthorized, http.StatusUnauthorized},
		{"unavailable", service.ErrUnavailable, http.StatusServiceUnavailable},
		{"deadline exceeded", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"generic error", errors.New("some error"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			mapServiceError(w, r, tc.err)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}
