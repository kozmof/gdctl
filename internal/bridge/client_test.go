package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientAddNodeRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Marker"},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	props := map[string]any{"position": map[string]any{"kind": "Vector2", "value": []any{1, 2}}}
	result, err := client.AddNode(context.Background(), "cli-test", "/root/Main", "Node2D", "Marker", props, true)
	if err != nil {
		t.Fatal(err)
	}

	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.RequestID != "cli-test" || gotEnvelope.Op != "node.add" {
		t.Fatalf("envelope = %#v", gotEnvelope)
	}
	if gotEnvelope.Params["dry_run"] != true {
		t.Fatalf("dry_run = %#v", gotEnvelope.Params["dry_run"])
	}
	if _, ok := gotEnvelope.Params["props"].(map[string]any); !ok {
		t.Fatalf("props = %#v", gotEnvelope.Params["props"])
	}
	if result["path"] != "/root/Main/Marker" {
		t.Fatalf("result path = %#v", result["path"])
	}
}

func TestClientUpdateAddonRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/addon/update" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
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

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.UpdateAddon(context.Background(), "cli-test", map[string]any{"name": "godot_tcp_bridge"}, []AddonUpdateFile{
		{Path: "plugin.cfg", ContentBase64: "Zm9v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "addon.update" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if result.FilesWritten != 4 || !result.ReloadRequired {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientLogsRequestUsesAuth(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/logs" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(LogsResponse{
			OK: true,
			Entries: []LogEntry{
				{Time: "now", Level: "info", Source: "test", Message: "hello"},
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	entries, err := client.Logs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if len(entries) != 1 || entries[0].Message != "hello" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestClientClearLogsRequest(t *testing.T) {
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/logs/clear" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"cleared": true},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	if err := client.ClearLogs(context.Background(), "cli-test"); err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "logs.clear" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
}

func TestClientRunStartRequest(t *testing.T) {
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/start" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"running":       true,
				"scene":         "res://main.tscn",
				"playing_scene": "main",
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.RunStart(context.Background(), "cli-test", "res://main.tscn", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "run.start" || gotEnvelope.Params["scene"] != "res://main.tscn" || gotEnvelope.Params["clear_logs"] != true {
		t.Fatalf("envelope = %#v", gotEnvelope)
	}
	if !result.Running || result.Scene != "res://main.tscn" {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientRunLogsRequest(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/logs" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(LogsResponse{
			OK:      true,
			Entries: []LogEntry{{Time: "now", Level: "error", Source: "runtime.error", Message: "boom"}},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	entries, err := client.RunLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if len(entries) != 1 || entries[0].Message != "boom" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestClientClearRunLogsRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/logs/clear" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"cleared": true},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	if err := client.ClearRunLogs(context.Background(), "cli-test"); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "run.logs.clear" {
		t.Fatalf("envelope = %#v", gotEnvelope)
	}
}

func TestClientRunScreenshotRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/screenshot" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"queued": true,
				"job_id": "run-shot-1",
				"screen": 1,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.RunScreenshot(context.Background(), "cli-test", "screen", 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "run.screenshot" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["source"] != "screen" || gotEnvelope.Params["screen"] != float64(1) {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if result.JobID != "run-shot-1" || !result.Queued || result.Screen != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientJobRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jobs/save-1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(JobResponse{
			OK: true,
			Job: Job{
				ID:     "save-1",
				Kind:   "scene.save",
				Status: "succeeded",
				Result: map[string]any{"path": "res://main.tscn"},
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http"}
	client := NewClient(cfg)
	job, err := client.Job(context.Background(), "save-1")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "succeeded" || job.Result["path"] != "res://main.tscn" {
		t.Fatalf("job = %#v", job)
	}
}

func TestClientRenameNodeRequest(t *testing.T) {
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/rename" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Renamed"},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	_, err := client.RenameNode(context.Background(), "cli-test", "/root/Main/Old", "Renamed", true)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.rename" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["name"] != "Renamed" || gotEnvelope.Params["dry_run"] != true {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
}

func TestClientMoveNodeRequest(t *testing.T) {
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/move" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/NewParent/Child"},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	_, err := client.MoveNode(context.Background(), "cli-test", "/root/Main/Child", "/root/Main/NewParent", 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.move" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["parent"] != "/root/Main/NewParent" || gotEnvelope.Params["index"] != float64(2) {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
}

func TestClientGetNodePropertyRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/get" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":     "/root/Main/Player",
				"property": "position",
				"value":    map[string]any{"kind": "Vector2", "value": []any{200, 400}},
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.GetNodeProperty(context.Background(), "cli-test", "/root/Main/Player", "position")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "node.get" || gotEnvelope.Params["property"] != "position" {
		t.Fatalf("envelope = %#v", gotEnvelope)
	}
	if result.Path != "/root/Main/Player" || result.Property != "position" {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientSetNodePropertyRequest(t *testing.T) {
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/set" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":     "/root/Main/Player",
				"property": "position",
				"value":    map[string]any{"kind": "Vector2", "value": []any{200, 400}},
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	_, err := client.SetNodeProperty(context.Background(), "cli-test", "/root/Main/Player", "position", map[string]any{
		"kind":  "Vector2",
		"value": []any{200, 400},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "node.set" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	value, ok := gotEnvelope.Params["value"].(map[string]any)
	if !ok || value["kind"] != "Vector2" {
		t.Fatalf("value = %#v", gotEnvelope.Params["value"])
	}
}

func TestClientAttachScriptRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/attach-script" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":     "/root/Main",
				"script":   "res://scripts/player.gd",
				"attached": true,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.AttachScript(context.Background(), "cli-test", "/root/Main", "res://scripts/player.gd")
	if err != nil {
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
	if result.Path != "/root/Main" || result.Script != "res://scripts/player.gd" || !result.Attached {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientSaveSceneRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/save" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":  "res://main.tscn",
				"root":  "/root/Main",
				"saved": true,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.SaveScene(context.Background(), "cli-test", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "scene.save" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if _, ok := gotEnvelope.Params["path"]; ok {
		t.Fatalf("path param should be omitted: %#v", gotEnvelope.Params)
	}
	if result.Path != "res://main.tscn" || result.Root != "/root/Main" || !result.Saved {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientCreateSceneRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
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

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.CreateScene(context.Background(), "cli-test", "res://scenes/Main.tscn", "Node2D", "Main", true)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "scene.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scenes/Main.tscn" || gotEnvelope.Params["root_type"] != "Node2D" || gotEnvelope.Params["root_name"] != "Main" || gotEnvelope.Params["force"] != true {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if result.Path != "res://scenes/Main.tscn" || result.RootType != "Node2D" || result.RootName != "Main" || result.RootPath != "/root/Main" || !result.Created {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientOpenSceneRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/open" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":   "res://scenes/Main.tscn",
				"queued": true,
				"job_id": "open-1",
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.OpenScene(context.Background(), "cli-test", "res://scenes/Main.tscn")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "scene.open" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scenes/Main.tscn" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if result.Path != "res://scenes/Main.tscn" || !result.Queued || result.JobID != "open-1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientScreenshotViewportRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/viewport/screenshot" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"queued": true,
				"job_id": "shot-1",
				"kind":   "3d",
				"index":  1,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.ScreenshotViewport(context.Background(), "cli-test", "3d", 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "viewport.screenshot" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["kind"] != "3d" || gotEnvelope.Params["index"] != float64(1) {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if result.JobID != "shot-1" || !result.Queued || result.Kind != "3d" || result.Index != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientApplySceneRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/apply" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"root":    "/root/Main",
				"created": 2,
				"updated": 3,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	tree := map[string]any{"root": map[string]any{"children": []any{map[string]any{"name": "Platform", "type": "StaticBody3D"}}}}
	result, err := client.ApplyScene(context.Background(), "cli-test", tree, false)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.Op != "scene.apply" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["dry_run"] != false {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if _, ok := gotEnvelope.Params["tree"].(map[string]any); !ok {
		t.Fatalf("tree missing: %#v", gotEnvelope.Params)
	}
	if result.Root != "/root/Main" || result.Created != 2 || result.Updated != 3 {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientInstanceSceneRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scene/instance" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
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

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.InstanceScene(context.Background(), "cli-test", "/root/Main", "res://scenes/Child.tscn", "Child")
	if err != nil {
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
	if result.Path != "/root/Main/Child" || result.Scene != "res://scenes/Child.tscn" || !result.Instanced {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientCheckScriptRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/script/check" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":  "res://scripts/player.gd",
				"valid": true,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.CheckScript(context.Background(), "cli-test", "res://scripts/player.gd")
	if err != nil {
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
	if result.Path != "res://scripts/player.gd" || !result.Valid {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientCreateScriptRequest(t *testing.T) {
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/script/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://scripts/player.gd",
				"valid":   true,
				"created": true,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http"}
	client := NewClient(cfg)
	result, err := client.CreateScript(context.Background(), "cli-test", "res://scripts/player.gd", "Node2D", true)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "script.create" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scripts/player.gd" || gotEnvelope.Params["extends"] != "Node2D" || gotEnvelope.Params["force"] != true {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if result.Path != "res://scripts/player.gd" || !result.Valid || !result.Created {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientWriteScriptRequest(t *testing.T) {
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/script/write" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK: true,
			Result: map[string]any{
				"path":    "res://scripts/player.gd",
				"valid":   true,
				"written": true,
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http"}
	client := NewClient(cfg)
	result, err := client.WriteScript(context.Background(), "cli-test", "res://scripts/player.gd", "extends Node2D\n")
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvelope.Op != "script.write" {
		t.Fatalf("op = %q", gotEnvelope.Op)
	}
	if gotEnvelope.Params["path"] != "res://scripts/player.gd" || gotEnvelope.Params["body"] != "extends Node2D\n" {
		t.Fatalf("params = %#v", gotEnvelope.Params)
	}
	if result.Path != "res://scripts/player.gd" || !result.Valid || !result.Written {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientBridgeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(BridgeResponse[any]{
			OK: false,
			Error: &BridgeError{
				Code:    "NODE_PARENT_NOT_FOUND",
				Message: "Parent node does not exist",
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http"}
	client := NewClient(cfg)
	_, err := client.RemoveNode(context.Background(), "cli-test", "/root/Main/Missing", false)
	if err == nil {
		t.Fatal("expected error")
	}
	bridgeErr, ok := err.(*BridgeError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if bridgeErr.Code != "NODE_PARENT_NOT_FOUND" {
		t.Fatalf("code = %s", bridgeErr.Code)
	}
}
