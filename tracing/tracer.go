package tracing

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/efficientgo/tools/core/pkg/errcapture"
	"github.com/efficientgo/tools/core/pkg/merrors"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Option sets the value of an option for a Config.
type Option func(*options)

type ExporterBuilder func() (Exporter, error)

type options struct {
	newExporterFns []ExporterBuilder
	sampler        Sampler
	svcName        string
}

// WithWriter sets the writer exporter for spans.
func WithWriter(w io.Writer) Option {
	return func(o *options) {
		o.newExporterFns = append(o.newExporterFns, func() (Exporter, error) {
			e, err := stdouttrace.New(stdouttrace.WithWriter(w))
			if err != nil {
				return nil, errors.Wrap(err, "writer exporter creation")
			}
			return e, nil
		})
	}
}

// WithExporter sets additional exporter builders for spans. E.g. otlp.Exporter and Thrift
func WithExporter(startExporterFn ExporterBuilder) Option {
	return func(o *options) {
		o.newExporterFns = append(o.newExporterFns, startExporterFn)
	}
}

// WithSampler sets sampler, by default there is no sampler.
func WithSampler(s Sampler) Option {
	return func(o *options) {
		o.sampler = s
	}
}

// TraceIDRatioBasedSampler samples a given fraction of traces. Fractions >= 1 will
// always sample. Fractions < 0 are treated as zero. To respect the
// parent trace's `SampledFlag`, the `TraceIDRatioBased` sampler should be used
// as a delegate of a `Parent` sampler.
//nolint:golint // golint complains about stutter of `trace.TraceIDRatioBased`
func TraceIDRatioBasedSampler(fraction float64) Sampler {
	return sdktrace.TraceIDRatioBased(fraction)
}

// WithServiceName sets service name that will be in attributes of all spans created by this tracer
// in "service.name" key. Usually it comes in format of service:app.
func WithServiceName(s string) Option {
	return func(o *options) {
		o.svcName = s
	}
}

// Tracer is the root tracing entity that can enables creation
// of spans, and its export to the desired backends in a form of traces.
type Tracer struct {
	tr   trace.TracerProvider
	prop propagation.TextMapPropagator
}

// NewTracer creates new instance of Tracer with given exporter builder.
// Tracer returns tracer and close function that releases all resources or error.
func NewTracer(exporter ExporterBuilder, opts ...Option) (*Tracer, func() error, error) {
	o := options{
		newExporterFns: []ExporterBuilder{exporter},
	}
	for _, opt := range opts {
		opt(&o)
	}

	var closers []func() error
	closeFn := func() error {
		errs := merrors.New()
		for _, cl := range closers {
			errs.Add(cl())
		}
		return errs.Err()
	}

	svcName := o.svcName
	if svcName == "" {
		executable, err := os.Executable()
		if err != nil {
			svcName = "unknown_service:go"
		} else {
			svcName = "unknown_service:" + filepath.Base(executable)
		}
	}

	tpOpts := []sdktrace.TracerProviderOption{
		// TODO(bwplotka): Detect process info etc.
		sdktrace.WithResource(resource.NewSchemaless(attribute.KeyValue{Key: "service.name" /*semconv.ServiceNameKey*/, Value: attribute.StringValue(svcName)})),
	}
	for _, ne := range o.newExporterFns {
		exporter, err := ne()
		if err != nil {
			errcapture.Do(&err, closeFn, "close")
			return nil, func() error { return nil }, err
		}
		closers = append(closers, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			return exporter.Shutdown(ctx)
		})

		// TODO(bwplotka): Allow different batch options too.
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exporter))
	}

	if o.sampler != nil {
		tpOpts = append(tpOpts, sdktrace.WithSampler(o.sampler))
	} else {
		tpOpts = append(tpOpts, sdktrace.WithSampler(sdktrace.AlwaysSample()))
	}

	tr := &Tracer{
		tr: sdktrace.NewTracerProvider(tpOpts...),
		// TODO(bwplotka): Allow different propagations.
		prop: propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	}

	// Globals kill people, but some tools still use those.
	// TODO(bwplotka): Do we really need this?
	otel.SetTracerProvider(tr.tr)
	otel.SetTextMapPropagator(tr.prop)

	return tr, closeFn, nil
}

// TracerStartSpanOption sets the value in tracerStartSpanOptions.
type TracerStartSpanOption func(*tracerStartSpanOptions)

type tracerStartSpanOptions struct {
	ctx context.Context
}

func WithTracerStartSpanContext(ctx context.Context) TracerStartSpanOption {
	return func(spanOptions *tracerStartSpanOptions) {
		spanOptions.ctx = ctx
	}
}

// StartSpan creates a new root span that can add more spans using returned context. Returned context
func (tr *Tracer) StartSpan(spanName string, opts ...TracerStartSpanOption) (context.Context, Span) {
	o := tracerStartSpanOptions{ctx: context.Background()}

	for _, opt := range opts {
		opt(&o)
	}

	sctx, s := tr.tr.Tracer(instrumentationID).Start(o.ctx, spanName)
	return sctx, &span{Span: s}
}

// DoInSpan does `f` function that can return error inside span using tracer in the context.
func (tr *Tracer) DoInSpan(spanName string, f func(context.Context, Span) error, opts ...TracerStartSpanOption) error {
	sctx, s := tr.StartSpan(spanName, opts...)
	err := f(sctx, s)
	s.End(err)
	return err
}
