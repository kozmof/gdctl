package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gdctl/internal/bridge"
)

// ---------------------------------------------------------------------------
// runDoctor: additional ping branches
// ---------------------------------------------------------------------------

func TestDoctorPingWithProjectName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ping":
			_ = json.NewEncoder(w).Encode(bridge.PingResponse{
				OK:            true,
				ProjectName:   "MyGame",
				PluginVersion: "0.1.9",
			})
		default:
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "doctor")
	_ = Run(context.Background(), args, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "Godot TCP Bridge Doctor") {
		t.Fatalf("expected doctor header, got: %s", out)
	}
	if !strings.Contains(out, "project is open") {
		t.Fatalf("expected project is open in output: %s", out)
	}
}

func TestDoctorPingNotOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{OK: false})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "doctor")
	_ = Run(context.Background(), args, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "ping returned not ok") {
		t.Fatalf("expected ping not ok in output: %s", out)
	}
}

func TestDoctorWithProjectFix(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ping":
			_ = json.NewEncoder(w).Encode(bridge.PingResponse{
				OK:              true,
				PluginVersion:   "0.1.9",
				ProtocolVersion: "gdctl.v1",
			})
		default:
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "doctor", "--project", project, "--fix")
	_ = Run(context.Background(), args, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "Godot TCP Bridge Doctor") {
		t.Fatalf("expected doctor header, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runRunSmoke: assert path that matches
// ---------------------------------------------------------------------------

func TestRunSmokeWithAssertPass(t *testing.T) {
	logsCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/stop":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"stopped": true}})
		case "/run/start":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStartResult]{OK: true, Result: bridge.RunStartResult{Scene: "res://main.tscn"}})
		case "/run/logs":
			logsCount++
			entries := []bridge.LogEntry{}
			if logsCount >= 1 {
				entries = []bridge.LogEntry{
					{Source: "engine", Message: "probe", Detail: map[string]any{"health": float64(100)}},
				}
			}
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{OK: true, Entries: entries})
		default:
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "smoke", "--main", "--assert", "engine:health>=50", "--timeout", "10s")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Smoke: PASS") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunSmokeWithSplitAssertPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/stop":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"stopped": true}})
		case "/run/start":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStartResult]{OK: true, Result: bridge.RunStartResult{Scene: "res://main.tscn"}})
		case "/run/logs":
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{OK: true, Entries: []bridge.LogEntry{
				{Source: "engine", Message: "p", Detail: map[string]any{"score": float64(99)}},
			}})
		default:
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "smoke", "--main",
		"--assert-source", "engine",
		"--assert-key", "score",
		"--assert-op", ">=",
		"--assert-value", "50",
		"--timeout", "10s")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Smoke: PASS") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// parseNameResourcePairs
// ---------------------------------------------------------------------------

func TestParseNameResourcePairsValid(t *testing.T) {
	out, err := parseNameResourcePairs([]string{"texture=res://icon.png", "mesh=res://mesh.tres"})
	if err != nil {
		t.Fatal(err)
	}
	if out["texture"] != "res://icon.png" {
		t.Errorf("unexpected: %v", out)
	}
}

func TestParseNameResourcePairsMissingEquals(t *testing.T) {
	_, err := parseNameResourcePairs([]string{"noequals"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// runViewportScreenshot: validation branches
// ---------------------------------------------------------------------------

func TestViewportScreenshotRequiresOut(t *testing.T) {
	err := Run(context.Background(), []string{"viewport", "screenshot"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--out") {
		t.Fatalf("expected --out error, got %v", err)
	}
}

func TestViewportScreenshotInvalidKind(t *testing.T) {
	err := Run(context.Background(), []string{"viewport", "screenshot", "--out", "/tmp/x.png", "--kind", "invalid"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--kind") {
		t.Fatalf("expected --kind error, got %v", err)
	}
}

func TestViewportScreenshotRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"viewport"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// runNodeGroupRemove: empty groups branch
// ---------------------------------------------------------------------------

func TestNodeGroupRemoveResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.NodeGroupResult]{
			OK:     true,
			Result: bridge.NodeGroupResult{Path: "/root/Player", Group: "players", Removed: true},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "node", "group", "remove", "--path", "/root/Player", "--group", "players")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "players") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// graph-edit node remove: normal branch
// ---------------------------------------------------------------------------

func TestGraphEditNodeRemoveOK(t *testing.T) {
	server := singleHandler("/graph-edit/node-remove", map[string]any{"name": "MyNode"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "graph-edit", "node-remove", "--path", "/root/Graph", "--name", "MyNode")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "MyNode") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// node duplicate: dry-run branch
// ---------------------------------------------------------------------------

func TestNodeDuplicateDryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.NodeDuplicateResult]{
			OK:     true,
			Result: bridge.NodeDuplicateResult{Path: "/root/PlayerCopy", SourcePath: "/root/Player", DryRun: true},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "node", "duplicate", "--path", "/root/Player", "--name", "PlayerCopy", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Dry run") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// scene apply: requires --path and --file
// ---------------------------------------------------------------------------

func TestSceneApplyRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"scene", "apply", "--path", "res://t.tscn"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--file") {
		t.Fatalf("expected --file error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// scene apply-blueprint: json body path
// ---------------------------------------------------------------------------

func TestSceneApplyBlueprintRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"scene", "apply-blueprint", "--path", "/root"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--blueprint") {
		t.Fatalf("expected --blueprint error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// run instantiate: no path in result
// ---------------------------------------------------------------------------

func TestRunInstantiateNoPath(t *testing.T) {
	server := makeJobServerAt("/run/instantiate", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "run", "instantiate", "--scene", "res://player.tscn", "--parent", "/root")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Instantiated") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// autoload list: empty
// ---------------------------------------------------------------------------

func TestAutoloadListEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.AutoloadListResult]{
			OK:     true,
			Result: bridge.AutoloadListResult{Autoloads: []bridge.AutoloadResult{}},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "autoload", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No autoloads") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// openSceneAndWait: path from result vs result.Path
// ---------------------------------------------------------------------------

func TestOpenSceneResultPathFromJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scene/open":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"job_id": "j1", "path": "res://fallback.tscn"}})
		case "/jobs/j1":
			// Return path in job result
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{Status: "succeeded", Result: map[string]any{"path": "res://actual.tscn", "root": "/root/Main"}}})
		case "/scene/save":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{"job_id": "j2"}})
		case "/jobs/j2":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{OK: true, Job: bridge.Job{Status: "succeeded", Result: map[string]any{"path": "res://actual.tscn"}}})
		default:
			ok200(w, map[string]any{})
		}
	}))
	defer server.Close()
	// scene batch uses openSceneAndWait + saveSceneAndWait
	batchFile := makeBatchFile(t, []map[string]any{
		{"op": "node.add", "parent": "/root", "type": "Node", "name": "Test"},
	})
	out, err := runCmd(t, server, "scene", "batch", "--path", "res://actual.tscn", "--file", batchFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "actual") {
		t.Fatalf("stdout: %s", out)
	}
}
