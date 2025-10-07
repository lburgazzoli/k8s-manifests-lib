package yaml_test

import (
	"context"
	"testing"
	"testing/fstest"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	corev1 "k8s.io/api/core/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"

	. "github.com/onsi/gomega"
)

const podYAML = `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test-app
    component: frontend
spec:
  containers:
  - name: nginx
    image: nginx:latest
`

const configMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  labels:
    app: test-app
    component: backend
data:
  config.yaml: "port: 8080"
`

const multiDocYAML = `
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  ports:
  - port: 80
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
type: Opaque
data:
  password: cGFzc3dvcmQ=
`

func TestRenderer(t *testing.T) {
	g := NewWithT(t)
	ctx := t.Context()

	t.Run("should load single YAML file", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "pod.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].Object).To(And(
			jqmatcher.Match(`.kind == "Pod"`),
			jqmatcher.Match(`.metadata.name == "test-pod"`),
			jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
		))
	})

	t.Run("should load multiple YAML files with glob", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})

	t.Run("should load multi-document YAML", func(t *testing.T) {
		testFS := fstest.MapFS{
			"resources.yaml": &fstest.MapFile{Data: []byte(multiDocYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "resources.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
		g.Expect(objects[0].GetKind()).To(Equal("Service"))
		g.Expect(objects[1].GetKind()).To(Equal("Secret"))
	})

	t.Run("should apply filters", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithFilter(gvk.Filter(corev1.SchemeGroupVersion.WithKind("Pod"))),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply transformers", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithTransformer(labels.Transform(map[string]string{
				"managed-by": "yaml-renderer",
				"env":        "test",
			})),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].Object).To(And(
			jqmatcher.Match(`.metadata.labels["managed-by"] == "yaml-renderer"`),
			jqmatcher.Match(`.metadata.labels["env"] == "test"`),
			jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
		))
	})

	t.Run("should handle .yml extension", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "pod.yml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
	})

	t.Run("should return error for non-existent pattern", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "nonexistent.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = renderer.Process(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("no files matched pattern"))
	})

	t.Run("should process multiple inputs", func(t *testing.T) {
		testFS1 := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}
		testFS2 := fstest.MapFS{
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS1, Path: "*.yaml"},
			{FS: testFS2, Path: "*.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})
}

func TestCacheIntegration(t *testing.T) {
	g := NewWithT(t)

	t.Run("should cache identical renders", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		},
			yaml.WithCache(),
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

	t.Run("should miss cache on different paths", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "pod.yaml"},
		},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render with pod.yaml
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(1))
		g.Expect(result1[0].GetKind()).To(Equal("Pod"))

		// Create new renderer with different path - cache miss
		renderer2, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "configmap.yaml"},
		},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		result2, err := renderer2.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(1))
		g.Expect(result2[0].GetKind()).To(Equal("ConfigMap"))
	})

	t.Run("should work with cache disabled", func(t *testing.T) {
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{
				{FS: testFS, Path: "*.yaml"},
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
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		},
			yaml.WithCache(),
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

func BenchmarkYamlRenderWithoutCache(b *testing.B) {
	testFS := fstest.MapFS{
		"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
		"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		"multi.yaml":     &fstest.MapFile{Data: []byte(multiDocYAML)},
	}

	renderer, err := yaml.New([]yaml.Source{
		{FS: testFS, Path: "*.yaml"},
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

func BenchmarkYamlRenderWithCache(b *testing.B) {
	testFS := fstest.MapFS{
		"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
		"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		"multi.yaml":     &fstest.MapFile{Data: []byte(multiDocYAML)},
	}

	renderer, err := yaml.New(
		[]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		},
		yaml.WithCache(),
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

func BenchmarkYamlRenderCacheMiss(b *testing.B) {
	testFS := fstest.MapFS{
		"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
		"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		// Create new renderer each time to ensure cache miss
		renderer, err := yaml.New(
			[]yaml.Source{
				{FS: testFS, Path: "*.yaml"},
			},
			yaml.WithCache(),
		)
		if err != nil {
			b.Fatalf("failed to create renderer: %v", err)
		}

		_, err = renderer.Process(context.Background())
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}
