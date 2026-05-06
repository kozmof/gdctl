package addon

import (
	"context"
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
	manager := NewManager(embeddedaddons.FS)
	manager.NewClient = func(bridge.Config) bridgePinger {
		return fakePinger{ping: bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.0",
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

	manager := NewManager(embeddedaddons.FS)
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
	manager := NewManager(embeddedaddons.FS)
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
