package helm

import (
	"context"
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// renderSingle performs the rendering for a single Helm chart.
func (r *Renderer) renderSingle(ctx context.Context, data Data) ([]unstructured.Unstructured, error) {
	// Create a temporary directory for chart operations
	tmpDir, err := os.MkdirTemp("", "helm-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Configure Helm action
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(r.settings.RESTClientGetter(), data.Namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {}); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	// Create a chart downloader
	dl := downloader.ChartDownloader{
		Out:     os.Stdout,
		Verify:  downloader.VerifyNever,
		Getters: getter.All(r.settings),
	}

	// Download the chart
	chartRef := data.PathOrOCIReference
	if _, err := os.Stat(chartRef); err != nil {
		// If not a local path, try to download it
		chartRef, _, err = dl.DownloadTo(chartRef, "", tmpDir)
		if err != nil {
			return nil, fmt.Errorf("failed to download chart: %w", err)
		}
	}

	// Load the chart
	chart, err := loader.Load(chartRef)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Create a template action
	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = data.ReleaseName
	client.Namespace = data.Namespace
	client.ClientOnly = true
	client.IncludeCRDs = true

	// Create a release
	rel, err := client.Run(chart, data.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to run helm install: %w", err)
	}

	// Split the manifest into individual resources
	manifests := releaseutil.SplitManifests(rel.Manifest)

	var objects []unstructured.Unstructured
	for _, manifest := range manifests {
		// Skip empty manifests
		if manifest == "" {
			continue
		}

		// Parse the manifest into an unstructured object
		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(manifest), obj); err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}

		objects = append(objects, *obj)
	}

	return objects, nil
}
