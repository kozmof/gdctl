package cli

import (
	"bytes"
	"context"
	"encoding/base64"
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

func TestRunSceneSave(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"queued": true,
					"job_id": "save-1",
					"path":   "res://main.tscn",
				},
			})
		case "/jobs/save-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "save-1",
					Kind:   "scene.save",
					Status: "succeeded",
					Result: map[string]any{"path": "res://main.tscn"},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "scene", "save"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Scene saved: res://main.tscn") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestRunSceneCreate(t *testing.T) {
	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":      "res://scenes/Main.tscn",
				"root_type": "Node2D",
				"root_name": "Main",
				"root_path": "/root/Main",
				"created":   true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "scene", "create", "--path", "res://scenes/Main.tscn", "--root", "Node2D", "--name", "Main")
	err := Run(context.Background(), args, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "scene.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scenes/Main.tscn" || gotEnvelope.Params["root_type"] != "Node2D" || gotEnvelope.Params["root_name"] != "Main" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Scene created: res://scenes/Main.tscn") || !strings.Contains(stdout.String(), "Root: /root/Main Node2D") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunSceneCreateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"scene", "create", "--path", "res://scenes/Main.tscn", "--root", "Node2D"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path, --root, and --name") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunSceneOpen(t *testing.T) {
	requests := 0
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/scene/open":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"queued": true,
					"job_id": "open-1",
					"path":   "res://scenes/Main.tscn",
				},
			})
		case "/jobs/open-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "open-1",
					Kind:   "scene.open",
					Status: "succeeded",
					Result: map[string]any{"path": "res://scenes/Main.tscn", "root": "/root/Main"},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "scene", "open", "--path", "res://scenes/Main.tscn"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "scene.open" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scenes/Main.tscn" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Scene opened: res://scenes/Main.tscn") || !strings.Contains(stdout.String(), "Root: /root/Main") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestRunSceneOpenRequiresPathBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"scene", "open"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunSceneSavePathUnsupportedBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"scene", "save", "--path", "res://main.tscn"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected unsupported path error")
	}
	if !strings.Contains(err.Error(), "--path is temporarily unsupported") {
		t.Fatalf("err = %v", err)
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

func TestNodeRenameRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "rename", "--path", "/root/Main/Old"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --name") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeMoveRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "move", "--path", "/root/Main/Child"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --parent") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeGetRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "get", "--path", "/root/Main"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --property") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeSetRequiresTypedJSONBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "set", "--path", "/root/Main", "--property", "position", "--value", "not-json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "typed JSON") {
		t.Fatalf("err = %v", err)
	}
}

func TestScriptCheckRequiresPathBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"script", "check"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunScriptCheck(t *testing.T) {
	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/script/check" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":  "res://scripts/player.gd",
				"valid": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "script", "check", "--path", "res://scripts/player.gd")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "script.check" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scripts/player.gd" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Script OK: res://scripts/player.gd") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunScriptCreate(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/script/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://scripts/player.gd",
				"valid":   true,
				"created": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "script", "create", "--path", "res://scripts/player.gd", "--extends", "Node2D", "--force")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "script.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scripts/player.gd" || gotEnvelope.Params["extends"] != "Node2D" || gotEnvelope.Params["force"] != true {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Script created: res://scripts/player.gd") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestScriptCreateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"script", "create", "--path", "res://scripts/player.gd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --extends") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunScriptWriteBodyFile(t *testing.T) {
	bodyPath := filepath.Join(t.TempDir(), "player.gd")
	if err := os.WriteFile(bodyPath, []byte("extends Node2D\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/script/write" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://scripts/player.gd",
				"valid":   true,
				"written": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "script", "write", "--path", "res://scripts/player.gd", "--body-file", bodyPath)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "script.write" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scripts/player.gd" || gotEnvelope.Params["body"] != "extends Node2D\n" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Script written: res://scripts/player.gd") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestScriptWriteRequiresOneBodySourceBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"script", "write", "--path", "res://scripts/player.gd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunNodeRename(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/rename" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Renamed"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "rename", "--path", "/root/Main/Old", "--name", "Renamed", "--dry-run")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.rename" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Dry run ok: /root/Main/Renamed") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunNodeMove(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/move" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/NewParent/Child"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "move", "--path", "/root/Main/Child", "--parent", "/root/Main/NewParent", "--index", "1")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.move" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Moved node: /root/Main/NewParent/Child") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunNodeGet(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/get" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":     "/root/Main/Player",
				"property": "position",
				"value":    map[string]any{"kind": "Vector2", "value": []any{200, 400}},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "get", "--path", "/root/Main/Player", "--property", "position")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.get" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), `"kind": "Vector2"`) {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunNodeSet(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":     "/root/Main/Player",
				"property": "position",
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "set", "--path", "/root/Main/Player", "--property", "position", "--value", `{"kind":"Vector2","value":[200,400]}`)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.set" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Set position on /root/Main/Player") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestProjectTokenIsUsedForMutationRequests(t *testing.T) {
	project := newCLIProject(t)
	if err := os.WriteFile(filepath.Join(project, bridge.ProjectTokenFile), []byte("project-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Smoke"},
		})
	}))
	defer server.Close()

	args := append(serverArgs(server), "--project", project, "node", "add", "--parent", "/root/Main", "--type", "Node2D", "--name", "Smoke", "--dry-run")
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer project-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestExplicitTokenBeatsProjectToken(t *testing.T) {
	project := newCLIProject(t)
	if err := os.WriteFile(filepath.Join(project, bridge.ProjectTokenFile), []byte("project-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Smoke"},
		})
	}))
	defer server.Close()

	args := append(serverArgs(server), "--project", project, "--token", "explicit-token", "node", "add", "--parent", "/root/Main", "--type", "Node2D", "--name", "Smoke", "--dry-run")
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer explicit-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestBridgeAddonUpdateSendsFixtureAddon(t *testing.T) {
	useTestAddon(t)

	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/addon/update" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"updated":         true,
				"files_written":   4,
				"backup":          "res://addons/.godot_tcp_bridge_backup/now/",
				"reload_required": true,
			},
		})
	}))
	defer server.Close()

	args := append(serverArgs(server), "--token", "secret", "bridge", "addon-update")
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "addon.update" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	files, ok := gotEnvelope.Params["files"].([]any)
	if !ok {
		t.Fatalf("files param = %#v", gotEnvelope.Params["files"])
	}
	foundFixture := false
	for _, item := range files {
		file, ok := item.(map[string]any)
		if !ok || file["path"] != "bridge_plugin.gd" {
			continue
		}
		content, err := base64.StdEncoding.DecodeString(file["content_base64"].(string))
		if err != nil {
			t.Fatal(err)
		}
		foundFixture = strings.Contains(string(content), "TEST_FIXTURE")
	}
	if !foundFixture {
		t.Fatal("bridge addon-update did not send fixture bridge_plugin.gd")
	}
	if !strings.Contains(stdout.String(), "Addon updated over bridge: 4 files written") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestAddonUpdateWithoutProjectUsesBridge(t *testing.T) {
	useTestAddon(t)

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"updated":         true,
				"files_written":   4,
				"reload_required": true,
			},
		})
	}))
	defer server.Close()

	args := append(serverArgs(server), "--token", "secret", "addon", "update")
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/addon/update" {
		t.Fatalf("path = %s", gotPath)
	}
	if !strings.Contains(stdout.String(), "Addon updated over bridge") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestBridgeInfoProjectless(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:              true,
			Service:         "godot-bridge",
			Engine:          "Godot",
			EngineVersion:   "4.6",
			PluginVersion:   "0.1.0",
			ProjectName:     "demo",
			ProjectPath:     "C:/demo/",
			AuthEnabled:     true,
			Host:            "0.0.0.0",
			Port:            7777,
			ProtocolVersion: "gdctl.v1",
			SceneOpen:       true,
			Capabilities:    []string{"addon.update"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "bridge", "info"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Godot bridge", "project_path: C:/demo/", "capabilities: addon.update"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestAddonStatusWithoutProjectUsesRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.0",
			ProtocolVersion: "gdctl.v1",
			ProjectPath:     "C:/demo/",
			Capabilities:    []string{"addon.update"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "addon", "status", "--json"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var status map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status["mode"] != "runtime" || status["reachable"] != true {
		t.Fatalf("status = %#v", status)
	}
}

func TestAddonDoctorWithoutProjectUsesRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:            true,
			PluginVersion: "0.1.0",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "addon", "doctor"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "projectless runtime mode") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestBridgeLogs(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
			OK: true,
			Entries: []bridge.LogEntry{
				{Time: "2026-05-07T10:00:00", Level: "error", Source: "bridge.response", Message: "Request failed", Detail: map[string]any{"path": "/node/add"}},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "bridge", "logs")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if !strings.Contains(stdout.String(), "bridge.response: Request failed") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestBridgeLogsJSONAndClear(t *testing.T) {
	var gotClear bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/logs":
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
				OK:      true,
				Entries: []bridge.LogEntry{{Time: "now", Level: "info", Source: "test", Message: "hello"}},
			})
		case "/logs/clear":
			gotClear = true
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"cleared": true},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "bridge", "logs", "--json", "--clear")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !gotClear {
		t.Fatal("expected clear request")
	}
	if !strings.Contains(stdout.String(), `"message": "hello"`) || !strings.Contains(stdout.String(), "Logs cleared") {
		t.Fatalf("stdout:\n%s", stdout.String())
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
