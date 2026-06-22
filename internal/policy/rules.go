package policy

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Violation describes a single policy rule failure.
type Violation struct {
	Rule    string
	Path    string
	Message string
}

func (v Violation) String() string {
	if v.Path != "" {
		return fmt.Sprintf("%s: %s (%s)", v.Rule, v.Path, v.Message)
	}
	return fmt.Sprintf("%s: %s", v.Rule, v.Message)
}

// CheckAssetFileSize returns violations for files that exceed the size limit.
func CheckAssetFileSize(policy *AssetPolicy, files []FileInfo) []Violation {
	if policy == nil || policy.MaxFileSizeMB <= 0 {
		return nil
	}
	limitBytes := int64(policy.MaxFileSizeMB * 1024 * 1024)
	var violations []Violation
	for _, f := range files {
		if f.SizeBytes > limitBytes {
			violations = append(violations, Violation{
				Rule:    "assets.max_file_size_mb",
				Path:    f.Path,
				Message: fmt.Sprintf("%.1f MB > limit %.1f MB", float64(f.SizeBytes)/1024/1024, policy.MaxFileSizeMB),
			})
		}
	}
	return violations
}

// CheckTextureFormat returns violations for textures with disallowed formats.
func CheckTextureFormat(policy *TexturePolicy, files []string) []Violation {
	if policy == nil || len(policy.AllowedFormats) == 0 {
		return nil
	}
	allowed := make(map[string]bool)
	for _, f := range policy.AllowedFormats {
		allowed[strings.ToLower(f)] = true
	}
	var violations []Violation
	for _, path := range files {
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
		if ext != "" && !allowed[ext] {
			violations = append(violations, Violation{
				Rule:    "textures.allowed_formats",
				Path:    path,
				Message: fmt.Sprintf("format %q not in allowed list", ext),
			})
		}
	}
	return violations
}

// CheckSceneNodeCount returns violations for scenes that exceed the node limit.
func CheckSceneNodeCount(policy *ScenePolicy, scenePath string, nodeCount int) []Violation {
	if policy == nil || policy.MaxNodeCount <= 0 {
		return nil
	}
	if nodeCount > policy.MaxNodeCount {
		return []Violation{{
			Rule:    "scenes.max_node_count",
			Path:    scenePath,
			Message: fmt.Sprintf("%d nodes > limit %d", nodeCount, policy.MaxNodeCount),
		}}
	}
	return nil
}

// FileInfo holds metadata for a single project file.
type FileInfo struct {
	Path      string
	SizeBytes int64
}
