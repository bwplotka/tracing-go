package tracing

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationID = "tracing-go"

// StartSpan creates spans using tracer in the context.
// WARNING: ctx has to be chained to root Tracer.StartSpan or Tracer.DoInSpan.
func StartSpan(ctx context.Context, spanName string) (context.Context, Span) {
	sctx, s := trace.SpanFromContext(ctx).TracerProvider().Tracer(instrumentationID).Start(ctx, spanName)
	return sctx, &span{Span: s}
}

// DoInSpan does `f` function inside span using tracer in the context.
// WARNING: ctx has to be chained to root Tracer.StartSpan or Tracer.DoInSpan.
func DoInSpan(ctx context.Context, spanName string, f func(context.Context) error) error {
	sctx, s := StartSpan(ctx, spanName)
	err := f(sctx)
	s.End(err)
	return err
}

// GetSpan returns current span or noopSpan if no span was created.
func GetSpan(ctx context.Context) Span {
	return &span{Span: trace.SpanFromContext(ctx)}
}

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
	End(err error)

	// Context returns span context that contains useful information about span and belonging trace.
	// This information is available even after span End.
	// NOTE: Do not confuse with Go context.Context which is important, but has to be tracked outside of Span.
	Context() Context

	// AddEvent adds an event to the span. This was previously (in OpenTracing) known as
	// structured logs attached to the span.
	AddEvent(name string, keyvals ...interface{})

	// SetAttributes sets kv as attributes of the Span. If a key from kv
	// already exists for an attribute of the Span it should be overwritten with
	// the value contained in kv.
	SetAttributes(keyvals ...interface{})
}

type span struct {
	trace.Span
}

func (s *span) End(err error) {
	if err != nil {
		s.Span.SetStatus(codes.Error, err.Error())
	} else {
		s.Span.SetStatus(codes.Ok, "")
	}
	s.Span.End()
}

func (s *span) Context() Context {
	sctx := s.SpanContext()
	if !sctx.IsSampled() {
		return ctx{}
	}
	return ctx{
		traceID: sctx.TraceID().String(),
		spanID:  sctx.SpanID().String(),
	}

}
func (s *span) AddEvent(name string, keyvals ...interface{}) {
	s.Span.AddEvent(name, trace.WithAttributes(kvToAttr(keyvals...)...))
}

func (s *span) SetAttributes(keyvals ...interface{}) { s.Span.SetAttributes(kvToAttr(keyvals...)...) }

type Context interface {
	IsSampled() bool
	TraceID() string
	SpanID() string
}

type ctx struct {
	traceID string
	spanID  string
}

func (c ctx) IsSampled() bool { return c.traceID != "" }
func (c ctx) TraceID() string { return c.traceID }
func (c ctx) SpanID() string  { return c.spanID }
