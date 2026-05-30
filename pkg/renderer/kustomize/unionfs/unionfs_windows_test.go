//go:build windows

package unionfs_test

import (
	"io/fs"
	"path/filepath"
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

	t.Run("should handle override with unsafe windows path segments", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		windowsPath := `C:\Users\RUNNER~1\AppData\Local\Temp\kustomization.yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(windowsPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})

	t.Run("should handle override with space in windows path segment", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		windowsPath := `C:\Program Files\app\kustomization.yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(windowsPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})

	t.Run("should handle override with dot-dot in windows path segment", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		windowsPath := `C:\repo\file..yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(windowsPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})

	t.Run("should handle override with internal prefix in windows path segment", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		windowsPath := `C:\repo\__unionfs_windows_segment__real\kustomization.yaml`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		content, err := ufs.ReadFile(windowsPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).Should(Equal(testContent1))
	})

	t.Run("should glob overrides under unsafe windows path segment", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsInMemory()
		windowsPath := `C:\Users\RUNNER~1\AppData\Local\Temp\file.txt`
		pattern := `C:\Users\RUNNER~1\AppData\Local\Temp\*.txt`

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		files, err := ufs.Glob(pattern)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(files).Should(ConsistOf(windowsPath))
	})

	t.Run("should walk overrides under unsafe windows path segment", func(t *testing.T) {
		g := NewWithT(t)
		delegate := filesys.MakeFsOnDisk()
		root := `C:\Users\RUNNER~1\AppData\Local\Temp`
		windowsPath := filepath.Join(root, "file.txt")

		ufs, err := unionfs.NewBuilder(delegate).
			WithOverride(windowsPath, []byte(testContent1)).
			Build()
		g.Expect(err).ToNot(HaveOccurred())

		var walked []string
		err = ufs.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				walked = append(walked, path)
			}

			return nil
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(walked).Should(ConsistOf(windowsPath))
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
