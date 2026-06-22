// Package policy defines organizational rules for gdctl's policy layer.
// Policy files are JSON; YAML support can be added via gopkg.in/yaml.v3 when available.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
)

// Policy is the top-level structure for a gdctl policy file.
//
// Example policy.json:
//
//	{
//	  "textures": { "max_size": 2048 },
//	  "scenes":   { "max_node_count": 500 },
//	  "scripts":  { "require_type_hints": true },
//	  "assets":   { "max_file_size_mb": 10 }
//	}
type Policy struct {
	Textures *TexturePolicy `json:"textures,omitempty"`
	Scenes   *ScenePolicy   `json:"scenes,omitempty"`
	Scripts  *ScriptPolicy  `json:"scripts,omitempty"`
	Assets   *AssetPolicy   `json:"assets,omitempty"`
}

type TexturePolicy struct {
	MaxSize        int      `json:"max_size,omitempty"`
	AllowedFormats []string `json:"allowed_formats,omitempty"`
}

type ScenePolicy struct {
	MaxNodeCount int `json:"max_node_count,omitempty"`
}

type ScriptPolicy struct {
	RequireTypeHints bool `json:"require_type_hints,omitempty"`
}

type AssetPolicy struct {
	MaxFileSizeMB float64 `json:"max_file_size_mb,omitempty"`
}

// Load reads and parses a policy file.
func Load(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("policy: read %s: %w", path, err)
	}
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("policy: parse %s: %w", path, err)
	}
	return &p, nil
}
