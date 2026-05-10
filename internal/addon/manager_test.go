package addon

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	embeddedaddons "gdctl/addons"
	"gdctl/internal/bridge"
)

type fakePinger struct {
	ping bridge.PingResponse
	err  error
}

func (f fakePinger) Ping(context.Context) (bridge.PingResponse, error) {
	return f.ping, f.err
}

func TestResolveProjectRequiresProjectGodot(t *testing.T) {
	_, err := ResolveProject(t.TempDir())
	if err == nil {
		t.Fatal("expected missing project.godot error")
	}
	if !strings.Contains(err.Error(), "missing project.godot") {
		t.Fatalf("err = %v", err)
	}
}

func TestInstallEnableStatusAndRemove(t *testing.T) {
	project := newProject(t)
	manager := NewManager(testAddonFS())
	manager.NewClient = func(bridge.Config) bridgePinger {
		return fakePinger{ping: bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.3",
			ProtocolVersion: "gdctl.v1",
			Capabilities:    []string{"scene.tree"},
		}}
	}

	result, err := manager.Install(InstallOptions{ProjectPath: project})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatal("expected install to change files")
	}
	if _, err := os.Stat(filepath.Join(project, AddonDir, "plugin.cfg")); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(project, AddonDir, "bridge_plugin.gd")); !strings.Contains(got, "TEST_FIXTURE") {
		t.Fatalf("expected test addon fixture to be installed, got:\n%s", got)
	}

	if _, err := manager.Enable(project); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(context.Background(), StatusOptions{ProjectPath: project, CheckRuntime: true})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Installed || !status.Enabled || !status.Reachable || !status.Compatible {
		t.Fatalf("status = %#v", status)
	}

	result, err = manager.Remove(RemoveOptions{ProjectPath: project})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatal("expected remove to change files")
	}
	if _, err := os.Stat(filepath.Join(project, AddonDir, "plugin.cfg")); !os.IsNotExist(err) {
		t.Fatalf("plugin.cfg still exists, err = %v", err)
	}
}

func TestInstallRefusesUnknownExistingFileWithoutForce(t *testing.T) {
	project := newProject(t)
	addonPath := filepath.Join(project, AddonDir)
	if err := os.MkdirAll(addonPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(addonPath, "plugin.cfg"), []byte("local edit"), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(testAddonFS())
	_, err := manager.Install(InstallOptions{ProjectPath: project})
	if err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateCreatesBackupAndPreservesUnknownFiles(t *testing.T) {
	project := newProject(t)
	manager := NewManager(testAddonFS())
	manager.Now = func() time.Time {
		return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	}
	if _, err := manager.Install(InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(project, AddonDir, "local_notes.txt")
	if err := os.WriteFile(unknown, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, AddonDir, "plugin.cfg"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := manager.Update(UpdateOptions{ProjectPath: project})
	if err != nil {
		t.Fatal(err)
	}
	if result.Backup == "" {
		t.Fatal("expected backup path")
	}
	if _, err := os.Stat(filepath.Join(result.Backup, "plugin.cfg")); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, unknown); got != "keep me" {
		t.Fatalf("unknown file = %q", got)
	}
}

func TestPackageEmbeddedUpdateUsesManifestFiles(t *testing.T) {
	manifest, files, err := PackageEmbeddedUpdate(testAddonFS())
	if err != nil {
		t.Fatal(err)
	}
	if manifest["name"] != "godot_tcp_bridge" {
		t.Fatalf("manifest = %#v", manifest)
	}
	if len(files) != 4 {
		t.Fatalf("files len = %d", len(files))
	}
	foundPlugin := false
	for _, file := range files {
		if file.Path == "bridge_plugin.gd" {
			foundPlugin = true
			if file.ContentBase64 == "" {
				t.Fatal("bridge_plugin.gd content is empty")
			}
		}
	}
	if !foundPlugin {
		t.Fatal("bridge_plugin.gd missing from packaged update")
	}
}

func TestEmbeddedAddonManifestIncludesCommandDirectory(t *testing.T) {
	manifest, files, err := PackageEmbeddedUpdate(embeddedaddons.FS)
	if err != nil {
		t.Fatal(err)
	}
	manifestFiles := map[string]bool{}
	for _, value := range manifest["files"].([]any) {
		manifestFiles[value.(string)] = true
	}
	packagedFiles := map[string]bool{}
	for _, file := range files {
		packagedFiles[file.Path] = true
		if strings.HasPrefix(file.Path, "commands/") && file.ContentBase64 == "" {
			t.Fatalf("%s content is empty", file.Path)
		}
	}
	required := []string{
		"commands/request.gd",
		"commands/bridge_commands.gd",
		"commands/log_commands.gd",
		"commands/scene_commands.gd",
		"commands/node_commands.gd",
		"commands/script_commands.gd",
		"commands/shader_commands.gd",
		"commands/resource_commands.gd",
		"commands/file_commands.gd",
		"commands/viewport_commands.gd",
	}
	for _, path := range required {
		if !manifestFiles[path] {
			t.Fatalf("manifest missing %s", path)
		}
		if !packagedFiles[path] {
			t.Fatalf("packaged update missing %s", path)
		}
	}
}

func newProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "project.godot"), []byte(`[application]
config/name="demo"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func testAddonFS() fs.FS {
	return os.DirFS("testdata")
}
