package addon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetEnabledAppendsAndPreservesPlugins(t *testing.T) {
	projectFile := writeProjectGodot(t, `[application]
config/name="demo"

[editor_plugins]

enabled=PackedStringArray("res://addons/other/plugin.cfg")
`)

	changed, err := SetEnabled(projectFile, true)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected enable to change project.godot")
	}
	data := readFile(t, projectFile)
	for _, want := range []string{
		`"res://addons/other/plugin.cfg"`,
		`"res://addons/godot_tcp_bridge/plugin.cfg"`,
		`[application]`,
	} {
		if !strings.Contains(data, want) {
			t.Fatalf("project.godot missing %q:\n%s", want, data)
		}
	}

	changed, err = SetEnabled(projectFile, true)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected repeated enable to be idempotent")
	}
}

func TestSetEnabledCreatesSectionAndDisablesOnlyBridge(t *testing.T) {
	projectFile := writeProjectGodot(t, `[application]
config/name="demo"
`)

	changed, err := SetEnabled(projectFile, true)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	enabled, err := IsEnabled(projectFile)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected plugin to be enabled")
	}

	changed, err = SetEnabled(projectFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected disable to change project.godot")
	}
	enabled, err = IsEnabled(projectFile)
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected plugin to be disabled")
	}
}

func writeProjectGodot(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "project.godot")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
