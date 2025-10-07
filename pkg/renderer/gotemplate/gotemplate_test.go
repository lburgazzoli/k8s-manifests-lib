package gotemplate_test

import (
	"context"
	"testing"
	"testing/fstest"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/onsi/gomega/types"
	"github.com/rs/xid"

	corev1 "k8s.io/api/core/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/gotemplate"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"

	. "github.com/onsi/gomega"
)

const podTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{ .Repo }}-pod
  labels:
    app: {{ .Repo }}
    component: {{ .Component }}
spec:
  containers:
  - name: nginx
    image: nginx:latest
`

const configMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Repo }}-config
  labels:
    app: {{ .Repo }}
    component: {{ .Component }}
data:
  config.yaml: |
    port: {{ .Port }}
`

const invalidTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{ .InvalidField }}-pod
`

func TestRenderer(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name          string
		data          gotemplate.Source
		opts          []gotemplate.RendererOption
		expectedCount int
		validation    types.GomegaMatcher
	}{
		{
			name: "should render single template",
			data: gotemplate.Source{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo":      "test-app",
					"Component": "frontend",
				}),
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.name == "test-app-pod"`),
				jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
				jqmatcher.Match(`.metadata.labels["component"] == "frontend"`),
			),
		},
		{
			name: "should render multiple templates",
			data: gotemplate.Source{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl":       &fstest.MapFile{Data: []byte(podTemplate)},
					"templates/configmap.yaml.tpl": &fstest.MapFile{Data: []byte(configMapTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo":      "test-app",
					"Component": "frontend",
					"Port":      8080,
				}),
			},
			expectedCount: 2,
			validation: Or(
				And(
					jqmatcher.Match(`.kind == "Pod"`),
					jqmatcher.Match(`.metadata.name == "test-app-pod"`),
				),
				And(
					jqmatcher.Match(`.kind == "ConfigMap"`),
					jqmatcher.Match(`.metadata.name == "test-app-config"`),
					jqmatcher.Match(`.data["config.yaml"] == "port: 8080\n"`),
				),
			),
		},
		{
			name: "should handle invalid template",
			data: gotemplate.Source{
				FS: fstest.MapFS{
					"templates/invalid.yaml.tpl": &fstest.MapFile{Data: []byte(invalidTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo": "test-app",
				}),
			},
			expectedCount: 0,
			validation:    nil,
		},
		{
			name: "should apply filters",
			data: gotemplate.Source{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl":       &fstest.MapFile{Data: []byte(podTemplate)},
					"templates/configmap.yaml.tpl": &fstest.MapFile{Data: []byte(configMapTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo":      "test-app",
					"Component": "frontend",
					"Port":      8080,
				}),
			},
			opts: []gotemplate.RendererOption{
				gotemplate.WithFilter(gvk.Filter(corev1.SchemeGroupVersion.WithKind("Pod"))),
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.name == "test-app-pod"`),
			),
		},
		{
			name: "should apply transformers",
			data: gotemplate.Source{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo":      "test-app",
					"Component": "frontend",
				}),
			},
			opts: []gotemplate.RendererOption{
				gotemplate.WithTransformer(labels.Set(map[string]string{
					"managed-by": "gotemplate-renderer",
					"env":        "test",
				})),
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.labels["managed-by"] == "gotemplate-renderer"`),
				jqmatcher.Match(`.metadata.labels["env"] == "test"`),
				jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
			),
		},
		{
			name: "should handle empty template",
			data: gotemplate.Source{
				FS:   fstest.MapFS{},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo": "test-app",
				}),
			},
			expectedCount: 0,
			validation:    nil,
		},
		{
			name: "should handle non-existent template",
			data: gotemplate.Source{
				FS: fstest.MapFS{
					"templates/other.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo": "test-app",
				}),
			},
			expectedCount: 0,
			validation:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer, err := gotemplate.New([]gotemplate.Source{tt.data}, tt.opts...)
			g.Expect(err).ToNot(HaveOccurred())

			objects, err := renderer.Process(t.Context())

			if tt.validation == nil {
				g.Expect(err).To(HaveOccurred())
				g.Expect(objects).To(BeEmpty())

				return
			}

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(objects).To(HaveLen(tt.expectedCount))

			for _, obj := range objects {
				g.Expect(obj.Object).To(tt.validation)
			}
		})
	}
}

func TestNew(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		inputs      []gotemplate.Source
		expectError bool
	}{
		{
			name: "should reject input without FS",
			inputs: []gotemplate.Source{{
				Path: "templates/*.yaml",
			}},
			expectError: true,
		},
		{
			name: "should reject input without path",
			inputs: []gotemplate.Source{{
				FS: fstest.MapFS{},
			}},
			expectError: true,
		},
		{
			name: "should accept valid input",
			inputs: []gotemplate.Source{{
				FS:   fstest.MapFS{},
				Path: "templates/*.yaml",
			}},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer, err := gotemplate.New(tt.inputs)
			if tt.expectError {
				g.Expect(err).To(HaveOccurred())
				g.Expect(renderer).To(BeNil())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(renderer).ToNot(BeNil())
			}
		})
	}
}

func TestCacheIntegration(t *testing.T) {
	g := NewWithT(t)

	t.Run("should cache identical renders", func(t *testing.T) {
		renderer, err := gotemplate.New([]gotemplate.Source{
			{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo":      "cache-app",
					"Component": "frontend",
				}),
			},
		},
			gotemplate.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render - cache miss
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Second render - cache hit (should be identical)
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(len(result1)))

		// Results should be equal
		for i := range result1 {
			g.Expect(result2[i]).To(Equal(result1[i]))
		}
	})

	t.Run("should miss cache on different values", func(t *testing.T) {
		callCount := 0
		dynamicValues := func(_ context.Context) (any, error) {
			callCount++
			return map[string]interface{}{
				"Repo":      xid.New().String(),
				"Component": "frontend",
			}, nil
		}

		renderer, err := gotemplate.New([]gotemplate.Source{
			{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path:   "templates/*.tpl",
				Values: dynamicValues,
			},
		},
			gotemplate.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Second render with different values - cache miss
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).ToNot(BeEmpty())

		// Values function should be called twice (no cache hits)
		g.Expect(callCount).To(Equal(2))
	})

	t.Run("should work with cache disabled", func(t *testing.T) {
		renderer, err := gotemplate.New(
			[]gotemplate.Source{
				{
					FS: fstest.MapFS{
						"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
					},
					Path: "templates/*.tpl",
					Values: gotemplate.Values(map[string]interface{}{
						"Repo":      "no-cache-app",
						"Component": "frontend",
					}),
				},
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Second render - should work even without cache
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(len(result1)))
	})

	t.Run("should return clones from cache", func(t *testing.T) {
		renderer, err := gotemplate.New([]gotemplate.Source{
			{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo":      "clone-app",
					"Component": "frontend",
				}),
			},
		},
			gotemplate.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Modify first result
		if len(result1) > 0 {
			result1[0].SetName("modified-name")
		}

		// Second render - should not be affected by modification
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).ToNot(BeEmpty())

		if len(result2) > 0 {
			g.Expect(result2[0].GetName()).ToNot(Equal("modified-name"))
		}
	})
}

func BenchmarkGoTemplateRenderWithoutCache(b *testing.B) {
	renderer, err := gotemplate.New([]gotemplate.Source{
		{
			FS: fstest.MapFS{
				"templates/pod.yaml.tpl":       &fstest.MapFile{Data: []byte(podTemplate)},
				"templates/configmap.yaml.tpl": &fstest.MapFile{Data: []byte(configMapTemplate)},
			},
			Path: "templates/*.tpl",
			Values: gotemplate.Values(map[string]interface{}{
				"Repo":      "bench-app",
				"Component": "backend",
				"Port":      8080,
			}),
		},
	})
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_, err := renderer.Process(context.Background())
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}

func BenchmarkGoTemplateRenderWithCache(b *testing.B) {
	renderer, err := gotemplate.New(
		[]gotemplate.Source{
			{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl":       &fstest.MapFile{Data: []byte(podTemplate)},
					"templates/configmap.yaml.tpl": &fstest.MapFile{Data: []byte(configMapTemplate)},
				},
				Path: "templates/*.tpl",
				Values: gotemplate.Values(map[string]interface{}{
					"Repo":      "bench-app",
					"Component": "backend",
					"Port":      8080,
				}),
			},
		},
		gotemplate.WithCache(),
	)
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_, err := renderer.Process(context.Background())
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}

func BenchmarkGoTemplateRenderCacheMiss(b *testing.B) {
	renderer, err := gotemplate.New(
		[]gotemplate.Source{
			{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl":       &fstest.MapFile{Data: []byte(podTemplate)},
					"templates/configmap.yaml.tpl": &fstest.MapFile{Data: []byte(configMapTemplate)},
				},
				Path: "templates/*.tpl",
				Values: func(_ context.Context) (any, error) {
					return map[string]interface{}{
						"Repo":      xid.New().String(),
						"Component": "backend",
						"Port":      8080,
					}, nil
				},
			},
		},
		gotemplate.WithCache(),
	)
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_, err := renderer.Process(context.Background())
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}
