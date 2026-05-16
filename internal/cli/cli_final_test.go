package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdctl/internal/bridge"
)

// ---------------------------------------------------------------------------
// propertiesMapFromValue
// ---------------------------------------------------------------------------

func TestPropertiesMapFromValueValid(t *testing.T) {
	m, err := propertiesMapFromValue(map[string]any{"health": float64(100)})
	if err != nil {
		t.Fatal(err)
	}
	if m["health"] != float64(100) {
		t.Fatalf("unexpected: %v", m)
	}
}

func TestPropertiesMapFromValueEmpty(t *testing.T) {
	_, err := propertiesMapFromValue(map[string]any{})
	if err == nil {
		t.Fatal("expected error for empty map")
	}
}

func TestPropertiesMapFromValueNotMap(t *testing.T) {
	_, err := propertiesMapFromValue("not a map")
	if err == nil {
		t.Fatal("expected error for non-map")
	}
}

// ---------------------------------------------------------------------------
// runInputActionList: JSON output and with events using text/type labels
// ---------------------------------------------------------------------------

func TestInputActionListJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.InputActionListResult]{
			OK: true,
			Result: bridge.InputActionListResult{
				Actions: []bridge.InputActionResult{
					{Action: "jump", Deadzone: 0.5},
				},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "input", "action", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected JSON: %s", out)
	}
}

func TestInputActionListEventLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.InputActionListResult]{
			OK: true,
			Result: bridge.InputActionListResult{
				Actions: []bridge.InputActionResult{
					{Action: "fire", Events: []bridge.InputEventInfo{
						{Key: ""},
						{Text: "Space", Type: "key"},
					}},
					{Action: "dash", Events: []bridge.InputEventInfo{
						{Type: "joypad_button"},
					}},
				},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "input", "action", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "fire") || !strings.Contains(out, "dash") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runSceneBatchOperation: remaining branches
// ---------------------------------------------------------------------------

func makeSceneServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"job_id": "job-open-1"},
			})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"job_id": "job-save-1"},
			})
		case "/jobs/job-open-1", "/jobs/job-save-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					Status: "succeeded",
					Result: map[string]any{"path": "res://test.tscn"},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"path": "/root/Node"},
			})
		}
	}))
}

func makeBatchFile(t *testing.T, ops []map[string]any) string {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"operations": ops})
	f := filepath.Join(t.TempDir(), "batch.json")
	if err := os.WriteFile(f, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestSceneBatchNodeSet(t *testing.T) {
	server := makeSceneServer()
	defer server.Close()
	batchFile := makeBatchFile(t, []map[string]any{
		{"op": "node.set", "path": "/root/Node", "property": "visible", "value": true},
	})
	out, err := runCmd(t, server, "scene", "batch", "--path", "res://test.tscn", "--file", batchFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "node.set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestSceneBatchNodeSetMany(t *testing.T) {
	server := makeSceneServer()
	defer server.Close()
	batchFile := makeBatchFile(t, []map[string]any{
		{"op": "node.set-many", "path": "/root/Node", "properties": map[string]any{"visible": true, "modulate": "#fff"}},
	})

	// Need a server that returns updated count for set-many
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true, Result: map[string]any{"job_id": "j1"},
			})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK: true, Result: map[string]any{"job_id": "j2"},
			})
		case "/jobs/j1", "/jobs/j2":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{Status: "succeeded", Result: map[string]any{"path": "res://test.tscn"}},
			})
		case "/node/set-many":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.NodeSetManyResult]{
				OK:     true,
				Result: bridge.NodeSetManyResult{Updated: 2},
			})
		default:
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{}})
		}
	}))
	defer server2.Close()

	out, err := runCmd(t, server2, "scene", "batch", "--path", "res://test.tscn", "--file", batchFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "node.set-many") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestSceneBatchNodeAttachScript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"job_id": "j1"}})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"job_id": "j2"}})
		case "/jobs/j1", "/jobs/j2":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{Status: "succeeded", Result: map[string]any{"path": "res://t.tscn"}}})
		default:
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{}})
		}
	}))
	defer server.Close()
	batchFile := makeBatchFile(t, []map[string]any{
		{"op": "node.attach-script", "path": "/root/Node", "script": "res://player.gd"},
	})
	out, err := runCmd(t, server, "scene", "batch", "--path", "res://t.tscn", "--file", batchFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "attach-script") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestSceneBatchNodeSetResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"job_id": "j1"}})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"job_id": "j2"}})
		case "/jobs/j1", "/jobs/j2":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{Status: "succeeded", Result: map[string]any{"path": "res://t.tscn"}}})
		default:
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{}})
		}
	}))
	defer server.Close()
	batchFile := makeBatchFile(t, []map[string]any{
		{"op": "node.set-resource", "path": "/root/Node", "property": "texture", "resource": "res://icon.png"},
	})
	out, err := runCmd(t, server, "scene", "batch", "--path", "res://t.tscn", "--file", batchFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "set-resource") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestSceneBatchUnsupportedOp(t *testing.T) {
	server := makeSceneServer()
	defer server.Close()
	batchFile := makeBatchFile(t, []map[string]any{
		{"op": "node.unsupported"},
	})
	_, err := runCmd(t, server, "scene", "batch", "--path", "res://test.tscn", "--file", batchFile)
	if err == nil || !strings.Contains(err.Error(), "unsupported op") {
		t.Fatalf("expected unsupported op error, got %v", err)
	}
}

func TestSceneBatchMissingOp(t *testing.T) {
	server := makeSceneServer()
	defer server.Close()
	batchFile := makeBatchFile(t, []map[string]any{
		{"path": "/root/Node"},
	})
	_, err := runCmd(t, server, "scene", "batch", "--path", "res://test.tscn", "--file", batchFile)
	if err == nil || !strings.Contains(err.Error(), "requires op") {
		t.Fatalf("expected requires op error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// run wait-probe: requires source
// ---------------------------------------------------------------------------

func TestRunWaitProbeRequiresSource(t *testing.T) {
	err := Run(context.Background(), []string{"run", "wait-probe", "--assert", "k>=1"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--source") {
		t.Fatalf("expected --source error, got %v", err)
	}
}

func TestRunWaitProbeRequiresAssert(t *testing.T) {
	err := Run(context.Background(), []string{"run", "wait-probe", "--source", "engine"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--assert") {
		t.Fatalf("expected --assert error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// run start variants
// ---------------------------------------------------------------------------

func TestRunStartWithScene(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStartResult]{
			OK:     true,
			Result: bridge.RunStartResult{Scene: "res://main.tscn"},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "start", "--scene", "res://main.tscn")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Run started") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStartWithPlayingScene(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStartResult]{
			OK:     true,
			Result: bridge.RunStartResult{PlayingScene: "res://level1.tscn"},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "start")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "level1") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStartNoScene(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStartResult]{
			OK:     true,
			Result: bridge.RunStartResult{},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "start")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Run started") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStartBothSceneAndMain(t *testing.T) {
	err := Run(context.Background(), []string{"run", "start", "--scene", "x.tscn", "--main"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "at most one") {
		t.Fatalf("expected at-most-one error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// scene run: validation errors
// ---------------------------------------------------------------------------

func TestSceneRunRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"scene", "run"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--path") {
		t.Fatalf("expected --path error, got %v", err)
	}
}

func TestSceneRunRequiresGodot(t *testing.T) {
	err := Run(context.Background(), []string{"scene", "run", "--path", "res://main.tscn"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "godot") {
		t.Fatalf("expected godot error, got %v", err)
	}
}

func TestSceneRunRequiresProject(t *testing.T) {
	err := Run(context.Background(), []string{"scene", "run", "--path", "res://main.tscn", "--godot", "godot"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("expected --project error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// printRuntimeAddonDoctor: not-ok ping branch
// ---------------------------------------------------------------------------

func TestPrintRuntimeAddonDoctorNotOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{OK: false})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "addon", "doctor")
	_ = Run(context.Background(), args, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "gdctl Addon Doctor") {
		t.Fatalf("expected Addon Doctor header, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// help: unknown topic, usecase with group
// ---------------------------------------------------------------------------

func TestHelpUnknownTopic(t *testing.T) {
	err := Run(context.Background(), []string{"help", "nonexistentxyz"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown help topic") {
		t.Fatalf("expected unknown help topic error, got %v", err)
	}
}

func TestHelpUnknownSubcmdInGroup(t *testing.T) {
	err := Run(context.Background(), []string{"help", "scene", "nonexistent-sub"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// accessibility tts-stop: stopped=false branch
// ---------------------------------------------------------------------------

func TestAccessibilityTTSStopNotStopped(t *testing.T) {
	server := singleHandler("/accessibility/tts-stop", map[string]any{"stopped": false})
	defer server.Close()
	out, err := runCmd(t, server, "accessibility", "tts-stop")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected some output")
	}
}

// ---------------------------------------------------------------------------
// runRunSceneReload: no job ID error case
// ---------------------------------------------------------------------------

func TestRunSceneReloadNoJobID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{},
		})
	}))
	defer server.Close()
	_, err := runCmd(t, server, "run", "scene-reload")
	if err == nil || !strings.Contains(err.Error(), "job id") {
		t.Fatalf("expected job id error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// runInputEventAddKey: event not added case
// ---------------------------------------------------------------------------

func TestInputEventAddKeyNotAdded(t *testing.T) {
	server := singleHandler("/input/event-add-key", map[string]any{"action": "jump", "key": "Space", "event_added": false})
	defer server.Close()
	out, err := runCmd(t, server, "input", "event", "add-key", "--action", "jump", "--key", "Space")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected some output")
	}
}

// ---------------------------------------------------------------------------
// runFileWriteBytes
// ---------------------------------------------------------------------------

func TestFileWriteBytes(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "data.bin")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.FileWriteBytesResult]{
			OK:     true,
			Result: bridge.FileWriteBytesResult{Path: "res://data.bin", Bytes: 5},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "file", "write-bytes", "--path", "res://data.bin", "--in", tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "File written") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// autoload add
// ---------------------------------------------------------------------------

func TestAutoloadAdd(t *testing.T) {
	server := singleHandler("/autoload/add", map[string]any{"name": "GameManager", "path": "res://game_manager.gd"})
	defer server.Close()
	out, err := runCmd(t, server, "autoload", "add", "--name", "GameManager", "--path", "res://game_manager.gd")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Autoload added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAutoloadAddRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"autoload", "add", "--name", "GM"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--path") {
		t.Fatalf("expected --path error, got %v", err)
	}
}
