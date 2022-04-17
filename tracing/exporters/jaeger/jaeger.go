package jaeger

import (
	"net/http"

	"github.com/bwplotka/tracing-go/tracing"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/jaeger"
)

// Option represents Jaeger Thrift exporter option.
type Option struct {
	jaegerOpt jaeger.CollectorEndpointOption
}

// Exporter sets the Jaeger exporter builder for spans that can be used in tracing.WithExporter().
func Exporter(opts ...Option) tracing.ExporterBuilder {
	jopts := make([]jaeger.CollectorEndpointOption, 0, len(opts))
	for _, o := range opts {
		jopts = append(jopts, o.jaegerOpt)
	}
	return func() (tracing.Exporter, error) {
		e, err := jaeger.New(jaeger.WithCollectorEndpoint(jopts...))
		if err != nil {
			return nil, errors.Wrap(err, "OTLP exporter creation")
		}
		return e, nil
	}
}

// WithCollectorEndpoint allows setting the endpoint that the exporter will
// connect to the collector on.
// "http://localhost:14268/api/traces" will be used by default.
func WithCollectorEndpoint(endpoint string) Option {
	return Option{jaegerOpt: jaeger.WithEndpoint(endpoint)}
}

// WithHTTPClient sets the http client to be used to make request to the collector endpoint.
func WithHTTPClient(client *http.Client) Option {
	return Option{jaegerOpt: jaeger.WithHTTPClient(client)}
}
