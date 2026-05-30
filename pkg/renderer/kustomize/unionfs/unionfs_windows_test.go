//go:build windows

package unionfs_test

import (
	"testing"

	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize/unionfs"

	. "github.com/onsi/gomega"
)

func TestBuilderWindowsPaths(t *testing.T) {

	t.Run("should handle override with windows absolute path", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		windowsPath := `C:\repo\app\kustomization.yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(windowsPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})

	t.Run("should keep windows volume paths isolated", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		cDrivePath := `C:\repo\app\kustomization.yaml`
		dDrivePath := `D:\repo\app\kustomization.yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(cDrivePath, []byte(testContent1)).
			WithOverride(dDrivePath, []byte(testContent2)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(cDrivePath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))

		content, err = ufs.ReadFile(dDrivePath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent2))
	})

	t.Run("should handle override with windows volume only path", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		windowsVolumePath := `C:`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsVolumePath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(windowsVolumePath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})

	t.Run("should handle override with windows UNC path", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		uncPath := `\\server\share\repo\app\kustomization.yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(uncPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(uncPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})

	t.Run("should handle override with windows extended-length path", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		extendedPath := `\\?\C:\repo\app\kustomization.yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(extendedPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(extendedPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})
}
