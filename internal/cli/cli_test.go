package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/color"
	"image/png"
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
			PluginVersion: "0.1.3",
			ProjectName:   "my-game",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "ping"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Godot bridge: ok", "Engine: Godot 4.4.1", "Project: my-game", "Plugin: 0.1.3"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunHelpIncludesDottedAliasNote(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Dotted aliases are supported",
		"gdctl file.mkdir",
		"gdctl project.setting.get",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunHelpScriptWriteIncludesDiagnosticsNote(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "script.write"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"script write --path PATH",
		"syntax-check and write a GDScript file body",
		"Godot's diagnostic, line number, and nearby source context",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunHelpNodeAttachScriptIncludesSceneOption(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "node.attach-script"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"node attach-script --path PATH --script SCRIPT [--scene SCENE]",
		"--scene SCENE",
		"opens that scene, attaches the script, and saves it",
		"Invalid GDScript reports Godot's diagnostic",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunHelpRunStart(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "run.start"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"run start [--scene SCENE | --main]",
		"already-open Godot editor",
		"does not require GDCTL_GODOT_PATH",
	} {
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

func TestRunSceneInstance(t *testing.T) {
	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/instance" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":      "/root/Main/Child",
				"scene":     "res://scenes/Child.tscn",
				"parent":    "/root/Main",
				"name":      "Child",
				"instanced": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "scene", "instance", "--parent", "/root/Main", "--scene", "res://scenes/Child.tscn", "--name", "Child")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "scene.instance" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["parent"] != "/root/Main" || gotEnvelope.Params["scene"] != "res://scenes/Child.tscn" || gotEnvelope.Params["name"] != "Child" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Scene instanced: /root/Main/Child") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunSceneInstanceRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"scene", "instance", "--parent", "/root/Main", "--scene", "res://scenes/Child.tscn"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--parent, --scene, and --name") {
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

func TestNodeSetResourceRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "set-resource", "--path", "/root/Main"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path, --property, and --resource") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunNodeSetResource(t *testing.T) {
	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set-resource" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":     "/root/Main/Body",
				"property": "material",
				"resource": "res://materials/edge_mix.tres",
				"set":      true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "set-resource", "--path", "/root/Main/Body", "--property", "material", "--resource", "res://materials/edge_mix.tres")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "node.set_resource" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "/root/Main/Body" || gotEnvelope.Params["property"] != "material" || gotEnvelope.Params["resource"] != "res://materials/edge_mix.tres" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Set material on /root/Main/Body to res://materials/edge_mix.tres") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNodeAttachScriptRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "attach-script", "--path", "/root/Main"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --script") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunNodeAttachScript(t *testing.T) {
	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/attach-script" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":     "/root/Main",
				"script":   "res://scripts/player.gd",
				"attached": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "attach-script", "--path", "/root/Main", "--script", "res://scripts/player.gd")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "node.attach_script" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "/root/Main" || gotEnvelope.Params["script"] != "res://scripts/player.gd" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Attached script: res://scripts/player.gd -> /root/Main") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunNodeAttachScriptWithScene(t *testing.T) {
	var ops []string
	var attachEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			ops = append(ops, "open")
			var envelope bridge.RequestEnvelope
			if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Params["path"] != "res://scenes/Player.tscn" {
				t.Fatalf("open params = %#v", envelope.Params)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "open-1", "path": "res://scenes/Player.tscn"},
			})
		case "/jobs/open-1":
			ops = append(ops, "open-job")
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "open-1",
					Kind:   "scene.open",
					Status: "succeeded",
					Result: map[string]any{"path": "res://scenes/Player.tscn", "root": "/root/Player"},
				},
			})
		case "/node/attach-script":
			ops = append(ops, "attach")
			if err := json.NewDecoder(r.Body).Decode(&attachEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"path":     "/root/Player",
					"script":   "res://scripts/player.gd",
					"attached": true,
				},
			})
		case "/scene/save":
			ops = append(ops, "save")
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "save-1", "path": "res://scenes/Player.tscn"},
			})
		case "/jobs/save-1":
			ops = append(ops, "save-job")
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "save-1",
					Kind:   "scene.save",
					Status: "succeeded",
					Result: map[string]any{"path": "res://scenes/Player.tscn"},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "attach-script", "--scene", "res://scenes/Player.tscn", "--path", "/root/Player", "--script", "res://scripts/player.gd")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(ops, ","), "open,open-job,attach,save,save-job"; got != want {
		t.Fatalf("ops = %s", got)
	}
	if attachEnvelope.Op != "node.attach_script" {
		t.Fatalf("op = %q", attachEnvelope.Op)
	}
	if attachEnvelope.Params["path"] != "/root/Player" || attachEnvelope.Params["script"] != "res://scripts/player.gd" {
		t.Fatalf("attach params = %#v", attachEnvelope.Params)
	}
	for _, want := range []string{
		"Scene opened: res://scenes/Player.tscn",
		"Attached script: res://scripts/player.gd -> /root/Player",
		"Scene saved: res://scenes/Player.tscn",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
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

func TestShaderCheckRequiresPathBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"shader", "check"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunShaderCheck(t *testing.T) {
	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shader/check" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":  "res://shaders/edge_mix_3d.gdshader",
				"valid": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "shader", "check", "--path", "res://shaders/edge_mix_3d.gdshader")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "shader.check" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://shaders/edge_mix_3d.gdshader" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Shader OK: res://shaders/edge_mix_3d.gdshader") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunShaderWriteBodyFile(t *testing.T) {
	bodyPath := filepath.Join(t.TempDir(), "edge_mix_3d.gdshader")
	if err := os.WriteFile(bodyPath, []byte("shader_type spatial;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shader/write" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://shaders/edge_mix_3d.gdshader",
				"valid":   true,
				"written": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "shader", "write", "--path", "res://shaders/edge_mix_3d.gdshader", "--body-file", bodyPath)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "shader.write" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://shaders/edge_mix_3d.gdshader" || gotEnvelope.Params["body"] != "shader_type spatial;\n" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Shader written: res://shaders/edge_mix_3d.gdshader") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestShaderWriteRequiresBodyBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"shader", "write", "--path", "res://shaders/edge_mix_3d.gdshader"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "exactly one of --body or --body-file") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunResourceCreateShaderMaterial(t *testing.T) {
	var gotAuth string
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/resource/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://materials/edge_mix.tres",
				"type":    "ShaderMaterial",
				"created": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "resource", "create",
		"--path", "res://materials/edge_mix.tres",
		"--type", "ShaderMaterial",
		"--prop", `shader={"kind":"Resource","value":"res://shaders/edge_mix_3d.gdshader"}`,
		"--shader-param", "edge_lut=res://textures/edge_lut.png",
	)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "resource.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://materials/edge_mix.tres" || gotEnvelope.Params["type"] != "ShaderMaterial" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	shaderParams, ok := gotEnvelope.Params["shader_params"].(map[string]any)
	if !ok {
		t.Fatalf("shader_params = %#v", gotEnvelope.Params["shader_params"])
	}
	if shaderParams["edge_lut"] != "res://textures/edge_lut.png" {
		t.Fatalf("shader_params = %#v", shaderParams)
	}
	if !strings.Contains(stdout.String(), "Resource created: res://materials/edge_mix.tres (ShaderMaterial)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunFileWriteBytes(t *testing.T) {
	inPath := filepath.Join(t.TempDir(), "edge_lut.png")
	if err := os.WriteFile(inPath, []byte("png-data"), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/write-bytes" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://mini_3d/edge_lut.png",
				"bytes":   8,
				"written": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "file", "write-bytes", "--path", "res://mini_3d/edge_lut.png", "--in", inPath)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "file.write_bytes" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://mini_3d/edge_lut.png" || gotEnvelope.Params["content_base64"] != base64.StdEncoding.EncodeToString([]byte("png-data")) {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "File written: res://mini_3d/edge_lut.png (8 bytes)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunLUTWrite(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "edge_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`[{"id":2,"mode":"bleed","mix":0.5,"blur":0.25,"width":1.0}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/write-bytes" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://mini_3d/edge_lut.png",
				"bytes":   93,
				"written": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "file", "lut-write", "--path", "res://mini_3d/edge_lut.png", "--profiles", profilesPath)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "file.write_bytes" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	content, ok := gotEnvelope.Params["content_base64"].(string)
	if !ok || content == "" {
		t.Fatalf("content_base64 missing: %#v", gotEnvelope.Params)
	}
	pngData, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 1 {
		t.Fatalf("bounds = %v", img.Bounds())
	}
	pixel := color.NRGBAModel.Convert(img.At(2, 0)).(color.NRGBA)
	if pixel.R != 128 || pixel.G != 64 || pixel.B != 255 || pixel.A == 0 {
		t.Fatalf("pixel = %d %d %d %d", pixel.R, pixel.G, pixel.B, pixel.A)
	}
	if !strings.Contains(stdout.String(), "LUT written: res://mini_3d/edge_lut.png") || !strings.Contains(stdout.String(), "Profiles: 1") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestLUTWriteValidatesProfileIDBeforeNetwork(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "edge_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`[{"id":300}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"file", "lut-write", "--path", "res://mini_3d/edge_lut.png", "--profiles", profilesPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "between 0 and 255") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunViewportScreenshot(t *testing.T) {
	requests := 0
	var gotEnvelope bridge.RequestEnvelope
	outPath := filepath.Join(t.TempDir(), "status.png")
	pngData := []byte("fake-png")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/viewport/screenshot":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"queued": true,
					"job_id": "shot-1",
					"kind":   "2d",
				},
			})
		case "/jobs/shot-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "shot-1",
					Kind:   "viewport.screenshot",
					Status: "succeeded",
					Result: map[string]any{
						"format":         "png",
						"width":          640,
						"height":         360,
						"content_base64": base64.StdEncoding.EncodeToString(pngData),
					},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "viewport", "screenshot", "--out", outPath), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "viewport.screenshot" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["kind"] != "2d" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	gotData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotData) != string(pngData) {
		t.Fatalf("png data = %q", gotData)
	}
	if !strings.Contains(stdout.String(), "Screenshot written:") || !strings.Contains(stdout.String(), "640x360") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestViewportScreenshotRequiresOutBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"viewport", "screenshot"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--out") {
		t.Fatalf("err = %v", err)
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
			PluginVersion:   "0.1.3",
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
			PluginVersion:   "0.1.3",
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
			PluginVersion: "0.1.3",
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
			PluginVersion:   "0.1.3",
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

func TestAddonRollbackRestoresLatestBackup(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"addon", "install", "--project", project}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	originalPlugin := readFile(t, filepath.Join(project, addon.AddonDir, "plugin.cfg"))
	stdout.Reset()
	if err := Run(context.Background(), []string{"addon", "update", "--project", project}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Backup:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if err := os.WriteFile(filepath.Join(project, addon.AddonDir, "plugin.cfg"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"addon", "rollback", "--project", project}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "addon rolled back") || !strings.Contains(stdout.String(), "Restored:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if got := readFile(t, filepath.Join(project, addon.AddonDir, "plugin.cfg")); got != originalPlugin {
		t.Fatalf("plugin.cfg = %q, want %q", got, originalPlugin)
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

func TestRunNodeGroupAdd(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/group-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":  "/root/Main/Player",
				"group": "enemies",
				"added": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "group", "add", "--path", "/root/Main/Player", "--group", "enemies")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.group_add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "/root/Main/Player" || gotEnvelope.Params["group"] != "enemies" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Added to group: enemies on /root/Main/Player") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunNodeGroupRemove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/group-remove" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "/root/Main/Player",
				"group":   "enemies",
				"removed": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "group", "remove", "--path", "/root/Main/Player", "--group", "enemies")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Removed from group: enemies on /root/Main/Player") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunNodeGroupList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/group-list" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":   "/root/Main/Player",
				"groups": []any{"enemies", "pickups"},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "group", "list", "--path", "/root/Main/Player")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "enemies") || !strings.Contains(stdout.String(), "pickups") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNodeGroupAddRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "group", "add", "--path", "/root/Main/Player"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --group") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunSignalConnect(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/signal/connect" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"from":      "/root/Main/Timer",
				"signal":    "timeout",
				"to":        "/root/Main/Player",
				"method":    "_on_timer_timeout",
				"connected": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "signal", "connect",
		"--from", "/root/Main/Timer", "--signal", "timeout",
		"--to", "/root/Main/Player", "--method", "_on_timer_timeout")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "signal.connect" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["from"] != "/root/Main/Timer" || gotEnvelope.Params["signal"] != "timeout" ||
		gotEnvelope.Params["to"] != "/root/Main/Player" || gotEnvelope.Params["method"] != "_on_timer_timeout" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Connected: /root/Main/Timer::timeout -> /root/Main/Player::_on_timer_timeout") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunSignalDisconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/signal/disconnect" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"from":         "/root/Main/Timer",
				"signal":       "timeout",
				"to":           "/root/Main/Player",
				"method":       "_on_timer_timeout",
				"disconnected": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "signal", "disconnect",
		"--from", "/root/Main/Timer", "--signal", "timeout",
		"--to", "/root/Main/Player", "--method", "_on_timer_timeout")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Disconnected: /root/Main/Timer::timeout -> /root/Main/Player::_on_timer_timeout") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestSignalConnectRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"signal", "connect", "--from", "/root/Main/Timer", "--signal", "timeout", "--to", "/root/Main/Player"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--from, --signal, --to, and --method") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunProjectSettingGet(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project/setting-get" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"key":   "display/window/size/viewport_width",
				"value": map[string]any{"kind": "int", "value": float64(1920)},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "project", "setting", "get", "--key", "display/window/size/viewport_width")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "project.setting_get" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["key"] != "display/window/size/viewport_width" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "display/window/size/viewport_width") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunProjectSettingSet(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project/setting-set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"key":   "display/window/size/viewport_width",
				"value": map[string]any{"kind": "int", "value": float64(1280)},
				"set":   true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "project", "setting", "set",
		"--key", "display/window/size/viewport_width", "--value", `{"kind":"int","value":1280}`)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "project.setting_set" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["key"] != "display/window/size/viewport_width" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Set display/window/size/viewport_width") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestProjectSettingSetRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"project", "setting", "set", "--value", `{"kind":"int","value":1}`}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--key and --value") {
		t.Fatalf("err = %v", err)
	}
}

func TestProjectSettingSetRequiresValidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"project", "setting", "set", "--key", "some/key", "--value", "not-json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	if !strings.Contains(err.Error(), "typed JSON") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeDuplicateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "duplicate", "--path", "/root/Scene/Enemy"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --name") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunNodeDuplicate(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/duplicate" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"source_path": "/root/Scene/Enemy",
				"path":        "/root/Scene/Enemy2",
				"duplicated":  true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "duplicate",
		"--path", "/root/Scene/Enemy", "--name", "Enemy2")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.duplicate" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "/root/Scene/Enemy" || gotEnvelope.Params["name"] != "Enemy2" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Duplicated: /root/Scene/Enemy2 (source: /root/Scene/Enemy)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNodeListPropertiesRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "list-properties"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunNodeListProperties(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/list-properties" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path": "/root/Scene/Player",
				"properties": []any{
					map[string]any{"name": "position", "type": "Vector2", "usage": 8},
				},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "list-properties", "--path", "/root/Scene/Player")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.list_properties" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "position") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestFileListRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"file", "list"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunFileList(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/list" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":  "res://scenes",
				"files": []any{"res://scenes/player.tscn"},
				"dirs":  []any{},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "file", "list", "--path", "res://scenes")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "file.list" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "res://scenes") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestFileMkdirRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"file", "mkdir"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunFileMkdir(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/mkdir" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://scenes/level1", "created": true},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "file", "mkdir", "--path", "res://scenes/level1")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "file.mkdir" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Created: res://scenes/level1") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunDottedCommandAlias(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/mkdir" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://scenes/level1", "created": true},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "file.mkdir", "--path", "res://scenes/level1")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "file.mkdir" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Created: res://scenes/level1") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestFileDeleteRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"file", "delete"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunFileDelete(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/delete" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://old.tscn", "deleted": true},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "file", "delete", "--path", "res://old.tscn")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "file.delete" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Deleted: res://old.tscn") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestFileExistsRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"file", "exists"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunFileExists(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/exists" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://scenes/player.tscn",
				"exists":  true,
				"is_file": true,
				"is_dir":  false,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "file", "exists", "--path", "res://scenes/player.tscn")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "file.exists" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "res://scenes/player.tscn") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNavigationBakeRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"navigation", "bake"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunNavigationBake(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/navigation/bake" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":  "/root/Level/NavRegion",
				"kind":  "NavigationRegion3D",
				"baked": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "navigation", "bake", "--path", "/root/Level/NavRegion")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "navigation.bake" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "/root/Level/NavRegion" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Baked: /root/Level/NavRegion (NavigationRegion3D)") {
		t.Fatalf("stdout:\n%s", stdout.String())
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

func TestRunResourceCreate(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/resource/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://materials/ground.tres",
				"type":    "StandardMaterial3D",
				"created": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "resource", "create",
		"--path", "res://materials/ground.tres",
		"--type", "StandardMaterial3D",
	)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "resource.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://materials/ground.tres" || gotEnvelope.Params["type"] != "StandardMaterial3D" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Resource created: res://materials/ground.tres (StandardMaterial3D)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunResourceCreateWithProp(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://materials/red.tres",
				"type":    "StandardMaterial3D",
				"created": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "resource", "create",
		"--path", "res://materials/red.tres",
		"--type", "StandardMaterial3D",
		"--prop", `albedo_color={"kind":"Color","value":[1,0,0,1]}`,
	)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	props, ok := gotEnvelope.Params["props"].(map[string]any)
	if !ok {
		t.Fatalf("props = %#v", gotEnvelope.Params["props"])
	}
	if _, ok := props["albedo_color"]; !ok {
		t.Fatalf("props missing albedo_color: %#v", props)
	}
}

func TestResourceCreateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"resource", "create", "--path", "res://materials/ground.tres"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --type") {
		t.Fatalf("err = %v", err)
	}
}

func TestResourceCreateRequiresPathAndType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"resource", "create", "--type", "StandardMaterial3D"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --type") {
		t.Fatalf("err = %v", err)
	}
}

// Feature 10: import set

func TestRunImportSet(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/import/set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://textures/player.png",
				"params":  float64(2),
				"applied": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "import", "set",
		"--path", "res://textures/player.png",
		"--param", "compress/mode=0",
		"--param", "filter/mode=1",
	)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "import.set" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://textures/player.png" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	params, ok := gotEnvelope.Params["params"].(map[string]any)
	if !ok {
		t.Fatalf("params.params = %#v", gotEnvelope.Params["params"])
	}
	if params["compress/mode"] != float64(0) || params["filter/mode"] != float64(1) {
		t.Fatalf("params.params = %#v", params)
	}
	if !strings.Contains(stdout.String(), "Import set: res://textures/player.png (2 params)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestImportSetRequiresPathBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"import", "set", "--param", "compress/mode=0"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestImportSetParamRequiresValidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"import", "set", "--path", "res://textures/player.png", "--param", "compress/mode=not-json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Fatalf("err = %v", err)
	}
}

// Feature 12: scene list / resource list

func TestRunSceneList(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/list" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"dir":    "res://",
				"scenes": []any{"res://scenes/main.tscn", "res://scenes/level1.tscn"},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "scene", "list", "--dir", "res://")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "scene.list" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["dir"] != "res://" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "main.tscn") || !strings.Contains(stdout.String(), "level1.tscn") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunResourceList(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/resource/list" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"dir":       "res://",
				"resources": []any{"res://materials/ground.tres", "res://materials/wall.tres"},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "resource", "list", "--dir", "res://", "--ext", ".tres")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "resource.list" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["dir"] != "res://" || gotEnvelope.Params["ext"] != ".tres" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "ground.tres") || !strings.Contains(stdout.String(), "wall.tres") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunStartCommand(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/start" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"running":       true,
				"scene":         "res://main.tscn",
				"playing_scene": "main",
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "run", "start", "--scene", "res://main.tscn")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "run.start" || gotEnvelope.Params["scene"] != "res://main.tscn" {
		t.Fatalf("envelope = %#v", gotEnvelope)
	}
	if !strings.Contains(stdout.String(), "Run started: res://main.tscn") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunStartRejectsSceneAndMain(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"run", "start", "--scene", "res://main.tscn", "--main"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--scene or --main") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunStatusCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"running": true, "playing_scene": "main"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "status"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Run status: running (main)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunLogsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/logs" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
			OK:      true,
			Entries: []bridge.LogEntry{{Time: "now", Level: "error", Source: "runtime.error", Message: "boom"}},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "logs"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "runtime.error: boom") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

// Feature 13: project run / scene run

func TestProjectRunRequiresGodotBeforeExec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"--project", t.TempDir(), "project", "run"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when godot not configured")
	}
	if !strings.Contains(err.Error(), "headless Godot") {
		t.Fatalf("err = %v", err)
	}
}

func TestProjectRunRequiresProjectBeforeExec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"--godot", "/usr/bin/godot4", "project", "run"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when project not configured")
	}
	if !strings.Contains(err.Error(), "--project") {
		t.Fatalf("err = %v", err)
	}
}

func TestSceneRunRequiresPathBeforeExec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"--godot", "/usr/bin/godot4", "--project", t.TempDir(), "scene", "run"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestSceneRunRequiresGodotBeforeExec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"--project", t.TempDir(), "scene", "run", "--path", "res://main.tscn"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when godot not configured")
	}
	if !strings.Contains(err.Error(), "headless Godot") {
		t.Fatalf("err = %v", err)
	}
}
