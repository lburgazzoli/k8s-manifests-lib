package k8s

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCloneUnstructuredSlice creates a deep copy of a slice of unstructured objects.
// This is necessary because unstructured.Unstructured contains map[string]interface{}
// which needs deep copying to prevent mutations from affecting the original.
func DeepCloneUnstructuredSlice(objects []unstructured.Unstructured) []unstructured.Unstructured {
	if objects == nil {
		return nil
	}

	result := make([]unstructured.Unstructured, len(objects))
	for i, obj := range objects {
		result[i] = *obj.DeepCopy()
	}

	return result
}

// DecodeYAML decodes YAML content into a slice of unstructured objects.
func DecodeYAML(decoder runtime.Decoder, content []byte) ([]unstructured.Unstructured, error) {
	results := make([]unstructured.Unstructured, 0)

	r := bytes.NewReader(content)
	yd := yaml.NewDecoder(r)

	docIndex := 0
	for {
		var out map[string]interface{}

		err := yd.Decode(&out)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("unable to decode YAML document[%d]: %w", docIndex, err)
		}

		docIndex++

		if len(out) == 0 {
			continue
		}

		if out["kind"] == "" {
			continue
		}

		encoded, err := yaml.Marshal(out)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal YAML document[%d]: %w", docIndex-1, err)
		}

		var obj unstructured.Unstructured

		if _, _, err = decoder.Decode(encoded, nil, &obj); err != nil {
			if runtime.IsMissingKind(err) {
				continue
			}

			return nil, fmt.Errorf("unable to decode YAML document[%d]: %w", docIndex-1, err)
		}

		results = append(results, obj)
	}

	return results, nil
}
