package util_test

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"

	. "github.com/onsi/gomega"
)

const kindPod = "Pod"

func TestApplyFilters(t *testing.T) {
	g := NewWithT(t)
	ctx := t.Context()

	t.Run("should return all objects when no filters", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
		}

		result, err := util.ApplyFilters(ctx, objects, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result).To(Equal(objects))
	})

	t.Run("should filter objects with single filter", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
			makeObject("Pod", "pod2"),
		}

		podFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == kindPod, nil
		}

		result, err := util.ApplyFilters(ctx, objects, []types.Filter{podFilter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result[0].GetKind()).To(Equal("Pod"))
		g.Expect(result[1].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply multiple filters with AND logic", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObjectWithNamespace("Pod", "pod1", "default"),
			makeObjectWithNamespace("Pod", "pod2", "kube-system"),
			makeObjectWithNamespace("Service", "svc1", "default"),
		}

		podFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == kindPod, nil
		}

		namespaceFilter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetNamespace() == "default", nil
		}

		result, err := util.ApplyFilters(ctx, objects, []types.Filter{podFilter, namespaceFilter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetKind()).To(Equal("Pod"))
		g.Expect(result[0].GetName()).To(Equal("pod1"))
		g.Expect(result[0].GetNamespace()).To(Equal("default"))
	})

	t.Run("should return error when filter fails", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		errorFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, errors.New("filter error")
		}

		result, err := util.ApplyFilters(ctx, objects, []types.Filter{errorFilter})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("filter error"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should handle empty objects slice", func(t *testing.T) {
		objects := []unstructured.Unstructured{}

		filter := func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
			return obj.GetKind() == kindPod, nil
		}

		result, err := util.ApplyFilters(ctx, objects, []types.Filter{filter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})

	t.Run("should reject all objects if any filter rejects", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		acceptFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return true, nil
		}

		rejectFilter := func(_ context.Context, _ unstructured.Unstructured) (bool, error) {
			return false, nil
		}

		result, err := util.ApplyFilters(ctx, objects, []types.Filter{acceptFilter, rejectFilter})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})
}

func TestApplyTransformers(t *testing.T) {
	g := NewWithT(t)
	ctx := t.Context()

	t.Run("should return objects unchanged when no transformers", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
			makeObject("Service", "svc1"),
		}

		result, err := util.ApplyTransformers(ctx, objects, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(2))
		g.Expect(result).To(Equal(objects))
	})

	t.Run("should apply single transformer", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		addLabelTransformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["env"] = "test"
			obj.SetLabels(labels)
			return obj, nil
		}

		result, err := util.ApplyTransformers(ctx, objects, []types.Transformer{addLabelTransformer})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetLabels()).To(HaveKeyWithValue("env", "test"))
	})

	t.Run("should chain multiple transformers", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		addLabel1 := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["label1"] = "value1"
			obj.SetLabels(labels)
			return obj, nil
		}

		addLabel2 := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["label2"] = "value2"
			obj.SetLabels(labels)
			return obj, nil
		}

		result, err := util.ApplyTransformers(ctx, objects, []types.Transformer{addLabel1, addLabel2})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetLabels()).To(HaveKeyWithValue("label1", "value1"))
		g.Expect(result[0].GetLabels()).To(HaveKeyWithValue("label2", "value2"))
	})

	t.Run("should return error when transformer fails", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		errorTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("transformer error")
		}

		result, err := util.ApplyTransformers(ctx, objects, []types.Transformer{errorTransformer})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("transformer error"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should handle empty objects slice", func(t *testing.T) {
		objects := []unstructured.Unstructured{}

		transformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["test"] = "value"
			obj.SetLabels(labels)
			return obj, nil
		}

		result, err := util.ApplyTransformers(ctx, objects, []types.Transformer{transformer})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeEmpty())
	})

	t.Run("should stop on first transformer error", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		successTransformer := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			labels := obj.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["success"] = "true"
			obj.SetLabels(labels)
			return obj, nil
		}

		errorTransformer := func(_ context.Context, _ unstructured.Unstructured) (unstructured.Unstructured, error) {
			return unstructured.Unstructured{}, errors.New("second transformer failed")
		}

		result, err := util.ApplyTransformers(ctx, objects, []types.Transformer{successTransformer, errorTransformer})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("second transformer failed"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should preserve transformer order", func(t *testing.T) {
		objects := []unstructured.Unstructured{
			makeObject("Pod", "pod1"),
		}

		setAnnotation := func(key string, value string) types.Transformer {
			return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
				annotations := obj.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				annotations[key] = value
				obj.SetAnnotations(annotations)
				return obj, nil
			}
		}

		overwriteAnnotation := func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
			annotations := obj.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations["key"] = "overwritten"
			obj.SetAnnotations(annotations)
			return obj, nil
		}

		result, err := util.ApplyTransformers(ctx, objects, []types.Transformer{
			setAnnotation("key", "original"),
			overwriteAnnotation,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetAnnotations()).To(HaveKeyWithValue("key", "overwritten"))
	})
}

// Helper functions

func makeObject(kind string, name string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}

func makeObjectWithNamespace(kind string, name string, namespace string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}
