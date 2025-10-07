package mem_test

import (
	"testing"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/mem"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"

	. "github.com/onsi/gomega"
)

func TestRenderer(t *testing.T) {
	g := NewWithT(t)

	// Test objects
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Labels: map[string]string{
				"app":       "test-app",
				"component": "frontend",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-config",
			Labels: map[string]string{
				"app":       "test-app",
				"component": "frontend",
			},
		},
		Data: map[string]string{
			"config.yaml": "port: 8080",
		},
	}

	tests := []struct {
		name          string
		objects       []runtime.Object
		opts          []mem.RendererOption
		expectedCount int
		validation    types.GomegaMatcher
	}{
		{
			name:          "should return empty list for no objects",
			objects:       []runtime.Object{},
			expectedCount: 0,
			validation:    nil,
		},
		{
			name:          "should return single object unchanged",
			objects:       []runtime.Object{pod},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.name == "test-pod"`),
				jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
				jqmatcher.Match(`.metadata.labels["component"] == "frontend"`),
			),
		},
		{
			name:          "should return multiple objects unchanged",
			objects:       []runtime.Object{pod, configMap},
			expectedCount: 2,
			validation: Or(
				And(
					jqmatcher.Match(`.kind == "Pod"`),
					jqmatcher.Match(`.metadata.name == "test-pod"`),
				),
				And(
					jqmatcher.Match(`.kind == "ConfigMap"`),
					jqmatcher.Match(`.metadata.name == "test-config"`),
				),
			),
		},
		{
			name:    "should apply filters",
			objects: []runtime.Object{pod, configMap},
			opts: []mem.RendererOption{
				mem.WithFilter(gvk.Filter(corev1.SchemeGroupVersion.WithKind("Pod"))),
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.name == "test-pod"`),
			),
		},
		{
			name:    "should apply transformers",
			objects: []runtime.Object{pod},
			opts: []mem.RendererOption{
				mem.WithTransformer(labels.Set(map[string]string{
					"managed-by": "mem-renderer",
					"env":        "test",
				})),
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.labels["managed-by"] == "mem-renderer"`),
				jqmatcher.Match(`.metadata.labels["env"] == "test"`),
				jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert typed objects to unstructured inline
			unstructuredObjects := make([]unstructured.Unstructured, len(tt.objects))
			for i, obj := range tt.objects {
				unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				g.Expect(err).ToNot(HaveOccurred())

				unstructuredObjects[i] = unstructured.Unstructured{Object: unstr}
			}

			renderer, err := mem.New([]mem.Source{{Objects: unstructuredObjects}}, tt.opts...)
			g.Expect(err).ToNot(HaveOccurred())

			objects, err := renderer.Process(t.Context())

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(objects).To(HaveLen(tt.expectedCount))

			if tt.validation != nil {
				for _, obj := range objects {
					g.Expect(obj.Object).To(tt.validation)
				}
			}
		})
	}
}
