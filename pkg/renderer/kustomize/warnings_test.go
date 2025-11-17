package kustomize_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"

	. "github.com/onsi/gomega"
)

const deprecatedKustomization = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

commonLabels:
  app: test

resources:
- configmap.yaml
`

const simpleConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  foo: bar
`

func TestWarningHandler_Ignore(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "kustomization.yaml"),
		[]byte(deprecatedKustomization),
		0600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "configmap.yaml"),
		[]byte(simpleConfigMap),
		0600,
	)).To(Succeed())

	r, err := kustomize.New(
		[]kustomize.Source{{Path: tmpDir}},
		kustomize.WithWarningHandler(kustomize.WarningIgnore()),
	)
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := r.Process(ctx, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objects).To(HaveLen(1))
}

func TestWarningHandler_Log(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "kustomization.yaml"),
		[]byte(deprecatedKustomization),
		0600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "configmap.yaml"),
		[]byte(simpleConfigMap),
		0600,
	)).To(Succeed())

	var buf bytes.Buffer
	r, err := kustomize.New(
		[]kustomize.Source{{Path: tmpDir}},
		kustomize.WithWarningHandler(kustomize.WarningLog(&buf)),
	)
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := r.Process(ctx, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objects).To(HaveLen(1))

	output := buf.String()
	g.Expect(output).To(ContainSubstring("commonLabels"))
	g.Expect(output).To(ContainSubstring("deprecated"))
	g.Expect(output).To(ContainSubstring("labels"))
}

func TestWarningHandler_Fail(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "kustomization.yaml"),
		[]byte(deprecatedKustomization),
		0600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "configmap.yaml"),
		[]byte(simpleConfigMap),
		0600,
	)).To(Succeed())

	r, err := kustomize.New(
		[]kustomize.Source{{Path: tmpDir}},
		kustomize.WithWarningHandler(kustomize.WarningFail()),
	)
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := r.Process(ctx, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("kustomize warnings detected"))
	g.Expect(err.Error()).To(ContainSubstring("commonLabels"))
	g.Expect(objects).To(BeNil())
}

func TestWarningHandler_Custom(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "kustomization.yaml"),
		[]byte(deprecatedKustomization),
		0600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "configmap.yaml"),
		[]byte(simpleConfigMap),
		0600,
	)).To(Succeed())

	var collectedWarnings []string
	customHandler := func(warnings []string) error {
		collectedWarnings = append(collectedWarnings, warnings...)

		return nil
	}

	r, err := kustomize.New(
		[]kustomize.Source{{Path: tmpDir}},
		kustomize.WithWarningHandler(customHandler),
	)
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := r.Process(ctx, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objects).To(HaveLen(1))
	g.Expect(collectedWarnings).To(HaveLen(1))
	g.Expect(collectedWarnings[0]).To(ContainSubstring("commonLabels"))
}

func TestWarningHandler_Default(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "kustomization.yaml"),
		[]byte(deprecatedKustomization),
		0600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "configmap.yaml"),
		[]byte(simpleConfigMap),
		0600,
	)).To(Succeed())

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	renderer, err := kustomize.New([]kustomize.Source{{Path: tmpDir}})
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := renderer.Process(ctx, nil)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objects).To(HaveLen(1))

	output := buf.String()
	g.Expect(output).To(ContainSubstring("commonLabels"))
	g.Expect(output).To(ContainSubstring("deprecated"))
}

func TestWarningHandler_NoWarnings(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	nonDeprecatedKustomization := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- configmap.yaml
`

	tmpDir := t.TempDir()
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "kustomization.yaml"),
		[]byte(nonDeprecatedKustomization),
		0600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(tmpDir, "configmap.yaml"),
		[]byte(simpleConfigMap),
		0600,
	)).To(Succeed())

	var handlerCalled bool
	customHandler := func(_ []string) error {
		handlerCalled = true

		return errors.New("should not be called")
	}

	r, err := kustomize.New(
		[]kustomize.Source{{Path: tmpDir}},
		kustomize.WithWarningHandler(customHandler),
	)
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := r.Process(ctx, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objects).To(HaveLen(1))
	g.Expect(handlerCalled).To(BeFalse())
}

func TestWarningHandler_MultipleDeprecatedFields(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	multiDeprecatedKustomization := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

commonLabels:
  app: test

bases:
- ../base
`

	baseKust := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- configmap.yaml
`

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	g.Expect(os.MkdirAll(baseDir, 0750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(baseDir, "kustomization.yaml"),
		[]byte(baseKust),
		0600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(baseDir, "configmap.yaml"),
		[]byte(simpleConfigMap),
		0600,
	)).To(Succeed())

	overlayDir := filepath.Join(tmpDir, "overlay")
	g.Expect(os.MkdirAll(overlayDir, 0750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(overlayDir, "kustomization.yaml"),
		[]byte(multiDeprecatedKustomization),
		0600,
	)).To(Succeed())

	var buf bytes.Buffer
	r, err := kustomize.New(
		[]kustomize.Source{{Path: overlayDir}},
		kustomize.WithWarningHandler(kustomize.WarningLog(&buf)),
	)
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := r.Process(ctx, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(objects).To(HaveLen(1))

	output := buf.String()
	warnings := strings.Split(strings.TrimSpace(output), "\n")
	g.Expect(warnings).To(HaveLen(2))
	g.Expect(output).To(ContainSubstring("commonLabels"))
	g.Expect(output).To(ContainSubstring("bases"))
}
