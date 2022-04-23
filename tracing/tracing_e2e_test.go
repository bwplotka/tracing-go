package tracing_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/bwplotka/tracing-go/tracing"
	"github.com/bwplotka/tracing-go/tracing/exporters/jaeger"
	"github.com/efficientgo/e2e"
	e2einteractive "github.com/efficientgo/e2e/interactive"
	"github.com/efficientgo/tools/core/pkg/testutil"
	"github.com/pkg/errors"
)

//nolint
func dummyOperation(ctx context.Context) (err error) {
	ctx, span := tracing.StartSpan(ctx, "dummy operation")
	defer func() { span.End(err) }()

	alloc := make([]byte, 1e6)
	iters := int(rand.Float64() * 100)
	span.SetAttributes("iterations", iters)
	for i := 0; i < iters; i++ {
		_ = fmt.Sprintf("doing stuff! %+v", alloc)
	}

	tracing.DoInSpan(ctx, "sub operation1", func(ctx context.Context, span tracing.Span) error {
		time.Sleep(1200 * time.Millisecond)
		return nil
	})
	tracing.DoInSpan(ctx, "sub operation2", func(ctx context.Context, span tracing.Span) error {
		time.Sleep(300 * time.Millisecond)
		return nil
	})

	switch rand.Intn(3) {
	case 0:
		return nil
	case 1:
		return errors.New("dummy error1")
	case 2:
		return errors.New("dummy error2")
	}
	return nil
}

func runInstrumentedApp(t *testing.T, jaegerEndpoint string) {
	tr, closeFn, err := tracing.NewTracer(
		tracing.WithServiceName("app"),
		tracing.WithExporter(jaeger.Exporter(jaegerEndpoint)),
	)
	testutil.Ok(t, err)
	t.Cleanup(func() {
		testutil.Ok(t, closeFn())
	})

	ctx, root := tr.StartSpan("app")
	defer root.End(nil)

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			_ = dummyOperation(ctx)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestTracingOTLPWithJaeger(t *testing.T) {
	e, err := e2e.NewDockerEnvironment("e2e_otlp")
	testutil.Ok(t, err)
	t.Cleanup(e.Close)

	// Setup in-memory Jaeger to check if backend can understand our client traces.
	jaeger := e.Runnable("tracing").
		WithPorts(
			map[string]int{
				"http.front":    16686,
				"jaeger.thrift": 16000,
			}).
		Init(e2e.StartOptions{
			Image:   "jaegertracing/all-in-one:1.33",
			Command: e2e.NewCommand("--collector.http-server.host-port=:16000"),
		})

	testutil.Ok(t, e2e.StartAndWaitReady(jaeger))

	runInstrumentedApp(t, "http://"+jaeger.Endpoint("jaeger.thrift")+"/api/traces")

	// TODO(bwplotka): Make it non-interactive and expect certain Jaeger output.
	testutil.Ok(t, e2einteractive.OpenInBrowser("http://"+jaeger.Endpoint("http.front")))
	testutil.Ok(t, e2einteractive.RunUntilEndpointHit())
}
