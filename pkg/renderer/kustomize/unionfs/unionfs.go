//nolint:wrapcheck
package unionfs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/filesys"

	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	// ErrRemoveAllNotSupported is returned when RemoveAll is called on a union filesystem.
	ErrRemoveAllNotSupported = errors.New("RemoveAll not supported on union filesystem")
)

// unionFS provides a union filesystem that layers an in-memory FS over a delegate FS.
// Writes go to the memory layer, reads check memory first then fall back to delegate.
type unionFS struct {
	memory   filesys.FileSystem
	delegate filesys.FileSystem
}

func (u *unionFS) ReadFile(path string) ([]byte, error) {
	memoryPath := toMemoryPath(path)
	if u.memory.Exists(memoryPath) {
		return u.memory.ReadFile(memoryPath)
	}

	return u.delegate.ReadFile(path)
}

func (u *unionFS) WriteFile(path string, data []byte) error {
	return u.memory.WriteFile(toMemoryPath(path), data)
}

func (u *unionFS) Mkdir(path string) error {
	return u.memory.Mkdir(toMemoryPath(path))
}

func (u *unionFS) MkdirAll(path string) error {
	return u.memory.MkdirAll(toMemoryPath(path))
}

func (u *unionFS) RemoveAll(_ string) error {
	return ErrRemoveAllNotSupported
}

func (u *unionFS) Create(path string) (filesys.File, error) {
	return u.memory.Create(toMemoryPath(path))
}

func (u *unionFS) Open(path string) (filesys.File, error) {
	memoryPath := toMemoryPath(path)
	if u.memory.Exists(memoryPath) {
		return u.memory.Open(memoryPath)
	}

	return u.delegate.Open(path)
}

func (u *unionFS) Exists(path string) bool {
	return u.memory.Exists(toMemoryPath(path)) || u.delegate.Exists(path)
}

func (u *unionFS) IsDir(path string) bool {
	memoryPath := toMemoryPath(path)
	if u.memory.Exists(memoryPath) {
		return u.memory.IsDir(memoryPath)
	}

	return u.delegate.IsDir(path)
}

func (u *unionFS) ReadDir(path string) ([]string, error) {
	res := sets.New[string]()

	// Get files from memory layer
	memoryPath := toMemoryPath(path)
	if u.memory.Exists(memoryPath) && u.memory.IsDir(memoryPath) {
		files, err := u.memory.ReadDir(memoryPath)
		if err != nil {
			return nil, err
		}

		res.Insert(files...)
	}

	// Get files from delegate layer (deduplicate)
	if u.delegate.Exists(path) && u.delegate.IsDir(path) {
		files, err := u.delegate.ReadDir(path)
		if err != nil {
			return nil, err
		}

		res.Insert(files...)
	}

	return res.UnsortedList(), nil
}

func (u *unionFS) Glob(pattern string) ([]string, error) {
	res := sets.New[string]()

	// Get matches from memory layer
	files, err := u.memory.Glob(toMemoryPath(pattern))
	if err != nil {
		return nil, err
	}
	for i := range files {
		files[i] = fromMemoryPath(pattern, files[i])
	}

	res.Insert(files...)

	// Get matches from delegate layer (deduplicate)
	files, err = u.delegate.Glob(pattern)
	if err != nil {
		return nil, err
	}

	res.Insert(files...)

	return res.UnsortedList(), nil
}

func (u *unionFS) Walk(path string, walkFn filepath.WalkFunc) error {
	visited := make(map[string]bool)

	// Walk memory layer first
	memoryPath := toMemoryPath(path)
	if u.memory.Exists(memoryPath) {
		err := u.memory.Walk(memoryPath, func(p string, info fs.FileInfo, err error) error {
			p = fromMemoryPath(path, p)
			visited[p] = true

			return walkFn(p, info, err)
		})
		if err != nil {
			return err
		}
	}

	// Walk delegate layer (skip visited)
	if u.delegate.Exists(path) {
		err := u.delegate.Walk(path, func(p string, info fs.FileInfo, err error) error {
			if visited[p] {
				return nil
			}

			return walkFn(p, info, err)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *unionFS) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	return u.delegate.CleanedAbs(path)
}

const memoryVolumeRoot = "__unionfs_windows_volumes__"

// toMemoryPath converts host paths into paths that are safe for kyaml's in-memory FS.
// Examples: `C:\repo\app` -> `\__unionfs_windows_volumes__\Qzo\repo\app`,
// `\\server\share\repo` -> `\__unionfs_windows_volumes__\XFxzZXJ2ZXJcc2hhcmU\repo`.
func toMemoryPath(path string) string {
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	if volume == "" {
		return path
	}

	rest := strings.TrimPrefix(path, volume)
	rest = strings.TrimLeft(rest, `\/`)

	return filepath.Join(toMemoryVolumePath(volume), rest)
}

// fromMemoryPath converts memory FS results back to the caller's original volume.
// Example: reference `C:\repo\*.yaml` maps `\__unionfs_windows_volumes__\Qzo\repo\a.yaml` back to `C:\repo\a.yaml`.
func fromMemoryPath(referencePath, memoryPath string) string {
	volume := filepath.VolumeName(filepath.Clean(referencePath))
	if volume == "" {
		return memoryPath
	}

	prefix := toMemoryVolumePath(volume)
	if memoryPath == prefix {
		return volume + string(filepath.Separator)
	}
	if strings.HasPrefix(memoryPath, prefix+string(filepath.Separator)) {
		return volume + strings.TrimPrefix(memoryPath, prefix)
	}

	return memoryPath
}

// toMemoryVolumePath builds the internal memory FS root path for a Windows volume.
// Example: `C:` -> `\__unionfs_windows_volumes__\Qzo`.
func toMemoryVolumePath(volume string) string {
	return filepath.Join(string(filepath.Separator), memoryVolumeRoot, volumePathSegment(volume))
}

// volumePathSegment encodes a Windows volume name into a safe memory FS path segment.
// Examples: `C:` -> `Qzo`, `\\server\share` -> `XFxzZXJ2ZXJcc2hhcmU`.
func volumePathSegment(volume string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(volume))
}

// Builder provides a fluent API for constructing a union filesystem.
type Builder struct {
	delegate  filesys.FileSystem
	overrides map[string][]byte
}

// NewBuilder creates a new union FS builder wrapping the given delegate filesystem.
func NewBuilder(delegate filesys.FileSystem) *Builder {
	return &Builder{
		delegate:  delegate,
		overrides: make(map[string][]byte),
	}
}

// WithOverride adds a virtual file to the memory layer.
func (b *Builder) WithOverride(path string, content []byte) *Builder {
	b.overrides[path] = content

	return b
}

// WithOverrides adds multiple virtual files to the memory layer.
func (b *Builder) WithOverrides(overrides map[string][]byte) *Builder {
	maps.Copy(b.overrides, overrides)

	return b
}

// Build creates the union filesystem with all configured overrides.
func (b *Builder) Build() (filesys.FileSystem, error) {
	memory := filesys.MakeFsInMemory()

	for path, content := range b.overrides {
		if err := memory.WriteFile(toMemoryPath(path), content); err != nil {
			return nil, fmt.Errorf("failed to write override %s: %w", path, err)
		}
	}

	return &unionFS{
		memory:   memory,
		delegate: b.delegate,
	}, nil
}
