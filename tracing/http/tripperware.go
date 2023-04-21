package tracinghttp

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/bwplotka/tracing-go/tracing"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

// Tripperware instruments net/http client's transport called http.RoundTripper.
// The "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp" module instruments many too things.
// Here we want to just focus on tracing.
type Tripperware struct {
}

func NewTripperware() *Tripperware {
	return &Tripperware{}
}

type rtFunc func(*http.Request) (*http.Response, error)

func (rt rtFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

func (Tripperware) WrapRoundTipper(name string, next http.RoundTripper) http.RoundTripper {
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	return rtFunc(func(r *http.Request) (*http.Response, error) {
		ctx, span := tracing.StartSpan(r.Context(), name)
		span.SetAttributes(attrToKv(semconv.NetAttributesFromHTTPRequest("tcp", r)...))
		span.SetAttributes(attrToKv(semconv.EndUserAttributesFromHTTPRequest(r)...))
		span.SetAttributes(attrToKv(semconv.HTTPServerAttributesFromHTTPRequest(name, "", r)...))

		propagator.Inject(ctx, propagation.HeaderCarrier(r.Header))

		// Perform round trip.
		res, err := next.RoundTrip(r.WithContext(ctx))
		if err != nil {
			span.End(err)
			return res, err
		}

		// TODO(bwplotka): Add option to turn this off, this might be too much - we are getting in the world of profiling.
		var bw bodyWrapper
		bw.ReadCloser = res.Body
		bw.record = func(n int64) {
			span.AddEvent("read", string(otelhttp.ReadBytesKey), n)
		}
		res.Body = &bw
		span.End(nil)
		return res, nil
	})
}
