package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
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
			PluginVersion: "0.1.9",
			ProjectName:   "my-game",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "ping"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Godot bridge: ok", "Engine: Godot 4.4.1", "Project: my-game", "Plugin: 0.1.9"} {
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

func TestRunSceneApply(t *testing.T) {
	requests := 0
	var applyEnvelope bridge.RequestEnvelope
	treePath := filepath.Join(t.TempDir(), "tree.json")
	treeJSON := `{"root":{"path":"/root/Main","children":[{"name":"Platform","type":"StaticBody3D","properties":{"position":{"kind":"Vector3","value":[1,2,3]}}}]}}`
	if err := os.WriteFile(treePath, []byte(treeJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/scene/open":
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
		case "/scene/apply":
			if err := json.NewDecoder(r.Body).Decode(&applyEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"root":    "/root/Main",
					"created": 1,
					"updated": 1,
				},
			})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"queued": true,
					"job_id": "save-1",
					"path":   "res://scenes/Main.tscn",
				},
			})
		case "/jobs/save-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "save-1",
					Kind:   "scene.save",
					Status: "succeeded",
					Result: map[string]any{"path": "res://scenes/Main.tscn"},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "scene", "apply", "--path", "res://scenes/Main.tscn", "--file", treePath), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if applyEnvelope.Op != "scene.apply" {
		t.Fatalf("op = %q", applyEnvelope.Op)
	}
	if applyEnvelope.Params["dry_run"] != false {
		t.Fatalf("params = %#v", applyEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Scene applied: res://scenes/Main.tscn") || !strings.Contains(stdout.String(), "Created: 1") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if requests != 5 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestRunSceneApplyDryRunDoesNotSave(t *testing.T) {
	requests := 0
	treePath := filepath.Join(t.TempDir(), "tree.json")
	if err := os.WriteFile(treePath, []byte(`{"children":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "open-1", "path": "res://scenes/Main.tscn"},
			})
		case "/jobs/open-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{ID: "open-1", Kind: "scene.open", Status: "succeeded", Result: map[string]any{"path": "res://scenes/Main.tscn"}}})
		case "/scene/apply":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"root": "/root/Main", "created": 0, "updated": 0, "dry_run": true},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "scene", "apply", "--path", "res://scenes/Main.tscn", "--file", treePath, "--dry-run"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Dry run ok") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if requests != 3 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestRunSceneBatchOpensAndSavesOnce(t *testing.T) {
	requests := map[string]int{}
	batchPath := filepath.Join(t.TempDir(), "ops.json")
	batchJSON := `{"operations":[{"op":"node.add","parent":"/root/Main","type":"Node3D","name":"Anchor"},{"op":"node.set","path":"/root/Main/Anchor","property":"position","value":{"kind":"Vector3","value":[1,2,3]}}]}`
	if err := os.WriteFile(batchPath, []byte(batchJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "open-batch"},
			})
		case "/jobs/open-batch":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{ID: "open-batch", Kind: "scene.open", Status: "succeeded", Result: map[string]any{"path": "res://main.tscn"}},
			})
		case "/node/add":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"path": "/root/Main/Anchor"},
			})
		case "/node/set":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"path": "/root/Main/Anchor", "property": "position"},
			})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "save-batch"},
			})
		case "/jobs/save-batch":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{ID: "save-batch", Kind: "scene.save", Status: "succeeded", Result: map[string]any{"path": "res://main.tscn"}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "scene", "batch", "--path", "res://main.tscn", "--file", batchPath)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if requests["/scene/open"] != 1 || requests["/scene/save"] != 1 {
		t.Fatalf("requests = %#v", requests)
	}
	if !strings.Contains(stdout.String(), "Scene batch saved: res://main.tscn (2 operations)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunSceneBatchSupportsNodeSetMany(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	batchPath := filepath.Join(t.TempDir(), "ops.json")
	batchJSON := `{"operations":[{"op":"node.set-many","path":"/root/Main/HUD","properties":{"text":{"kind":"String","value":"Market"},"position":{"kind":"Vector2","value":[10,20]}}}]}`
	if err := os.WriteFile(batchPath, []byte(batchJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "open-batch"},
			})
		case "/jobs/open-batch":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{ID: "open-batch", Kind: "scene.open", Status: "succeeded", Result: map[string]any{"path": "res://main.tscn"}}})
		case "/node/set-many":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"path": "/root/Main/HUD", "updated": 2},
			})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "save-batch"},
			})
		case "/jobs/save-batch":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{ID: "save-batch", Kind: "scene.save", Status: "succeeded", Result: map[string]any{"path": "res://main.tscn"}}})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "scene", "batch", "--path", "res://main.tscn", "--file", batchPath)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.set_many" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	props := gotEnvelope.Params["properties"].(map[string]any)
	if len(props) != 2 {
		t.Fatalf("properties = %#v", props)
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

func TestNodeSetManyHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	propsPath := filepath.Join(t.TempDir(), "props.json")
	propsJSON := `{"properties":{"text":{"kind":"String","value":"Hollow Market"},"position":{"kind":"Vector2","value":[10,20]}}}`
	if err := os.WriteFile(propsPath, []byte(propsJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set-many" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/HUD", "updated": 2},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "set-many", "--path", "/root/Main/HUD", "--file", propsPath)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.set_many" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "/root/Main/HUD" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Set 2 properties") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNodeSetManyRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "set-many", "--path", "/root/Main"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path and --file") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeSetManyRejectsMalformedJSONBeforeNetwork(t *testing.T) {
	propsPath := filepath.Join(t.TempDir(), "props.json")
	if err := os.WriteFile(propsPath, []byte(`{"properties":`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "set-many", "--path", "/root/Main", "--file", propsPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "must be JSON") {
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

func TestRunScreenshot(t *testing.T) {
	requests := 0
	var gotEnvelope bridge.RequestEnvelope
	outPath := filepath.Join(t.TempDir(), "run.png")
	pngData := []byte("fake-run-png")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
		case "/run/screenshot":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"queued": true,
					"job_id": "run-shot-1",
					"source": "game",
					"screen": 1,
				},
			})
		case "/jobs/run-shot-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "run-shot-1",
					Kind:   "run.screenshot",
					Status: "succeeded",
					Result: map[string]any{
						"format":         "png",
						"source":         "game",
						"screen":         1,
						"width":          1280,
						"height":         720,
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
	err := Run(context.Background(), append(serverArgs(server), "run", "screenshot", "--out", outPath), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "run.screenshot" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["source"] != "game" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	gotData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotData) != string(pngData) {
		t.Fatalf("png data = %q", gotData)
	}
	if !strings.Contains(stdout.String(), "Run screenshot written:") || !strings.Contains(stdout.String(), "1280x720") || !strings.Contains(stdout.String(), "game viewport") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if requests != 3 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestRunScreenshotScreenSource(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	outPath := filepath.Join(t.TempDir(), "screen.png")
	pngData := []byte("fake-screen-png")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/screenshot":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true,
				Result: map[string]any{
					"queued": true,
					"job_id": "run-shot-screen",
					"source": "screen",
					"screen": 1,
				},
			})
		case "/jobs/run-shot-screen":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "run-shot-screen",
					Kind:   "run.screenshot",
					Status: "succeeded",
					Result: map[string]any{
						"format":         "png",
						"source":         "screen",
						"screen":         1,
						"width":          1920,
						"height":         1080,
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
	err := Run(context.Background(), append(serverArgs(server), "run", "screenshot", "--source", "screen", "--screen", "1", "--out", outPath), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Params["source"] != "screen" || gotEnvelope.Params["screen"] != float64(1) {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "host screen") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunScreenshotRejectsInvalidSourceBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"run", "screenshot", "--source", "window"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--source") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunScreenshotTimeoutIncludesDebuggerDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
		case "/run/screenshot":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "run-shot-failed"},
			})
		case "/jobs/run-shot-failed":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "run-shot-failed",
					Kind:   "run.screenshot",
					Status: "failed",
					Error: &bridge.BridgeError{
						Code:    "RUN_SCREENSHOT_HELPER_TIMEOUT",
						Message: "Runtime helper did not return a game viewport screenshot.",
						Detail: map[string]any{
							"debugger": map[string]any{
								"paused":  true,
								"file":    "res://main.gd",
								"line":    9,
								"message": "boom",
							},
						},
					},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "run", "screenshot", "--out", filepath.Join(t.TempDir(), "run.png")), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "debugger paused at res://main.gd:9: boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInputCommand(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{"steps":[{"type":"key","key":"W","action":"tap"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
		case "/run/input":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "input-1", "steps": 1},
			})
		case "/jobs/input-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "input-1",
					Kind:   "run.input",
					Status: "succeeded",
					Result: map[string]any{"steps": 1, "duration_ms": 75},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "run", "input", "--file", inputPath), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "run.input" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	steps, ok := gotEnvelope.Params["steps"].([]any)
	if !ok || len(steps) != 1 {
		t.Fatalf("steps = %#v", gotEnvelope.Params["steps"])
	}
	if !strings.Contains(stdout.String(), "Run input completed: 1 steps") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunInputRequiresFileBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"run", "input"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--file") {
		t.Fatalf("err = %v", err)
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

func TestRunNodeSetVector3Shorthand(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Player", "property": "position"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "node", "set", "--path", "/root/Main/Player", "--property", "position", "--vector3", "1,2.5,3")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	value, ok := gotEnvelope.Params["value"].(map[string]any)
	if !ok || value["kind"] != "Vector3" {
		t.Fatalf("value = %#v", gotEnvelope.Params["value"])
	}
}

func TestRunNodeSetArrayVector3Shorthand(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Drone", "property": "route"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "node", "set", "--path", "/root/Main/Drone", "--property", "route", "--array-vector3", "0,1,2;3,4,5")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	value, ok := gotEnvelope.Params["value"].(map[string]any)
	if !ok || value["kind"] != "Array[Vector3]" {
		t.Fatalf("value = %#v", gotEnvelope.Params["value"])
	}
	items, ok := value["value"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items = %#v", value["value"])
	}
}

func TestRunNodeSetArrayVector3RejectsInvalidBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "set", "--path", "/root/Main/Drone", "--property", "route", "--array-vector3", "0,1;3,4,5"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--array-vector3") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunNodeSetArrayBoolUsesSemicolonSeparator(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Switches", "property": "active_flags"},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "node", "set", "--path", "/root/Main/Switches", "--property", "active_flags", "--array-bool", "true;false")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	value, ok := gotEnvelope.Params["value"].(map[string]any)
	if !ok || value["kind"] != "Array[bool]" {
		t.Fatalf("value = %#v", gotEnvelope.Params["value"])
	}
	items, ok := value["value"].([]any)
	if !ok || len(items) != 2 || items[0] != true || items[1] != false {
		t.Fatalf("items = %#v", value["value"])
	}
}

func TestRunNodeAddWithProp(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Marker", "properties": 1},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "node", "add", "--parent", "/root/Main", "--type", "Node2D", "--name", "Marker", "--prop", `position={"kind":"Vector2","value":[1,2]}`)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	props, ok := gotEnvelope.Params["props"].(map[string]any)
	if !ok {
		t.Fatalf("props = %#v", gotEnvelope.Params["props"])
	}
	if _, ok := props["position"]; !ok {
		t.Fatalf("props = %#v", props)
	}
	if !strings.Contains(stdout.String(), "Added node: /root/Main/Marker") {
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
		if r.URL.Path == "/ping" {
			_ = json.NewEncoder(w).Encode(bridge.PingResponse{OK: true, Service: "godot_tcp_bridge", ProjectPath: t.TempDir()})
			return
		}
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
			PluginVersion:   "0.1.9",
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
			PluginVersion:   "0.1.9",
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
			PluginVersion: "0.1.9",
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
			PluginVersion:   "0.1.9",
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

func TestRunProjectSettingSetIntShorthand(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project/setting-set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"key": "display/window/size/viewport_width", "set": true},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "project", "setting", "set", "--key", "display/window/size/viewport_width", "--int", "1280")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	value, ok := gotEnvelope.Params["value"].(map[string]any)
	if !ok || value["kind"] != "int" || value["value"] != float64(1280) {
		t.Fatalf("value = %#v", gotEnvelope.Params["value"])
	}
}

func TestProjectSettingSetRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"project", "setting", "set", "--value", `{"kind":"int","value":1}`}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--key") {
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

func TestRunResourceCreateWithScript(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://data/room_a.tres",
				"type":    "Resource",
				"script":  "res://scripts/room_data.gd",
				"created": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "resource", "create",
		"--path", "res://data/room_a.tres",
		"--script", "res://scripts/room_data.gd",
	)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Params["script"] != "res://scripts/room_data.gd" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "via res://scripts/room_data.gd") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestResourceCreateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"resource", "create", "--path", "res://materials/ground.tres"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--type or --script") {
		t.Fatalf("err = %v", err)
	}
}

func TestResourceCreateRequiresPathAndType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"resource", "create", "--type", "StandardMaterial3D"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunAutoloadAdd(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/autoload/add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"name":  "GameState",
				"path":  "res://scripts/game_state.gd",
				"added": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "autoload", "add",
		"--name", "GameState",
		"--path", "res://scripts/game_state.gd",
	)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "autoload.add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["name"] != "GameState" || gotEnvelope.Params["path"] != "res://scripts/game_state.gd" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Autoload added: GameState -> res://scripts/game_state.gd") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunAutoloadList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/autoload/list" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"autoloads": []map[string]any{
					{"name": "GameState", "path": "res://scripts/game_state.gd"},
				},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "autoload", "list")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "GameState -> res://scripts/game_state.gd") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunInputActionAdd(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/input/action-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"action":   "grav_down",
				"deadzone": 0.5,
				"added":    true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "input", "action", "add", "--name", "grav_down")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "input.action_add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["action"] != "grav_down" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Input action added: grav_down") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunInputEventAddKey(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/input/event-add-key" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"action":      "grav_down",
				"key":         "W",
				"event_added": true,
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "input", "event", "add-key", "--action", "grav_down", "--key", "W")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "input.event_add_key" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["action"] != "grav_down" || gotEnvelope.Params["key"] != "W" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Input key added: grav_down -> W") {
		t.Fatalf("stdout:\n%s", stdout.String())
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
			OK: true,
			Result: map[string]any{"running": true, "playing_scene": "main", "runtime_helper": map[string]any{
				"present":             true,
				"autoload_configured": true,
				"path":                "res://addons/godot_tcp_bridge/runtime/runtime_bridge.gd",
				"last_seen":           "2026-05-14T10:30:00Z",
				"last_message":        "heartbeat",
			}},
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
	if !strings.Contains(stdout.String(), "Runtime helper: present") {
		t.Fatalf("stdout missing helper status:\n%s", stdout.String())
	}
}

func TestRunStatusCommandJSONIncludesRuntimeHelper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"running":       true,
				"playing_scene": "main",
				"runtime_helper": map[string]any{
					"present": true,
					"path":    "res://addons/godot_tcp_bridge/runtime/runtime_bridge.gd",
				},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "status", "--json"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var got bridge.RunStatusResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json output invalid: %v\n%s", err, stdout.String())
	}
	if !got.RuntimeHelper.Present {
		t.Fatalf("runtime helper not decoded from json: %#v", got.RuntimeHelper)
	}
}

func TestRunHelperStatusCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{"running": true, "runtime_helper": map[string]any{
				"present":             false,
				"autoload_configured": true,
				"path":                "res://addons/godot_tcp_bridge/runtime/runtime_bridge.gd",
				"error":               "runtime helper has not checked in",
			}},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "helper-status"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Runtime helper: not present", "Autoload: configured", "Issue: runtime helper has not checked in"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunStatusCommandPausedDebugger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"running":       true,
				"playing_scene": "main",
				"debugger": map[string]any{
					"paused":  true,
					"file":    "res://main.gd",
					"line":    12,
					"message": "typed array mismatch",
				},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "status"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Run status: paused (main) res://main.gd:12 typed array mismatch") {
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

func TestRunLogsCommandClear(t *testing.T) {
	var cleared bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/logs":
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
				OK:      true,
				Entries: []bridge.LogEntry{{Time: "now", Level: "info", Source: "runtime.skyline", Message: "ready"}},
			})
		case "/run/logs/clear":
			cleared = true
			var env bridge.RequestEnvelope
			if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
				t.Fatal(err)
			}
			if env.Op != "run.logs.clear" {
				t.Fatalf("envelope = %#v", env)
			}
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
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "logs", "--clear"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "runtime.skyline: ready") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if !cleared {
		t.Fatal("expected run logs to be cleared")
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

// ---------------------------------------------------------------------------
// Phase 1: run logs filters (#3)
// ---------------------------------------------------------------------------

func TestRunLogsSourceFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
			OK: true,
			Entries: []bridge.LogEntry{
				{Time: "t1", Level: "info", Source: "runtime.game", Message: "keep"},
				{Time: "t2", Level: "info", Source: "runtime.other", Message: "drop"},
			},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "logs", "--source", "runtime.game"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "keep") {
		t.Fatalf("expected 'keep' in output:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "drop") {
		t.Fatalf("expected 'drop' to be filtered out:\n%s", stdout.String())
	}
}

func TestRunLogsLatestFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
			OK: true,
			Entries: []bridge.LogEntry{
				{Time: "t1", Level: "info", Source: "runtime.game", Message: "first"},
				{Time: "t2", Level: "info", Source: "runtime.game", Message: "latest"},
			},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "logs", "--latest"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "latest") {
		t.Fatalf("expected 'latest' in output:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "first") {
		t.Fatalf("expected 'first' to be filtered out by --latest:\n%s", stdout.String())
	}
}

func TestRunLogsSinceStartFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
			OK: true,
			Entries: []bridge.LogEntry{
				{Time: "2024-01-01T00:00:00Z", Level: "info", Source: "run.start", Message: "started"},
				{Time: "2024-01-01T00:00:01Z", Level: "info", Source: "runtime.game", Message: "after"},
				{Time: "2023-12-31T23:59:59Z", Level: "info", Source: "runtime.game", Message: "before"},
			},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "logs", "--since-start"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "after") {
		t.Fatalf("expected 'after' in output:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "before") {
		t.Fatalf("expected pre-start entry 'before' filtered out:\n%s", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// Phase 1: run input --summary-probe (#4)
// ---------------------------------------------------------------------------

func TestRunInputSummaryProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
		case "/run/input":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "inp-1"},
			})
		case "/jobs/inp-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{ID: "inp-1", Kind: "run.input", Status: "succeeded",
					Result: map[string]any{"steps": 1, "duration_ms": 50}},
			})
		case "/run/logs":
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
				OK: true,
				Entries: []bridge.LogEntry{
					{Time: "t1", Level: "info", Source: "runtime.probe", Message: "state", Detail: map[string]any{"hp": float64(100)}},
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	f := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(f, []byte(`{"steps":[{"type":"key","key":"ui_accept"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "run", "input", "--file", f, "--summary-probe", "runtime.probe")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Probe [runtime.probe]:") {
		t.Fatalf("expected probe summary in stdout:\n%s", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// Phase 1: run wait-probe (#2)
// ---------------------------------------------------------------------------

func TestRunWaitProbeHappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/run/status" {
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
			return
		}
		if r.URL.Path != "/run/logs" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
			OK: true,
			Entries: []bridge.LogEntry{
				{Time: "t1", Level: "info", Source: "runtime.game", Message: "state",
					Detail: map[string]any{"score": float64(10)}},
			},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "run", "wait-probe",
		"--source", "runtime.game", "--assert", "score>=5", "--timeout", "5s")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "runtime.game") {
		t.Fatalf("expected source in output:\n%s", stdout.String())
	}
}

func TestRunWaitProbeRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"run", "wait-probe", "--source", "runtime.game"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --assert is missing")
	}
	if !strings.Contains(err.Error(), "--assert") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunWaitProbeRequiresSourceBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"run", "wait-probe", "--assert", "score>=5"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --source is missing")
	}
	if !strings.Contains(err.Error(), "--source") {
		t.Fatalf("err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Phase 1: node set transform shorthands (#7)
// ---------------------------------------------------------------------------

func TestNodeSetPositionShorthand(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Player", "property": "position"},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "set",
		"--path", "/root/Player", "--position", "1,2,3")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Params["property"] != "position" {
		t.Fatalf("property = %v", gotEnvelope.Params["property"])
	}
}

func TestNodeSetPositionConflictsWithProperty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "set",
		"--path", "/root/Player", "--position", "1,2,3", "--property", "scale"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when both --position and --property are given")
	}
}

// ---------------------------------------------------------------------------
// Phase 1: --scene on node commands (#8)
// ---------------------------------------------------------------------------

func sceneJobServer(t *testing.T, mutation string, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "open-s"},
			})
		case "/jobs/open-s":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{ID: "open-s", Kind: "scene.open", Status: "succeeded", Result: map[string]any{"opened": true, "path": "res://main.tscn"}},
			})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "save-s"},
			})
		case "/jobs/save-s":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{ID: "save-s", Kind: "scene.save", Status: "succeeded", Result: map[string]any{"saved": true, "path": "res://main.tscn"}},
			})
		case mutation:
			handler(w, r)
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
}

func TestNodeAddWithScene(t *testing.T) {
	server := sceneJobServer(t, "/node/add", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Player/Temp", "created": true},
		})
	})
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "add",
		"--parent", "/root/Player", "--type", "Node3D", "--name", "Temp",
		"--scene", "res://main.tscn")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Added node:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNodeRemoveWithScene(t *testing.T) {
	server := sceneJobServer(t, "/node/remove", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Player/Temp", "removed": true},
		})
	})
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "node", "remove",
		"--path", "/root/Player/Temp", "--scene", "res://main.tscn")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Removed node:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// Phase 1: screenshot sanity check (#11)
// ---------------------------------------------------------------------------

func TestRunScreenshotSanityCheckWarns(t *testing.T) {
	// Build a solid-colour 100x100 PNG.
	img := makeUniformPNG(100, 100, color.RGBA{R: 30, G: 30, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	pngData := buf.Bytes()
	outPath := filepath.Join(t.TempDir(), "out.png")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
		case "/run/screenshot":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "shot-sanity"},
			})
		case "/jobs/shot-sanity":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{ID: "shot-sanity", Kind: "run.screenshot", Status: "succeeded",
					Result: map[string]any{
						"format":         "png",
						"source":         "game",
						"width":          100,
						"height":         100,
						"content_base64": base64.StdEncoding.EncodeToString(pngData),
					}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "screenshot", "--out", outPath), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "screenshot may be the desktop") {
		t.Fatalf("expected sanity warning on stderr:\n%s", stderr.String())
	}
}

func makeUniformPNG(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// ---------------------------------------------------------------------------
// Phase 3: script write --allow-missing-preloads (#5)
// ---------------------------------------------------------------------------

func TestScriptWriteAllowMissingPreloads(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/script/write" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://scripts/player.gd", "valid": true, "written": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "script", "write",
		"--path", "res://scripts/player.gd", "--body", "extends Node\n", "--allow-missing-preloads")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Params["allow_missing_preloads"] != true {
		t.Fatalf("expected allow_missing_preloads=true in params: %#v", gotEnvelope.Params)
	}
}

// ---------------------------------------------------------------------------
// Phase 4: run probe raycast (#10)
// ---------------------------------------------------------------------------

func TestRunProbeRaycastHit(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/probe/raycast":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "ray-1"},
			})
		case "/jobs/ray-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{ID: "ray-1", Kind: "run.probe.raycast", Status: "succeeded",
					Result: map[string]any{
						"hit":          true,
						"camera_path":  "/root/Camera3D",
						"hit_collider": "/root/Wall",
						"hit_distance": 5.0,
					}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "probe", "raycast"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "run.probe.raycast" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Raycast hit:") || !strings.Contains(stdout.String(), "/root/Wall") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunProbeNode(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/probe/node":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "node-1"},
			})
		case "/jobs/node-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{ID: "node-1", Kind: "run.probe.node", Status: "succeeded",
					Result: map[string]any{
						"path": "/root/Main/Player",
						"type": "CharacterBody3D",
						"properties": map[string]any{
							"global_position": []any{1.0, 2.0, 3.0},
						},
					}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "run", "probe", "node",
		"--path", "/root/Main/Player", "--property", "global_position")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "run.probe.node" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "/root/Main/Player" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Node probe: /root/Main/Player (CharacterBody3D)") ||
		!strings.Contains(stdout.String(), "global_position") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestRunProbeRaycastNoHit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/probe/raycast":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "ray-2"},
			})
		case "/jobs/ray-2":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{ID: "ray-2", Kind: "run.probe.raycast", Status: "succeeded",
					Result: map[string]any{"hit": false, "camera_path": "/root/Camera3D"}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "run", "probe", "raycast"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "no hit") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// Phase 4: scene apply-blueprint (#6)
// ---------------------------------------------------------------------------

func TestSceneApplyBlueprintHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "open-bp"},
			})
		case "/jobs/open-bp":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{ID: "open-bp", Kind: "scene.open", Status: "succeeded", Result: map[string]any{"opened": true, "path": "res://main.tscn"}},
			})
		case "/scene/apply/blueprint":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"path": "res://main.tscn", "blueprint": "player3d", "created": 3},
			})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "save-bp"},
			})
		case "/jobs/save-bp":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{ID: "save-bp", Kind: "scene.save", Status: "succeeded", Result: map[string]any{"saved": true, "path": "res://main.tscn"}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "scene", "apply-blueprint",
		"--path", "res://main.tscn", "--blueprint", "player3d")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "scene.apply.blueprint" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["blueprint"] != "player3d" {
		t.Fatalf("blueprint param = %v", gotEnvelope.Params["blueprint"])
	}
	if !strings.Contains(stdout.String(), "Blueprint applied:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestSceneApplyBlueprintRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"scene", "apply-blueprint", "--path", "res://main.tscn"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --blueprint is missing")
	}
	if !strings.Contains(err.Error(), "--blueprint") {
		t.Fatalf("err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Phase 4: theme commands (#14)
// ---------------------------------------------------------------------------

func TestThemeCreateHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/theme/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://ui/main.tres", "created": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "theme", "create", "--path", "res://ui/main.tres")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "theme.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Theme created:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestThemeCreateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"theme", "create"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --path is missing")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestThemeSetColorHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/theme/set-color" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://ui/main.tres", "set": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "theme", "set-color",
		"--path", "res://ui/main.tres", "--node-type", "Label", "--name", "font_color", "--value", "1,0,0,1")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "theme.set-color" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["node_type"] != "Label" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
}

func TestThemeSetColorRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"theme", "set-color", "--path", "res://ui/main.tres"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when required flags are missing")
	}
}

func TestThemeSetFontSizeHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/theme/set-font-size" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://ui/main.tres", "set": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "theme", "set-font-size",
		"--path", "res://ui/main.tres", "--node-type", "Label", "--name", "font_size", "--value", "18")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "theme.set-font-size" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

func TestThemeSetConstantHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/theme/set-constant" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://ui/main.tres", "set": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "theme", "set-constant",
		"--path", "res://ui/main.tres", "--node-type", "MarginContainer", "--name", "margin_top", "--value", "8")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "theme.set-constant" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

// ---------------------------------------------------------------------------
// Phase 4: animation commands (#15)
// ---------------------------------------------------------------------------

func TestAnimationCreateHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/animation/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://anim/player.tres", "name": "walk", "created": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "animation", "create",
		"--path", "res://anim/player.tres", "--name", "walk", "--length", "1.0")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "animation.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["name"] != "walk" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Animation created:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestAnimationCreateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"animation", "create", "--path", "res://anim/player.tres"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --name is missing")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnimationTrackAddHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/animation/track-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://anim/player.tres", "animation": "walk", "track_idx": 0},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "animation", "track-add",
		"--path", "res://anim/player.tres", "--animation", "walk",
		"--node-path", "Player", "--property", "position")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "animation.track-add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

func TestAnimationKeyframeAddHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/animation/keyframe-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://anim/player.tres", "animation": "walk", "track_idx": 0, "time": 0.5, "added": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "animation", "keyframe-add",
		"--path", "res://anim/player.tres", "--animation", "walk",
		"--track-idx", "0", "--time", "0.5", "--value", `{"kind":"Vector3","value":[0,1,0]}`)
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "animation.keyframe-add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

func TestAnimationPlayerPlayHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/animation/player-play" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Player/AnimationPlayer", "name": "walk"},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "animation", "player-play",
		"--node-path", "/root/Player/AnimationPlayer", "--animation", "walk")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "animation.player-play" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

func TestAnimationPlayerPlayRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"animation", "player-play"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --node-path is missing")
	}
	if !strings.Contains(err.Error(), "--node-path") {
		t.Fatalf("err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Phase 4: tilemap commands (#16)
// ---------------------------------------------------------------------------

func TestTilesetCreateHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tilemap/tileset-create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://tiles/world.tres", "created": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "tilemap", "tileset-create",
		"--path", "res://tiles/world.tres", "--tile-width", "16", "--tile-height", "16")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "tilemap.tileset-create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "TileSet created:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestTilesetCreateRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"tilemap", "tileset-create"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --path is missing")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func TestTilesetSourceAddHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tilemap/source-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "res://tiles/world.tres"},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "tilemap", "source-add",
		"--path", "res://tiles/world.tres", "--texture", "res://tiles/atlas.png")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "tilemap.source-add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

func TestTilemapCellSetHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tilemap/cell-set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"node": "/root/World/TileMap", "applied": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "tilemap", "cell-set",
		"--node", "/root/World/TileMap", "--x", "3", "--y", "4", "--source-id", "0")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "tilemap.cell-set" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Cell set:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestTilemapCellSetRectHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tilemap/cell-set-rect" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"node": "/root/World/TileMap", "layer": 0, "x": 3, "y": 4, "width": 5, "height": 2, "cells": 10, "applied": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "tilemap", "cell-set-rect",
		"--node", "/root/World/TileMap", "--x", "3", "--y", "4", "--width", "5", "--height", "2", "--source-id", "0")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "tilemap.cell-set-rect" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["width"] != float64(5) && gotEnvelope.Params["width"] != 5 {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Cell rect set:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestTilemapCellSetRectRequiresPositiveSizeBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"tilemap", "cell-set-rect", "--node", "/root/TileMap", "--width", "0", "--height", "2"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--width and --height") {
		t.Fatalf("err = %v", err)
	}
}

func TestTilemapCellClearHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tilemap/cell-clear" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"node": "/root/World/TileMap", "applied": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "tilemap", "cell-clear",
		"--node", "/root/World/TileMap", "--x", "3", "--y", "4")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "tilemap.cell-clear" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

func TestTilemapCellSetRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"tilemap", "cell-set", "--x", "0", "--y", "0", "--source-id", "0"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --node is missing")
	}
	if !strings.Contains(err.Error(), "--node") {
		t.Fatalf("err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Phase 4: audio commands (#17)
// ---------------------------------------------------------------------------

func TestAudioBusAddHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/bus-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"bus": "Music", "applied": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "audio", "bus-add", "--name", "Music")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "audio.bus-add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Audio bus added:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestAudioBusAddIfMissingHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/bus-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"bus": "Ambience", "applied": false, "created": false},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "audio", "bus-add", "--name", "Ambience", "--if-missing")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "audio.bus-add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["if_missing"] != true {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if !strings.Contains(stdout.String(), "Audio bus already exists: Ambience") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestAudioBusAddRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"audio", "bus-add"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --name is missing")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Fatalf("err = %v", err)
	}
}

func TestAudioBusVolumeSetHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/bus-volume-set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"bus": "Music", "applied": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "audio", "bus-volume-set",
		"--name", "Music", "--volume-db", "-6.0")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "audio.bus-volume-set" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["volume_db"] != -6.0 {
		t.Fatalf("volume_db = %v", gotEnvelope.Params["volume_db"])
	}
}

func TestAudioBusEffectAddHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/bus-effect-add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"bus": "Music", "applied": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "audio", "bus-effect-add",
		"--name", "Music", "--effect-type", "AudioEffectReverb")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "audio.bus-effect-add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

// ---------------------------------------------------------------------------
// Phase 4: viewport set-size / add (#18)
// ---------------------------------------------------------------------------

func TestViewportSetSizeHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/viewport/set-size" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"width": 1280, "height": 720},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "viewport", "set-size",
		"--width", "1280", "--height", "720")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "viewport.set-size" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "Viewport size set:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestViewportSetSizeRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"viewport", "set-size", "--width", "1280"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --height is missing")
	}
	if !strings.Contains(err.Error(), "--height") {
		t.Fatalf("err = %v", err)
	}
}

func TestViewportAddHappyPath(t *testing.T) {
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/viewport/add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/SubViewport", "width": 320, "height": 240, "added": true},
		})
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--token", "secret", "viewport", "add",
		"--width", "320", "--height", "240")
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "viewport.add" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if !strings.Contains(stdout.String(), "SubViewport added:") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestViewportAddRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"viewport", "add", "--width", "320"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --height is missing")
	}
	if !strings.Contains(err.Error(), "--height") {
		t.Fatalf("err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Help entries for new command groups
// ---------------------------------------------------------------------------

func TestHelpRunSmoke(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "run.smoke"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"run smoke", "--assert", "--screenshot", "PASS"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestHelpThemeGroup(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "theme"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"theme create", "theme set-color", "theme set-font-size", "theme set-constant"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestHelpAnimationGroup(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "animation"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"animation create", "animation track-add", "animation keyframe-add", "animation player-play"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestHelpTilemapGroup(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "tilemap"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"tilemap tileset-create", "tilemap source-add", "tilemap cell-set", "tilemap cell-set-rect", "tilemap cell-clear"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestHelpAudioGroup(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "audio"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"audio bus-add", "--if-missing", "audio bus-volume-set", "audio bus-effect-add"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestHelpViewportSetSizeAndAdd(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help", "viewport"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"viewport set-size", "viewport add"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}
