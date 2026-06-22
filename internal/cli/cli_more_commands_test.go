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
// LOD commands
// ---------------------------------------------------------------------------

func TestLODSet(t *testing.T) {
	server := singleHandler("/lod/set", map[string]any{"begin": 10.0, "end": 50.0})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "lod", "set", "--path", "/root/Mesh", "--begin", "10", "--end", "50")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "LOD set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestLODSetRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "lod", "set"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--path") {
		t.Fatalf("expected --path error, got %v", err)
	}
}

func TestLODSetMany(t *testing.T) {
	lodFile := filepath.Join(t.TempDir(), "lod.json")
	content := `[{"path":"/root/Mesh","begin":5,"end":50}]`
	if err := os.WriteFile(lodFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	server := singleHandler("/lod/set-many", map[string]any{"updated": 1.0})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "lod", "set-many", "--file", lodFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "LOD set-many: 1") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestLODSetManyRequiresFile(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "lod", "set-many"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// SoftBody commands
// ---------------------------------------------------------------------------

func TestSoftBodyPinPoint(t *testing.T) {
	server := singleHandler("/softbody/pin-point", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "softbody", "pin-point", "--path", "/root/SoftBody3D", "--point", "0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "pinned") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestSoftBodyUnpinPoint(t *testing.T) {
	server := singleHandler("/softbody/unpin-point", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "recipe", "softbody", "unpin-point", "--path", "/root/SoftBody3D", "--point", "0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unpinned") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestSoftBodyRequiresSubcommand(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "softbody"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSoftBodyPinRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"recipe", "softbody", "pin-point", "--path", "/root/Body"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Animation: length-set
// ---------------------------------------------------------------------------

func TestAnimationLengthSet(t *testing.T) {
	server := singleHandler("/animation/length-set", map[string]any{"path": "res://animations.tres", "name": "walk"})
	defer server.Close()
	out, err := runCmd(t, server, "animation", "length-set", "--path", "res://animations.tres", "--animation", "walk", "--length", "2.5")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Animation length set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAnimationLengthSetRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"animation", "length-set", "--path", "res://a.tres", "--animation", "walk"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--length") {
		t.Fatalf("expected --length error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Viewport: camera-assign
// ---------------------------------------------------------------------------

func TestViewportCameraAssign(t *testing.T) {
	server := singleHandler("/viewport/camera-assign", map[string]any{"camera": "/root/Camera", "viewport": "/root/Viewport"})
	defer server.Close()
	out, err := runCmd(t, server, "viewport", "camera-assign", "--viewport", "/root/Viewport", "--camera", "/root/Camera")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Camera assigned") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestViewportCameraAssignRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"viewport", "camera-assign", "--viewport", "/root/V"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Audio: listener-make-current, playlist-add, playlist-autoplay
// ---------------------------------------------------------------------------

func TestAudioListenerMakeCurrent(t *testing.T) {
	server := singleHandler("/audio/listener-make-current", map[string]any{"path": "/root/Listener", "type": "AudioListener3D"})
	defer server.Close()
	out, err := runCmd(t, server, "audio", "listener-make-current", "--path", "/root/Listener")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Audio listener active") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAudioListenerMakeCurrentRequiresPath(t *testing.T) {
	err := Run(context.Background(), []string{"audio", "listener-make-current"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAudioPlaylistAdd(t *testing.T) {
	server := singleHandler("/audio/playlist-add", map[string]any{"stream_count": 2.0})
	defer server.Close()
	out, err := runCmd(t, server, "audio", "playlist-add", "--bus", "Music", "--stream", "res://music/theme.ogg")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "playlist stream added") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAudioPlaylistAddRequiresFlags(t *testing.T) {
	err := Run(context.Background(), []string{"audio", "playlist-add", "--bus", "Music"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAudioPlaylistAutoplay(t *testing.T) {
	server := singleHandler("/audio/playlist-autoplay", map[string]any{})
	defer server.Close()
	out, err := runCmd(t, server, "audio", "playlist-autoplay", "--bus", "Music", "--mode", "sequential")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "playlist autoplay set") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAudioPlaylistAutoplayRequiresBus(t *testing.T) {
	err := Run(context.Background(), []string{"audio", "playlist-autoplay"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Autoload: remove
// ---------------------------------------------------------------------------

func TestAutoloadRemove(t *testing.T) {
	server := singleHandler("/autoload/remove", map[string]any{"name": "GameManager"})
	defer server.Close()
	out, err := runCmd(t, server, "autoload", "remove", "--name", "GameManager")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Autoload removed") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAutoloadRemoveRequiresName(t *testing.T) {
	err := Run(context.Background(), []string{"autoload", "remove"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// Input: action remove, action list, event add-joypad
// ---------------------------------------------------------------------------

func TestInputActionRemove(t *testing.T) {
	server := singleHandler("/input/action-remove", map[string]any{"action": "jump"})
	defer server.Close()
	out, err := runCmd(t, server, "input", "action", "remove", "--name", "jump")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Input action removed") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestInputActionRemoveRequiresName(t *testing.T) {
	err := Run(context.Background(), []string{"input", "action", "remove"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInputActionListEmpty(t *testing.T) {
	server := singleHandler("/input/action-list", map[string]any{"actions": []any{}})
	defer server.Close()
	out, err := runCmd(t, server, "input", "action", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No input actions") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestInputActionListWithActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.InputActionListResult]{
			OK: true,
			Result: bridge.InputActionListResult{
				Actions: []bridge.InputActionResult{
					{Action: "jump", Deadzone: 0.5, Events: []bridge.InputEventInfo{{Key: "Space"}}},
				},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "input", "action", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "jump") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestInputEventAddJoypad(t *testing.T) {
	server := singleHandler("/input/event-add-joypad", map[string]any{"action": "dash", "event_added": true})
	defer server.Close()
	out, err := runCmd(t, server, "input", "event", "add-joypad", "--action", "dash", "--button", "0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dash") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestInputEventAddJoypadRequiresButtonOrAxis(t *testing.T) {
	err := Run(context.Background(), []string{"input", "event", "add-joypad", "--action", "dash"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--button or --axis") {
		t.Fatalf("expected button/axis error, got %v", err)
	}
}
