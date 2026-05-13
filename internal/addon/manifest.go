package addon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"

	"gdctl/internal/bridge"
)

const (
	AddonDir       = "addons/godot_tcp_bridge"
	PluginResPath  = "res://addons/godot_tcp_bridge/plugin.cfg"
	ManifestName   = "gdctl_manifest.json"
	EmbeddedRoot   = "godot_tcp_bridge"
	BackupRootName = ".godot_tcp_bridge_backup"
)

type Manifest struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Files   []string `json:"files"`
}

func LoadInstalledManifest(projectPath string) (Manifest, error) {
	project, err := ResolveProject(projectPath)
	if err != nil {
		return Manifest{}, err
	}
	m := loadInstalledManifest(project)
	return m, nil
}

func LoadEmbeddedManifest(source fs.FS) (Manifest, error) {
	data, err := fs.ReadFile(source, EmbeddedRoot+"/"+ManifestName)
	if err != nil {
		return Manifest{}, fmt.Errorf("read embedded manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse embedded manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func PackageEmbeddedUpdate(source fs.FS) (map[string]any, []bridge.AddonUpdateFile, error) {
	manifest, err := LoadEmbeddedManifest(source)
	if err != nil {
		return nil, nil, err
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, err
	}
	var manifestMap map[string]any
	if err := json.Unmarshal(manifestJSON, &manifestMap); err != nil {
		return nil, nil, err
	}
	files := make([]bridge.AddonUpdateFile, 0, len(manifest.Files))
	for _, rel := range manifest.Files {
		data, err := fs.ReadFile(source, EmbeddedRoot+"/"+rel)
		if err != nil {
			return nil, nil, fmt.Errorf("read embedded addon file %s: %w", rel, err)
		}
		files = append(files, bridge.AddonUpdateFile{
			Path:          rel,
			ContentBase64: base64.StdEncoding.EncodeToString(data),
		})
	}
	return manifestMap, files, nil
}

func (m Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("manifest name is empty")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest version is empty")
	}
	if len(m.Files) == 0 {
		return fmt.Errorf("manifest files are empty")
	}
	for _, file := range m.Files {
		if file == "" {
			return fmt.Errorf("manifest contains empty file path")
		}
		if file[0] == '/' {
			return fmt.Errorf("manifest file must be relative: %s", file)
		}
	}
	return nil
}
