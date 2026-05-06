package addon

import (
	"encoding/json"
	"fmt"
	"io/fs"
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
