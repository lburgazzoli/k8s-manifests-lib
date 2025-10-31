# Example Refactoring Status

## Objective

Refactor all 30 examples to follow a testable pattern:
- `main()` calls exported `Run(context.Context) error` function
- `Run()` contains the logic and returns errors (no `log.Fatal`)
- `main_test.go` (package `main_test`) tests the `Run()` function

## Completed (9/30)

### ✅ 01-basic (4/4)
- [x] gotemplate - `main.go` + `main_test.go`
- [x] helm - `main.go` + `main_test.go`
- [x] kustomize - `main.go` + `main_test.go` + created missing kustomization files
- [x] yaml - `main.go` + `main_test.go`

### ✅ 02-filtering (4/4)
- [x] gvk - `main.go` + `main_test.go`
- [x] jq - `main.go` + `main_test.go`
- [x] labels - `main.go` + `main_test.go`
- [x] namespace - `main.go` + `main_test.go`

### 🔄 03-transformation (1/4)
- [x] labels - `main.go` + `main_test.go`
- [ ] annotations - needs refactoring
- [ ] name - needs refactoring
- [ ] namespace - needs refactoring

## Remaining (21/30)

### ⏳ 03-transformation (3 remaining)
- [ ] annotations/main.go
- [ ] name/main.go
- [ ] namespace/main.go

### ⏳ 04-composition (4)
- [ ] filter-boolean/main.go
- [ ] filter-conditional/main.go
- [ ] transformer-chain/main.go
- [ ] transformer-switch/main.go

### ⏳ 05-advanced (4)
- [ ] complex-nested/main.go
- [ ] conditional-transformations/main.go
- [ ] multi-environment/main.go
- [ ] three-level-pipeline/main.go

### ⏳ 06-renderers (4)
- [ ] dynamic-values/main.go
- [ ] multiple-renderers/main.go
- [ ] multiple-sources/main.go
- [ ] render-time-values/main.go

### ⏳ 07-caching (2)
- [ ] basic/main.go
- [ ] performance/main.go

### ⏳ 08-parallel (1)
- [ ] main.go

### ⏳ 09-metrics (1)
- [ ] basic/main.go

### ⏳ 10-source-annotations (2)
- [ ] basic/main.go
- [ ] kustomize-hierarchy/main.go

## Documentation

### ✅ Completed
- [x] Created `examples/CONTRIBUTING.md` with detailed pattern documentation
- [x] Updated `examples/README.md` with testing instructions
- [x] Pattern documented with complete examples

### Pattern Reference

**main.go**:
```go
package main

import (
	"context"
	"fmt"
	"log"
)

func main() {
	if err := Run(context.Background()); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	// Example logic here
	// Return errors instead of log.Fatal
	return nil
}
```

**main_test.go**:
```go
package main_test

import (
	"context"
	"testing"
	"time"

	example "github.com/lburgazzoli/k8s-manifests-lib/examples/<category>/<name>"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := example.Run(ctx); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
}
```

## Refactoring Steps (for each example)

1. **Read the original main.go** to understand the example
2. **Extract Run function**:
   - Change `func main()` logic → `func Run(ctx context.Context) error`
   - Replace `log.Fatal(err)` → `return fmt.Errorf("...: %w", err)`
   - Replace `context.Background()` → use `ctx` parameter
3. **Update main()**:
   ```go
   func main() {
       if err := Run(context.Background()); err != nil {
           log.Fatalf("Error: %v", err)
       }
   }
   ```
4. **Create main_test.go**:
   - Package: `main_test`
   - Import the example package
   - Test with timeout (30s for simple, 60s for Helm)
5. **Verify**:
   ```bash
   go run examples/<category>/<name>/main.go
   go test ./examples/<category>/<name>
   ```

## Notes

- All completed examples follow the pattern correctly
- Tests pass for all completed examples
- Helm examples need 60s timeout (network dependency)
- Simple examples (YAML, GoTemplate) use 30s timeout
- Missing files (like kustomization dirs) were created as needed

## Next Steps

Continue refactoring remaining 21 examples following the established pattern. Start with:
1. Complete 03-transformation (3 remaining)
2. Move to 04-composition (4 examples)
3. Continue through categories 05-10

Reference completed examples (01-basic, 02-filtering) for the pattern.

