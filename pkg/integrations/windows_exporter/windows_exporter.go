package win_exporter

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/agent/pkg/integrations/config"
	"github.com/prometheus-community/windows_exporter/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"gopkg.in/yaml.v3"
	"net/http"
	"sort"
)

// Integration is the windows_export integration. The integration scrapes metrics
// from the host windows ased system.
type Integration struct {
	c      *Config
	logger log.Logger
	we     *exporter.WindowsCollector

	exporterMetricsRegistry *prometheus.Registry
}

// New creates a new windows_exporter integration.
func New(log log.Logger, c *Config) (*Integration, error) {

	we,_ := yaml.Marshal(c.RawConfig)

	windows := exporter.CreateLibrary(string(we))
	level.Info(log).Log("msg", "Enabled windows_exporter collectors")
	collectors := []string{}
	for n := range windows.Collectors {
		collectors = append(collectors, n)
	}
	sort.Strings(collectors)
	for _, c := range collectors {
		level.Info(log).Log("collector", c)
	}

	return &Integration{
		c:      c,
		logger: log,
		we:     windows,

		exporterMetricsRegistry: prometheus.NewRegistry(),
	}, nil
}

// RegisterRoutes satisfies Integration.RegisterRoutes. The mux.Router provided
// here is expected to be a subrouter, where all registered paths will be
// registered within that subroute.
func (i *Integration) RegisterRoutes(r *mux.Router) error {
	handler, err := i.handler()
	if err != nil {
		return err
	}

	r.Handle("/metrics", handler)
	return nil
}

func (i *Integration) handler() (http.Handler, error) {
	r := prometheus.NewRegistry()
	if err := r.Register(i.we); err != nil {
		return nil, fmt.Errorf("couldn't register node_exporter node collector: %w", err)
	}
	handler := promhttp.HandlerFor(
		prometheus.Gatherers{i.exporterMetricsRegistry, r},
		promhttp.HandlerOpts{
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: 0,
			Registry:            i.exporterMetricsRegistry,
		},
	)

	// Register node_exporter_build_info metrics, generally useful for
	// dashboards that depend on them for discovering targets.
	if err := r.Register(version.NewCollector(i.c.Name())); err != nil {
		return nil, fmt.Errorf("couldn't register %s: %w", i.c.Name(), err)
	}

	/*
	if i.c.IncludeExporterMetrics {
		// Note that we have to use reg here to use the same promhttp metrics for
		// all expositions.
		handler = promhttp.InstrumentMetricHandler(i.exporterMetricsRegistry, handler)
	}*/

	return handler, nil
}

// ScrapeConfigs satisfies Integration.ScrapeConfigs.
func (i *Integration) ScrapeConfigs() []config.ScrapeConfig {
	return []config.ScrapeConfig{{
		JobName:     i.c.Name(),
		MetricsPath: "/metrics",
	}}
}

// Run satisfies Integration.Run.
func (i *Integration) Run(ctx context.Context) error {
	// We don't need to do anything here, so we can just wait for the context to
	// finish.
	<-ctx.Done()
	return ctx.Err()
}
