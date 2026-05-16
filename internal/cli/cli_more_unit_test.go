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
// parseVec2
// ---------------------------------------------------------------------------

func TestParseVec2Valid(t *testing.T) {
	x, y, err := parseVec2("10.5,20.3")
	if err != nil {
		t.Fatal(err)
	}
	if x != 10.5 || y != 20.3 {
		t.Fatalf("got %v,%v", x, y)
	}
}

func TestParseVec2Missing(t *testing.T) {
	_, _, err := parseVec2("10")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseVec2InvalidX(t *testing.T) {
	_, _, err := parseVec2("x,10")
	if err == nil {
		t.Fatal("expected error for invalid X")
	}
}

func TestParseVec2InvalidY(t *testing.T) {
	_, _, err := parseVec2("10,y")
	if err == nil {
		t.Fatal("expected error for invalid Y")
	}
}

// ---------------------------------------------------------------------------
// Dispatcher unknown subcommand branches
// ---------------------------------------------------------------------------

func TestAutoloadUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"autoload", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown autoload") {
		t.Fatalf("expected unknown autoload error, got %v", err)
	}
}

func TestAutoloadRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"autoload"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGraphEditUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"graph-edit", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown graph-edit") {
		t.Fatalf("expected unknown graph-edit error, got %v", err)
	}
}

func TestGraphEditRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"graph-edit"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestI18nUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"i18n", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown i18n") {
		t.Fatalf("expected unknown i18n error, got %v", err)
	}
}

func TestLODUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"lod", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown lod") {
		t.Fatalf("expected unknown lod error, got %v", err)
	}
}

func TestDecalUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"decal", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown decal") {
		t.Fatalf("expected unknown decal error, got %v", err)
	}
}

func TestDecalRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"decal"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCSGUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"csg", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown csg") {
		t.Fatalf("expected unknown csg error, got %v", err)
	}
}

func TestCSGRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"csg"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBridgeUnknownSubcmd(t *testing.T) {
	// Need a server because bridge commands require a valid connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok200(w, map[string]any{})
	}))
	defer server.Close()
	err := Run(context.Background(), append(serverArgs(server), "bridge", "unknown"), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown bridge command") {
		t.Fatalf("expected unknown bridge command error, got %v", err)
	}
}

func TestBridgeRequiresSubcmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok200(w, map[string]any{})
	}))
	defer server.Close()
	err := Run(context.Background(), append(serverArgs(server), "bridge"), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "bridge requires a subcommand") {
		t.Fatalf("expected bridge requires subcommand error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// run status: running without playing scene
// ---------------------------------------------------------------------------

func TestRunStatusRunningNoScene(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
			OK: true,
			Result: bridge.RunStatusResult{
				Running: true,
				// no PlayingScene, no Debugger
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "running") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// run status: debugger with stack frames
// ---------------------------------------------------------------------------

func TestRunStatusPausedWithStackFrames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
			OK: true,
			Result: bridge.RunStatusResult{
				Running:      true,
				PlayingScene: "res://main.tscn",
				Debugger: bridge.DebuggerState{
					Paused: true,
					File:   "res://player.gd",
					Line:   10,
					StackFrames: []bridge.DebuggerFrame{
						{File: "res://player.gd", Line: 10, Function: "_process"},
						{File: "res://player.gd", Line: 0, Function: "_ready"},
					},
				},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "_process") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStatusPausedWithRawStack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
			OK: true,
			Result: bridge.RunStatusResult{
				Running: true,
				Debugger: bridge.DebuggerState{
					Paused: true,
					Stack: []map[string]any{
						{"file": "res://main.gd", "function": "_process", "line": float64(5)},
						{"file": "res://main.gd", "line": float64(0)},
					},
				},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "paused") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// run smoke: minimal path (no scene, no assert, no screenshot, no input)
// ---------------------------------------------------------------------------

func makeSmokeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/stop":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"stopped": true},
			})
		case "/run/start":
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStartResult]{
				OK:     true,
				Result: bridge.RunStartResult{Scene: "res://main.tscn"},
			})
		default:
			ok200(w, map[string]any{})
		}
	}))
}

func TestRunSmokeBasic(t *testing.T) {
	server := makeSmokeServer(t)
	defer server.Close()
	out, err := runCmd(t, server, "run", "smoke", "--main")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Smoke: PASS") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunSmokeSceneAndMainError(t *testing.T) {
	server := makeSmokeServer(t)
	defer server.Close()
	_, err := runCmd(t, server, "run", "smoke", "--scene", "res://x.tscn", "--main")
	if err == nil || !strings.Contains(err.Error(), "at most one") {
		t.Fatalf("expected at-most-one error, got %v", err)
	}
}

func TestRunSmokeMissingAssertSource(t *testing.T) {
	server := makeSmokeServer(t)
	defer server.Close()
	_, err := runCmd(t, server, "run", "smoke", "--main", "--assert-key", "hp", "--assert-op", ">=", "--assert-value", "1")
	if err == nil || !strings.Contains(err.Error(), "--assert-source") {
		t.Fatalf("expected --assert-source error, got %v", err)
	}
}

func TestRunSmokeAssertCombinedError(t *testing.T) {
	server := makeSmokeServer(t)
	defer server.Close()
	_, err := runCmd(t, server, "run", "smoke", "--main", "--assert", "src:k>=1", "--assert-key", "k")
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("expected cannot be combined error, got %v", err)
	}
}

func TestRunSmokeAssertInvalidFormat(t *testing.T) {
	server := makeSmokeServer(t)
	defer server.Close()
	// --assert without colon separator
	_, err := runCmd(t, server, "run", "smoke", "--main", "--assert", "no-colon")
	if err == nil || !strings.Contains(err.Error(), "SOURCE:KEY") {
		t.Fatalf("expected SOURCE:KEY error, got %v", err)
	}
}

func TestRunSmokeAssertInvalidExpr(t *testing.T) {
	server := makeSmokeServer(t)
	defer server.Close()
	// source:predicate with no valid operator
	_, err := runCmd(t, server, "run", "smoke", "--main", "--assert", "src:k~=1")
	if err == nil || !strings.Contains(err.Error(), "--assert") {
		t.Fatalf("expected --assert error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// smokeHelperFailureSummary
// ---------------------------------------------------------------------------

func TestSmokeHelperFailureSummaryPresent(t *testing.T) {
	server := makeRunStatusServer(true, bridge.RuntimeHelperStatus{Present: true})
	defer server.Close()
	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.SplitN(hostPort, ":", 2)
	port := 9001
	if len(parts) == 2 {
		import_strconv_atoi(&port, parts[1])
	}
	client := bridge.NewClient(bridge.Config{Host: parts[0], Port: port, Protocol: "http"})
	result := smokeHelperFailureSummary(context.Background(), client)
	if result != "runtime helper present" {
		t.Fatalf("got %q", result)
	}
}

func TestSmokeHelperFailureSummaryError(t *testing.T) {
	server := makeRunStatusServer(false, bridge.RuntimeHelperStatus{Error: "crashed"})
	defer server.Close()
	client := clientForSmokeServer(server)
	result := smokeHelperFailureSummary(context.Background(), client)
	if !strings.Contains(result, "crashed") {
		t.Fatalf("got %q", result)
	}
}

func TestSmokeHelperFailureSummaryNotPresent(t *testing.T) {
	server := makeRunStatusServer(false, bridge.RuntimeHelperStatus{})
	defer server.Close()
	client := clientForSmokeServer(server)
	result := smokeHelperFailureSummary(context.Background(), client)
	if result != "runtime helper not present" {
		t.Fatalf("got %q", result)
	}
}

func clientForSmokeServer(server *httptest.Server) *bridge.Client {
	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.SplitN(hostPort, ":", 2)
	port := bridge.DefaultPort
	import_strconv_atoi(&port, parts[1])
	return bridge.NewClient(bridge.Config{Host: parts[0], Port: port, Protocol: "http"})
}

func import_strconv_atoi(port *int, s string) {
	// strconv.Atoi helper without import collision
	p := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return
		}
		p = p*10 + int(c-'0')
	}
	*port = p
}
