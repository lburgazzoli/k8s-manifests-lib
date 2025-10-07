package kustomize_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/rs/xid"

	corev1 "k8s.io/api/core/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"

	. "github.com/onsi/gomega"
)

const basicKustomization = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namePrefix: test-

resources:
- configmap.yaml
- pod.yaml
`

const basicConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap
data:
  key: value
`

const basicPod = `
apiVersion: v1
kind: Pod
metadata:
  name: pod
spec:
  containers:
  - name: nginx
    image: nginx:latest
`

const baseKustomization = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- configmap.yaml
`

const baseConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  env: dev
`

const overlayKustomization = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

commonLabels:
  environment: production

resources:
- ../base

patches:
- patch: |-
    - op: replace
      path: /data/env
      value: prod
  target:
    kind: ConfigMap
`

const secondKustomization = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- service.yaml
`

const secondService = `
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  ports:
  - port: 80
`

const labelsNamespaceKustomization = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: custom-namespace

commonLabels:
  app: myapp

resources:
- deployment.yaml
`

const labelsNamespaceDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: placeholder
  template:
    metadata:
      labels:
        app: placeholder
    spec:
      containers:
      - name: nginx
        image: nginx:latest
`

func TestRenderer(t *testing.T) {
	g := NewWithT(t)

	t.Run("should render basic kustomization", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New([]kustomize.Source{
			{Path: dir},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		// Check that namespace prefix was applied
		g.Expect(objects[0].Object).To(Or(
			jqmatcher.Match(`.metadata.name == "test-configmap"`),
			jqmatcher.Match(`.metadata.name == "test-pod"`),
		))
	})

	t.Run("should render kustomization with overlay", func(t *testing.T) {
		dir := setupOverlayKustomization(t)

		overlayDir := filepath.Join(dir, "overlay")
		renderer, err := kustomize.New([]kustomize.Source{
			{Path: overlayDir},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))

		// Check that overlay label was applied
		g.Expect(objects[0].Object).To(And(
			jqmatcher.Match(`.kind == "ConfigMap"`),
			jqmatcher.Match(`.metadata.labels["environment"] == "production"`),
		))
	})

	t.Run("should apply filters", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New(
			[]kustomize.Source{{Path: dir}},
			kustomize.WithFilter(gvk.Filter(corev1.SchemeGroupVersion.WithKind("ConfigMap"))),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetKind()).To(Equal("ConfigMap"))
	})

	t.Run("should apply transformers", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New(
			[]kustomize.Source{{Path: dir}},
			kustomize.WithTransformer(labels.Transform(map[string]string{
				"managed-by": "kustomize-renderer",
			})),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		for _, obj := range objects {
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("managed-by", "kustomize-renderer"))
		}
	})

	t.Run("should process multiple inputs", func(t *testing.T) {
		dir1 := setupBasicKustomization(t)
		dir2 := setupSecondKustomization(t)

		renderer, err := kustomize.New([]kustomize.Source{
			{Path: dir1},
			{Path: dir2},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(3))
	})

	t.Run("should return error for non-existent path", func(t *testing.T) {
		renderer, err := kustomize.New([]kustomize.Source{
			{Path: "/non/existent/path"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = renderer.Process(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to run kustomize"))
	})

	t.Run("should apply kustomize labels and namespace", func(t *testing.T) {
		dir := setupKustomizationWithLabelsAndNamespace(t)

		renderer, err := kustomize.New([]kustomize.Source{
			{Path: dir},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))

		g.Expect(objects[0].Object).To(And(
			jqmatcher.Match(`.metadata.namespace == "custom-namespace"`),
			jqmatcher.Match(`.metadata.labels["app"] == "myapp"`),
		))
	})
}

func TestNew(t *testing.T) {
	g := NewWithT(t)

	t.Run("should reject input without path", func(t *testing.T) {
		renderer, err := kustomize.New([]kustomize.Source{{}})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("Path is required"))
		g.Expect(renderer).To(BeNil())
	})

	t.Run("should accept valid input", func(t *testing.T) {
		renderer, err := kustomize.New([]kustomize.Source{
			{Path: "/some/path"},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(renderer).ToNot(BeNil())
	})
}

// Helper functions to set up test kustomizations

func setupBasicKustomization(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, dir, "kustomization.yaml", basicKustomization)
	writeFile(t, dir, "configmap.yaml", basicConfigMap)
	writeFile(t, dir, "pod.yaml", basicPod)

	return dir
}

func setupOverlayKustomization(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create base
	baseDir := filepath.Join(dir, "base")
	err := os.Mkdir(baseDir, 0750)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, baseDir, "kustomization.yaml", baseKustomization)
	writeFile(t, baseDir, "configmap.yaml", baseConfigMap)

	// Create overlay
	overlayDir := filepath.Join(dir, "overlay")
	err = os.Mkdir(overlayDir, 0750)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, overlayDir, "kustomization.yaml", overlayKustomization)

	return dir
}

func setupSecondKustomization(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, dir, "kustomization.yaml", secondKustomization)
	writeFile(t, dir, "service.yaml", secondService)

	return dir
}

func setupKustomizationWithLabelsAndNamespace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, dir, "kustomization.yaml", labelsNamespaceKustomization)
	writeFile(t, dir, "deployment.yaml", labelsNamespaceDeployment)

	return dir
}

func writeFile(t *testing.T, dir string, name string, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0600)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValuesConfigMap(t *testing.T) {
	g := NewWithT(t)

	t.Run("should write values as ConfigMap", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		values := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		renderer, err := kustomize.New([]kustomize.Source{
			{
				Path:   dir,
				Values: kustomize.Values(values),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Render should create values.yaml
		_, err = renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())

		// values.yaml should be cleaned up after render
		valuesPath := filepath.Join(dir, "values.yaml")
		g.Expect(valuesPath).ToNot(BeAnExistingFile())
	})

	t.Run("should fail if values.yaml exists", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		// Pre-create values.yaml
		writeFile(t, dir, "values.yaml", "existing content")

		renderer, err := kustomize.New([]kustomize.Source{
			{
				Path: dir,
				Values: kustomize.Values(map[string]string{
					"key": "value",
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = renderer.Process(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("values.yaml already exists"))
	})

	t.Run("should clean up values.yaml after render", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New([]kustomize.Source{
			{
				Path: dir,
				Values: kustomize.Values(map[string]string{
					"test": "value",
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify cleanup
		valuesPath := filepath.Join(dir, "values.yaml")
		g.Expect(valuesPath).ToNot(BeAnExistingFile())
	})

	t.Run("should clean up values.yaml on error", func(t *testing.T) {
		dir := t.TempDir()

		// Create invalid kustomization
		writeFile(t, dir, "kustomization.yaml", "invalid: yaml: content:")

		renderer, err := kustomize.New([]kustomize.Source{
			{
				Path: dir,
				Values: kustomize.Values(map[string]string{
					"key": "value",
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = renderer.Process(t.Context())
		g.Expect(err).To(HaveOccurred())

		// Verify cleanup even on error
		valuesPath := filepath.Join(dir, "values.yaml")
		g.Expect(valuesPath).ToNot(BeAnExistingFile())
	})

	t.Run("should work without values", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New([]kustomize.Source{
			{Path: dir},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		// No values.yaml should be created
		valuesPath := filepath.Join(dir, "values.yaml")
		g.Expect(valuesPath).ToNot(BeAnExistingFile())
	})
}

func TestCacheIntegration(t *testing.T) {
	g := NewWithT(t)

	t.Run("should cache identical renders", func(t *testing.T) {
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New([]kustomize.Source{
			{
				Path: dir,
				Values: kustomize.Values(map[string]string{
					"key": "value",
				}),
			},
		},
			kustomize.WithCache(),
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
		dir := setupBasicKustomization(t)

		callCount := 0
		dynamicValues := func(_ context.Context) (map[string]string, error) {
			callCount++
			return map[string]string{
				"key": xid.New().String(),
			}, nil
		}

		renderer, err := kustomize.New([]kustomize.Source{
			{
				Path:   dir,
				Values: dynamicValues,
			},
		},
			kustomize.WithCache(),
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
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New(
			[]kustomize.Source{
				{
					Path: dir,
					Values: kustomize.Values(map[string]string{
						"key": "value",
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
		dir := setupBasicKustomization(t)

		renderer, err := kustomize.New([]kustomize.Source{
			{
				Path: dir,
				Values: kustomize.Values(map[string]string{
					"key": "value",
				}),
			},
		},
			kustomize.WithCache(),
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

func BenchmarkKustomizeRenderWithoutCache(b *testing.B) {
	dir := b.TempDir()

	writeFileB(b, dir, "kustomization.yaml", basicKustomization)
	writeFileB(b, dir, "configmap.yaml", basicConfigMap)
	writeFileB(b, dir, "pod.yaml", basicPod)

	renderer, err := kustomize.New([]kustomize.Source{
		{
			Path: dir,
			Values: kustomize.Values(map[string]string{
				"key1": "value1",
				"key2": "value2",
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

func BenchmarkKustomizeRenderWithCache(b *testing.B) {
	dir := b.TempDir()

	writeFileB(b, dir, "kustomization.yaml", basicKustomization)
	writeFileB(b, dir, "configmap.yaml", basicConfigMap)
	writeFileB(b, dir, "pod.yaml", basicPod)

	renderer, err := kustomize.New(
		[]kustomize.Source{
			{
				Path: dir,
				Values: kustomize.Values(map[string]string{
					"key1": "value1",
					"key2": "value2",
				}),
			},
		},
		kustomize.WithCache(),
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

func BenchmarkKustomizeRenderCacheMiss(b *testing.B) {
	dir := b.TempDir()

	writeFileB(b, dir, "kustomization.yaml", basicKustomization)
	writeFileB(b, dir, "configmap.yaml", basicConfigMap)
	writeFileB(b, dir, "pod.yaml", basicPod)

	renderer, err := kustomize.New(
		[]kustomize.Source{
			{
				Path: dir,
				Values: func(_ context.Context) (map[string]string, error) {
					return map[string]string{
						"key": xid.New().String(),
					}, nil
				},
			},
		},
		kustomize.WithCache(),
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

// Helper for benchmarks.
func writeFileB(b *testing.B, dir string, name string, content string) {
	b.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0600)
	if err != nil {
		b.Fatal(err)
	}
}
