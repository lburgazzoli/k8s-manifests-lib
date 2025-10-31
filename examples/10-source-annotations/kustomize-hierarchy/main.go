package main

import (
	"context"
	"fmt"
	"log"

	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

func main() {
	fmt.Println("=== Kustomize Hierarchy with Source Annotations ===")
	fmt.Println("Demonstrates: Tracking source files in hierarchical Kustomize structures with relative path imports")
	fmt.Println()

	renderer, err := kustomize.New(
		[]kustomize.Source{
			{
				Path:             "./kustomize-example/base",
				LoadRestrictions: kustomizetypes.LoadRestrictionsNone,
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to create Kustomize renderer: %v", err)
	}

	e, err := engine.New(
		engine.WithRenderer(renderer),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	fmt.Println("=== Rendering Kustomize Hierarchy ===")
	fmt.Println("Structure:")
	fmt.Println("  base/kustomization.yaml")
	fmt.Println("  ├── resources:")
	fmt.Println("  │   ├── deployment.yaml")
	fmt.Println("  │   └── service.yaml")
	fmt.Println("  └── imports: ../addons/kustomization.yaml")
	fmt.Println("      └── resources:")
	fmt.Println("          └── configmap.yaml")
	fmt.Println()

	objects, err := e.Render(ctx)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	fmt.Printf("Successfully rendered %d objects with source tracking:\n\n", len(objects))

	for i, obj := range objects {
		fmt.Printf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			fmt.Printf(" (namespace: %s)", obj.GetNamespace())
		}
		fmt.Println()

		annotations := obj.GetAnnotations()
		if annotations != nil {
			fmt.Println("   Source Annotations:")
			if sourceType, ok := annotations[types.AnnotationSourceType]; ok {
				fmt.Printf("   - Type: %s\n", sourceType)
			}
			if sourcePath, ok := annotations[types.AnnotationSourcePath]; ok {
				fmt.Printf("   - Path: %s\n", sourcePath)
			}
			if sourceFile, ok := annotations[types.AnnotationSourceFile]; ok {
				fmt.Printf("   - File: %s\n", sourceFile)
			}
		}
		fmt.Println()
	}

	fmt.Println("=== Key Observations ===")
	fmt.Println("✓ base/kustomization.yaml imports ../addons/kustomization.yaml using relative path")
	fmt.Println("✓ Source annotations show the exact file each manifest originated from")
	fmt.Println("✓ Files from addons/ show relative path: ../addons/configmap.yaml")
	fmt.Println("✓ Files from base/ show relative path: deployment.yaml, service.yaml")
	fmt.Println("✓ All objects track the base path as the source (render entry point)")
	fmt.Println()
	fmt.Println("Note: LoadRestrictionsNone is required to allow importing resources from ../addons/")
}
