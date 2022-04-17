package otlp

import (
	"context"

	"github.com/bwplotka/tracing-go/tracing"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Option represents gRPC OTLP exporter option.
type Option struct {
	otelOpt otlptracegrpc.Option
}

// Exporter sets the gRPC OTLP exporter builder for spans that can be used in tracing.WithExporter().
func Exporter(opts ...Option) tracing.ExporterBuilder {
	oopts := make([]otlptracegrpc.Option, 0, len(opts))
	for _, o := range opts {
		oopts = append(oopts, o.otelOpt)
	}
	return func() (tracing.Exporter, error) {
		e, err := otlptrace.New(context.TODO(), otlptracegrpc.NewClient(oopts...))
		if err != nil {
			return nil, errors.Wrap(err, "OTLP exporter creation")
		}
		return e, nil
	}
}

// WithEndpoint allows setting the endpoint that the exporter will
// connect to the collector on. If unset, it will instead try to use
// connect to DefaultCollectorHost:DefaultCollectorPort.
func WithEndpoint(endpoint string) Option {
	return Option{otelOpt: otlptracegrpc.WithEndpoint(endpoint)}
}

// WithInsecure disables client transport security for the exporter's gRPC connection
// just like grpc.WithInsecure() https://pkg.go.dev/google.golang.org/grpc#WithInsecure
// does. Note, by default, client security is required unless WithInsecure is used.
func WithInsecure() Option {
	return Option{otelOpt: otlptracegrpc.WithInsecure()}
}

// WithDialOption opens support to any grpc.DialOption to be used. If it conflicts
// with some other configuration the GRPC specified via the collector the ones here will
// take preference since they are set last.
func WithDialOption(opts ...grpc.DialOption) Option {
	return Option{otelOpt: otlptracegrpc.WithDialOption(opts...)}
}

// WithHeaders will send the provided headers with gRPC requests/
func WithHeaders(headers map[string]string) Option {
	return Option{otelOpt: otlptracegrpc.WithHeaders(headers)}
}

// WithTLSCredentials allows the connection to use TLS credentials
// when talking to the server. It takes in grpc.TransportCredentials instead
// of say a Certificate file or a tls.Certificate, because the retrieving
// these credentials can be done in many ways e.g. plain file, in code tls.Config
// or by certificate rotation, so it is up to the caller to decide what to use.
func WithTLSCredentials(creds credentials.TransportCredentials) Option {
	return Option{otelOpt: otlptracegrpc.WithTLSCredentials(creds)}
}
