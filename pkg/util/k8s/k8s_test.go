package k8s_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/k8s"

	. "github.com/onsi/gomega"
)

const singleDocumentYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value
`

const multipleDocumentsYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy1
`

const emptyDocumentsYAML = `
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
---
`

const missingKindYAML = `
apiVersion: v1
metadata:
  name: no-kind
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: with-kind
`

const missingAPIVersionYAML = `
kind: ConfigMap
metadata:
  name: no-apiversion
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: with-apiversion
`

const emptyAPIVersionYAML = `
apiVersion: ""
kind: ConfigMap
metadata:
  name: empty-apiversion
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: valid
`

const nonStringFieldsYAML = `
apiVersion: 123
kind: ConfigMap
metadata:
  name: numeric-apiversion
---
apiVersion: v1
kind: 456
metadata:
  name: numeric-kind
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: valid
`

const invalidYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  invalid: [unclosed bracket
`

const yamlWithComments = `
# This is a comment
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  # Another comment
data:
  key: value # inline comment
`

const complexNestedYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy
  labels:
    app: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: app
        image: nginx:latest
        ports:
        - containerPort: 80
`

func TestDeepCloneUnstructuredSlice(t *testing.T) {
	t.Run("returns nil for nil input", func(t *testing.T) {
		g := NewWithT(t)

		result := k8s.DeepCloneUnstructuredSlice(nil)

		g.Expect(result).Should(BeNil())
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		g := NewWithT(t)

		result := k8s.DeepCloneUnstructuredSlice([]unstructured.Unstructured{})

		g.Expect(result).Should(BeEmpty())
	})

	t.Run("creates independent copy of objects", func(t *testing.T) {
		g := NewWithT(t)

		original := []unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "test",
					},
					"data": map[string]interface{}{
						"key": "value",
					},
				},
			},
		}

		cloned := k8s.DeepCloneUnstructuredSlice(original)

		g.Expect(cloned).Should(HaveLen(1))
		g.Expect(cloned[0].GetName()).Should(Equal("test"))

		// Modify the cloned object
		cloned[0].SetName("modified")
		cloned[0].Object["data"].(map[string]interface{})["key"] = "modified-value"

		// Verify original is unchanged
		g.Expect(original[0].GetName()).Should(Equal("test"))
		g.Expect(original[0].Object["data"].(map[string]interface{})["key"]).Should(Equal("value"))
	})

	t.Run("clones multiple objects", func(t *testing.T) {
		g := NewWithT(t)

		original := []unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "config1",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]interface{}{
						"name": "secret1",
					},
				},
			},
		}

		cloned := k8s.DeepCloneUnstructuredSlice(original)

		g.Expect(cloned).Should(HaveLen(2))
		g.Expect(cloned[0].GetKind()).Should(Equal("ConfigMap"))
		g.Expect(cloned[1].GetKind()).Should(Equal("Secret"))
	})
}

func TestDecodeYAML(t *testing.T) {
	t.Run("decodes single YAML document", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(singleDocumentYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(1))
		g.Expect(result[0].GetKind()).Should(Equal("ConfigMap"))
		g.Expect(result[0].GetName()).Should(Equal("test-config"))
	})

	t.Run("decodes multiple YAML documents", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(multipleDocumentsYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(3))
		g.Expect(result[0].GetKind()).Should(Equal("ConfigMap"))
		g.Expect(result[1].GetKind()).Should(Equal("Secret"))
		g.Expect(result[2].GetKind()).Should(Equal("Deployment"))
	})

	t.Run("skips empty documents", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(emptyDocumentsYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(2))
	})

	t.Run("skips documents without kind", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(missingKindYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(1))
		g.Expect(result[0].GetName()).Should(Equal("with-kind"))
	})

	t.Run("skips documents without apiVersion", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(missingAPIVersionYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(1))
		g.Expect(result[0].GetName()).Should(Equal("with-apiversion"))
	})

	t.Run("skips documents with empty apiVersion", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(emptyAPIVersionYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(1))
		g.Expect(result[0].GetName()).Should(Equal("valid"))
	})

	t.Run("skips documents with non-string kind or apiVersion", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(nonStringFieldsYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(1))
		g.Expect(result[0].GetName()).Should(Equal("valid"))
	})

	t.Run("handles empty content", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte{})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(BeEmpty())
	})

	t.Run("handles nil content", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML(nil)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(BeEmpty())
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		g := NewWithT(t)

		_, err := k8s.DecodeYAML([]byte(invalidYAML))

		g.Expect(err).Should(HaveOccurred())
		g.Expect(err.Error()).Should(ContainSubstring("unable to decode YAML document"))
	})

	t.Run("handles YAML with comments", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(yamlWithComments))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(1))
		g.Expect(result[0].GetKind()).Should(Equal("ConfigMap"))
	})

	t.Run("decodes complex nested structures", func(t *testing.T) {
		g := NewWithT(t)

		result, err := k8s.DecodeYAML([]byte(complexNestedYAML))

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).Should(HaveLen(1))
		g.Expect(result[0].GetKind()).Should(Equal("Deployment"))

		spec, found, err := unstructured.NestedMap(result[0].Object, "spec")
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(found).Should(BeTrue())
		g.Expect(spec).Should(HaveKey("replicas"))
	})
}

func TestToUnstructured(t *testing.T) {
	t.Run("converts map to unstructured", func(t *testing.T) {
		g := NewWithT(t)

		obj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
			},
		}

		result, err := k8s.ToUnstructured(&obj)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).ShouldNot(BeNil())
		g.Expect(result.GetKind()).Should(Equal("ConfigMap"))
		g.Expect(result.GetName()).Should(Equal("test"))
		g.Expect(result.GetNamespace()).Should(Equal("default"))
	})

	t.Run("converts struct to unstructured", func(t *testing.T) {
		g := NewWithT(t)

		type TestStruct struct {
			APIVersion string            `json:"apiVersion"`
			Kind       string            `json:"kind"`
			Metadata   map[string]string `json:"metadata"`
		}

		obj := TestStruct{
			APIVersion: "v1",
			Kind:       "Pod",
			Metadata: map[string]string{
				"name": "test-pod",
			},
		}

		result, err := k8s.ToUnstructured(&obj)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).ShouldNot(BeNil())
		g.Expect(result.GetKind()).Should(Equal("Pod"))
		g.Expect(result.GetName()).Should(Equal("test-pod"))
	})

	t.Run("handles empty map", func(t *testing.T) {
		g := NewWithT(t)

		obj := map[string]interface{}{}

		result, err := k8s.ToUnstructured(&obj)

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(result).ShouldNot(BeNil())
		g.Expect(result.Object).Should(BeEmpty())
	})

	t.Run("preserves nested structures", func(t *testing.T) {
		g := NewWithT(t)

		obj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port":       80,
						"targetPort": 8080,
					},
				},
				"selector": map[string]interface{}{
					"app": "test",
				},
			},
		}

		result, err := k8s.ToUnstructured(&obj)

		g.Expect(err).ShouldNot(HaveOccurred())

		spec, found, err := unstructured.NestedMap(result.Object, "spec")
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(found).Should(BeTrue())
		g.Expect(spec).Should(HaveKey("ports"))
		g.Expect(spec).Should(HaveKey("selector"))
	})
}
