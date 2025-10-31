package main

import (
	"context"
	"fmt"
	"log"

	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Kustomize Hierarchy with Source Annotations ===")
	l.Log("Demonstrates: Tracking source files in hierarchical Kustomize structures with relative path imports")
	l.Log()

	renderer, err := kustomize.New(
		[]kustomize.Source{
			{
				Path:             "./kustomize-example/base",
				LoadRestrictions: kustomizetypes.LoadRestrictionsNone,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create kustomize renderer: %w", err)
	}

	e, err := engine.New(
		engine.WithRenderer(renderer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	l.Log("=== Rendering Kustomize Hierarchy ===")
	l.Log("Structure:")
	l.Log("  base/kustomization.yaml")
	l.Log("  ├── resources:")
	l.Log("  │   ├── deployment.yaml")
	l.Log("  │   └── service.yaml")
	l.Log("  └── imports: ../addons/kustomization.yaml")
	l.Log("      └── resources:")
	l.Log("          └── configmap.yaml")
	l.Log()

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("Successfully rendered %d objects with source tracking:\n\n", len(objects))

	for i, obj := range objects {
		l.Logf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			l.Logf(" (namespace: %s)", obj.GetNamespace())
		}
		l.Log()

		annotations := obj.GetAnnotations()
		if annotations != nil {
			l.Log("   Source Annotations:")
			if sourceType, ok := annotations[types.AnnotationSourceType]; ok {
				l.Logf("   - Type: %s\n", sourceType)
			}
			if sourcePath, ok := annotations[types.AnnotationSourcePath]; ok {
				l.Logf("   - Path: %s\n", sourcePath)
			}
			if sourceFile, ok := annotations[types.AnnotationSourceFile]; ok {
				l.Logf("   - File: %s\n", sourceFile)
			}
		}
		l.Log()
	}

	l.Log("=== Key Observations ===")
	l.Log("✓ base/kustomization.yaml imports ../addons/kustomization.yaml using relative path")
	l.Log("✓ Source annotations show the exact file each manifest originated from")
	l.Log("✓ Files from addons/ show relative path: ../addons/configmap.yaml")
	l.Log("✓ Files from base/ show relative path: deployment.yaml, service.yaml")
	l.Log("✓ All objects track the base path as the source (render entry point)")
	l.Log()
	l.Log("Note: LoadRestrictionsNone is required to allow importing resources from ../addons/")

	return nil
}
