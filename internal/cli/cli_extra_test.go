package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gdctl/internal/bridge"
)

// ---------------------------------------------------------------------------
// run probe node: additional branches (json output, no type, no path in result)
// ---------------------------------------------------------------------------

func makeJobServerAt(path string, jobResult map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case path:
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"job_id": "job-probe-1"},
			})
		case "/jobs/job-probe-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{Status: "succeeded", Result: jobResult},
			})
		default:
			ok200(w, map[string]any{})
		}
	}))
}

func TestRunProbeNodeJSON(t *testing.T) {
	server := makeJobServerAt("/run/probe/node", map[string]any{
		"path":       "/root/Player",
		"type":       "CharacterBody3D",
		"properties": map[string]any{"speed": float64(10)},
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "probe", "node", "--path", "/root/Player", "--property", "speed", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected JSON, got: %s", out)
	}
}

func TestRunProbeNodeNoType(t *testing.T) {
	server := makeJobServerAt("/run/probe/node", map[string]any{
		"properties": map[string]any{"visible": true},
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "probe", "node", "--path", "/root/Player", "--property", "visible")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Node probe") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunProbeNodeRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"run", "probe", "node"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--path") {
		t.Fatalf("expected --path error, got %v", err)
	}
}

func TestRunProbeNodeRequiresProperty(t *testing.T) {
	err := Run(context.Background(), []string{"run", "probe", "node", "--path", "/root/Node"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--property") {
		t.Fatalf("expected --property error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// run probe raycast: json output, direct result (no job_id)
// ---------------------------------------------------------------------------

func TestRunProbeRaycastJSON(t *testing.T) {
	server := makeJobServerAt("/run/probe/raycast", map[string]any{
		"hit": true, "hit_collider": "/root/Wall", "hit_distance": 5.0,
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "probe", "raycast", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected JSON, got: %s", out)
	}
}

func TestRunProbeRaycastDirectResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunRaycastResult]{
			OK:     true,
			Result: bridge.RunRaycastResult{Hit: false, CameraPath: "/root/Camera3D"},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "probe", "raycast")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no hit") {
		t.Fatalf("stdout: %s", out)
	}
	if !strings.Contains(out, "Camera") {
		t.Fatalf("expected camera in output: %s", out)
	}
}

// ---------------------------------------------------------------------------
// run wait-probe: happy path (match found immediately)
// ---------------------------------------------------------------------------

func TestRunWaitProbeMatchFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/run/status" {
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
		} else if r.URL.Path == "/run/logs" {
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
				OK: true,
				Entries: []bridge.LogEntry{
					{
						Time:    "2024-01-01T00:00:00Z",
						Level:   "info",
						Source:  "engine",
						Message: "probe",
						Detail:  map[string]any{"health": float64(80)},
					},
				},
			})
		} else {
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "wait-probe", "--source", "engine", "--assert", "health>=50", "--timeout", "5s")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "matched") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunWaitProbeMatchFoundJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/run/status" {
			writeRunStatusResponse(t, w, true, "res://main.tscn", bridge.RuntimeHelperStatus{Present: true})
		} else if r.URL.Path == "/run/logs" {
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{
				OK: true,
				Entries: []bridge.LogEntry{
					{Source: "engine", Message: "probe", Detail: map[string]any{"count": float64(5)}},
				},
			})
		} else {
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "wait-probe", "--source", "engine", "--assert", "count==5", "--json", "--timeout", "5s")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected JSON, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// run smoke: input file path
// ---------------------------------------------------------------------------

func TestRunSmokeWithInputFile(t *testing.T) {
	tmpFile := writeTempJSONFile(t, map[string]any{
		"steps": []any{map[string]any{"type": "wait", "ms": 100}},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/stop":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"stopped": true}})
		case "/run/start":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStartResult]{OK: true, Result: bridge.RunStartResult{Scene: "res://main.tscn"}})
		case "/run/input":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{}})
		default:
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "smoke", "--main", "--input", tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Smoke: PASS") {
		t.Fatalf("stdout: %s", out)
	}
}

func writeTempJSONFile(t *testing.T, v any) string {
	t.Helper()
	data, _ := json.Marshal(v)
	f, err := os.CreateTemp(t.TempDir(), "*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

// ---------------------------------------------------------------------------
// animation tree: unknown subcommand
// ---------------------------------------------------------------------------

func TestAnimationTreeUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"animation", "tree", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown animation tree") {
		t.Fatalf("expected unknown animation tree error, got %v", err)
	}
}

func TestAnimationTreeRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"animation", "tree"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// runNodeGroupList: with groups
// ---------------------------------------------------------------------------

func TestNodeGroupListWithGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.NodeGroupListResult]{
			OK: true,
			Result: bridge.NodeGroupListResult{
				Path:   "/root/Player",
				Groups: []string{"players", "movers"},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "node", "group", "list", "--path", "/root/Player")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "players") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// audio bus volume set and effect add
// ---------------------------------------------------------------------------

func TestAudioBusVolumeSet(t *testing.T) {
	server := singleHandler("/audio/bus-volume-set", map[string]any{"bus": "Music"})
	defer server.Close()
	out, err := runCmd(t, server, "audio", "bus-volume-set", "--name", "Music", "--volume-db", "-6")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Music") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAudioBusVolumeSetRequiresName(t *testing.T) {
	err := Run(context.Background(), []string{"audio", "bus-volume-set"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAudioBusEffectAdd(t *testing.T) {
	server := singleHandler("/audio/bus-effect-add", map[string]any{"bus": "SFX"})
	defer server.Close()
	out, err := runCmd(t, server, "audio", "bus-effect-add", "--name", "SFX", "--effect-type", "AudioEffectReverb")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "AudioEffectReverb") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAudioBusEffectAddRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"audio", "bus-effect-add", "--name", "SFX"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// project setting get
// ---------------------------------------------------------------------------

func TestProjectSettingGet(t *testing.T) {
	server := singleHandler("/project/setting-get", map[string]any{"key": "application/config/name", "value": "Demo"})
	defer server.Close()
	out, err := runCmd(t, server, "project", "setting", "get", "--key", "application/config/name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Demo") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestProjectSettingGetRequiresKey(t *testing.T) {
	err := Run(context.Background(), []string{"project", "setting", "get"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--key") {
		t.Fatalf("expected --key error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// input action add: already exists case
// ---------------------------------------------------------------------------

func TestInputActionAddRequiresName(t *testing.T) {
	err := Run(context.Background(), []string{"input", "action", "add"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--name") {
		t.Fatalf("expected --name error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// runInputEventAddKey: requires flags
// ---------------------------------------------------------------------------

func TestInputEventAddKeyRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"input", "event", "add-key", "--action", "jump"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--key") {
		t.Fatalf("expected --key error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// runLODSetMany: empty file error
// ---------------------------------------------------------------------------

func TestLODSetManyNoContent(t *testing.T) {
	tmpFile := writeTempJSONFile(t, []any{})
	err := Run(context.Background(), []string{"recipe", "lod", "set-many", "--file", tmpFile}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no entries") {
		t.Fatalf("expected no entries error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// openSceneAndWait: no job_id error branch
// ---------------------------------------------------------------------------

func TestOpenSceneNoJobID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{},
		})
	}))
	defer server.Close()
	// node add with --scene triggers openSceneAndWait
	_, err := runCmd(t, server, "node", "add", "--parent", "/root", "--type", "Node", "--name", "Test", "--scene", "res://test.tscn")
	if err == nil || !strings.Contains(err.Error(), "job id") {
		t.Fatalf("expected job id error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// saveSceneAndWait: no job_id error branch
// ---------------------------------------------------------------------------

func TestSaveSceneNoJobID(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"job_id": "jopen"}})
		case "/jobs/jopen":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{Status: "succeeded", Result: map[string]any{"path": "res://t.tscn"}}})
		case "/node/add":
			callCount++
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"path": "/root/Test"}})
		case "/scene/save":
			// returns no job_id
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{}})
		}
	}))
	defer server.Close()
	_, err := runCmd(t, server, "node", "add", "--parent", "/root", "--type", "Node", "--name", "Test", "--scene", "res://t.tscn")
	if err == nil || !strings.Contains(err.Error(), "job id") {
		t.Fatalf("expected job id error from save, got %v (callCount=%d)", err, callCount)
	}
}
