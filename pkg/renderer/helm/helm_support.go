package helm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

// Values returns a Values function that always returns the provided static values.
// This is a convenience helper for the common case of non-dynamic values.
func Values(values map[string]any) func(context.Context) (map[string]any, error) {
	return func(_ context.Context) (map[string]any, error) {
		return values, nil
	}
}

// sourceHolder wraps a Source with internal state for lazy loading and thread-safety.
type sourceHolder struct {
	Source

	// Mutex protects concurrent access to chart field
	mu *sync.RWMutex

	// The loaded Helm chart (protected by mu)
	chart *chart.Chart
}

// Validate checks if the Source configuration is valid.
func (h *sourceHolder) Validate() error {
	if len(strings.TrimSpace(h.Chart)) == 0 {
		return errors.New("chart cannot be empty or whitespace-only")
	}

	releaseName := strings.TrimSpace(h.ReleaseName)
	if len(releaseName) == 0 {
		return errors.New("release name cannot be empty or whitespace-only")
	}
	if len(releaseName) > 53 {
		return fmt.Errorf("release name must not exceed 53 characters (got %d)", len(releaseName))
	}

	return nil
}

// LoadChart returns the loaded Helm chart, loading it lazily if needed.
// Thread-safe for concurrent use.
func (h *sourceHolder) LoadChart(settings *cli.EnvSettings) (*chart.Chart, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.chart != nil {
		return h.chart, nil
	}

	opt, err := createChartPathOptions(&h.Source)
	if err != nil {
		return nil, err
	}

	path, err := opt.LocateChart(h.Chart, settings)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to locate chart (repo: %s, name: %s, version: %s): %w",
			h.Repo,
			h.Chart,
			h.ReleaseVersion,
			err,
		)
	}

	c, err := loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load chart (repo: %s, name: %s, version: %s): %w",
			h.Repo,
			h.Chart,
			h.ReleaseVersion,
			err,
		)
	}

	h.chart = c
	return h.chart, nil
}

// createChartPathOptions creates ChartPathOptions for a Source.
// Creates a fresh registry client and install instance per call.
// This allows each Source to have different credential/authentication requirements.
func createChartPathOptions(source *Source) (action.ChartPathOptions, error) {
	c, err := registry.NewClient()
	if err != nil {
		return action.ChartPathOptions{}, fmt.Errorf("unable to create registry client: %w", err)
	}

	install := action.NewInstall(&action.Configuration{
		RegistryClient: c,
	})

	opt := install.ChartPathOptions
	opt.RepoURL = source.Repo
	opt.Version = source.ReleaseVersion

	return opt, nil
}
