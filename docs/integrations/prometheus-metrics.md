# Prometheus Metrics Recorder

httputil's `MetricsRecorder` interface lets you plug in any metrics backend. The built-in middleware records method, path, status, and duration per request.

## Interface

```go
type MetricsRecorder interface {
    Record(method, path string, status int, duration time.Duration)
}
```

## Prometheus Implementation

```go
// This example references github.com/prometheus/client_golang, which is NOT a
// dependency of httputil — add it to your go.mod to compile.
package main

import (
    "net/http"
    "time"

    "github.com/larsartmann/httputil"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusRecorder struct {
    requests *prometheus.HistogramVec
}

func NewPrometheusRecorder(reg prometheus.Registerer) *PrometheusRecorder {
    return &PrometheusRecorder{
        requests: promauto.With(reg).NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "http_request_duration_seconds",
                Help:    "HTTP request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "path", "status"},
        ),
    }
}

func (p *PrometheusRecorder) Record(method, path string, status int, duration time.Duration) {
    p.requests.WithLabelValues(method, path, statusLabel(status)).Observe(duration.Seconds())
}

func statusLabel(status int) string {
    switch {
    case status >= 200 && status < 300:
        return "2xx"
    case status >= 300 && status < 400:
        return "3xx"
    case status >= 400 && status < 500:
        return "4xx"
    case status >= 500:
        return "5xx"
    default:
        return "unknown"
    }
}

// Usage:
func main() {
    recorder := NewPrometheusRecorder(prometheus.DefaultRegisterer)

    cfg := httputil.DefaultMetricsConfig()
    cfg.Recorder = recorder

    mux := http.NewServeMux()
    handler := httputil.Metrics(cfg)(mux)
    _ = handler
}
```

## Cardinality Control

Prometheus labels become time series. High-cardinality values (e.g., full request paths with IDs) cause unbounded label growth. Use `PathFunc` to normalize:

```go
cfg := httputil.DefaultMetricsConfig()
cfg.Recorder = recorder
cfg.PathFunc = func(r *http.Request) string {
    // Strip dynamic segments: /users/123 → /users/:id
    return normalizePath(r.URL.Path)
}
```

## Counter Alternative

If you prefer counters over histograms (for a simple request total):

```go
type CounterRecorder struct {
    total *prometheus.CounterVec
}

func NewCounterRecorder(reg prometheus.Registerer) *CounterRecorder {
    return &CounterRecorder{
        total: promauto.With(reg).NewCounterVec(
            prometheus.CounterOpts{
                Name: "http_requests_total",
                Help: "Total HTTP requests",
            },
            []string{"method", "status"},
        ),
    }
}

func (c *CounterRecorder) Record(method, path string, status int, duration time.Duration) {
    c.total.WithLabelValues(method, statusLabel(status)).Inc()
}
```

## Combining with Prometheus Middleware

If you already use the official Prometheus middleware (`promhttp.InstrumentHandlerDuration`), you don't need this integration — both serve the same purpose. Use `httputil.Metrics` when you want a metrics-agnostic recorder (e.g., for switching backends without changing middleware) or when you want metrics without a Prometheus dependency in your core package.
