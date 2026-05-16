package addon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdctl/internal/bridge"
)

// helpers

func newProjectWithAddon(t *testing.T) string {
	t.Helper()
	project := newProject(t)
	m := NewManager(testAddonFS())
	if _, err := m.Install(InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}
	return project
}

// NewManager

func TestNewManagerSetsDefaultNow(t *testing.T) {
	m := NewManager(testAddonFS())
	if m.Now == nil {
		t.Fatal("Now should be set")
	}
	if m.NewClient == nil {
		t.Fatal("NewClient should be set")
	}
	if m.Source == nil {
		t.Fatal("Source should be set")
	}
}

// Disable

func TestDisableAddon(t *testing.T) {
	project := newProject(t)
	m := NewManager(testAddonFS())
	if _, err := m.Install(InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Enable(project); err != nil {
		t.Fatal(err)
	}

	result, err := m.Disable(project)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Error("expected Changed=true when disabling an enabled addon")
	}
	if !strings.Contains(result.Message, "disabled") {
		t.Errorf("message = %q", result.Message)
	}
}

func TestDisableAlreadyDisabled(t *testing.T) {
	project := newProject(t)
	m := NewManager(testAddonFS())

	result, err := m.Disable(project)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed {
		t.Error("expected Changed=false when addon was not enabled")
	}
}

func TestDisableInvalidProject(t *testing.T) {
	m := NewManager(testAddonFS())
	_, err := m.Disable(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "project.godot") {
		t.Fatalf("expected project.godot error, got %v", err)
	}
}

// Doctor

func TestDoctorReportsProblems(t *testing.T) {
	project := newProject(t)
	m := NewManager(testAddonFS())
	m.NewClient = func(bridge.Config) bridgePinger {
		return fakePinger{err: errFake}
	}

	_, _, err := m.Doctor(context.Background(), DoctorOptions{ProjectPath: project})
	if err == nil || !strings.Contains(err.Error(), "problems") {
		t.Fatalf("expected doctor problems, got %v", err)
	}
}

func TestDoctorWithFixInstallsAndEnables(t *testing.T) {
	project := newProject(t)
	m := NewManager(testAddonFS())
	m.NewClient = func(bridge.Config) bridgePinger {
		return fakePinger{ping: bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.9",
			ProtocolVersion: "gdctl.v1",
		}}
	}

	status, actions, err := m.Doctor(context.Background(), DoctorOptions{ProjectPath: project, Fix: true})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Installed || !status.Enabled {
		t.Errorf("status = %#v", status)
	}
	if len(actions) == 0 {
		t.Error("expected at least one action taken")
	}
}

var errFake = errFakeType{}

type errFakeType struct{}

func (errFakeType) Error() string { return "fake error" }

// LoadInstalledManifest

func TestLoadInstalledManifestSuccess(t *testing.T) {
	project := newProjectWithAddon(t)
	m, err := LoadInstalledManifest(project)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "godot_tcp_bridge" {
		t.Errorf("Name = %q", m.Name)
	}
	if len(m.Files) == 0 {
		t.Error("Files should not be empty")
	}
}

func TestLoadInstalledManifestInvalidProject(t *testing.T) {
	_, err := LoadInstalledManifest(t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid project")
	}
}

// Manifest.Validate

func TestManifestValidateMissingName(t *testing.T) {
	m := Manifest{Version: "1.0", Files: []string{"a.gd"}}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected name error, got %v", err)
	}
}

func TestManifestValidateMissingVersion(t *testing.T) {
	m := Manifest{Name: "test", Files: []string{"a.gd"}}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected version error, got %v", err)
	}
}

func TestManifestValidateEmptyFiles(t *testing.T) {
	m := Manifest{Name: "test", Version: "1.0"}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "files") {
		t.Fatalf("expected files error, got %v", err)
	}
}

func TestManifestValidateEmptyFilePath(t *testing.T) {
	m := Manifest{Name: "test", Version: "1.0", Files: []string{""}}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "empty file path") {
		t.Fatalf("expected empty path error, got %v", err)
	}
}

func TestManifestValidateAbsoluteFilePath(t *testing.T) {
	m := Manifest{Name: "test", Version: "1.0", Files: []string{"/absolute/path"}}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "relative") {
		t.Fatalf("expected relative path error, got %v", err)
	}
}

// compatibility

func TestCompatibilityNotInstalled(t *testing.T) {
	ok, reason := compatibility(Status{Installed: false})
	if ok || !strings.Contains(reason, "not installed") {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestCompatibilityVersionMismatch(t *testing.T) {
	ok, reason := compatibility(Status{Installed: true, DeclaredVersion: "0.0.1"})
	if ok || !strings.Contains(reason, "installed addon version") {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestCompatibilityManifestVersionMismatch(t *testing.T) {
	ok, reason := compatibility(Status{Installed: true, ManifestVersion: "0.0.1"})
	if ok || !strings.Contains(reason, "manifest version") {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestCompatibilityRuntimeVersionMismatch(t *testing.T) {
	ok, reason := compatibility(Status{Installed: true, RuntimeVersion: "0.0.1"})
	if ok || !strings.Contains(reason, "runtime bridge version") {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestCompatibilityProtocolVersionMismatch(t *testing.T) {
	ok, reason := compatibility(Status{Installed: true, ProtocolVersion: "bad.proto"})
	if ok || !strings.Contains(reason, "protocol version") {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestCompatibilityCompatible(t *testing.T) {
	ok, reason := compatibility(Status{Installed: true})
	if !ok || reason != "compatible" {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

// removeEmptyParents

func TestRemoveEmptyParentsStopsAtRoot(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	removeEmptyParents(root, sub)
	if _, err := os.Stat(root); err != nil {
		t.Errorf("root should still exist: %v", err)
	}
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Error("sub should have been removed")
	}
}

func TestRemoveEmptyParentsKeepsNonEmpty(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// put a file in sub so it can't be removed
	if err := os.WriteFile(filepath.Join(sub, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	removeEmptyParents(root, sub)
	if _, err := os.Stat(sub); err != nil {
		t.Errorf("non-empty sub should still exist: %v", err)
	}
}

// ResolveProject

func TestResolveProjectNotExist(t *testing.T) {
	_, err := ResolveProject("/nonexistent/path/that/does/not/exist/xyz")
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected does-not-exist error, got %v", err)
	}
}

func TestResolveProjectNotDirectory(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	_, err = ResolveProject(f.Name())
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

func TestResolveProjectDefaultsToCurrent(t *testing.T) {
	// Just verify empty string doesn't panic (resolves to ".")
	// This will likely fail because . has no project.godot, but won't crash.
	_, _ = ResolveProject("")
}

// IsEnabled

func TestIsEnabled(t *testing.T) {
	dir := t.TempDir()
	projectFile := filepath.Join(dir, "project.godot")

	content := `[application]
config/name="demo"

[editor_plugins]

enabled=PackedStringArray("res://addons/godot_tcp_bridge/plugin.cfg")
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ok, err := IsEnabled(projectFile)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected plugin to be enabled")
	}
}

func TestIsEnabledFalseWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	projectFile := filepath.Join(dir, "project.godot")
	if err := os.WriteFile(projectFile, []byte(`[application]
config/name="demo"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	ok, err := IsEnabled(projectFile)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected plugin to not be enabled")
	}
}

// latestBackup — exercise via Update/Rollback roundtrip (already in existing tests)
// but test the no-backup case directly:

func TestRollbackNoBackups(t *testing.T) {
	project := newProject(t)
	m := NewManager(testAddonFS())
	_, err := m.Rollback(RollbackOptions{ProjectPath: project})
	if err == nil || !strings.Contains(err.Error(), "no addon backups") {
		t.Fatalf("expected no-backups error, got %v", err)
	}
}

func TestRollbackSpecificBackupNotExist(t *testing.T) {
	project := newProject(t)
	m := NewManager(testAddonFS())
	_, err := m.Rollback(RollbackOptions{ProjectPath: project, BackupPath: "/nonexistent"})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected does-not-exist error, got %v", err)
	}
}

func TestRollbackBackupNotDirectory(t *testing.T) {
	project := newProject(t)
	f, err := os.CreateTemp(t.TempDir(), "backup")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	m := NewManager(testAddonFS())
	_, err = m.Rollback(RollbackOptions{ProjectPath: project, BackupPath: f.Name()})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

func TestRollbackEmptyBackupDir(t *testing.T) {
	project := newProject(t)
	backupDir := t.TempDir()
	m := NewManager(testAddonFS())
	_, err := m.Rollback(RollbackOptions{ProjectPath: project, BackupPath: backupDir})
	if err == nil || !strings.Contains(err.Error(), "no files") {
		t.Fatalf("expected no-files error, got %v", err)
	}
}
