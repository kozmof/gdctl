package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gdctl/internal/bridge"
)

// ok200 returns a minimal BridgeResponse with OK:true and the given result fields.
func ok200(w http.ResponseWriter, result map[string]any) {
	_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: result})
}

func singleHandler(path string, result map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			panic("unexpected path: " + r.URL.Path)
		}
		ok200(w, result)
	}))
}

func runCmd(t *testing.T, server *httptest.Server, args ...string) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), args...), &stdout, &stderr)
	return stdout.String(), err
}

// TestPingUnreachableGivesGuidance verifies that when the bridge is down, the
// CLI turns the raw socket error into actionable guidance while still matching
// bridge.ErrUnreachable in the wrapped chain.
func TestPingUnreachableGivesGuidance(t *testing.T) {
	// Start then immediately stop a server to get a port that refuses connections.
	server := singleHandler("/ping", map[string]any{})
	args := serverArgs(server)
	server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(args, "ping"), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error pinging a closed bridge")
	}
	if !strings.Contains(err.Error(), "gdctl doctor") {
		t.Fatalf("expected guidance mentioning 'gdctl doctor', got: %v", err)
	}
	if !errors.Is(err, bridge.ErrUnreachable) {
		t.Fatalf("expected wrapped error to match bridge.ErrUnreachable, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Accessibility commands
// ---------------------------------------------------------------------------

func TestAccessibilityTTSSpeak(t *testing.T) {
	// The bridge auto-selects a voice; the CLI must surface it rather than
	// only echoing the requested text.
	server := singleHandler("/accessibility/tts-speak", map[string]any{"voice": "en-US-1", "spoken": true})
	defer server.Close()
	out, err := runCmd(t, server, "accessibility", "tts-speak", "--text", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "TTS speak") || !strings.Contains(out, "en-US-1") {
		t.Fatalf("expected text and server-chosen voice in output, got: %s", out)
	}
}

func TestAccessibilityTTSSpeakRequiresText(t *testing.T) {
	err := Run(context.Background(), []string{"accessibility", "tts-speak"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--text") {
		t.Fatalf("expected --text error, got %v", err)
	}
}

func TestAccessibilityTTSConfigure(t *testing.T) {
	// The bridge echoes the effective configuration (filling in unchanged
	// values); the CLI must report what actually took effect.
	server := singleHandler("/accessibility/tts-configure", map[string]any{
		"pitch": 1.2, "rate": 1.0, "volume": 50.0, "voice": "en-US-1", "applied": true,
	})
	defer server.Close()
	out, err := runCmd(t, server, "accessibility", "tts-configure", "--pitch", "1.2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "TTS configured") || !strings.Contains(out, "pitch=1.20") || !strings.Contains(out, "voice=en-US-1") {
		t.Fatalf("expected effective config in output, got: %s", out)
	}
}

func TestAccessibilityTTSStop(t *testing.T) {
	server := singleHandler("/accessibility/tts-stop", map[string]any{"stopped": true})
	defer server.Close()
	out, err := runCmd(t, server, "accessibility", "tts-stop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "TTS stopped") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAccessibilityUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"accessibility", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// localization commands
// ---------------------------------------------------------------------------

func TestI18nLocaleSet(t *testing.T) {
	server := singleHandler("/i18n/locale-set", map[string]any{"locale": "ja"})
	defer server.Close()
	out, err := runCmd(t, server, "localization", "locale-set", "--locale", "ja")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Locale set: ja") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestI18nLocaleSetRequiresLocale(t *testing.T) {
	err := Run(context.Background(), []string{"localization", "locale-set"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--locale") {
		t.Fatalf("expected --locale error, got %v", err)
	}
}

func TestI18nStringAdd(t *testing.T) {
	server := singleHandler("/i18n/string-add", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "localization", "string-add", "--key", "GREET", "--locale", "en", "--text", "Hello")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Translation added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestI18nStringAddRequiresAllFlags(t *testing.T) {
	err := Run(context.Background(), []string{"localization", "string-add", "--key", "K"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Window commands
// ---------------------------------------------------------------------------

func TestWindowCreate(t *testing.T) {
	server := singleHandler("/window/create", map[string]any{"window_id": 1.0, "path": "/root/Window"})
	defer server.Close()
	out, err := runCmd(t, server, "window", "create", "--title", "Test", "--width", "800", "--height", "600")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Window created") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestWindowAssignViewport(t *testing.T) {
	server := singleHandler("/window/assign-viewport", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "window", "assign-viewport", "--window-id", "1", "--viewport", "/root/Main/Viewport")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Viewport assigned") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestWindowAssignViewportSurfacesServerNote(t *testing.T) {
	// A server-side caveat must reach the user instead of being swallowed.
	server := singleHandler("/window/assign-viewport", map[string]any{
		"assigned": true, "note": "window already had a viewport",
	})
	defer server.Close()
	out, err := runCmd(t, server, "window", "assign-viewport", "--window-id", "1", "--viewport", "/root/Main/Viewport")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "window already had a viewport") {
		t.Fatalf("expected server note in output, got: %s", out)
	}
}

func TestWindowAssignViewportRequiresViewport(t *testing.T) {
	err := Run(context.Background(), []string{"window", "assign-viewport"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--viewport") {
		t.Fatalf("expected --viewport error, got %v", err)
	}
}

func TestWindowCreateInvalidPosition(t *testing.T) {
	err := Run(context.Background(), []string{"--host", "127.0.0.1", "--port", "0", "window", "create", "--position", "bad"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Graph-edit commands (recipe layer)
// ---------------------------------------------------------------------------

func TestGraphEditNodeAdd(t *testing.T) {
	server := singleHandler("/graph-edit/node-add", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "graph-edit", "node-add", "--path", "/root/Graph", "--name", "MyNode")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GraphNode added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestGraphEditNodeAddRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "graph-edit", "node-add", "--path", "/root/Graph"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--path and --name") {
		t.Fatalf("expected path+name error, got %v", err)
	}
}

func TestGraphEditConnectionAdd(t *testing.T) {
	server := singleHandler("/graph-edit/connection-add", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "graph-edit", "connection-add", "--graph", "/root/Graph", "--from", "A", "--to", "B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "connection added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestGraphEditNodeRemove(t *testing.T) {
	server := singleHandler("/graph-edit/node-remove", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "graph-edit", "node-remove", "--path", "/root/Graph", "--name", "A")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GraphNode removed") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Animation tree commands
// ---------------------------------------------------------------------------

func TestAnimationTreeAddState(t *testing.T) {
	server := singleHandler("/animation-tree/add-state", map[string]any{"created": true})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "add-state", "--tree", "/root/AnimTree", "--name", "Idle", "--animation", "idle")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "state added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeAddStateAlreadyExists(t *testing.T) {
	server := singleHandler("/animation-tree/add-state", map[string]any{"created": false})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "add-state", "--tree", "/root/AnimTree", "--name", "Idle")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "already exists") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeAddTransition(t *testing.T) {
	server := singleHandler("/animation-tree/add-transition", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "add-transition", "--tree", "/root/AnimTree", "--from", "Idle", "--to", "Walk")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "transition added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeBlendSpace2DAdd(t *testing.T) {
	server := singleHandler("/animation-tree/blend-space-2d-add", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "blend-space-2d-add", "--tree", "/root/AnimTree", "--state", "Blend")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "BlendSpace2D added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeSetParamFloat(t *testing.T) {
	server := singleHandler("/animation-tree/set-param", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "set-param", "--tree", "/root/AnimTree", "--param", "parameters/blend", "--float", "0.5")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "param set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeSetParamVector2(t *testing.T) {
	server := singleHandler("/animation-tree/set-param", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "set-param", "--tree", "/root/AnimTree", "--param", "parameters/blend", "--vector2", "0.5,1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "param set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeSetParamBool(t *testing.T) {
	server := singleHandler("/animation-tree/set-param", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "set-param", "--tree", "/root/AnimTree", "--param", "parameters/active", "--bool")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "param set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeSetParamInt(t *testing.T) {
	server := singleHandler("/animation-tree/set-param", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "tree", "set-param", "--tree", "/root/AnimTree", "--param", "parameters/current", "--int", "2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "param set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationTreeSetParamRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"animation", "tree", "set-param"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Recipe: terrain, lightmap, voxelgi, reflection-probe, decal, fog-volume, occluder
// ---------------------------------------------------------------------------

func TestTerrainHeightmapImport(t *testing.T) {
	server := singleHandler("/terrain/heightmap-import", map[string]any{"width": 128.0, "height": 128.0})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "terrain", "heightmap-import", "--path", "/root/Terrain", "--texture", "res://hmap.png")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Terrain heightmap imported") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestTerrainHeightmapImportRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "terrain", "heightmap-import", "--path", "/root/T"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLightmapBake(t *testing.T) {
	server := singleHandler("/lightmap/bake", map[string]any{"status": "ok", "note": "done"})
	defer server.Close()
	out, err := runCmd(t, server, "lightmap", "bake", "--path", "/root/LightmapGI")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "LightmapGI bake") {
		t.Fatalf("stdout: %s", out)
	}
	if !strings.Contains(out, "Note:") {
		t.Fatalf("note not shown: %s", out)
	}
}

func TestLightmapBakeRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"lightmap", "bake"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--path") {
		t.Fatalf("expected --path error, got %v", err)
	}
}

func TestVoxelGIBake(t *testing.T) {
	server := singleHandler("/voxelgi/bake", map[string]any{"status": "ok"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "voxelgi", "bake", "--path", "/root/VoxelGI")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "VoxelGI bake") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestVoxelGIBakeRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "voxelgi", "bake"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReflectionProbeBake(t *testing.T) {
	server := singleHandler("/reflection-probe/bake", map[string]any{"status": "ok", "note": ""})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "reflection-probe", "bake", "--path", "/root/ReflectionProbe")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ReflectionProbe bake") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestReflectionProbeBakeRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "reflection-probe", "bake"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecalAdd(t *testing.T) {
	server := singleHandler("/decal/add", map[string]any{"path": "/root/Main/Decal"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "decal", "add", "--parent", "/root/Main", "--texture", "res://tex.png")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Decal added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestDecalAddRequiresParent(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "decal", "add"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecalSetNormalFade(t *testing.T) {
	server := singleHandler("/decal/set-normal-fade", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "decal", "set-normal-fade", "--path", "/root/Main/Decal", "--fade", "0.5")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "normal fade set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestDecalSetNormalFadeRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "decal", "set-normal-fade"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFogVolumeAdd(t *testing.T) {
	server := singleHandler("/fog-volume/add", map[string]any{"path": "/root/Main/FogVolume"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "fog-volume", "add", "--parent", "/root/Main")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "FogVolume added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestFogVolumeAddRequiresParent(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "fog-volume", "add"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOccluderAdd(t *testing.T) {
	server := singleHandler("/occluder/add", map[string]any{"path": "/root/Main/Occluder"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "occluder", "add", "--parent", "/root/Main")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "OccluderInstance3D added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestOccluderAddRequiresParent(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "occluder", "add"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// CSG commands (recipe layer)
// ---------------------------------------------------------------------------

func TestCSGNodeAdd(t *testing.T) {
	server := singleHandler("/node/add", map[string]any{"path": "/root/Main/CSGBox3D"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "csg", "node-add", "--parent", "/root/Main", "--name", "Box")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Added node") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestCSGNodeAddRequiresParent(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "csg", "node-add"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCSGOperationSet(t *testing.T) {
	server := singleHandler("/node/set", map[string]any{"path": "/root/Main/Box", "property": "operation"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "csg", "operation-set", "--path", "/root/Main/Box", "--operation", "union")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CSG operation set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestCSGOperationSetRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "csg", "operation-set"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCSGSizeSet(t *testing.T) {
	server := singleHandler("/node/set", map[string]any{"path": "/root/Main/Box", "property": "size"})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "csg", "size-set", "--path", "/root/Main/Box", "--size", "2,2,2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CSG size set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestCSGSizeSetRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "csg", "size-set"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Run: stop, status, instantiate, scene-reload
// ---------------------------------------------------------------------------

func makeJobServer(t *testing.T, actionPath string, actionResult map[string]any, jobKind string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case actionPath:
			ok200(w, map[string]any{"queued": true, "job_id": "j1"})
		case "/jobs/j1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "j1",
					Kind:   jobKind,
					Status: "succeeded",
					Result: actionResult,
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
}

func TestRunStop(t *testing.T) {
	server := singleHandler("/run/stop", map[string]any{"stopped": true, "running": false})
	defer server.Close()
	out, err := runCmd(t, server, "run", "stop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Run stopped") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStopNotRunning(t *testing.T) {
	server := singleHandler("/run/stop", map[string]any{"stopped": false, "running": false})
	defer server.Close()
	out, err := runCmd(t, server, "run", "stop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "not running") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStatus(t *testing.T) {
	server := singleHandler("/run/status", map[string]any{"running": true, "playing_scene": "res://main.tscn"})
	defer server.Close()
	out, err := runCmd(t, server, "run", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "running") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStatusStopped(t *testing.T) {
	server := singleHandler("/run/status", map[string]any{"running": false})
	defer server.Close()
	out, err := runCmd(t, server, "run", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "stopped") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunStatusJSON(t *testing.T) {
	server := singleHandler("/run/status", map[string]any{"running": false})
	defer server.Close()
	out, err := runCmd(t, server, "run", "status", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"running"`) {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunInstantiate(t *testing.T) {
	server := makeJobServer(t, "/run/instantiate", map[string]any{"path": "/root/Main/Enemy"}, "run.instantiate")
	defer server.Close()
	out, err := runCmd(t, server, "run", "instantiate", "--scene", "res://enemies/Enemy.tscn", "--parent", "/root/Main")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Instantiated") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunInstantiateRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"run", "instantiate", "--scene", "res://foo.tscn"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunSceneReload(t *testing.T) {
	server := makeJobServer(t, "/run/scene-reload", map[string]any{}, "run.scene-reload")
	defer server.Close()
	out, err := runCmd(t, server, "run", "scene-reload")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Scene reloaded") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Run profile
// ---------------------------------------------------------------------------

func TestRunProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/profile":
			ok200(w, map[string]any{"job_id": "pj1"})
		case "/jobs/pj1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID: "pj1", Kind: "run.profile", Status: "succeeded",
					Result: map[string]any{"sample_count": 10.0, "fps_avg": 60.0, "fps_min": 55.0},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	out, err := runCmd(t, server, "run", "profile", "--metric", "fps", "--duration", "100ms", "--timeout", "5s")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Profile result") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// parseWindowPos / parseVec3 helpers via command validation
// ---------------------------------------------------------------------------

func TestWindowCreateInvalidWindowPos(t *testing.T) {
	// needs a server but will fail at parseWindowPos before HTTP
	server := singleHandler("/window/create", map[string]any{})
	defer server.Close()
	_, err := runCmd(t, server, "window", "create", "--position", "notapos")
	if err == nil || !strings.Contains(err.Error(), "--position") {
		t.Fatalf("expected --position error, got %v", err)
	}
}

func TestDecalAddInvalidSize(t *testing.T) {
	server := singleHandler("/decal/add", map[string]any{})
	defer server.Close()
	_, err := runCmd(t, server, "recipe", "decal", "add", "--parent", "/root/Main", "--size", "1,2")
	if err == nil || !strings.Contains(err.Error(), "--size") {
		t.Fatalf("expected size error, got %v", err)
	}
}
