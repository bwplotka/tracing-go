# tracing-go

Pragmatic and minimalistic module for collecting and exporting trace data from the Go code.

> [prometheus/client_golang](https://github.com/prometheus/client_golang) but for Traces

NOTE: This project follows semver, but it is in experimental v0.x.y phase. API might change.

## Background

This library was born from the fact that the current state of Go clients for tracing are far from perfection.

The success of the [Prometheus client_golang library](https://github.com/prometheus/client_golang) (this package is used more than [51,000 repositories](https://github.com/prometheus/client_golang/network/dependents?package_id=UGFja2FnZS0yMjY0ODEyOTE4)) was in some way thanks to the simplicity, stability and efficiency of that Go client for metrics. Strict compatibility, clear API and error semantics, no scope creep and single module are the things that enabled massive value to so many people and organizations in the community. The key is to make the best user (developer) experience possible.

The above learnings was what motivated the creation of `github.com/bwplotka/tracing-go`.

## Features

* Manual span instrumentation with contextualized tracer and clear error semantics.
* Export of traces to the desired tracing backend or collector:
  * Using [gRPC OTLP](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md) protocol
  * Using Jaeger Thrift Collector, because Jaeger does [not support OTLP yet](https://github.com/jaegertracing/jaeger/issues/3625) 🙃
  * Writing to file e.g stdout/stderr.

At least for now, this project wraps [multiple https://github.com/open-telemetry/opentelemetry-go](https://github.com/open-telemetry/opentelemetry-go) modules, (almost) fully hiding those from the public interface. Yet, if you import `github.com/bwplotka/tracing-go` module you will transiently import OpenTelemetry modules.

## Usage

First, create tracer with exporter(s) you want e.g. Jaeger HTTP Thrift.

```go
// import "github.com/bwplotka/tracing-go/tracing"
// import "github.com/bwplotka/tracing-go/tracing/exporters/jaeger"

tr, closeFn, err := tracing.NewTracer(
		tracing.WithServiceName("app"),
		tracing.WithExporter(jaeger.Exporter(
			jaeger.WithCollectorEndpoint(jaegerEndpoint),
		)),
		// Further options.
	)
if err != nil {
	// Handle err...
}

defer closeFn()
```

Then use it to create root span that also gives context that can be used to create more sub-spans. 
NOTE: Only context has power to create sub spans.

```go
// import "github.com/bwplotka/tracing-go/tracing"

ctx, root := tr.StartSpan("app")
defer root.End()

// ...
ctx, span := tracing.StartSpan(ctx, "dummy operation")
defer func() {
  span.SetAttributes("err", err)
  span.End()
}()
```

Use `DoInSpan` if you want to do something in the dedicated span. 

```go
// import "github.com/bwplotka/tracing-go/tracing"

tracing.DoInSpan(ctx, "sub operation1", func(ctx context.Context, span tracing.Span) {
    // ...
})
tracing.DoInSpan(ctx, "sub operation2", func(ctx context.Context, span tracing.Span) { 
	// ...
})
```

See (and run if you want) an [example instrumented application](https://github.com/bwplotka/tracing-go/blob/e4932502118d0cf62706a342c04107b0727cd230/tracing/tracing_e2e_test.go#L78) using our docker based [e2e suite](https://github.com/efficientgo/e2e).  

## Credits

* Initial version of this library was written for @AnaisUrlichs and @bwplotka demo of [monitoring Argo Rollout jobs](https://github.com/AnaisUrlichs/observe-argo-rollout/blob/main/app/tracing/tracing.go)
* OpenTelemetry project for providing OTLP trace protocol and Go implementation for writing gRPC OTLP.
