//nolint:wrapcheck
package kustomize

import (
	"fmt"
	"path/filepath"
	"slices"

	goyaml "gopkg.in/yaml.v3"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type overrideFS struct {
	wrapped               filesys.FileSystem
	modifiedKustomization []byte
	kustomizationDir      string
}

func (o *overrideFS) ReadFile(path string) ([]byte, error) {
	cleanPath := filepath.Clean(path)
	base := filepath.Base(cleanPath)

	if slices.Contains(kustomizationFiles, base) {
		dir := filepath.Dir(cleanPath)
		if dir == o.kustomizationDir || dir == "." || cleanPath == base {
			return o.modifiedKustomization, nil
		}
	}

	return o.wrapped.ReadFile(path)
}

func (o *overrideFS) WriteFile(path string, data []byte) error {
	return o.wrapped.WriteFile(path, data)
}

func (o *overrideFS) Mkdir(path string) error {
	return o.wrapped.Mkdir(path)
}

func (o *overrideFS) MkdirAll(path string) error {
	return o.wrapped.MkdirAll(path)
}

func (o *overrideFS) RemoveAll(path string) error {
	return o.wrapped.RemoveAll(path)
}

func (o *overrideFS) Create(path string) (filesys.File, error) {
	return o.wrapped.Create(path)
}

func (o *overrideFS) Open(path string) (filesys.File, error) {
	return o.wrapped.Open(path)
}

func (o *overrideFS) Exists(path string) bool {
	return o.wrapped.Exists(path)
}

func (o *overrideFS) IsDir(path string) bool {
	return o.wrapped.IsDir(path)
}

func (o *overrideFS) ReadDir(path string) ([]string, error) {
	return o.wrapped.ReadDir(path)
}

func (o *overrideFS) Glob(pattern string) ([]string, error) {
	return o.wrapped.Glob(pattern)
}

func (o *overrideFS) Walk(path string, walkFn filepath.WalkFunc) error {
	return o.wrapped.Walk(path, walkFn)
}

func (o *overrideFS) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	return o.wrapped.CleanedAbs(path)
}

func newOverrideFS(fs filesys.FileSystem, path string, kust *kustomizetypes.Kustomization) (filesys.FileSystem, error) {
	modifiedContent, err := goyaml.Marshal(kust)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modified kustomization: %w", err)
	}

	ofs := &overrideFS{
		wrapped:               fs,
		modifiedKustomization: modifiedContent,
		kustomizationDir:      path,
	}

	return ofs, nil
}
