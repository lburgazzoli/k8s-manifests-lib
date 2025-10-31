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
	log := logger.FromContext(ctx)
	log.Log("=== Kustomize Hierarchy with Source Annotations ===")
	log.Log("Demonstrates: Tracking source files in hierarchical Kustomize structures with relative path imports")
	log.Log()

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

	log.Log("=== Rendering Kustomize Hierarchy ===")
	log.Log("Structure:")
	log.Log("  base/kustomization.yaml")
	log.Log("  ├── resources:")
	log.Log("  │   ├── deployment.yaml")
	log.Log("  │   └── service.yaml")
	log.Log("  └── imports: ../addons/kustomization.yaml")
	log.Log("      └── resources:")
	log.Log("          └── configmap.yaml")
	log.Log()

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	log.Logf("Successfully rendered %d objects with source tracking:\n\n", len(objects))

	for i, obj := range objects {
		log.Logf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			log.Logf(" (namespace: %s)", obj.GetNamespace())
		}
		log.Log()

		annotations := obj.GetAnnotations()
		if annotations != nil {
			log.Log("   Source Annotations:")
			if sourceType, ok := annotations[types.AnnotationSourceType]; ok {
				log.Logf("   - Type: %s\n", sourceType)
			}
			if sourcePath, ok := annotations[types.AnnotationSourcePath]; ok {
				log.Logf("   - Path: %s\n", sourcePath)
			}
			if sourceFile, ok := annotations[types.AnnotationSourceFile]; ok {
				log.Logf("   - File: %s\n", sourceFile)
			}
		}
		log.Log()
	}

	log.Log("=== Key Observations ===")
	log.Log("✓ base/kustomization.yaml imports ../addons/kustomization.yaml using relative path")
	log.Log("✓ Source annotations show the exact file each manifest originated from")
	log.Log("✓ Files from addons/ show relative path: ../addons/configmap.yaml")
	log.Log("✓ Files from base/ show relative path: deployment.yaml, service.yaml")
	log.Log("✓ All objects track the base path as the source (render entry point)")
	log.Log()
	log.Log("Note: LoadRestrictionsNone is required to allow importing resources from ../addons/")

	return nil
}
