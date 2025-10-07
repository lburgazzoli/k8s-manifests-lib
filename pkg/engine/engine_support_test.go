package engine_test

import (
	"context"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/gotemplate"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/mem"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"

	. "github.com/onsi/gomega"
)

func TestHelm(t *testing.T) {
	g := NewWithT(t)

	t.Run("should create engine with Helm renderer", func(t *testing.T) {
		e, err := engine.Helm(helm.Source{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "test-release",
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())
	})

	t.Run("should return error for invalid source", func(t *testing.T) {
		e, err := engine.Helm(helm.Source{
			Chart: "", // Missing chart
		})

		g.Expect(err).Should(HaveOccurred())
		g.Expect(e).Should(BeNil())
	})
}

func TestKustomize(t *testing.T) {
	g := NewWithT(t)

	t.Run("should create engine with Kustomize renderer", func(t *testing.T) {
		e, err := engine.Kustomize(kustomize.Source{
			Path: "/some/path",
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())
	})

	t.Run("should return error for invalid source", func(t *testing.T) {
		e, err := engine.Kustomize(kustomize.Source{
			Path: "", // Missing path
		})

		g.Expect(err).Should(HaveOccurred())
		g.Expect(e).Should(BeNil())
	})
}

func TestYaml(t *testing.T) {
	g := NewWithT(t)

	t.Run("should create engine with YAML renderer", func(t *testing.T) {
		e, err := engine.Yaml(yaml.Source{
			FS:   os.DirFS("."),
			Path: "*.go",
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())
	})

	t.Run("should return error for invalid source", func(t *testing.T) {
		e, err := engine.Yaml(yaml.Source{
			// Missing FS and Path
		})

		g.Expect(err).Should(HaveOccurred())
		g.Expect(e).Should(BeNil())
	})
}

func TestGoTemplate(t *testing.T) {
	g := NewWithT(t)

	t.Run("should create engine with GoTemplate renderer", func(t *testing.T) {
		e, err := engine.GoTemplate(gotemplate.Source{
			FS:   os.DirFS("."),
			Path: "*.go",
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())
	})

	t.Run("should return error for invalid source", func(t *testing.T) {
		e, err := engine.GoTemplate(gotemplate.Source{
			// Missing FS and Path
		})

		g.Expect(err).Should(HaveOccurred())
		g.Expect(e).Should(BeNil())
	})
}

func TestMem(t *testing.T) {
	g := NewWithT(t)

	t.Run("should create engine with Mem renderer", func(t *testing.T) {
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}

		e, err := engine.Mem(mem.Source{
			Objects: []unstructured.Unstructured{
				{
					Object: map[string]any{
						"apiVersion": pod.APIVersion,
						"kind":       pod.Kind,
						"metadata": map[string]any{
							"name":      pod.Name,
							"namespace": pod.Namespace,
						},
					},
				},
			},
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())

		// Verify it can render
		objects, err := e.Render(context.Background())
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(HaveLen(1))
		g.Expect(objects[0].GetName()).Should(Equal("test-pod"))
	})

	t.Run("should create engine with empty objects", func(t *testing.T) {
		e, err := engine.Mem(mem.Source{
			Objects: []unstructured.Unstructured{},
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())

		// Verify it renders empty
		objects, err := e.Render(context.Background())
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).Should(BeEmpty())
	})
}
