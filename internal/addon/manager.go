package addon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gdctl/internal/bridge"
	"gdctl/internal/version"
)

type Manager struct {
	Source    fs.FS
	Now       func() time.Time
	NewClient func(bridge.Config) bridgePinger
}

type bridgePinger interface {
	Ping(context.Context) (bridge.PingResponse, error)
}

type InstallOptions struct {
	ProjectPath string
	Force       bool
}

type UpdateOptions struct {
	ProjectPath string
}

type StatusOptions struct {
	ProjectPath  string
	BridgeConfig bridge.Config
	CheckRuntime bool
}

type RemoveOptions struct {
	ProjectPath string
}

type DoctorOptions struct {
	ProjectPath  string
	BridgeConfig bridge.Config
	Fix          bool
}

type Status struct {
	Installed       bool     `json:"installed"`
	AddonPath       string   `json:"addon_path"`
	Enabled         bool     `json:"enabled"`
	DeclaredVersion string   `json:"declared_version,omitempty"`
	ManifestVersion string   `json:"manifest_version,omitempty"`
	EmbeddedVersion string   `json:"embedded_version"`
	RuntimeVersion  string   `json:"runtime_version,omitempty"`
	Reachable       bool     `json:"reachable"`
	Compatible      bool     `json:"compatible"`
	Compatibility   string   `json:"compatibility"`
	ProtocolVersion string   `json:"protocol_version"`
	Capabilities    []string `json:"capabilities,omitempty"`
	RuntimeError    string   `json:"runtime_error,omitempty"`
}

type Result struct {
	Changed bool
	Message string
	Backup  string
}

func NewManager(source fs.FS) Manager {
	return Manager{
		Source: source,
		Now:    time.Now,
		NewClient: func(cfg bridge.Config) bridgePinger {
			return bridge.NewClient(cfg)
		},
	}
}

func (m Manager) Install(opts InstallOptions) (Result, error) {
	project, manifest, err := m.projectAndManifest(opts.ProjectPath)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(project.AddonPath(), 0o755); err != nil {
		return Result{}, err
	}
	changed := false
	hasInstalledManifest := fileExists(filepath.Join(project.AddonPath(), ManifestName))
	for _, rel := range manifest.Files {
		srcPath := EmbeddedRoot + "/" + filepath.ToSlash(rel)
		src, err := fs.ReadFile(m.Source, srcPath)
		if err != nil {
			return Result{}, fmt.Errorf("read embedded addon file %s: %w", rel, err)
		}
		dst := filepath.Join(project.AddonPath(), filepath.FromSlash(rel))
		if fileExists(dst) {
			current, err := os.ReadFile(dst)
			if err != nil {
				return Result{}, err
			}
			if string(current) == string(src) {
				continue
			}
			if !opts.Force && !hasInstalledManifest {
				return Result{}, fmt.Errorf("refusing to overwrite existing addon file without manifest: %s", dst)
			}
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(dst, src, 0o644); err != nil {
			return Result{}, err
		}
		changed = true
	}
	if changed {
		return Result{Changed: true, Message: "addon installed"}, nil
	}
	return Result{Message: "addon already installed"}, nil
}

func (m Manager) Update(opts UpdateOptions) (Result, error) {
	project, embedded, err := m.projectAndManifest(opts.ProjectPath)
	if err != nil {
		return Result{}, err
	}
	installed := loadInstalledManifest(project)
	if len(installed.Files) == 0 {
		installed = embedded
	}
	backup, err := m.backupManagedFiles(project, installed)
	if err != nil {
		return Result{}, err
	}
	result, err := m.Install(InstallOptions{ProjectPath: opts.ProjectPath, Force: true})
	if err != nil {
		return Result{}, err
	}
	result.Backup = backup
	if !result.Changed {
		result.Message = "addon already up to date"
	}
	return result, nil
}

func (m Manager) Enable(projectPath string) (Result, error) {
	project, err := ResolveProject(projectPath)
	if err != nil {
		return Result{}, err
	}
	changed, err := SetEnabled(project.ProjectGodot, true)
	if err != nil {
		return Result{}, err
	}
	if changed {
		return Result{Changed: true, Message: "addon enabled"}, nil
	}
	return Result{Message: "addon already enabled"}, nil
}

func (m Manager) Disable(projectPath string) (Result, error) {
	project, err := ResolveProject(projectPath)
	if err != nil {
		return Result{}, err
	}
	changed, err := SetEnabled(project.ProjectGodot, false)
	if err != nil {
		return Result{}, err
	}
	if changed {
		return Result{Changed: true, Message: "addon disabled"}, nil
	}
	return Result{Message: "addon already disabled"}, nil
}

func (m Manager) Remove(opts RemoveOptions) (Result, error) {
	project, err := ResolveProject(opts.ProjectPath)
	if err != nil {
		return Result{}, err
	}
	manifest := loadInstalledManifest(project)
	if len(manifest.Files) == 0 {
		embedded, err := LoadEmbeddedManifest(m.Source)
		if err != nil {
			return Result{}, err
		}
		manifest = embedded
	}
	changed := false
	for _, rel := range manifest.Files {
		dst := filepath.Join(project.AddonPath(), filepath.FromSlash(rel))
		if err := os.Remove(dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Result{}, err
		}
		changed = true
		removeEmptyParents(project.AddonPath(), filepath.Dir(dst))
	}
	_ = os.Remove(project.AddonPath())
	if changed {
		return Result{Changed: true, Message: "addon removed"}, nil
	}
	return Result{Message: "addon already absent"}, nil
}

func (m Manager) Status(ctx context.Context, opts StatusOptions) (Status, error) {
	project, err := ResolveProject(opts.ProjectPath)
	if err != nil {
		return Status{}, err
	}
	status := Status{
		AddonPath:       project.AddonPath(),
		EmbeddedVersion: version.EmbeddedBridgeVersion,
		ProtocolVersion: version.ProtocolVersion,
	}
	status.Installed = fileExists(project.PluginPath())
	status.DeclaredVersion = readPluginVersion(project.PluginPath())
	installed := loadInstalledManifest(project)
	status.ManifestVersion = installed.Version
	status.Enabled, _ = IsEnabled(project.ProjectGodot)

	if opts.CheckRuntime {
		client := m.NewClient(opts.BridgeConfig)
		ping, err := client.Ping(ctx)
		if err == nil && ping.OK {
			status.Reachable = true
			status.RuntimeVersion = ping.PluginVersion
			status.Capabilities = ping.Capabilities
			if ping.ProtocolVersion != "" {
				status.ProtocolVersion = ping.ProtocolVersion
			}
		} else if err != nil {
			status.RuntimeError = err.Error()
		}
	}
	status.Compatible, status.Compatibility = compatibility(status)
	return status, nil
}

func (m Manager) Doctor(ctx context.Context, opts DoctorOptions) (Status, []string, error) {
	var actions []string
	if opts.Fix {
		if result, err := m.Install(InstallOptions{ProjectPath: opts.ProjectPath}); err != nil {
			return Status{}, actions, err
		} else if result.Changed {
			actions = append(actions, result.Message)
		}
		if result, err := m.Enable(opts.ProjectPath); err != nil {
			return Status{}, actions, err
		} else if result.Changed {
			actions = append(actions, result.Message)
		}
	}
	status, err := m.Status(ctx, StatusOptions{ProjectPath: opts.ProjectPath, BridgeConfig: opts.BridgeConfig, CheckRuntime: true})
	if err != nil {
		return Status{}, actions, err
	}
	if !opts.Fix && (!status.Installed || !status.Enabled) {
		return status, actions, errors.New("addon doctor found problems")
	}
	return status, actions, nil
}

func (m Manager) projectAndManifest(projectPath string) (Project, Manifest, error) {
	project, err := ResolveProject(projectPath)
	if err != nil {
		return Project{}, Manifest{}, err
	}
	manifest, err := LoadEmbeddedManifest(m.Source)
	if err != nil {
		return Project{}, Manifest{}, err
	}
	return project, manifest, nil
}

func (m Manager) backupManagedFiles(project Project, manifest Manifest) (string, error) {
	timestamp := m.Now().UTC().Format("20060102T150405Z")
	backup := filepath.Join(project.Path, "addons", BackupRootName, timestamp)
	copied := false
	for _, rel := range manifest.Files {
		src := filepath.Join(project.AddonPath(), filepath.FromSlash(rel))
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		dst := filepath.Join(backup, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return "", err
		}
		copied = true
	}
	if !copied {
		return "", nil
	}
	return backup, nil
}

func loadInstalledManifest(project Project) Manifest {
	data, err := os.ReadFile(filepath.Join(project.AddonPath(), ManifestName))
	if err != nil {
		return Manifest{}
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}
	}
	return manifest
}

func readPluginVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version=") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "version=")), `"`)
		}
	}
	return ""
}

func compatibility(status Status) (bool, string) {
	if !status.Installed {
		return false, "addon is not installed"
	}
	if status.DeclaredVersion != "" && status.DeclaredVersion != version.EmbeddedBridgeVersion {
		return false, "installed addon version differs from embedded addon version"
	}
	if status.ManifestVersion != "" && status.ManifestVersion != version.EmbeddedBridgeVersion {
		return false, "installed manifest version differs from embedded addon version"
	}
	if status.RuntimeVersion != "" && status.RuntimeVersion != version.EmbeddedBridgeVersion {
		return false, "runtime bridge version differs from embedded addon version"
	}
	if status.ProtocolVersion != "" && status.ProtocolVersion != version.ProtocolVersion {
		return false, "runtime protocol version differs from CLI protocol version"
	}
	return true, "compatible"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func removeEmptyParents(root, dir string) {
	for {
		if dir == root || dir == "." || dir == string(filepath.Separator) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
