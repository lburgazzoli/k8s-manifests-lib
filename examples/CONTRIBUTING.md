# Contributing Examples

## Example Structure

Every example in this directory must follow a standard, testable pattern. This ensures examples stay in sync with the library and don't break over time.

### Required Files

Each example directory must contain:
1. `main.go` - The example implementation
2. `main_test.go` - Test file that validates the example works

### main.go Structure

```go
package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	// ... other imports
)

func main() {
	if err := Run(context.Background()); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// Run contains the example logic and is exported for testing.
// It accepts a context for cancellation/timeout and returns errors instead of calling log.Fatal.
func Run(ctx context.Context) error {
	fmt.Println("=== Example Title ===")
	fmt.Println("Demonstrates: What this example shows")
	fmt.Println()
	
	// Create renderer
	renderer, err := someRenderer.New(...)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}
	
	// Create engine
	e, err := engine.New(
		engine.WithRenderer(renderer),
		// ... options
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}
	
	// Render
	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	
	// Print results
	fmt.Printf("Rendered %d objects\n", len(objects))
	
	return nil
}
```

### main_test.go Structure

```go
package main_test

import (
	"context"
	"testing"
	"time"
	
	example "github.com/lburgazzoli/k8s-manifests-lib/examples/<category>/<name>"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := example.Run(ctx); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
}
```

**Note**: Adjust timeout based on example complexity:
- Simple examples (YAML, GoTemplate): 30 seconds
- Network-dependent (Helm, OCI): 60 seconds
- Complex/slow operations: 90+ seconds

## Key Requirements

### 1. Exported Run Function

The `Run` function must be **exported** (capitalized) so it can be tested from the `main_test` package.

❌ Bad:
```go
func run(ctx context.Context) error {  // lowercase - not exported
```

✅ Good:
```go
func Run(ctx context.Context) error {  // capitalized - exported
```

### 2. Separate Test Package

Tests must use `package main_test` (not `package main`) for proper isolation.

❌ Bad:
```go
package main  // same package
```

✅ Good:
```go
package main_test  // separate package
```

### 3. Error Returns, Not log.Fatal

The `Run` function must return errors, not call `log.Fatal`.

❌ Bad:
```go
func Run(ctx context.Context) error {
	if err := something(); err != nil {
		log.Fatal(err)  // Don't use log.Fatal in Run()
	}
}
```

✅ Good:
```go
func Run(ctx context.Context) error {
	if err := something(); err != nil {
		return fmt.Errorf("failed to do something: %w", err)
	}
	return nil
}
```

### 4. Context Usage

Always use the provided context, not `context.Background()`.

❌ Bad:
```go
func Run(ctx context.Context) error {
	objects, err := e.Render(context.Background())  // Ignores ctx
}
```

✅ Good:
```go
func Run(ctx context.Context) error {
	objects, err := e.Render(ctx)  // Uses provided context
}
```

### 5. Error Wrapping

Use `fmt.Errorf` with `%w` to wrap errors with context.

❌ Bad:
```go
return err  // No context about what failed
```

✅ Good:
```go
return fmt.Errorf("failed to create engine: %w", err)
```

## Complete Example

Here's a complete example showing all requirements:

**examples/01-basic/yaml/main.go**:
```go
package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
)

//go:embed manifests/*.yaml
var manifestsFS embed.FS

func main() {
	if err := Run(context.Background()); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	fmt.Println("=== Basic YAML Example ===")
	fmt.Println("Demonstrates: Simple YAML file loading")
	fmt.Println()

	e, err := engine.Yaml(yaml.Source{
		FS:   manifestsFS,
		Path: "manifests/*.yaml",
	})
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	fmt.Printf("Successfully loaded %d objects\n", len(objects))

	return nil
}
```

**examples/01-basic/yaml/main_test.go**:
```go
package main_test

import (
	"context"
	"testing"
	"time"

	example "github.com/lburgazzoli/k8s-manifests-lib/examples/01-basic/yaml"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := example.Run(ctx); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
}
```

## Testing Your Example

Before submitting, ensure your example:

1. **Runs successfully**:
   ```bash
   go run examples/<category>/<name>/main.go
   ```

2. **Tests pass**:
   ```bash
   go test ./examples/<category>/<name>
   ```

3. **Compiles**:
   ```bash
   go build ./examples/<category>/<name>
   ```

## Common Mistakes

### 1. Forgetting to Export Run

```go
func run(ctx context.Context) error {  // ❌ Not exported
```

This causes test compilation errors: `example.Run undefined`

### 2. Using package main in Tests

```go
package main  // ❌ Should be main_test
```

This allows tests to access private functions, defeating the purpose of the exported `Run` function.

### 3. Not Using Context

```go
func Run(ctx context.Context) error {
	objects, err := e.Render(context.Background())  // ❌ Ignores ctx
```

This prevents timeout/cancellation from working in tests.

### 4. log.Fatal in Run Function

```go
func Run(ctx context.Context) error {
	if err != nil {
		log.Fatal(err)  // ❌ Can't be tested
	}
}
```

This causes the entire test process to exit instead of reporting test failure.

## Pattern Benefits

Following this pattern provides:

1. **Testability**: Examples can be tested automatically
2. **Documentation rot prevention**: Tests catch breaking API changes
3. **Consistency**: All examples follow the same structure
4. **Reliability**: Users can trust examples work correctly
5. **CI Integration**: Examples can be validated in CI pipeline

## Questions?

If you have questions about the example pattern or need help:

1. Check existing examples for reference
2. Review the [examples README](README.md)
3. Open an issue on GitHub

## Refactoring Existing Examples

If you're updating an older example to follow this pattern:

1. Extract main logic into exported `Run(ctx context.Context) error`
2. Replace `log.Fatal` calls with `return fmt.Errorf(...)`
3. Change `context.Background()` to use the provided `ctx`
4. Update `main()` to call `Run()` and handle errors
5. Create `main_test.go` with `package main_test`
6. Run tests to verify it works

See completed examples in `01-basic/` and `02-filtering/` for reference.

