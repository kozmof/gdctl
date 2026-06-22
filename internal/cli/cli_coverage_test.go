package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	addonPkg "gdctl/internal/addon"
	"gdctl/internal/bridge"
)

// ---------------------------------------------------------------------------
// help --usecase
// ---------------------------------------------------------------------------

func TestHelpUsecaseAll(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"help", "--usecase"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	// printUsecaseAll prints group headers and use cases
	out := stdout.String()
	if len(out) == 0 {
		t.Fatal("expected usecase output")
	}
}

func TestHelpUsecaseGroup(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"help", "--usecase", "scene"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if len(stdout.String()) == 0 {
		t.Fatal("expected usecase output for scene group")
	}
}

func TestHelpUsecaseCommandWithFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Use a known command in a known group with --usecase flag
	err := Run(context.Background(), []string{"help", "--usecase", "node", "add"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// doctor command
// ---------------------------------------------------------------------------

func TestDoctorWithProject(t *testing.T) {
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
	args := append(serverArgs(server), "doctor", "--project", project)
	// doctor may fail (addon compatibility) but it runs the function
	_ = Run(context.Background(), args, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "Godot TCP Bridge Doctor") {
		t.Fatalf("expected doctor header, got: %s", out)
	}
}

func TestDoctorNoProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{OK: true, PluginVersion: "0.1.9"})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "doctor")
	_ = Run(context.Background(), args, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "Godot TCP Bridge Doctor") {
		t.Fatalf("expected doctor header, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// addon status
// ---------------------------------------------------------------------------

func TestAddonStatusWithProject(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)

	// Install the addon first so status works
	mgr := newAddonManager()
	if _, err := mgr.Install(addonPkg.InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.9",
			ProtocolVersion: "gdctl.v1",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--project", project, "addon", "status")
	err := Run(context.Background(), args, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "gdctl bridge addon") {
		t.Fatalf("expected addon status header, got: %s", out)
	}
}

func TestAddonStatusWithProjectJSON(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)

	mgr := newAddonManager()
	if _, err := mgr.Install(addonPkg.InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{OK: true})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "--project", project, "addon", "status", "--json")
	err := Run(context.Background(), args, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("expected JSON output, got: %s", stdout.String())
	}
}

func TestAddonStatusRuntimeMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.9",
			ProtocolVersion: "gdctl.v1",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "addon", "status")
	err := Run(context.Background(), args, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "gdctl bridge addon") {
		t.Fatalf("expected addon status header, got: %s", out)
	}
}

func TestAddonStatusRuntimeModeJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{OK: true, PluginVersion: "0.1.9"})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "addon", "status", "--json")
	err := Run(context.Background(), args, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("expected JSON, got: %s", stdout.String())
	}
	if m["mode"] != "runtime" {
		t.Errorf("expected mode=runtime, got %v", m["mode"])
	}
}


// ---------------------------------------------------------------------------
// run helper-status
// ---------------------------------------------------------------------------

func makeRunStatusServer(running bool, helper bridge.RuntimeHelperStatus) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
			OK: true,
			Result: bridge.RunStatusResult{
				Running:       running,
				RuntimeHelper: helper,
			},
		})
	}))
}

func TestRunHelperStatusNotPresent(t *testing.T) {
	server := makeRunStatusServer(false, bridge.RuntimeHelperStatus{
		Present:            false,
		AutoloadConfigured: false,
		Path:               "/addons/runtime_helper/helper.gd",
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "helper-status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Runtime helper") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunHelperStatusPresent(t *testing.T) {
	server := makeRunStatusServer(true, bridge.RuntimeHelperStatus{
		Present:            true,
		AutoloadConfigured: true,
		Path:               "/addons/runtime_helper/helper.gd",
		LastSeen:           "2024-01-01T00:00:00Z",
		LastMessage:        "heartbeat",
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "helper-status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "present") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunHelperStatusJSON(t *testing.T) {
	server := makeRunStatusServer(false, bridge.RuntimeHelperStatus{Present: true})
	defer server.Close()
	out, err := runCmd(t, server, "run", "helper-status", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected JSON, got: %s", out)
	}
}

func TestRunHelperStatusWithError(t *testing.T) {
	server := makeRunStatusServer(false, bridge.RuntimeHelperStatus{
		Present:            false,
		AutoloadConfigured: true,
		Path:               "/addons/runtime_helper/helper.gd",
		Error:              "script not found",
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "helper-status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "not configured") && !strings.Contains(out, "configured") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// printRuntimeHelperSummary via run status
// ---------------------------------------------------------------------------

func TestRunStatusWithRuntimeHelper(t *testing.T) {
	server := makeRunStatusServer(true, bridge.RuntimeHelperStatus{
		Present:  true,
		LastSeen: "2024-01-01T00:00:00Z",
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "running") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStatusStoppedWithHelper(t *testing.T) {
	server := makeRunStatusServer(false, bridge.RuntimeHelperStatus{
		Present:  false,
		LastSeen: "2024-01-01T00:00:00Z",
		Error:    "timed out",
	})
	defer server.Close()
	out, err := runCmd(t, server, "run", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "stopped") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStatusWithHelperNoLastSeen(t *testing.T) {
	server := makeRunStatusServer(true, bridge.RuntimeHelperStatus{
		Present: true,
	})
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
// CSG operation variants
// ---------------------------------------------------------------------------

func makeCSGServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok200(w, map[string]any{})
	}))
}

func TestCSGOperationSetIntersection(t *testing.T) {
	server := makeCSGServer()
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "csg", "operation-set", "--path", "/root/CSG", "--operation", "intersection")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CSG operation set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestCSGOperationSetSubtraction(t *testing.T) {
	server := makeCSGServer()
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "csg", "operation-set", "--path", "/root/CSG", "--operation", "subtraction")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CSG operation set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestCSGOperationSetSubtract(t *testing.T) {
	server := makeCSGServer()
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "csg", "operation-set", "--path", "/root/CSG", "--operation", "subtract")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CSG operation set") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// intFromJobResult unit tests
// ---------------------------------------------------------------------------

func TestIntFromJobResult(t *testing.T) {
	cases := []struct {
		input any
		want  int
	}{
		{42, 42},
		{int64(99), 99},
		{float64(3.7), 3},
		{"not a number", 0},
		{nil, 0},
	}
	for _, tc := range cases {
		got := intFromJobResult(tc.input)
		if got != tc.want {
			t.Errorf("intFromJobResult(%v) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// yesNo / valueOrDash unit tests
// ---------------------------------------------------------------------------

func TestYesNo(t *testing.T) {
	if yesNo(true) != "yes" {
		t.Error("yesNo(true) should be yes")
	}
	if yesNo(false) != "no" {
		t.Error("yesNo(false) should be no")
	}
}

func TestValueOrDash(t *testing.T) {
	if valueOrDash("hello") != "hello" {
		t.Error("valueOrDash(hello) should return hello")
	}
	if valueOrDash("") != "-" {
		t.Error("valueOrDash('') should return -")
	}
}

// ---------------------------------------------------------------------------
// recipe dispatcher coverage (terrain/voxelgi/reflectionprobe/fogvolume/occluder unknown verb)
// ---------------------------------------------------------------------------

func TestTerrainUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "terrain", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown terrain") {
		t.Fatalf("expected unknown terrain error, got %v", err)
	}
}

func TestLightmapUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"lightmap", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown lightmap") {
		t.Fatalf("expected unknown lightmap error, got %v", err)
	}
}

func TestVoxelGIUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "voxelgi", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown voxelgi") {
		t.Fatalf("expected unknown voxelgi error, got %v", err)
	}
}

func TestReflectionProbeUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "reflection-probe", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown reflection-probe") {
		t.Fatalf("expected unknown reflection-probe error, got %v", err)
	}
}

func TestFogVolumeUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "fog-volume", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown fog-volume") {
		t.Fatalf("expected unknown fog-volume error, got %v", err)
	}
}

func TestOccluderUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "occluder", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown occluder") {
		t.Fatalf("expected unknown occluder error, got %v", err)
	}
}

func TestTerrainRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "terrain"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLightmapRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"lightmap"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVoxelGIRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "voxelgi"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReflectionProbeRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "reflection-probe"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFogVolumeRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "fog-volume"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOccluderRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "occluder"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Window dispatcher
// ---------------------------------------------------------------------------

func TestWindowRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"window"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWindowUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"window", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// addon commands coverage
// ---------------------------------------------------------------------------

func TestAddonDoctorWithProject(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:              true,
			PluginVersion:   "0.1.9",
			ProtocolVersion: "gdctl.v1",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	args := append(serverArgs(server), "addon", "doctor", "--project", project)
	_ = Run(context.Background(), args, &stdout, &stderr)
	out := stdout.String()
	if !strings.Contains(out, "gdctl Addon Doctor") {
		t.Fatalf("expected Addon Doctor header, got: %s", out)
	}
}

func TestAddonDoctorRuntimeMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{OK: true, PluginVersion: "0.1.9"})
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
// run status paused branch
// ---------------------------------------------------------------------------

func TestRunStatusPaused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
			OK: true,
			Result: bridge.RunStatusResult{
				Running:      true,
				PlayingScene: "res://main.tscn",
				Debugger: bridge.DebuggerState{
					Paused:  true,
					File:    "res://player.gd",
					Line:    42,
					Message: "breakpoint",
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

func TestRunStatusPausedNoLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
			OK: true,
			Result: bridge.RunStatusResult{
				Running:      true,
				PlayingScene: "res://main.tscn",
				Debugger: bridge.DebuggerState{
					Paused:  true,
					Message: "assert failed",
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

func TestRunStatusPausedOnlyScene(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
			OK: true,
			Result: bridge.RunStatusResult{
				Running:      true,
				PlayingScene: "res://main.tscn",
				Debugger: bridge.DebuggerState{
					Paused: true,
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
