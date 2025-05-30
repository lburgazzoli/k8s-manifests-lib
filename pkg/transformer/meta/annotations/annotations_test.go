package annotations_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
)

func toUnstructured(t *testing.T, obj runtime.Object) unstructured.Unstructured {
	t.Helper()
	unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err, "failed to convert object to unstructured")
	return unstructured.Unstructured{Object: unstr}
}

func TestTransform(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name               string
		annotationsToApply map[string]string
		inputObject        runtime.Object
		expected           types.GomegaMatcher
	}{
		{
			name:               "should add new annotations to empty annotations",
			annotationsToApply: map[string]string{"key1": "value1", "key2": "value2"},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expected: And(
				jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.annotations["key2"] == "value2"`),
			),
		},
		{
			name:               "should merge with existing annotations",
			annotationsToApply: map[string]string{"key2": "new-value2", "key3": "value3"},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key1": "value1", "key2": "old-value2"},
				},
			},
			expected: And(
				jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
				jqmatcher.Match(`.metadata.annotations["key2"] == "new-value2"`),
				jqmatcher.Match(`.metadata.annotations["key3"] == "value3"`),
			),
		},
		{
			name:               "should handle nil annotations map",
			annotationsToApply: nil,
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			expected: jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
		},
		{
			name:               "should handle empty annotations map",
			annotationsToApply: map[string]string{},
			inputObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			expected: jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
		},
		{
			name:               "should handle object with nil metadata",
			annotationsToApply: map[string]string{"key1": "value1"},
			inputObject:        &corev1.ConfigMap{},
			expected:           jqmatcher.Match(`.metadata.annotations["key1"] == "value1"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := annotations.Transform(tt.annotationsToApply)
			unstrObj := toUnstructured(t, tt.inputObject)
			transformed, err := transformer(t.Context(), unstrObj)

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(transformed.Object).To(tt.expected)
		})
	}
}
