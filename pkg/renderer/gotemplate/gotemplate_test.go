package gotemplate_test

import (
	"context"
	"testing"

	"testing/fstest"

	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/gotemplate"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
)

const podTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{ .Name }}-pod
  labels:
    app: {{ .Name }}
    component: {{ .Component }}
spec:
  containers:
  - name: nginx
    image: nginx:latest
`

const configMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}-config
  labels:
    app: {{ .Name }}
    component: {{ .Component }}
data:
  config.yaml: |
    port: {{ .Port }}
`

const invalidTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{ .InvalidField }}-pod
`

func TestRenderer(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name          string
		data          gotemplate.Data
		opts          []gotemplate.Option
		expectedCount int
		validation    types.GomegaMatcher
	}{
		{
			name: "should render single template",
			data: gotemplate.Data{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: map[string]interface{}{
					"Name":      "test-app",
					"Component": "frontend",
				},
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.name == "test-app-pod"`),
				jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
				jqmatcher.Match(`.metadata.labels["component"] == "frontend"`),
			),
		},
		{
			name: "should render multiple templates",
			data: gotemplate.Data{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl":       &fstest.MapFile{Data: []byte(podTemplate)},
					"templates/configmap.yaml.tpl": &fstest.MapFile{Data: []byte(configMapTemplate)},
				},
				Path: "templates/*.tpl",
				Values: map[string]interface{}{
					"Name":      "test-app",
					"Component": "frontend",
					"Port":      8080,
				},
			},
			expectedCount: 2,
			validation: Or(
				And(
					jqmatcher.Match(`.kind == "Pod"`),
					jqmatcher.Match(`.metadata.name == "test-app-pod"`),
				),
				And(
					jqmatcher.Match(`.kind == "ConfigMap"`),
					jqmatcher.Match(`.metadata.name == "test-app-config"`),
					jqmatcher.Match(`.data["config.yaml"] == "port: 8080\n"`),
				),
			),
		},
		{
			name: "should handle invalid template",
			data: gotemplate.Data{
				FS: fstest.MapFS{
					"templates/invalid.yaml.tpl": &fstest.MapFile{Data: []byte(invalidTemplate)},
				},
				Path: "templates/*.tpl",
				Values: map[string]interface{}{
					"Name": "test-app",
				},
			},
			expectedCount: 0,
			validation:    nil,
		},
		{
			name: "should apply filters",
			data: gotemplate.Data{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl":       &fstest.MapFile{Data: []byte(podTemplate)},
					"templates/configmap.yaml.tpl": &fstest.MapFile{Data: []byte(configMapTemplate)},
				},
				Path: "templates/*.tpl",
				Values: map[string]interface{}{
					"Name":      "test-app",
					"Component": "frontend",
					"Port":      8080,
				},
			},
			opts: []gotemplate.Option{
				gotemplate.WithFilter(gvk.NewFilter(corev1.SchemeGroupVersion.WithKind("Pod"))),
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.name == "test-app-pod"`),
			),
		},
		{
			name: "should apply transformers",
			data: gotemplate.Data{
				FS: fstest.MapFS{
					"templates/pod.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: map[string]interface{}{
					"Name":      "test-app",
					"Component": "frontend",
				},
			},
			opts: []gotemplate.Option{
				gotemplate.WithTransformer(labels.NewTransformer(map[string]string{
					"managed-by": "gotemplate-renderer",
					"env":        "test",
				})),
			},
			expectedCount: 1,
			validation: And(
				jqmatcher.Match(`.kind == "Pod"`),
				jqmatcher.Match(`.metadata.labels["managed-by"] == "gotemplate-renderer"`),
				jqmatcher.Match(`.metadata.labels["env"] == "test"`),
				jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
			),
		},
		{
			name: "should handle empty template",
			data: gotemplate.Data{
				FS:   fstest.MapFS{},
				Path: "templates/*.tpl",
				Values: map[string]interface{}{
					"Name": "test-app",
				},
			},
			expectedCount: 0,
			validation:    nil,
		},
		{
			name: "should handle non-existent template",
			data: gotemplate.Data{
				FS: fstest.MapFS{
					"templates/other.yaml.tpl": &fstest.MapFile{Data: []byte(podTemplate)},
				},
				Path: "templates/*.tpl",
				Values: map[string]interface{}{
					"Name": "test-app",
				},
			},
			expectedCount: 0,
			validation:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := gotemplate.New([]gotemplate.Data{tt.data}, tt.opts...)
			objects, err := renderer.Process(context.Background())

			if tt.validation == nil {
				g.Expect(err).To(HaveOccurred())
				g.Expect(objects).To(BeEmpty())
				return
			}

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(objects).To(HaveLen(tt.expectedCount))

			for _, obj := range objects {
				g.Expect(obj.Object).To(tt.validation)
			}
		})
	}
}

func TestNew(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name   string
		inputs []gotemplate.Data
	}{
		{
			name:   "should accept empty inputs",
			inputs: []gotemplate.Data{},
		},
		{
			name: "should accept input without path",
			inputs: []gotemplate.Data{{
				FS:     fstest.MapFS{},
				Values: map[string]interface{}{},
			}},
		},
		{
			name: "should accept valid input",
			inputs: []gotemplate.Data{{
				FS:     fstest.MapFS{},
				Path:   "templates/*.yaml",
				Values: map[string]interface{}{},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := gotemplate.New(tt.inputs)
			g.Expect(renderer).ToNot(BeNil())
		})
	}
}
