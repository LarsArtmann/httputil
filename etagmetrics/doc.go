// Package etagmetrics turns go-etag's observability hooks into ready-made
// atomic counters for cache hit-ratio and buffer-overflow monitoring.
//
// go-etag's [etag.Config] exposes single-valued hooks (OnETagGenerated,
// On304, OnBufferOverflow). This package attaches counting implementations
// that preserve any hooks the consumer already installed, so a Prometheus
// exporter or a slog logger and these counters can coexist on one config:
//
//	cfg, counters := etagmetrics.Attach(etag.DefaultETagConfig())
//	handler := etag.New(cfg)(mux)
//
//	// elsewhere:
//	slog.Info("etag cache", "hit_ratio", counters.HitRatio())
//
// The package is intentionally metric-agnostic: it exposes plain atomic
// counters and leaves exposition format (Prometheus, OpenTelemetry, slog)
// to the consumer, mirroring httputil's zero-logging, zero-vendor stance.
package etagmetrics
