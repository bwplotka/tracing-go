package tracinghttp

import (
	"errors"
	"io"
	"net/http"

	"github.com/bwplotka/tracing-go/tracing"
	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

// Middleware instruments net/http server.
// The "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp" module instruments many too things.
// Here we want to just focus on tracing.
type Middleware struct {
	tracer *tracing.Tracer
}

func NewMiddleware(tracer *tracing.Tracer) *Middleware {
	return &Middleware{tracer: tracer}
}

func (m *Middleware) WrapHandler(name string, next http.Handler) http.HandlerFunc {
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := m.tracer.StartSpan(name, tracing.WithTracerStartSpanContext(
			propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))),
		)
		span.SetAttributes(attrToKv(semconv.NetAttributesFromHTTPRequest("tcp", r)...))
		span.SetAttributes(attrToKv(semconv.EndUserAttributesFromHTTPRequest(r)...))
		span.SetAttributes(attrToKv(semconv.HTTPServerAttributesFromHTTPRequest(name, "", r)...))

		// TODO(bwplotka): Add option to turn this off, this might be too much - we are getting in the world of profiling.
		readRecordFunc := func(n int64) {
			span.AddEvent("read", string(otelhttp.ReadBytesKey), n)
		}

		var bw bodyWrapper
		if r.Body != nil {
			bw.ReadCloser = r.Body
			bw.record = readRecordFunc
			r.Body = &bw
		}

		// TODO(bwplotka): Add option to turn this off, this might be too much - we are getting in the world of profiling.
		writeRecordFunc := func(n int64) {
			span.AddEvent("write", string(otelhttp.WroteBytesKey), n)
		}

		rww := &respWriterWrapper{ResponseWriter: w, record: writeRecordFunc}

		w = httpsnoop.Wrap(w, httpsnoop.Hooks{
			Header: func(httpsnoop.HeaderFunc) httpsnoop.HeaderFunc {
				return rww.Header
			},
			Write: func(httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return rww.Write
			},
			WriteHeader: func(httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return rww.WriteHeader
			},
		})

		// Perform handler.
		next.ServeHTTP(w, r.WithContext(ctx))

		var postServeAttrs []interface{}

		// Request behaviour.
		if bw.readBytes > 0 {
			postServeAttrs = append(postServeAttrs, string(otelhttp.ReadBytesKey), bw.readBytes)
		}
		if bw.lastReadError != nil && bw.lastReadError != io.EOF {
			postServeAttrs = append(postServeAttrs, string(otelhttp.ReadErrorKey), bw.lastReadError.Error())
		}

		// Writer behaviour.
		if rww.writtenBytes > 0 {
			postServeAttrs = append(postServeAttrs, string(otelhttp.WroteBytesKey), rww.writtenBytes)
		}
		if rww.lastWriteErr != nil && rww.lastWriteErr != io.EOF {
			postServeAttrs = append(postServeAttrs, string(otelhttp.WriteErrorKey), rww.lastWriteErr.Error())
		}
		if rww.statusCode > 0 {
			postServeAttrs = append(postServeAttrs, attrToKv(semconv.HTTPAttributesFromHTTPStatusCode(rww.statusCode)...))
		}
		span.SetAttributes(postServeAttrs...)

		if rww.statusCode == http.StatusOK {
			span.End(nil)
			return
		}
		span.End(errors.New("non-200 status code"))
	}
}

func attrToKv(kvs ...attribute.KeyValue) []interface{} {
	if len(kvs) == 0 {
		return nil
	}
	keyvals := make([]interface{}, 0, len(kvs)*2)
	for _, kv := range kvs {
		keyvals = append(keyvals, string(kv.Key))
		keyvals = append(keyvals, kv.Value.AsString())
	}
	return keyvals
}
