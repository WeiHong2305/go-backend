package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	provider *sdkmetric.MeterProvider

	httpRequestsTotal   otelmetric.Int64Counter
	httpRequestDuration otelmetric.Float64Histogram
	cacheHits           otelmetric.Int64Counter
	cacheMisses         otelmetric.Int64Counter
	jobsCompleted       otelmetric.Int64Counter
	jobsFailed          otelmetric.Int64Counter
	jobRetries          otelmetric.Int64Counter
}

func New() (*Metrics, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := provider.Meter("go-backend")

	httpRequestsTotal, err := meter.Int64Counter("http.server.requests.total",
		otelmetric.WithDescription("Total number of HTTP requests"))
	if err != nil {
		return nil, err
	}

	httpRequestDuration, err := meter.Float64Histogram("http.server.request.duration",
		otelmetric.WithDescription("HTTP request duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	cacheHits, err := meter.Int64Counter("cache.hits",
		otelmetric.WithDescription("Total cache hits"))
	if err != nil {
		return nil, err
	}

	cacheMisses, err := meter.Int64Counter("cache.misses",
		otelmetric.WithDescription("Total cache misses"))
	if err != nil {
		return nil, err
	}

	jobsCompleted, err := meter.Int64Counter("jobs.completed",
		otelmetric.WithDescription("Total jobs completed successfully"))
	if err != nil {
		return nil, err
	}

	jobsFailed, err := meter.Int64Counter("jobs.failed",
		otelmetric.WithDescription("Total jobs failed after retries exhausted"))
	if err != nil {
		return nil, err
	}

	jobRetries, err := meter.Int64Counter("jobs.retries",
		otelmetric.WithDescription("Total job retry attempts"))
	if err != nil {
		return nil, err
	}

	return &Metrics{
		provider:            provider,
		httpRequestsTotal:   httpRequestsTotal,
		httpRequestDuration: httpRequestDuration,
		cacheHits:           cacheHits,
		cacheMisses:         cacheMisses,
		jobsCompleted:       jobsCompleted,
		jobsFailed:          jobsFailed,
		jobRetries:          jobRetries,
	}, nil
}

func (m *Metrics) ShutDown(ctx context.Context) error {
	return m.provider.Shutdown(ctx)
}

func (m *Metrics) RecordHttpRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	attrs := otelmetric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("url.path", path),
		attribute.String("http.response.status_class", statusClass(statusCode)),
	)
	m.httpRequestsTotal.Add(ctx, 1, attrs)
	m.httpRequestDuration.Record(ctx, duration.Seconds(), attrs)
}

func statusClass(code int) string {
	return fmt.Sprintf("%dxx", code/100)
}
