package engine_test

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"

	. "github.com/onsi/gomega"
)

func TestNew(t *testing.T) {
	g := NewWithT(t)

	t.Run("should create empty engine", func(t *testing.T) {
		e := engine.New()
		g.Expect(e).ToNot(BeNil())
	})

	t.Run("should create engine with renderer", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("test-pod")})

		e := engine.New(engine.WithRenderer(renderer))
		g.Expect(e).ToNot(BeNil())
	})

	t.Run("should create engine with filter", func(t *testing.T) {
		filter := podFilter()
		e := engine.New(engine.WithFilter(filter))
		g.Expect(e).ToNot(BeNil())
	})

	t.Run("should create engine with transformer", func(t *testing.T) {
		transformer := addLabels(map[string]string{"test": "value"})
		e := engine.New(engine.WithTransformer(transformer))
		g.Expect(e).ToNot(BeNil())
	})
}

func TestEngineRender(t *testing.T) {
	g := NewWithT(t)

	t.Run("should render with single renderer", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{
			makePod("pod1"),
			makePod("pod2"),
		})

		e := engine.New(engine.WithRenderer(renderer))

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
		g.Expect(objects[1].GetName()).To(Equal("pod2"))
	})

	t.Run("should render with multiple renderers", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := newMockRenderer([]unstructured.Unstructured{makePod("pod2")})

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})

	t.Run("should apply engine-level filter", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{
			makePod("pod1"),
			makeService(),
		})

		filter := podFilter()
		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(filter),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply engine-level transformer", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})

		transformer := addLabels(map[string]string{
			"managed-by": "engine",
		})
		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(transformer),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("managed-by", "engine"))
	})

	t.Run("should apply render-time filter", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{
			makePod("pod1"),
			makeService(),
		})

		e := engine.New(engine.WithRenderer(renderer))

		filter := podFilter()
		objects, err := e.Render(t.Context(), engine.WithRenderFilter(filter))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply render-time transformer", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})

		e := engine.New(engine.WithRenderer(renderer))

		transformer := addLabels(map[string]string{
			"render-time": "true",
		})
		objects, err := e.Render(t.Context(), engine.WithRenderTransformer(transformer))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("render-time", "true"))
	})

	t.Run("should combine engine-level and render-time filters", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{
			makePod("pod1"),
			makeService(),
			makePodWithNamespace("pod2", "default"),
			makePodWithNamespace("pod3", "kube-system"),
		})

		// Engine-level: only Pods
		engineFilter := podFilter()
		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(engineFilter),
		)

		// Render-time: only default namespace
		renderFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == "default" || obj.GetNamespace() == "", nil
		}

		objects, err := e.Render(t.Context(), engine.WithRenderFilter(renderFilter))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2)) // pod1 (no namespace) and pod2 (default)
	})

	t.Run("should combine engine-level and render-time transformers", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})

		// Engine-level transformer
		engineTransformer := addLabels(map[string]string{
			"engine": "level",
		})
		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(engineTransformer),
		)

		// Render-time transformer
		renderTransformer := addLabels(map[string]string{
			"render": "time",
		})

		objects, err := e.Render(t.Context(), engine.WithRenderTransformer(renderTransformer))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("engine", "level"))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("render", "time"))
	})

	t.Run("should handle empty renderer", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{})

		e := engine.New(engine.WithRenderer(renderer))

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(BeEmpty())
	})

	t.Run("should handle no renderers", func(t *testing.T) {
		e := engine.New()

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(BeEmpty())
	})

	t.Run("should return error from failing renderer", func(t *testing.T) {
		failingRenderer := &mockRenderer{
			processFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
				return nil, errors.New("renderer failed")
			},
		}

		e := engine.New(engine.WithRenderer(failingRenderer))

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("renderer failed"))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should return error from failing filter", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})

		failingFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, errors.New("filter failed")
		}

		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(failingFilter),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("filter failed"))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should return error from failing transformer", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})

		failingTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("transformer failed")
		}

		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(failingTransformer),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("transformer failed"))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should apply multiple filters in sequence", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{
			makePodWithNamespace("pod1", "default"),
			makePodWithNamespace("pod2", "kube-system"),
			makeService(),
		})

		filter1 := podFilter()
		filter2 := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == "default", nil
		}

		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(filter1),
			engine.WithFilter(filter2),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
	})

	t.Run("should apply multiple transformers in sequence", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})

		transformer1 := addLabels(map[string]string{"label1": "value1"})
		transformer2 := addLabels(map[string]string{"label2": "value2"})

		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(transformer1),
			engine.WithTransformer(transformer2),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("label1", "value1"))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("label2", "value2"))
	})
}

// Helper functions

func makePod(name string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": name,
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))
	return obj
}

func makePodWithNamespace(name string, namespace string) unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))
	return obj
}

func makeService() unstructured.Unstructured {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": "svc1",
			},
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	return obj
}

// newMockRenderer creates a mock renderer that returns the given objects.
func newMockRenderer(objects []unstructured.Unstructured) *mockRenderer {
	return &mockRenderer{
		processFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return objects, nil
		},
	}
}

// podFilter returns a filter that only accepts Pod kind objects.
func podFilter() func(context.Context, unstructured.Unstructured) (bool, error) {
	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return obj.GetKind() == "Pod", nil
	}
}

// addLabels returns a transformer that adds the given labels to objects.
func addLabels(labels map[string]string) func(context.Context, unstructured.Unstructured) (unstructured.Unstructured, error) {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		existingLabels := obj.GetLabels()
		if existingLabels == nil {
			existingLabels = make(map[string]string)
		}
		for k, v := range labels {
			existingLabels[k] = v
		}
		obj.SetLabels(existingLabels)
		return obj, nil
	}
}

// mockRenderer is a mock implementation of types.Renderer for testing.
type mockRenderer struct {
	processFunc func(context.Context) ([]unstructured.Unstructured, error)
}

func (m *mockRenderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	return m.processFunc(ctx)
}
