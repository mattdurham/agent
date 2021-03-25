package tempoutils

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/grafana/agent/pkg/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/service/builder"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Server is a Tempo testing server that invokes a function every time a span
// is received.
type Server struct {
	receivers builder.Receivers
	pipelines builder.BuiltPipelines
	exporters builder.Exporters
}

// NewTestServer creates a new Server for testing, where received traces will
// call the callback function. The returned string is the address where traces
// can be sent using OTLP.
func NewTestServer(t *testing.T, callback func(pdata.Traces)) string {
	t.Helper()

	srv, listenAddr, err := NewServerWithRandomPort(callback)
	if err != nil {
		t.Fatalf("failed to create OTLP server: %s", err)
	}
	t.Cleanup(func() {
		err := srv.Stop()
		assert.NoError(t, err)
	})

	return listenAddr
}

// NewServerWithRandomPort calls NewServer with a random port >49152 and
// <65535. It will try up to five times before failing.
func NewServerWithRandomPort(callback func(pdata.Traces)) (srv *Server, addr string, err error) {
	var lastError error

	for i := 0; i < 5; i++ {
		port := rand.Intn(65535-49152) + 49152
		listenAddr := fmt.Sprintf("127.0.0.1:%d", port)

		srv, err = NewServer(listenAddr, callback)
		if err != nil {
			lastError = err
			continue
		}

		return srv, listenAddr, nil
	}

	return nil, "", fmt.Errorf("failed 5 times to create a server. last error: %w", lastError)
}

// NewServer creates an OTLP-accepting server that calls a function when a
// trace is received. This is primarily useful for testing.
func NewServer(addr string, callback func(pdata.Traces)) (*Server, error) {
	conf := util.Untab(fmt.Sprintf(`
processors:
	func_processor:
receivers:
  otlp:
		protocols:
			grpc:
				endpoint: %s
service:
	pipelines:
		traces:
			receivers: [otlp]
			processors: [func_processor]
			exporters: []
	`, addr))

	var cfg map[string]interface{}
	if err := yaml.NewDecoder(strings.NewReader(conf)).Decode(&cfg); err != nil {
		panic("could not decode config: " + err.Error())
	}

	v := viper.New()
	if err := v.MergeConfigMap(cfg); err != nil {
		return nil, fmt.Errorf("failed to merge in mapstructure config: %w", err)
	}

	extensionsFactory, err := component.MakeExtensionFactoryMap()
	if err != nil {
		return nil, fmt.Errorf("failed to make extension factory map: %w", err)
	}

	receiversFactory, err := component.MakeReceiverFactoryMap(otlpreceiver.NewFactory())
	if err != nil {
		return nil, fmt.Errorf("failed to make receiver factory map: %w", err)
	}

	exportersFactory, err := component.MakeExporterFactoryMap()
	if err != nil {
		return nil, fmt.Errorf("failed to make exporter factory map: %w", err)
	}

	processorsFactory, err := component.MakeProcessorFactoryMap(
		newFuncProcessorFactory(callback),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make processor factory map: %w", err)
	}

	factories := component.Factories{
		Extensions: extensionsFactory,
		Receivers:  receiversFactory,
		Processors: processorsFactory,
		Exporters:  exportersFactory,
	}
	otelCfg, err := config.Load(v, factories)
	if err != nil {
		return nil, fmt.Errorf("failed to make otel config: %w", err)
	}

	var (
		logger    = zap.NewNop()
		startInfo component.ApplicationStartInfo
	)

	exporters, err := builder.NewExportersBuilder(logger, startInfo, otelCfg, factories.Exporters).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build exporters: %w", err)
	}
	if err := exporters.StartAll(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("failed to start exporters: %w", err)
	}

	pipelines, err := builder.NewPipelinesBuilder(logger, startInfo, otelCfg, exporters, factories.Processors).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build pipelines: %w", err)
	}
	if err := pipelines.StartProcessors(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("failed to start pipelines: %w", err)
	}

	receivers, err := builder.NewReceiversBuilder(logger, startInfo, otelCfg, pipelines, factories.Receivers).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build receivers: %w", err)
	}
	if err := receivers.StartAll(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("failed to start receivers: %w", err)
	}

	return &Server{
		receivers: receivers,
		pipelines: pipelines,
		exporters: exporters,
	}, nil
}

// Stop stops the testing server.
func (s *Server) Stop() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var firstErr error

	deps := []func(context.Context) error{
		s.receivers.ShutdownAll,
		s.pipelines.ShutdownProcessors,
		s.exporters.ShutdownAll,
	}
	for _, dep := range deps {
		err := dep(shutdownCtx)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func newFuncProcessorFactory(callback func(pdata.Traces)) component.ProcessorFactory {
	return processorhelper.NewFactory(
		"func_processor",
		func() configmodels.Processor {
			return &configmodels.ProcessorSettings{
				TypeVal: "func_processor",
				NameVal: "func_processor",
			}
		},
		processorhelper.WithTraces(func(
			_ context.Context,
			_ component.ProcessorCreateParams,
			_ configmodels.Processor,
			next consumer.TracesConsumer,
		) (component.TracesProcessor, error) {
			return &funcProcessor{
				Callback: callback,
				Next:     next,
			}, nil
		}),
	)
}

type funcProcessor struct {
	Callback func(pdata.Traces)
	Next     consumer.TracesConsumer
}

func (p *funcProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	if p.Callback != nil {
		p.Callback(td)
	}
	return p.Next.ConsumeTraces(ctx, td)
}

func (p *funcProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (p *funcProcessor) Start(context.Context, component.Host) error { return nil }
func (p *funcProcessor) Shutdown(context.Context) error              { return nil }
