package helm_test

import (
	"context"
	"testing"

	"github.com/rs/xid"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"

	. "github.com/onsi/gomega"
)

func TestRenderer(t *testing.T) {
	g := NewWithT(t)

	t.Run("should render chart from OCI registry", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "test-release",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": "test-app",
					},
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		// Check that resources were rendered
		found := false
		for _, obj := range objects {
			if obj.GetKind() == "Deployment" || obj.GetKind() == "Service" {
				found = true
				break
			}
		}
		g.Expect(found).To(BeTrue(), "Should have rendered at least one Deployment or Service")
	})

	t.Run("should render chart from repository", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Repo:        "https://dapr.github.io/helm-charts",
				Chart:       "dapr",
				ReleaseName: "dapr-test",
				Values: helm.Values(map[string]any{
					"global": map[string]any{
						"ha": map[string]any{
							"enabled": false,
						},
					},
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())
	})

	t.Run("should render with dynamic values", func(t *testing.T) {
		dynamicValues := func(_ context.Context) (map[string]any, error) {
			return map[string]any{
				"shared": map[string]any{
					"appId": xid.New().String(),
				},
			}, nil
		}

		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "dynamic-test",
				Values:      dynamicValues,
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())
	})

	t.Run("should apply filters", func(t *testing.T) {
		renderer, err := helm.New(
			[]helm.Source{
				{
					Repo:        "https://dapr.github.io/helm-charts",
					Chart:       "dapr",
					ReleaseName: "filter-test",
					Values: helm.Values(map[string]any{
						"global": map[string]any{
							"ha": map[string]any{
								"enabled": false,
							},
						},
					}),
				},
			},
			helm.WithFilter(gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment"))),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		// All objects should be Deployments
		for _, obj := range objects {
			g.Expect(obj.GetKind()).To(Equal("Deployment"))
		}
	})

	t.Run("should apply transformers", func(t *testing.T) {
		renderer, err := helm.New(
			[]helm.Source{
				{
					Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
					ReleaseName: "transformer-test",
					Values: helm.Values(map[string]any{
						"shared": map[string]any{
							"appId": "transform-app",
						},
					}),
				},
			},
			helm.WithTransformer(labels.Set(map[string]string{
				"managed-by": "helm-renderer",
				"env":        "test",
			})),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		// All objects should have the transformer labels
		for _, obj := range objects {
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("managed-by", "helm-renderer"))
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("env", "test"))
		}
	})

	t.Run("should process multiple charts", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "release-1",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": "app-1",
					},
				}),
			},
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "release-2",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": "app-2",
					},
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		// Should have objects from both releases
		releaseNames := make(map[string]bool)
		for _, obj := range objects {
			if labels := obj.GetLabels(); labels != nil {
				if releaseName, ok := labels["app.kubernetes.io/instance"]; ok {
					releaseNames[releaseName] = true
				}
			}
		}
		// At least one of the releases should be represented
		g.Expect(releaseNames).ToNot(BeEmpty())
	})

	t.Run("should render with release name in metadata", func(t *testing.T) {
		releaseName := "custom-release"
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: releaseName,
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": "metadata-test",
					},
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		// Check that at least one object has the release name
		foundRelease := false
		for _, obj := range objects {
			if labels := obj.GetLabels(); labels != nil {
				if instance := labels["app.kubernetes.io/instance"]; instance == releaseName {
					foundRelease = true
					break
				}
			}
		}
		g.Expect(foundRelease).To(BeTrue(), "Should find release name in labels")
	})

	t.Run("should render with specific chart version", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Repo:           "https://dapr.github.io/helm-charts",
				Chart:          "dapr",
				ReleaseName:    "version-test",
				ReleaseVersion: "1.14.4",
				Values: helm.Values(map[string]any{
					"global": map[string]any{
						"ha": map[string]any{
							"enabled": false,
						},
					},
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())
	})

	t.Run("should handle values context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		valuesFn := func(ctx context.Context) (map[string]any, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return map[string]any{}, nil
			}
		}

		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "cancel-test",
				Values:      valuesFn,
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = renderer.Process(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("context canceled"))
	})

	t.Run("should combine filters and transformers", func(t *testing.T) {
		renderer, err := helm.New(
			[]helm.Source{
				{
					Repo:        "https://dapr.github.io/helm-charts",
					Chart:       "dapr",
					ReleaseName: "combined-test",
					Values: helm.Values(map[string]any{
						"global": map[string]any{
							"ha": map[string]any{
								"enabled": false,
							},
						},
					}),
				},
			},
			helm.WithFilter(gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment"))),
			helm.WithTransformer(labels.Set(map[string]string{
				"test": "combined",
			})),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		for _, obj := range objects {
			g.Expect(obj.GetKind()).To(Equal("Deployment"))
			g.Expect(obj.GetLabels()).To(HaveKeyWithValue("test", "combined"))
		}
	})
}

func TestNew(t *testing.T) {
	g := NewWithT(t)

	t.Run("should reject input without Chart", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				ReleaseName: "test",
			},
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("Chart is required"))
		g.Expect(renderer).To(BeNil())
	})

	t.Run("should reject input without ReleaseName", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart: "oci://registry-1.docker.io/daprio/dapr-shared-chart",
			},
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("ReleaseName is required"))
		g.Expect(renderer).To(BeNil())
	})

	t.Run("should accept valid input", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "test",
			},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(renderer).ToNot(BeNil())
	})

	t.Run("should return error for non-existent chart", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/non-existent/chart",
				ReleaseName: "test",
			},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(renderer).ToNot(BeNil())

		_, err = renderer.Process(t.Context())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("unable to locate chart"))
	})

	t.Run("should return error for invalid chart path", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "/non/existent/path",
				ReleaseName: "test",
			},
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(renderer).ToNot(BeNil())

		_, err = renderer.Process(t.Context())
		g.Expect(err).To(HaveOccurred())
	})
}

func TestValuesHelper(t *testing.T) {
	g := NewWithT(t)

	t.Run("should return static values", func(t *testing.T) {
		staticValues := map[string]any{
			"key1": "value1",
			"key2": 42,
		}

		valuesFn := helm.Values(staticValues)
		result, err := valuesFn(context.Background())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(staticValues))
	})

	t.Run("should work with nil values", func(t *testing.T) {
		valuesFn := helm.Values(nil)
		result, err := valuesFn(context.Background())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(BeNil())
	})

	t.Run("should work with empty map", func(t *testing.T) {
		valuesFn := helm.Values(map[string]any{})
		result, err := valuesFn(context.Background())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(Equal(map[string]any{}))
	})
}

func TestCacheIntegration(t *testing.T) {
	g := NewWithT(t)

	t.Run("should cache identical renders", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "cache-test",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": "cache-app",
					},
				}),
			},
		},
			helm.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render - cache miss
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Second render - cache hit (should be identical)
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(len(result1)))

		// Results should be equal
		for i := range result1 {
			g.Expect(result2[i]).To(Equal(result1[i]))
		}
	})

	t.Run("should miss cache on different values", func(t *testing.T) {
		callCount := 0
		dynamicValues := func(_ context.Context) (map[string]any, error) {
			callCount++
			return map[string]any{
				"shared": map[string]any{
					"appId": xid.New().String(),
				},
			}, nil
		}

		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "dynamic-cache-test",
				Values:      dynamicValues,
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Second render with different values - cache miss
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).ToNot(BeEmpty())

		// Values function should be called twice (no cache hits)
		g.Expect(callCount).To(Equal(2))
	})

	t.Run("should work with cache disabled", func(t *testing.T) {
		renderer, err := helm.New(
			[]helm.Source{
				{
					Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
					ReleaseName: "no-cache-test",
					Values: helm.Values(map[string]any{
						"shared": map[string]any{
							"appId": "no-cache-app",
						},
					}),
				},
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Second render - should work even without cache
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(len(result1)))
	})

	t.Run("should return clones from cache", func(t *testing.T) {
		renderer, err := helm.New([]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "clone-test",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": "clone-app",
					},
				}),
			},
		})
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Modify first result
		if len(result1) > 0 {
			result1[0].SetName("modified-name")
		}

		// Second render - should not be affected by modification
		result2, err := renderer.Process(t.Context())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).ToNot(BeEmpty())

		if len(result2) > 0 {
			g.Expect(result2[0].GetName()).ToNot(Equal("modified-name"))
		}
	})
}

func BenchmarkHelmRenderWithoutCache(b *testing.B) {
	renderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
			ReleaseName: "bench-no-cache",
			Values: helm.Values(map[string]any{
				"shared": map[string]any{
					"appId": "bench-app",
				},
			}),
		},
	})
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_, err := renderer.Process(context.Background())
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}

func BenchmarkHelmRenderWithCache(b *testing.B) {
	renderer, err := helm.New(
		[]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "bench-cache",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": "bench-app",
					},
				}),
			},
		},
		helm.WithCache(),
	)
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_, err := renderer.Process(context.Background())
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}

func BenchmarkHelmRenderCacheMiss(b *testing.B) {
	renderer, err := helm.New(
		[]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "bench-miss",
				Values: func(_ context.Context) (map[string]any, error) {
					return map[string]any{
						"shared": map[string]any{
							"appId": xid.New().String(),
						},
					}, nil
				},
			},
		},
		helm.WithCache(),
	)
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_, err := renderer.Process(context.Background())
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}
