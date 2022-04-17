package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

const instrumentationID = "tracing-go"

// StartSpan creates spans using tracer in the context.
// NOTE: Span has to be explicitly ended if you want to export it.
func StartSpan(ctx context.Context, spanName string) (context.Context, Span) {
	sctx, s := trace.SpanFromContext(ctx).TracerProvider().Tracer(instrumentationID).Start(ctx, spanName)
	return sctx, &span{Span: s}
}

// DoInSpan does `f` function inside span using tracer in the context.
func DoInSpan(ctx context.Context, spanName string, f func(context.Context, Span)) {
	sctx, s := StartSpan(ctx, spanName)
	f(sctx, s)
	s.End()
}

type span struct {
	trace.Span
}

func (s *span) End() { s.Span.End() }

func (s *span) AddEvent(name string, keyvals ...interface{}) {
	s.Span.AddEvent(name, trace.WithAttributes(kvToAttr(keyvals...)...))
}

func (s *span) SetAttributes(keyvals ...interface{}) { s.Span.SetAttributes(kvToAttr(keyvals...)...) }

// Span is the individual component of a trace. It represents a single named
// and timed operation of a workflow that is traced. A Tracer is used to
// create a Span and it is then up to the operation the Span represents to
// properly end the Span when the operation itself ends.
type Span interface {
	// End completes the Span. The Span is considered complete and ready to be
	// delivered through the rest of the telemetry pipeline after this method
	// is called. Therefore, updates to the Span are not allowed after this
	// method has been called.
	// TODO(bwplotka): Add Set status to End options.
	End()

	// AddEvent adds an event to the span. This was previously (in OpenTracing) known as
	// structured logs attached to the span.
	AddEvent(name string, keyvals ...interface{})

	// SetAttributes sets kv as attributes of the Span. If a key from kv
	// already exists for an attribute of the Span it should be overwritten with
	// the value contained in kv.
	SetAttributes(keyvals ...interface{})
}
