package unionfs

import (
	"encoding/base64"
	"path/filepath"
	"regexp"
	"strings"
)

const memoryVolumeRoot = "__unionfs_windows_volumes__"
const encodedPathSegmentPrefix = "__unionfs_windows_segment__"

// memoryPathSegmentPattern matches path segment names accepted by kyaml's in-memory FS.
var memoryPathSegmentPattern = regexp.MustCompile(`^[a-zA-Z0-9-_.:]+$`)

// toMemoryPath converts host paths into paths that are safe for kyaml's in-memory FS.
// Examples: `C:\repo\app` -> `\__unionfs_windows_volumes__\Qzo\repo\app`,
// `\\server\share\repo` -> `\__unionfs_windows_volumes__\XFxzZXJ2ZXJcc2hhcmU\repo`,
// `C:\Users\RUNNER~1` -> `\__unionfs_windows_volumes__\Qzo\Users\__unionfs_windows_segment__UlVOTkVSfjE`.
func toMemoryPath(path string) string {
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	if volume == "" {
		return path
	}

	rest := strings.TrimPrefix(path, volume)
	rest = strings.TrimLeft(rest, `\/`)

	return joinPath(toMemoryVolumePath(volume), toMemoryPathSegments(rest))
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
		rest := strings.TrimPrefix(memoryPath, prefix)
		rest = strings.TrimLeft(rest, `\/`)

		return joinPath(volume+string(filepath.Separator), fromMemoryPathSegments(rest))
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

// joinPath joins root and path segments into a single filesystem path.
func joinPath(root string, segments []string) string {
	return filepath.Join(append([]string{root}, segments...)...)
}

// toMemoryPathSegments encodes unsafe path segments.
func toMemoryPathSegments(path string) []string {
	segments := pathSegments(path)
	for i := range segments {
		if shouldEncodePathSegment(segments[i]) {
			segments[i] = encodedPathSegmentPrefix + base64.RawURLEncoding.EncodeToString([]byte(segments[i]))
		}
	}

	return segments
}

// fromMemoryPathSegments decodes previously encoded path segments.
func fromMemoryPathSegments(path string) []string {
	segments := pathSegments(path)
	for i := range segments {
		segment, ok := strings.CutPrefix(segments[i], encodedPathSegmentPrefix)
		if !ok {
			// Leave segments that were not encoded unchanged.
			continue
		}

		decoded, err := base64.RawURLEncoding.DecodeString(segment)
		if err != nil {
			// Leave segments with invalid encoding unchanged.
			continue
		}

		segments[i] = string(decoded)
	}

	return segments
}

// pathSegments splits path into segments.
func pathSegments(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	})
}

// shouldEncodePathSegment reports whether segment needs encoding before storing in memory FS.
func shouldEncodePathSegment(segment string) bool {
	// Do not encode glob wildcards: *, ?, and [.
	if strings.ContainsAny(segment, "*?[") {
		return false
	}

	return segment == "." || strings.Contains(segment, "..") ||
		!memoryPathSegmentPattern.MatchString(segment) ||
		// Encode segments with the internal prefix so decode works correctly.
		strings.HasPrefix(segment, encodedPathSegmentPrefix)
}
