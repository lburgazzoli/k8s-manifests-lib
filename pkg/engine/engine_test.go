package engine_test

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"

	. "github.com/onsi/gomega"
)

const (
	defaultNamespace = "default"
	systemNamespace  = "kube-system"
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
			makePodWithNamespace("pod2", defaultNamespace),
			makePodWithNamespace("pod3", systemNamespace),
		})

		// Engine-level: only Pods
		engineFilter := podFilter()
		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(engineFilter),
		)

		// Render-time: only default namespace
		renderFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == defaultNamespace || obj.GetNamespace() == "", nil
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
			processFunc: func(_ context.Context, _ map[string]any) ([]unstructured.Unstructured, error) {
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
			makePodWithNamespace("pod1", defaultNamespace),
			makePodWithNamespace("pod2", systemNamespace),
			makeService(),
		})

		filter1 := podFilter()
		filter2 := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == defaultNamespace, nil
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

	t.Run("should append struct-based RenderOptions filters to engine-level filters", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{
			makePodWithNamespace("pod1", defaultNamespace),
			makePodWithNamespace("pod2", systemNamespace),
			makeService(),
		})

		engineFilter := podFilter()
		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithFilter(engineFilter),
		)

		renderFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == defaultNamespace, nil
		}

		objects, err := e.Render(t.Context(), engine.RenderOptions{
			Filters: []types.Filter{renderFilter},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
	})

	t.Run("should append struct-based RenderOptions transformers to engine-level transformers", func(t *testing.T) {
		renderer := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})

		engineTransformer := addLabels(map[string]string{"engine": "level"})
		e := engine.New(
			engine.WithRenderer(renderer),
			engine.WithTransformer(engineTransformer),
		)

		renderTransformer := addLabels(map[string]string{"render": "time"})

		objects, err := e.Render(t.Context(), engine.RenderOptions{
			Transformers: []types.Transformer{renderTransformer},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("engine", "level"))
		g.Expect(objects[0].GetLabels()).To(HaveKeyWithValue("render", "time"))
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
		processFunc: func(_ context.Context, _ map[string]any) ([]unstructured.Unstructured, error) {
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
	processFunc func(context.Context, map[string]any) ([]unstructured.Unstructured, error)
	name        string
}

func (m *mockRenderer) Process(ctx context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
	return m.processFunc(ctx, values)
}

func (m *mockRenderer) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func TestParallelRendering(t *testing.T) {
	g := NewWithT(t)

	t.Run("should render with parallel enabled", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := newMockRenderer([]unstructured.Unstructured{makePod("pod2")})
		renderer3 := newMockRenderer([]unstructured.Unstructured{makePod("pod3")})

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
			engine.WithRenderer(renderer3),
			engine.WithParallel(true),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(3))

		names := []string{objects[0].GetName(), objects[1].GetName(), objects[2].GetName()}
		g.Expect(names).To(ContainElements("pod1", "pod2", "pod3"))
	})

	t.Run("should render sequentially with parallel disabled", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := newMockRenderer([]unstructured.Unstructured{makePod("pod2")})

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
			engine.WithParallel(false),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
		g.Expect(objects[1].GetName()).To(Equal("pod2"))
	})

	t.Run("should render sequentially by default", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := newMockRenderer([]unstructured.Unstructured{makePod("pod2")})

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
		g.Expect(objects[0].GetName()).To(Equal("pod1"))
		g.Expect(objects[1].GetName()).To(Equal("pod2"))
	})

	t.Run("should handle error in parallel mode", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := &mockRenderer{
			processFunc: func(_ context.Context, _ map[string]any) ([]unstructured.Unstructured, error) {
				return nil, errors.New("renderer2 failed")
			},
		}
		renderer3 := newMockRenderer([]unstructured.Unstructured{makePod("pod3")})

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
			engine.WithRenderer(renderer3),
			engine.WithParallel(true),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("renderer2 failed"))
		g.Expect(objects).To(BeNil())
	})

	t.Run("should apply filters after parallel rendering", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := newMockRenderer([]unstructured.Unstructured{makeService()})
		renderer3 := newMockRenderer([]unstructured.Unstructured{makePod("pod3")})

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
			engine.WithRenderer(renderer3),
			engine.WithFilter(podFilter()),
			engine.WithParallel(true),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		for _, obj := range objects {
			g.Expect(obj.GetKind()).To(Equal("Pod"))
		}
	})

	t.Run("should apply transformers after parallel rendering", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := newMockRenderer([]unstructured.Unstructured{makePod("pod2")})

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
			engine.WithTransformer(addLabels(map[string]string{"parallel": "true"})),
			engine.WithParallel(true),
		)

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		for _, obj := range objects {
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("parallel", "true"))
		}
	})

	t.Run("should handle empty renderers in parallel mode", func(t *testing.T) {
		e := engine.New(engine.WithParallel(true))

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(BeEmpty())
	})

	t.Run("should support struct-based option for parallel", func(t *testing.T) {
		renderer1 := newMockRenderer([]unstructured.Unstructured{makePod("pod1")})
		renderer2 := newMockRenderer([]unstructured.Unstructured{makePod("pod2")})

		e := engine.New(&engine.EngineOptions{
			Renderers: []types.Renderer{renderer1, renderer2},
			Parallel:  true,
		})

		objects, err := e.Render(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})
}

func TestRenderTimeValues(t *testing.T) {
	g := NewWithT(t)

	t.Run("should pass render-time values to renderer", func(t *testing.T) {
		var capturedValues map[string]any
		renderer := &mockRenderer{
			processFunc: func(_ context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
				capturedValues = values
				return []unstructured.Unstructured{makePod("test-pod")}, nil
			},
		}

		e := engine.New(engine.WithRenderer(renderer))

		renderValues := map[string]any{
			"replicaCount": 3,
			"image": map[string]any{
				"tag": "v2.0",
			},
		}

		objects, err := e.Render(t.Context(), engine.WithValues(renderValues))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))
		g.Expect(capturedValues).Should(Equal(renderValues))
	})

	t.Run("should pass empty map when no values provided", func(t *testing.T) {
		var capturedValues map[string]any
		renderer := &mockRenderer{
			processFunc: func(_ context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
				capturedValues = values
				return []unstructured.Unstructured{makePod("test-pod")}, nil
			},
		}

		e := engine.New(engine.WithRenderer(renderer))

		objects, err := e.Render(t.Context())

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))
		g.Expect(capturedValues).Should(BeEmpty())
	})

	t.Run("should pass same values to multiple renderers", func(t *testing.T) {
		var capturedValues1, capturedValues2 map[string]any

		renderer1 := &mockRenderer{
			processFunc: func(_ context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
				capturedValues1 = values
				return []unstructured.Unstructured{makePod("pod1")}, nil
			},
			name: "renderer1",
		}

		renderer2 := &mockRenderer{
			processFunc: func(_ context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
				capturedValues2 = values
				return []unstructured.Unstructured{makePod("pod2")}, nil
			},
			name: "renderer2",
		}

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
		)

		renderValues := map[string]any{
			"env": "production",
		}

		objects, err := e.Render(t.Context(), engine.WithValues(renderValues))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(2))
		g.Expect(capturedValues1).Should(Equal(renderValues))
		g.Expect(capturedValues2).Should(Equal(renderValues))
	})

	t.Run("should work with struct-based RenderOptions", func(t *testing.T) {
		var capturedValues map[string]any
		renderer := &mockRenderer{
			processFunc: func(_ context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
				capturedValues = values
				return []unstructured.Unstructured{makePod("test-pod")}, nil
			},
		}

		e := engine.New(engine.WithRenderer(renderer))

		renderValues := map[string]any{
			"key": "value",
		}

		objects, err := e.Render(t.Context(), engine.RenderOptions{
			Values: renderValues,
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))
		g.Expect(capturedValues).Should(Equal(renderValues))
	})

	t.Run("should pass values in parallel mode", func(t *testing.T) {
		var capturedValues1, capturedValues2 map[string]any

		renderer1 := &mockRenderer{
			processFunc: func(_ context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
				capturedValues1 = values
				return []unstructured.Unstructured{makePod("pod1")}, nil
			},
			name: "renderer1",
		}

		renderer2 := &mockRenderer{
			processFunc: func(_ context.Context, values map[string]any) ([]unstructured.Unstructured, error) {
				capturedValues2 = values
				return []unstructured.Unstructured{makePod("pod2")}, nil
			},
			name: "renderer2",
		}

		e := engine.New(
			engine.WithRenderer(renderer1),
			engine.WithRenderer(renderer2),
			engine.WithParallel(true),
		)

		renderValues := map[string]any{
			"parallel": true,
		}

		objects, err := e.Render(t.Context(), engine.WithValues(renderValues))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(2))
		g.Expect(capturedValues1).Should(Equal(renderValues))
		g.Expect(capturedValues2).Should(Equal(renderValues))
	})
}
