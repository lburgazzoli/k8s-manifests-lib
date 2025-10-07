package jq_test

import (
	"testing"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/jq"

	. "github.com/onsi/gomega"
)

func TestNewEngine(t *testing.T) {
	g := NewWithT(t)

	t.Run("should create engine with valid expression", func(t *testing.T) {
		engine, err := jq.NewEngine(`.field`)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(engine).ToNot(BeNil())
	})

	t.Run("should return error for invalid expression", func(t *testing.T) {
		engine, err := jq.NewEngine(`invalid jq expression[[[`)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to parse JQ expression"))
		g.Expect(engine).To(BeNil())
	})

	t.Run("should create engine with complex expression", func(t *testing.T) {
		engine, err := jq.NewEngine(`.items[] | select(.type == "pod")`)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(engine).ToNot(BeNil())
	})

	t.Run("should create engine with identity expression", func(t *testing.T) {
		engine, err := jq.NewEngine(`.`)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(engine).ToNot(BeNil())
	})
}

func TestEngineRun(t *testing.T) {
	g := NewWithT(t)

	t.Run("should extract field from object", func(t *testing.T) {
		engine, err := jq.NewEngine(`.name`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"name": "test-pod",
			"type": "pod",
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal("test-pod"))
	})

	t.Run("should return entire object with identity", func(t *testing.T) {
		engine, err := jq.NewEngine(`.`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"name": "test",
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(input))
	})

	t.Run("should handle nested fields", func(t *testing.T) {
		engine, err := jq.NewEngine(`.metadata.name`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"metadata": map[string]any{
				"name": "nested-pod",
			},
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal("nested-pod"))
	})

	t.Run("should handle array access", func(t *testing.T) {
		engine, err := jq.NewEngine(`.items[0]`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"items": []any{"first", "second", "third"},
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal("first"))
	})

	t.Run("should handle boolean expressions", func(t *testing.T) {
		engine, err := jq.NewEngine(`.count > 5`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"count": 10,
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle string comparison", func(t *testing.T) {
		engine, err := jq.NewEngine(`.kind == "Pod"`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"kind": "Pod",
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle null values", func(t *testing.T) {
		engine, err := jq.NewEngine(`.missing`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"name": "test",
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeNil())
	})

	t.Run("should return error for multiple results", func(t *testing.T) {
		engine, err := jq.NewEngine(`.items[]`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"items": []any{"first", "second"},
		}

		result, err := engine.Run(input)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("multiple results returned"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should handle type conversion", func(t *testing.T) {
		engine, err := jq.NewEngine(`.count | tostring`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"count": 42,
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal("42"))
	})

	t.Run("should handle logical operators", func(t *testing.T) {
		engine, err := jq.NewEngine(`.enabled and .valid`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"enabled": true,
			"valid":   true,
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle pipe operations", func(t *testing.T) {
		engine, err := jq.NewEngine(`.value | . * 2`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": float64(21),
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(float64(42)))
	})
}

func TestEngineWithVariables(t *testing.T) {
	g := NewWithT(t)

	t.Run("should use variable in expression", func(t *testing.T) {
		engine, err := jq.NewEngine(`.name == $expected`, jq.WithVariable("expected", "test-pod"))
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"name": "test-pod",
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle variable without $ prefix", func(t *testing.T) {
		engine, err := jq.NewEngine(`.count > $threshold`, jq.WithVariable("threshold", 5))
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"count": 10,
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle variable with $ prefix", func(t *testing.T) {
		engine, err := jq.NewEngine(`.value == $myvar`, jq.WithVariable("$myvar", "expected"))
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": "expected",
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle multiple variables", func(t *testing.T) {
		engine, err := jq.NewEngine(
			`.value > $min and .value < $max`,
			jq.WithVariable("min", 5),
			jq.WithVariable("max", 15),
		)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": 10,
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle string variables", func(t *testing.T) {
		engine, err := jq.NewEngine(`.type == $targetType`, jq.WithVariable("targetType", "deployment"))
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"type": "deployment",
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})

	t.Run("should handle complex variable types", func(t *testing.T) {
		engine, err := jq.NewEngine(
			`.items | contains($expected)`,
			jq.WithVariable("expected", []any{"test"}),
		)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"items": []any{"test", "other"},
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeTrue())
	})
}

func TestEngineWithFunctions(t *testing.T) {
	g := NewWithT(t)

	t.Run("should use custom function", func(t *testing.T) {
		double := func(input any, args []any) any {
			if num, ok := input.(float64); ok {
				return num * 2
			}
			if num, ok := input.(int); ok {
				return float64(num * 2)
			}
			return input
		}

		engine, err := jq.NewEngine(`.value | double`, jq.WithFunction("double", 0, 0, double))
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": float64(21),
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(float64(42)))
	})

	t.Run("should use function with arguments", func(t *testing.T) {
		multiply := func(input any, args []any) any {
			if num, ok := input.(float64); ok {
				if len(args) > 0 {
					if multiplier, ok := args[0].(float64); ok {
						return num * multiplier
					}
					if multiplier, ok := args[0].(int); ok {
						return num * float64(multiplier)
					}
				}
			}
			return input
		}

		engine, err := jq.NewEngine(`.value | multiply(3)`, jq.WithFunction("multiply", 1, 1, multiply))
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": float64(7),
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(float64(21)))
	})

	t.Run("should use multiple custom functions", func(t *testing.T) {
		add10 := func(input any, args []any) any {
			if num, ok := input.(float64); ok {
				return num + 10
			}
			return input
		}

		double := func(input any, args []any) any {
			if num, ok := input.(float64); ok {
				return num * 2
			}
			return input
		}

		engine, err := jq.NewEngine(
			`.value | add10 | double`,
			jq.WithFunction("add10", 0, 0, add10),
			jq.WithFunction("double", 0, 0, double),
		)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": float64(5),
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(float64(30))) // (5 + 10) * 2 = 30
	})
}

func TestEngineWithCombinedOptions(t *testing.T) {
	g := NewWithT(t)

	t.Run("should combine variables and functions", func(t *testing.T) {
		addValue := func(input any, args []any) any {
			if num, ok := input.(float64); ok {
				if len(args) > 0 {
					if add, ok := args[0].(float64); ok {
						return num + add
					}
					if add, ok := args[0].(int); ok {
						return num + float64(add)
					}
				}
			}
			return input
		}

		engine, err := jq.NewEngine(
			`.count | addValue($increment)`,
			jq.WithFunction("addValue", 1, 1, addValue),
			jq.WithVariable("increment", float64(5)),
		)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"count": float64(10),
		}

		result, err := engine.Run(input)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(float64(15)))
	})
}

func TestEngineErrorCases(t *testing.T) {
	g := NewWithT(t)

	t.Run("should handle division by zero", func(t *testing.T) {
		engine, err := jq.NewEngine(`.value / 0`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": float64(10),
		}

		result, err := engine.Run(input)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("error during execution"))
		g.Expect(result).To(BeNil())
	})

	t.Run("should handle type errors", func(t *testing.T) {
		engine, err := jq.NewEngine(`.value + "string"`)
		g.Expect(err).ToNot(HaveOccurred())

		input := map[string]any{
			"value": float64(10),
		}

		result, err := engine.Run(input)
		g.Expect(err).To(HaveOccurred())
		g.Expect(result).To(BeNil())
	})
}
