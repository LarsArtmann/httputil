// Fixture module: proof that the OpenTelemetry lanes documented in
// docs/integrations/otel.md compile and that httpspec.RunTracing passes against
// the real SDK middlewares. It deliberately sits outside the root module so the
// library keeps its four-dependency policy; nothing here ships to consumers.
//
// The patch-form go directive is REQUIRED, not a style choice: go-appkit/otel
// v0.3.0 declares `go 1.27.2`, and MVS rejects a lower floor. Do not let an
// automated directive sweep lower it; the CI job `otel-interop` fails loudly
// when this module stops building.
module github.com/larsartmann/httputil/testdata/otelinterop

go 1.27.3

require (
	github.com/larsartmann/go-appkit/otel v0.3.0
	github.com/larsartmann/httputil v1.5.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.72.0
	go.opentelemetry.io/otel v1.47.0
	go.opentelemetry.io/otel/trace v1.47.0
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.31.0 // indirect
	github.com/justinas/nosurf v1.2.0 // indirect
	github.com/larsartmann/go-error-family v0.12.0 // indirect
	github.com/larsartmann/go-etag/entitytag v0.6.1 // indirect
	github.com/larsartmann/go-etag/server v0.6.1 // indirect
	github.com/larsartmann/go-flightrecorder v0.2.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.1 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20261005182115-fad411399dd8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261005182115-fad411399dd8 // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/larsartmann/httputil => ../..
