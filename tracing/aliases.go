package tracing

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Sampler = sdktrace.Sampler
type Exporter = sdktrace.SpanExporter
