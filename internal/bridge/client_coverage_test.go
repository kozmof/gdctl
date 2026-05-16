package bridge

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// okJSON writes a minimal successful BridgeResponse[map[string]any] with an optional
// result payload merged on top of the empty map.
func okJSON(w http.ResponseWriter, result map[string]any) {
	if result == nil {
		result = map[string]any{}
	}
	_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{OK: true, Result: result})
}

// TestClientCoverage exercises every 0%-covered client method through a single
// httptest server that routes on r.URL.Path.
func TestClientCoverage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// --- client_assets.go ---
		case "/shader/check":
			okJSON(w, map[string]any{"path": "res://s.gdshader", "valid": true})
		case "/shader/write":
			okJSON(w, map[string]any{"path": "res://s.gdshader", "valid": true, "written": true})
		case "/resource/create":
			okJSON(w, map[string]any{"path": "res://r.tres", "type": "Resource", "created": true})
		case "/file/write-bytes":
			okJSON(w, map[string]any{"path": "res://f.bin", "bytes": 4, "written": true})
		case "/file/list":
			okJSON(w, map[string]any{"path": "res://", "files": []string{"a.gd"}, "dirs": []string{}})
		case "/file/mkdir":
			okJSON(w, map[string]any{"path": "res://new", "created": true})
		case "/file/delete":
			okJSON(w, map[string]any{"path": "res://old", "deleted": true})
		case "/file/exists":
			okJSON(w, map[string]any{"path": "res://f.gd", "exists": true, "is_file": true})
		case "/import/set":
			okJSON(w, map[string]any{"path": "res://img.png", "params": 1, "applied": true})
		case "/resource/list":
			okJSON(w, map[string]any{"dir": "res://", "resources": []string{}})
		case "/navigation/bake":
			okJSON(w, map[string]any{"path": "/root/Nav", "kind": "NavigationRegion3D", "baked": true})

		// --- client_media.go ---
		case "/run/probe/raycast":
			okJSON(w, map[string]any{"hit": false})
		case "/theme/create":
			okJSON(w, map[string]any{"path": "res://theme.tres", "created": true})
		case "/theme/set-color":
			okJSON(w, map[string]any{"path": "res://theme.tres", "set": true})
		case "/theme/set-font-size":
			okJSON(w, map[string]any{"path": "res://theme.tres", "set": true})
		case "/theme/set-constant":
			okJSON(w, map[string]any{"path": "res://theme.tres", "set": true})
		case "/animation/create":
			okJSON(w, map[string]any{"path": "res://anim.tres", "name": "Idle", "created": true})
		case "/animation/track-add":
			okJSON(w, map[string]any{"path": "res://anim.tres", "animation": "Idle", "track_idx": 0})
		case "/animation/keyframe-add":
			okJSON(w, map[string]any{"path": "res://anim.tres", "animation": "Idle", "track_idx": 0, "time": 0.0, "added": true})
		case "/animation/length-set":
			okJSON(w, map[string]any{"path": "res://anim.tres", "name": "Idle", "created": false})
		case "/animation/player-play":
			okJSON(w, map[string]any{"name": "Idle"})
		case "/tilemap/tileset-create":
			okJSON(w, map[string]any{"path": "res://ts.tres", "created": true})
		case "/tilemap/source-add":
			okJSON(w, map[string]any{"path": "res://ts.tres", "created": false})
		case "/tilemap/cell-set":
			okJSON(w, map[string]any{"node": "/root/TileMap", "layer": 0, "x": 1, "y": 2, "applied": true})
		case "/tilemap/cell-set-rect":
			okJSON(w, map[string]any{"node": "/root/TileMap", "layer": 0, "x": 0, "y": 0, "width": 3, "height": 3, "cells": 9, "applied": true})
		case "/tilemap/cell-clear":
			okJSON(w, map[string]any{"node": "/root/TileMap", "layer": 0, "x": 1, "y": 2, "applied": true})
		case "/audio/bus-add":
			okJSON(w, map[string]any{"bus": "Music", "created": true})
		case "/audio/bus-volume-set":
			okJSON(w, map[string]any{"bus": "Music", "applied": true})
		case "/audio/bus-effect-add":
			okJSON(w, map[string]any{"bus": "Music", "applied": true})
		case "/viewport/set-size":
			okJSON(w, map[string]any{"width": 1920, "height": 1080})
		case "/viewport/camera-assign":
			okJSON(w, map[string]any{"viewport": "/root/VP", "camera": "/root/Cam", "applied": true})
		case "/audio/listener-make-current":
			okJSON(w, map[string]any{"path": "/root/Listener", "applied": true})
		case "/viewport/add":
			okJSON(w, map[string]any{"path": "/root/VP", "width": 800, "height": 600, "added": true})

		// --- client_project.go ---
		case "/autoload/add":
			okJSON(w, map[string]any{"name": "Global", "path": "res://global.gd", "added": true})
		case "/autoload/remove":
			okJSON(w, map[string]any{"name": "Global", "removed": true})
		case "/autoload/list":
			okJSON(w, map[string]any{"autoloads": []any{}})
		case "/input/action-add":
			okJSON(w, map[string]any{"action": "jump", "added": true})
		case "/input/action-remove":
			okJSON(w, map[string]any{"action": "jump", "removed": true})
		case "/input/action-list":
			okJSON(w, map[string]any{"actions": []any{}})
		case "/input/event-add-key":
			okJSON(w, map[string]any{"action": "jump", "key": "Space", "event_added": true})
		case "/input/event-add-joypad":
			okJSON(w, map[string]any{"action": "jump", "event_added": true})
		case "/signal/connect":
			okJSON(w, map[string]any{"from": "/root/A", "signal": "pressed", "to": "/root/B", "method": "_on_pressed", "connected": true})
		case "/signal/disconnect":
			okJSON(w, map[string]any{"from": "/root/A", "signal": "pressed", "to": "/root/B", "method": "_on_pressed", "disconnected": true})
		case "/project/setting-get":
			okJSON(w, map[string]any{"key": "application/config/name", "value": "MyGame"})
		case "/project/setting-set":
			okJSON(w, map[string]any{"key": "application/config/name", "set": true})

		// --- client_new.go ---
		case "/animation-tree/add-state":
			okJSON(w, map[string]any{"state": "Idle"})
		case "/animation-tree/add-transition":
			okJSON(w, map[string]any{"from": "Idle", "to": "Run"})
		case "/animation-tree/blend-space-2d-add":
			okJSON(w, map[string]any{"state": "Walk"})
		case "/animation-tree/set-param":
			okJSON(w, map[string]any{"param": "blend_position"})
		case "/softbody/pin-point":
			okJSON(w, map[string]any{"path": "/root/SoftBody", "point": 0})
		case "/softbody/unpin-point":
			okJSON(w, map[string]any{"path": "/root/SoftBody", "point": 0})
		case "/lod/set":
			okJSON(w, map[string]any{"path": "/root/Mesh"})
		case "/lod/set-many":
			okJSON(w, map[string]any{"count": 1})
		case "/run/profile":
			_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"job_id": "profile-1"},
			})
		case "/terrain/heightmap-import":
			okJSON(w, map[string]any{"path": "/root/Terrain"})
		case "/lightmap/bake":
			okJSON(w, map[string]any{"path": "/root/LightmapGI"})
		case "/voxelgi/bake":
			okJSON(w, map[string]any{"path": "/root/VoxelGI"})
		case "/reflection-probe/bake":
			okJSON(w, map[string]any{"path": "/root/ReflectionProbe"})
		case "/window/create":
			okJSON(w, map[string]any{"title": "Test"})
		case "/window/assign-viewport":
			okJSON(w, map[string]any{"window_id": 1})
		case "/graph-edit/node-add":
			okJSON(w, map[string]any{"name": "MyNode"})
		case "/graph-edit/connection-add":
			okJSON(w, map[string]any{"from": "A", "to": "B"})
		case "/graph-edit/node-remove":
			okJSON(w, map[string]any{"name": "MyNode"})
		case "/accessibility/tts-speak":
			okJSON(w, map[string]any{})
		case "/accessibility/tts-configure":
			okJSON(w, map[string]any{})
		case "/accessibility/tts-stop":
			okJSON(w, map[string]any{})
		case "/i18n/locale-set":
			okJSON(w, map[string]any{"locale": "en"})
		case "/i18n/string-add":
			okJSON(w, map[string]any{"key": "hello"})
		case "/audio/playlist-add":
			okJSON(w, map[string]any{"bus": "Music"})
		case "/audio/playlist-autoplay":
			okJSON(w, map[string]any{"bus": "Music"})
		case "/decal/add":
			okJSON(w, map[string]any{"parent": "/root"})
		case "/decal/set-normal-fade":
			okJSON(w, map[string]any{"path": "/root/Decal"})
		case "/fog-volume/add":
			okJSON(w, map[string]any{"parent": "/root"})
		case "/occluder/add":
			okJSON(w, map[string]any{"parent": "/root"})

		// --- client_run.go ---
		case "/ping":
			_ = json.NewEncoder(w).Encode(PingResponse{OK: true, Service: "bridge", Engine: "Godot"})
		case "/run/status":
			okJSON(w, map[string]any{"running": true})
		case "/run/stop":
			okJSON(w, map[string]any{"stopped": true, "running": false})
		case "/scene/tree":
			_ = json.NewEncoder(w).Encode(SceneTreeResponse{OK: true, Root: NodeInfo{Name: "root", Type: "Node"}})
		case "/run/instantiate":
			okJSON(w, map[string]any{"queued": true, "job_id": "inst-1"})
		case "/run/scene-reload":
			okJSON(w, map[string]any{"queued": true, "job_id": "reload-1"})

		// --- client_scene.go ---
		case "/scene/apply/blueprint":
			okJSON(w, map[string]any{"path": "res://main.tscn", "created": 1})
		case "/scene/list":
			okJSON(w, map[string]any{"dir": "res://", "scenes": []string{"main.tscn"}})

		// --- client_node.go ---
		case "/node/set-many":
			okJSON(w, map[string]any{"path": "/root/Node", "updated": 2})
		case "/node/set-resource":
			okJSON(w, map[string]any{"path": "/root/Node", "property": "mesh", "resource": "res://mesh.tres", "set": true})
		case "/node/group-add":
			okJSON(w, map[string]any{"path": "/root/Node", "group": "enemies", "added": true})
		case "/node/group-remove":
			okJSON(w, map[string]any{"path": "/root/Node", "group": "enemies", "removed": true})
		case "/node/group-list":
			okJSON(w, map[string]any{"path": "/root/Node", "groups": []string{"enemies"}})

		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	ctx := context.Background()

	// --- client_assets.go ---

	if _, err := client.CheckShader(ctx, "r1", "res://s.gdshader"); err != nil {
		t.Fatalf("CheckShader: %v", err)
	}
	if _, err := client.WriteShader(ctx, "r2", "res://s.gdshader", "shader_type spatial;"); err != nil {
		t.Fatalf("WriteShader: %v", err)
	}
	if _, err := client.CreateResource(ctx, "r3", "res://r.tres", "Resource", "", nil, nil); err != nil {
		t.Fatalf("CreateResource: %v", err)
	}
	if _, err := client.WriteFileBytes(ctx, "r4", "res://f.bin", "AAAA"); err != nil {
		t.Fatalf("WriteFileBytes: %v", err)
	}
	if _, err := client.FileList(ctx, "r5", "res://", false); err != nil {
		t.Fatalf("FileList: %v", err)
	}
	if _, err := client.FileMkdir(ctx, "r6", "res://new"); err != nil {
		t.Fatalf("FileMkdir: %v", err)
	}
	if _, err := client.FileDelete(ctx, "r7", "res://old"); err != nil {
		t.Fatalf("FileDelete: %v", err)
	}
	if _, err := client.FileExists(ctx, "r8", "res://f.gd"); err != nil {
		t.Fatalf("FileExists: %v", err)
	}
	if _, err := client.ImportSet(ctx, "r9", "res://img.png", map[string]any{"compress/mode": 0}); err != nil {
		t.Fatalf("ImportSet: %v", err)
	}
	if _, err := client.ResourceList(ctx, "r10", "res://", "tres", false); err != nil {
		t.Fatalf("ResourceList: %v", err)
	}
	if _, err := client.NavigationBake(ctx, "r11", "/root/Nav"); err != nil {
		t.Fatalf("NavigationBake: %v", err)
	}

	// --- client_media.go ---

	if _, err := client.RunRaycast(ctx, "r12"); err != nil {
		t.Fatalf("RunRaycast: %v", err)
	}
	if _, err := client.ThemeCreate(ctx, "r13", "res://theme.tres", false); err != nil {
		t.Fatalf("ThemeCreate: %v", err)
	}
	if _, err := client.ThemeSetColor(ctx, "r14", "res://theme.tres", "Button", "font_color", [4]float64{1, 0, 0, 1}); err != nil {
		t.Fatalf("ThemeSetColor: %v", err)
	}
	if _, err := client.ThemeSetFontSize(ctx, "r15", "res://theme.tres", "Button", "font_size", 16); err != nil {
		t.Fatalf("ThemeSetFontSize: %v", err)
	}
	if _, err := client.ThemeSetConstant(ctx, "r16", "res://theme.tres", "Button", "h_separation", 4); err != nil {
		t.Fatalf("ThemeSetConstant: %v", err)
	}
	if _, err := client.AnimationCreate(ctx, "r17", "res://anim.tres", "Idle", 1.0, false); err != nil {
		t.Fatalf("AnimationCreate: %v", err)
	}
	if _, err := client.AnimationTrackAdd(ctx, "r18", "res://anim.tres", "Idle", "/root/Sprite", "position"); err != nil {
		t.Fatalf("AnimationTrackAdd: %v", err)
	}
	if _, err := client.AnimationKeyframeAdd(ctx, "r19", "res://anim.tres", "Idle", 0, 0.0, map[string]any{"x": 0, "y": 0}); err != nil {
		t.Fatalf("AnimationKeyframeAdd: %v", err)
	}
	if _, err := client.AnimationLengthSet(ctx, "r20", "res://anim.tres", "Idle", 2.0); err != nil {
		t.Fatalf("AnimationLengthSet: %v", err)
	}
	if _, err := client.AnimationPlayerPlay(ctx, "r21", "/root/AnimationPlayer", "Idle"); err != nil {
		t.Fatalf("AnimationPlayerPlay: %v", err)
	}
	if _, err := client.TilesetCreate(ctx, "r22", "res://ts.tres", 16, 16); err != nil {
		t.Fatalf("TilesetCreate: %v", err)
	}
	if _, err := client.TilesetSourceAdd(ctx, "r23", "res://ts.tres", "res://tiles.png", 16, 16); err != nil {
		t.Fatalf("TilesetSourceAdd: %v", err)
	}
	if _, err := client.TilemapCellSet(ctx, "r24", "/root/TileMap", 0, 1, 2, 0, 0, 0); err != nil {
		t.Fatalf("TilemapCellSet: %v", err)
	}
	if _, err := client.TilemapCellSetRect(ctx, "r25", "/root/TileMap", 0, 0, 0, 3, 3, 0, 0, 0); err != nil {
		t.Fatalf("TilemapCellSetRect: %v", err)
	}
	if _, err := client.TilemapCellClear(ctx, "r26", "/root/TileMap", 0, 1, 2); err != nil {
		t.Fatalf("TilemapCellClear: %v", err)
	}
	if _, err := client.AudioBusAdd(ctx, "r27", "Music", true); err != nil {
		t.Fatalf("AudioBusAdd: %v", err)
	}
	if _, err := client.AudioBusVolumeSet(ctx, "r28", "Music", -6.0); err != nil {
		t.Fatalf("AudioBusVolumeSet: %v", err)
	}
	if _, err := client.AudioBusEffectAdd(ctx, "r29", "Music", "AudioEffectReverb"); err != nil {
		t.Fatalf("AudioBusEffectAdd: %v", err)
	}
	if _, err := client.ViewportSetSize(ctx, "r30", 1920, 1080, ""); err != nil {
		t.Fatalf("ViewportSetSize: %v", err)
	}
	if _, err := client.ViewportCameraAssign(ctx, "r31", "/root/VP", "/root/Cam"); err != nil {
		t.Fatalf("ViewportCameraAssign: %v", err)
	}
	if _, err := client.AudioListenerMakeCurrent(ctx, "r32", "/root/Listener"); err != nil {
		t.Fatalf("AudioListenerMakeCurrent: %v", err)
	}
	if _, err := client.ViewportAdd(ctx, "r33", "/root", 800, 600, false); err != nil {
		t.Fatalf("ViewportAdd: %v", err)
	}

	// --- client_project.go ---

	if _, err := client.AutoloadAdd(ctx, "r34", "Global", "res://global.gd"); err != nil {
		t.Fatalf("AutoloadAdd: %v", err)
	}
	if _, err := client.AutoloadRemove(ctx, "r35", "Global"); err != nil {
		t.Fatalf("AutoloadRemove: %v", err)
	}
	if _, err := client.AutoloadList(ctx, "r36"); err != nil {
		t.Fatalf("AutoloadList: %v", err)
	}
	if _, err := client.InputActionAdd(ctx, "r37", "jump", 0.5); err != nil {
		t.Fatalf("InputActionAdd: %v", err)
	}
	if _, err := client.InputActionRemove(ctx, "r38", "jump"); err != nil {
		t.Fatalf("InputActionRemove: %v", err)
	}
	if _, err := client.InputActionList(ctx, "r39", false); err != nil {
		t.Fatalf("InputActionList: %v", err)
	}
	if _, err := client.InputEventAddKey(ctx, "r40", "jump", "Space", false); err != nil {
		t.Fatalf("InputEventAddKey: %v", err)
	}
	if _, err := client.InputEventAddJoypad(ctx, "r41", "jump", 0, 0, 0.5, 0); err != nil {
		t.Fatalf("InputEventAddJoypad: %v", err)
	}
	if _, err := client.SignalConnect(ctx, "r42", "/root/A", "pressed", "/root/B", "_on_pressed"); err != nil {
		t.Fatalf("SignalConnect: %v", err)
	}
	if _, err := client.SignalDisconnect(ctx, "r43", "/root/A", "pressed", "/root/B", "_on_pressed"); err != nil {
		t.Fatalf("SignalDisconnect: %v", err)
	}
	if _, err := client.ProjectSettingGet(ctx, "r44", "application/config/name"); err != nil {
		t.Fatalf("ProjectSettingGet: %v", err)
	}
	if _, err := client.ProjectSettingSet(ctx, "r45", "application/config/name", "MyGame"); err != nil {
		t.Fatalf("ProjectSettingSet: %v", err)
	}

	// --- client_new.go ---

	if _, err := client.AnimationTreeAddState(ctx, "r46", "/root/AnimTree", "Idle", "Idle"); err != nil {
		t.Fatalf("AnimationTreeAddState: %v", err)
	}
	if _, err := client.AnimationTreeAddTransition(ctx, "r47", "/root/AnimTree", "Idle", "Run", "is_running"); err != nil {
		t.Fatalf("AnimationTreeAddTransition: %v", err)
	}
	if _, err := client.AnimationTreeBlendSpace2DAdd(ctx, "r48", "/root/AnimTree", "Walk", "blend_x", "blend_y"); err != nil {
		t.Fatalf("AnimationTreeBlendSpace2DAdd: %v", err)
	}
	if _, err := client.AnimationTreeSetParam(ctx, "r49", "/root/AnimTree", "conditions/is_running", "value", true); err != nil {
		t.Fatalf("AnimationTreeSetParam: %v", err)
	}
	if _, err := client.SoftBodyPinPoint(ctx, "r50", "/root/SoftBody", 0); err != nil {
		t.Fatalf("SoftBodyPinPoint: %v", err)
	}
	if _, err := client.SoftBodyUnpinPoint(ctx, "r51", "/root/SoftBody", 0); err != nil {
		t.Fatalf("SoftBodyUnpinPoint: %v", err)
	}
	if _, err := client.LodSet(ctx, "r52", "/root/Mesh", 0.0, 100.0); err != nil {
		t.Fatalf("LodSet: %v", err)
	}
	if _, err := client.LodSetMany(ctx, "r53", []LodEntry{{Path: "/root/Mesh", Begin: 0, End: 100}}); err != nil {
		t.Fatalf("LodSetMany: %v", err)
	}
	profileResult, err := client.RunProfile(ctx, "r54", []string{"fps", "memory"}, 1000.0)
	if err != nil {
		t.Fatalf("RunProfile: %v", err)
	}
	if profileResult.JobID != "profile-1" {
		t.Fatalf("RunProfile JobID = %q, want profile-1", profileResult.JobID)
	}
	if _, err := client.TerrainHeightmapImport(ctx, "r55", "/root/Terrain", "res://hm.png", 0.0, 100.0); err != nil {
		t.Fatalf("TerrainHeightmapImport: %v", err)
	}
	if _, err := client.LightmapBake(ctx, "r56", "/root/LightmapGI"); err != nil {
		t.Fatalf("LightmapBake: %v", err)
	}
	if _, err := client.VoxelGIBake(ctx, "r57", "/root/VoxelGI"); err != nil {
		t.Fatalf("VoxelGIBake: %v", err)
	}
	if _, err := client.ReflectionProbeBake(ctx, "r58", "/root/ReflectionProbe"); err != nil {
		t.Fatalf("ReflectionProbeBake: %v", err)
	}
	if _, err := client.WindowCreate(ctx, "r59", "Test", 800, 600, 0, 0); err != nil {
		t.Fatalf("WindowCreate: %v", err)
	}
	if _, err := client.WindowAssignViewport(ctx, "r60", 1, "/root/VP"); err != nil {
		t.Fatalf("WindowAssignViewport: %v", err)
	}
	if _, err := client.GraphEditNodeAdd(ctx, "r61", "/root/GraphEdit", "MyNode", 100.0, 200.0); err != nil {
		t.Fatalf("GraphEditNodeAdd: %v", err)
	}
	if _, err := client.GraphEditConnectionAdd(ctx, "r62", "/root/GraphEdit", "A", 0, "B", 0); err != nil {
		t.Fatalf("GraphEditConnectionAdd: %v", err)
	}
	if _, err := client.GraphEditNodeRemove(ctx, "r63", "/root/GraphEdit", "MyNode"); err != nil {
		t.Fatalf("GraphEditNodeRemove: %v", err)
	}
	if _, err := client.AccessibilityTTSSpeak(ctx, "r64", "Hello world", false); err != nil {
		t.Fatalf("AccessibilityTTSSpeak: %v", err)
	}
	// Test AccessibilityTTSConfigure with non-zero params (exercises both branches)
	if _, err := client.AccessibilityTTSConfigure(ctx, "r65", 1.2, 1.0, "en-US"); err != nil {
		t.Fatalf("AccessibilityTTSConfigure (non-zero): %v", err)
	}
	// Test AccessibilityTTSConfigure with zero params (exercises the zero-skip branches)
	if _, err := client.AccessibilityTTSConfigure(ctx, "r65b", 0, 0, ""); err != nil {
		t.Fatalf("AccessibilityTTSConfigure (zeros): %v", err)
	}
	if _, err := client.AccessibilityTTSStop(ctx, "r66"); err != nil {
		t.Fatalf("AccessibilityTTSStop: %v", err)
	}
	if _, err := client.I18nLocaleSet(ctx, "r67", "en"); err != nil {
		t.Fatalf("I18nLocaleSet: %v", err)
	}
	if _, err := client.I18nStringAdd(ctx, "r68", "hello", "en", "Hello"); err != nil {
		t.Fatalf("I18nStringAdd: %v", err)
	}
	if _, err := client.AudioPlaylistAdd(ctx, "r69", "Music", "res://song.ogg"); err != nil {
		t.Fatalf("AudioPlaylistAdd: %v", err)
	}
	if _, err := client.AudioPlaylistAutoplay(ctx, "r70", "Music", "sequential"); err != nil {
		t.Fatalf("AudioPlaylistAutoplay: %v", err)
	}
	if _, err := client.DecalAdd(ctx, "r71", "/root", "res://decal.png", [3]float64{1, 1, 1}); err != nil {
		t.Fatalf("DecalAdd: %v", err)
	}
	if _, err := client.DecalSetNormalFade(ctx, "r72", "/root/Decal", 0.5); err != nil {
		t.Fatalf("DecalSetNormalFade: %v", err)
	}
	if _, err := client.FogVolumeAdd(ctx, "r73", "/root", "box", [3]float64{2, 2, 2}, 0.1); err != nil {
		t.Fatalf("FogVolumeAdd: %v", err)
	}
	if _, err := client.OccluderAdd(ctx, "r74", "/root", "box", [3]float64{1, 1, 1}); err != nil {
		t.Fatalf("OccluderAdd: %v", err)
	}

	// --- client_run.go ---

	pingResp, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !pingResp.OK {
		t.Fatalf("Ping OK = false")
	}
	if _, err := client.RunStatus(ctx, "r75"); err != nil {
		t.Fatalf("RunStatus: %v", err)
	}
	if _, err := client.RunStop(ctx, "r76"); err != nil {
		t.Fatalf("RunStop: %v", err)
	}
	root, err := client.SceneTree(ctx)
	if err != nil {
		t.Fatalf("SceneTree: %v", err)
	}
	if root.Name != "root" {
		t.Fatalf("SceneTree root.Name = %q, want root", root.Name)
	}
	if _, err := client.RunInstantiate(ctx, "r77", "res://enemy.tscn", "/root/Main", "Enemy"); err != nil {
		t.Fatalf("RunInstantiate: %v", err)
	}
	if _, err := client.RunSceneReload(ctx, "r78"); err != nil {
		t.Fatalf("RunSceneReload: %v", err)
	}

	// --- client_scene.go ---

	if _, err := client.ApplyBlueprint(ctx, "r79", "res://main.tscn", "platformer", map[string]any{"gravity": 9.8}, false); err != nil {
		t.Fatalf("ApplyBlueprint: %v", err)
	}
	if _, err := client.SceneList(ctx, "r80", "res://", true); err != nil {
		t.Fatalf("SceneList: %v", err)
	}

	// --- client_node.go ---

	if _, err := client.SetNodeProperties(ctx, "r81", "/root/Node", map[string]any{"visible": true, "scale": 2.0}); err != nil {
		t.Fatalf("SetNodeProperties: %v", err)
	}
	if _, err := client.SetNodeResource(ctx, "r82", "/root/Node", "mesh", "res://mesh.tres"); err != nil {
		t.Fatalf("SetNodeResource: %v", err)
	}
	if _, err := client.NodeGroupAdd(ctx, "r83", "/root/Node", "enemies"); err != nil {
		t.Fatalf("NodeGroupAdd: %v", err)
	}
	if _, err := client.NodeGroupRemove(ctx, "r84", "/root/Node", "enemies"); err != nil {
		t.Fatalf("NodeGroupRemove: %v", err)
	}
	if _, err := client.NodeGroupList(ctx, "r85", "/root/Node"); err != nil {
		t.Fatalf("NodeGroupList: %v", err)
	}
}

// TestNewClientWithHTTP verifies that NewClientWithHTTP stores the provided http.Client.
func TestNewClientWithHTTP(t *testing.T) {
	httpClient := &http.Client{Timeout: 42 * time.Second}
	cfg := Config{Host: "127.0.0.1:7777", Protocol: "http"}
	client := NewClientWithHTTP(cfg, httpClient)
	if client == nil {
		t.Fatal("NewClientWithHTTP returned nil")
	}
	if client.httpClient != httpClient {
		t.Fatalf("httpClient not stored: got %p, want %p", client.httpClient, httpClient)
	}
	if client.cfg.Host != "127.0.0.1:7777" {
		t.Fatalf("cfg.Host = %q", client.cfg.Host)
	}
}

// TestDial verifies Dial succeeds when a TCP listener is available and fails
// when no server is listening.
func TestDial(t *testing.T) {
	// Start a temporary TCP listener to accept the connection.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	cfg := Config{Host: addr, Protocol: "http"}
	client := NewClient(cfg)

	if err := client.Dial(context.Background()); err != nil {
		t.Fatalf("Dial succeeded expected: %v", err)
	}

	// Now close the listener and verify Dial fails.
	ln.Close()
	cfg2 := Config{Host: "127.0.0.1:1", Protocol: "http"}
	client2 := NewClient(cfg2)
	if err := client2.Dial(context.Background()); err == nil {
		t.Fatal("Dial should fail when nothing is listening")
	}
}

// TestConfigAddress verifies Config.Address returns the expected host:port string.
func TestConfigAddress(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"127.0.0.1", 7777, "127.0.0.1:7777"},
		{"::1", 8080, "[::1]:8080"},
		// When Host already contains a port, it is returned as-is.
		{"127.0.0.1:9999", 0, "127.0.0.1:9999"},
	}
	for _, tc := range tests {
		cfg := Config{Host: tc.host, Port: tc.port}
		got := cfg.Address()
		if got != tc.want {
			t.Errorf("Config{Host:%q, Port:%d}.Address() = %q, want %q", tc.host, tc.port, got, tc.want)
		}
	}
}

// TestDetailIntBranches exercises all branches in detailInt.
func TestDetailIntBranches(t *testing.T) {
	if got := detailInt(int(5)); got != 5 {
		t.Errorf("int branch: got %d, want 5", got)
	}
	if got := detailInt(int64(7)); got != 7 {
		t.Errorf("int64 branch: got %d, want 7", got)
	}
	if got := detailInt(float64(3.9)); got != 3 {
		t.Errorf("float64 branch: got %d, want 3", got)
	}
	if got := detailInt("not a number"); got != 0 {
		t.Errorf("unknown type branch: got %d, want 0", got)
	}
	if got := detailInt(nil); got != 0 {
		t.Errorf("nil branch: got %d, want 0", got)
	}
}

// TestFormatDebuggerContextBranches exercises all branches in formatDebuggerContext.
func TestFormatDebuggerContextBranches(t *testing.T) {
	// paused + file + line
	got := formatDebuggerContext(map[string]any{
		"paused": true,
		"file":   "res://enemy.gd",
		"line":   float64(42),
	})
	want := "debugger paused at res://enemy.gd:42"
	if got != want {
		t.Errorf("paused+file+line: got %q, want %q", got, want)
	}

	// paused + file only (no line)
	got = formatDebuggerContext(map[string]any{
		"paused": true,
		"file":   "res://enemy.gd",
	})
	want = "debugger paused at res://enemy.gd"
	if got != want {
		t.Errorf("paused+file only: got %q, want %q", got, want)
	}

	// paused only (no file, no line)
	got = formatDebuggerContext(map[string]any{
		"paused": true,
	})
	want = "debugger paused"
	if got != want {
		t.Errorf("paused only: got %q, want %q", got, want)
	}

	// paused + file + line + message
	got = formatDebuggerContext(map[string]any{
		"paused":  true,
		"file":    "res://enemy.gd",
		"line":    float64(10),
		"message": "breakpoint hit",
	})
	want = "debugger paused at res://enemy.gd:10: breakpoint hit"
	if got != want {
		t.Errorf("paused+file+line+message: got %q, want %q", got, want)
	}

	// not paused — should return empty string
	got = formatDebuggerContext(map[string]any{
		"paused": false,
		"file":   "res://enemy.gd",
		"line":   float64(10),
	})
	if got != "" {
		t.Errorf("not paused: got %q, want empty", got)
	}

	// nil value — should return empty string
	got = formatDebuggerContext(nil)
	if got != "" {
		t.Errorf("nil: got %q, want empty", got)
	}

	// empty map — should return empty string
	got = formatDebuggerContext(map[string]any{})
	if got != "" {
		t.Errorf("empty map: got %q, want empty", got)
	}
}
