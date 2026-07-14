package cache

import (
	"context"
	"errors"
	"go-backend/internal/metrics"
	"testing"
)

type mockCache struct {
	getFunc    func(ctx context.Context, key string) (string, error)
	setFunc    func(ctx context.Context, key string, value string) error
	deleteFunc func(ctx context.Context, key string) error
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	return m.getFunc(ctx, key)
}

func (m *mockCache) Set(ctx context.Context, key string, value string) error {
	return m.setFunc(ctx, key, value)
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return m.deleteFunc(ctx, key)
}

func newTestMetrics(t *testing.T) *metrics.Metrics {
	t.Helper()
	m, err := metrics.New()
	if err != nil {
		t.Fatalf("metrics.New: %v", err)
	}
	return m
}

func TestMetricsCacheGet_Hit(t *testing.T) {
	inner := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "cached-value", nil
		},
	}
	c := NewMetricsCache(inner, newTestMetrics(t), "test")
	val, err := c.Get(context.Background(), "some-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != "cached-value" {
		t.Errorf("unexpected %q, got %q", "cached-value", val)
	}
}

func TestMetricsCacheGet_Miss(t *testing.T) {
	inner := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "", ErrMiss
		},
	}
	c := NewMetricsCache(inner, newTestMetrics(t), "test")
	_, err := c.Get(context.Background(), "missing-key")
	if !errors.Is(err, ErrMiss) {
		t.Errorf("expected ErrMiss, got %v", err)
	}
}

func TestMetricsCacheSet(t *testing.T) {
	var gotKey, gotValue string
	inner := &mockCache{
		setFunc: func(_ context.Context, key, value string) error {
			gotKey = key
			gotValue = value
			return nil
		},
	}
	c := NewMetricsCache(inner, newTestMetrics(t), "test")
	if err := c.Set(context.Background(), "k", "v"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "k" || gotValue != "v" {
		t.Errorf("Set called with (%q, %q), want (%q, %q)", gotKey, gotValue, "k", "v")
	}
}

func TestMetricsCacheDelete(t *testing.T) {
	var gotKey string
	inner := &mockCache{
		deleteFunc: func(ctx context.Context, key string) error {
			gotKey = key
			return nil
		},
	}
	c := NewMetricsCache(inner, newTestMetrics(t), "test")
	if err := c.Delete(context.Background(), "k"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "k" {
		t.Errorf("Delete called with %q, want %q", gotKey, "k")
	}
}
