// +build !windows

package windows_exporter //nolint:golint

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/agent/pkg/integrations/config"
)

// Integration is the windows_exporter integration. On non-Windows platforms,
// this integration does nothing and will print a warning if enabled.
type Integration struct {
}

// New creates a fake windows_exporter integration.
func New(logger log.Logger, _ *Config) (*Integration, error) {
	level.Warn(logger).Log("msg", "the windows_exporter only works on Windows; enabling it otherwise will do nothing")
	return &Integration{}, nil
}

// RegisterRoutes satisfies Integration.RegisterRoutes.
func (i *Integration) RegisterRoutes(r *mux.Router) error {
	return nil
}

// ScrapeConfigs satisfies Integration.ScrapeConfigs.
func (i *Integration) ScrapeConfigs() []config.ScrapeConfig {
	// No-op: nothing to scrape.
	return []config.ScrapeConfig{}
}

// Run satisfies Integration.Run.
func (i *Integration) Run(ctx context.Context) error {
	// We don't need to do anything here, so we can just wait for the context to
	// finish.
	<-ctx.Done()
	return ctx.Err()
}
