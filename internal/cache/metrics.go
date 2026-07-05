package cache

import (
	"context"
	"errors"
	"go-backend/internal/metrics"
)

type metricsCache struct {
	inner Cache
	m     *metrics.Metrics
	name  string
}

func NewMetricsCache(inner Cache, m *metrics.Metrics, name string) Cache {
	return &metricsCache{inner: inner, m: m, name: name}
}

func (c *metricsCache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.inner.Get(ctx, key)
	if errors.Is(err, ErrMiss) {
		c.m.RecordCacheMiss(ctx, c.name)
	} else if err == nil {
		c.m.RecordCacheHit(ctx, c.name)
	}
	return val, err
}

func (c *metricsCache) Set(ctx context.Context, key string, value string) error {
	return c.inner.Set(ctx, key, value)
}

func (c *metricsCache) Delete(ctx context.Context, key string) error {
	return c.inner.Delete(ctx, key)
}
