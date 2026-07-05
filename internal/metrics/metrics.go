package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	provider *sdkmetric.MeterProvider

	httpRequestsTotal   otelmetric.Int64Counter
	httpRequestDuration otelmetric.Float64Histogram
	httpActiveRequests  otelmetric.Int64UpDownCounter
	httpResponseSize    otelmetric.Int64Histogram
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
		otelmetric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, err
	}

	httpActiveRequests, err := meter.Int64UpDownCounter("http.server.active_requests",
		otelmetric.WithDescription("Number of HTTP requests currently being processed"))
	if err != nil {
		return nil, err
	}

	httpResponseSize, err := meter.Int64Histogram("http.server.response.size",
		otelmetric.WithDescription("HTTP response body size in bytes"),
		otelmetric.WithUnit("By"),
		otelmetric.WithExplicitBucketBoundaries(100, 1000, 10000, 100000, 1000000),
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
		httpActiveRequests:  httpActiveRequests,
		httpResponseSize:    httpResponseSize,
		cacheHits:           cacheHits,
		cacheMisses:         cacheMisses,
		jobsCompleted:       jobsCompleted,
		jobsFailed:          jobsFailed,
		jobRetries:          jobRetries,
	}, nil
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) ShutDown(ctx context.Context) error {
	return m.provider.Shutdown(ctx)
}

func (m *Metrics) RecordHttpRequest(ctx context.Context, method, route string, statusCode int, duration time.Duration, responseBytes int64) {
	attrs := otelmetric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
		attribute.String("http.response.status_class", statusClass(statusCode)),
	)
	m.httpRequestsTotal.Add(ctx, 1, attrs)
	m.httpRequestDuration.Record(ctx, duration.Seconds(), attrs)
	m.httpResponseSize.Record(ctx, responseBytes, attrs)
}

func (m *Metrics) RecordActiveRequestStart(ctx context.Context) {
	m.httpActiveRequests.Add(ctx, 1)
}

func (m *Metrics) RecordActiveRequestEnd(ctx context.Context) {
	m.httpActiveRequests.Add(ctx, -1)
}

func (m *Metrics) RecordCacheHit(ctx context.Context, cacheName string) {
	m.cacheHits.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("cache.name", cacheName),
	))
}

func (m *Metrics) RecordCacheMiss(ctx context.Context, cacheName string) {
	m.cacheMisses.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("cache.name", cacheName),
	))
}

func (m *Metrics) RecordJobCompleted(ctx context.Context, jobType string) {
	m.jobsCompleted.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("job.type", jobType),
	))
}

func (m *Metrics) RecordJobFailed(ctx context.Context, jobType string) {
	m.jobsFailed.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("job.type", jobType),
	))
}

func (m *Metrics) RecordJobRetry(ctx context.Context, jobType string, attempt int) {
	m.jobRetries.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("job.type", jobType),
		attribute.Int("job.attempt", attempt),
	))
}

func statusClass(code int) string {
	return fmt.Sprintf("%dxx", code/100)
}
