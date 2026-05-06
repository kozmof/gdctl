package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdctl/internal/addon"
	"gdctl/internal/bridge"
)

func TestRunPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:            true,
			Engine:        "Godot",
			EngineVersion: "4.4.1",
			PluginVersion: "0.1.0",
			ProjectName:   "my-game",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "ping"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Godot bridge: ok", "Engine: Godot 4.4.1", "Project: my-game", "Plugin: 0.1.0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunSceneTree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.SceneTreeResponse{
			OK: true,
			Root: bridge.NodeInfo{
				Name: "Main",
				Type: "Node2D",
				Children: []bridge.NodeInfo{
					{Name: "Player", Type: "CharacterBody2D"},
				},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "scene", "tree"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "└── Player CharacterBody2D") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNodeAddRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "add", "--parent", "/root/Main"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--parent, --type, and --name") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeRemoveRequiresPathBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "remove"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestAddonStatusJSON(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"addon", "install", "--project", project}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(project, addon.AddonDir, "bridge_plugin.gd")); !strings.Contains(got, "TEST_FIXTURE") {
		t.Fatalf("expected CLI to install test addon fixture, got:\n%s", got)
	}
	stdout.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.0",
			ProtocolVersion: "gdctl.v1",
			Capabilities:    []string{"scene.tree"},
		})
	}))
	defer server.Close()

	args := append(serverArgs(server), "addon", "status", "--project", project, "--json")
	err = Run(context.Background(), args, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	var status map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status["installed"] != true || status["reachable"] != true || status["compatible"] != true {
		t.Fatalf("status = %#v\nstdout:\n%s", status, stdout.String())
	}
}

func TestAddonDoctorFixInstallsAndEnables(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"addon", "doctor", "--project", project, "--fix"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[fix] addon installed", "[fix] addon enabled", "[ok] addon installed", "[ok] addon enabled"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	data, err := os.ReadFile(filepath.Join(project, "project.godot"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "res://addons/godot_tcp_bridge/plugin.cfg") {
		t.Fatalf("project.godot not enabled:\n%s", string(data))
	}
}

func serverArgs(server *httptest.Server) []string {
	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(hostPort, ":")
	return []string{"--host", parts[0], "--port", parts[1]}
}

func newCLIProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "project.godot"), []byte(`[application]
config/name="demo"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func useTestAddon(t *testing.T) {
	t.Helper()
	previous := newAddonManager
	newAddonManager = func() addon.Manager {
		return addon.NewManager(cliTestAddonFS())
	}
	t.Cleanup(func() {
		newAddonManager = previous
	})
}

func cliTestAddonFS() fs.FS {
	return os.DirFS(filepath.Join("..", "addon", "testdata"))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
