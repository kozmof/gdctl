package bridge

import (
	"context"
)

func (c *Client) ScreenshotViewport(ctx context.Context, requestID, kind string, index int) (ViewportScreenshotResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.screenshot",
		Params:    map[string]any{"kind": kind, "index": index},
	}
	return callPost[ViewportScreenshotResult](ctx, c, "/viewport/screenshot", env)
}

func (c *Client) RunRaycast(ctx context.Context, requestID string) (RunRaycastResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "run.probe.raycast", Params: map[string]any{}}
	return callPost[RunRaycastResult](ctx, c, "/run/probe/raycast", env)
}

func (c *Client) ThemeCreate(ctx context.Context, requestID, path string, force bool) (ThemeCreateResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "theme.create", Params: map[string]any{"path": path, "force": force}}
	return callPost[ThemeCreateResult](ctx, c, "/theme/create", env)
}

func (c *Client) ThemeSetColor(ctx context.Context, requestID, path, nodeType, name string, rgba [4]float64) (ThemeSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "theme.set-color",
		Params:    map[string]any{"path": path, "node_type": nodeType, "name": name, "value": rgba},
	}
	return callPost[ThemeSetResult](ctx, c, "/theme/set-color", env)
}

func (c *Client) ThemeSetFontSize(ctx context.Context, requestID, path, nodeType, name string, size int) (ThemeSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "theme.set-font-size",
		Params:    map[string]any{"path": path, "node_type": nodeType, "name": name, "value": size},
	}
	return callPost[ThemeSetResult](ctx, c, "/theme/set-font-size", env)
}

func (c *Client) ThemeSetConstant(ctx context.Context, requestID, path, nodeType, name string, value int) (ThemeSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "theme.set-constant",
		Params:    map[string]any{"path": path, "node_type": nodeType, "name": name, "value": value},
	}
	return callPost[ThemeSetResult](ctx, c, "/theme/set-constant", env)
}

func (c *Client) AnimationCreate(ctx context.Context, requestID, libraryPath, name string, length float64, loop bool) (AnimationCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.create",
		Params:    map[string]any{"path": libraryPath, "name": name, "length": length, "loop": loop},
	}
	return callPost[AnimationCreateResult](ctx, c, "/animation/create", env)
}

func (c *Client) AnimationTrackAdd(ctx context.Context, requestID, libraryPath, animName, nodePath, property string) (AnimationTrackResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.track-add",
		Params:    map[string]any{"path": libraryPath, "animation": animName, "node_path": nodePath, "property": property},
	}
	return callPost[AnimationTrackResult](ctx, c, "/animation/track-add", env)
}

func (c *Client) AnimationKeyframeAdd(ctx context.Context, requestID, libraryPath, animName string, trackIdx int, timePos float64, value any) (AnimationKeyframeResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.keyframe-add",
		Params:    map[string]any{"path": libraryPath, "animation": animName, "track_idx": trackIdx, "time": timePos, "value": value},
	}
	return callPost[AnimationKeyframeResult](ctx, c, "/animation/keyframe-add", env)
}

func (c *Client) AnimationLengthSet(ctx context.Context, requestID, libraryPath, animName string, length float64) (AnimationCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.length-set",
		Params:    map[string]any{"path": libraryPath, "animation": animName, "length": length},
	}
	return callPost[AnimationCreateResult](ctx, c, "/animation/length-set", env)
}

func (c *Client) AnimationPlayerPlay(ctx context.Context, requestID, nodePath, animName string) (AnimationCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.player-play",
		Params:    map[string]any{"node_path": nodePath, "animation": animName},
	}
	return callPost[AnimationCreateResult](ctx, c, "/animation/player-play", env)
}

func (c *Client) TilesetCreate(ctx context.Context, requestID, path string, tileWidth, tileHeight int) (TilesetCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.tileset-create",
		Params:    map[string]any{"path": path, "tile_width": tileWidth, "tile_height": tileHeight},
	}
	return callPost[TilesetCreateResult](ctx, c, "/tilemap/tileset-create", env)
}

func (c *Client) TilesetSourceAdd(ctx context.Context, requestID, tilesetPath, texturePath string, tileWidth, tileHeight int) (TilesetCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.source-add",
		Params:    map[string]any{"path": tilesetPath, "texture": texturePath, "tile_width": tileWidth, "tile_height": tileHeight},
	}
	return callPost[TilesetCreateResult](ctx, c, "/tilemap/source-add", env)
}

func (c *Client) TilemapCellSet(ctx context.Context, requestID, nodePath string, layer, x, y, sourceID, atlasX, atlasY int) (TilemapCellResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.cell-set",
		Params:    map[string]any{"node": nodePath, "layer": layer, "x": x, "y": y, "source_id": sourceID, "atlas_x": atlasX, "atlas_y": atlasY},
	}
	return callPost[TilemapCellResult](ctx, c, "/tilemap/cell-set", env)
}

func (c *Client) TilemapCellSetRect(ctx context.Context, requestID, nodePath string, layer, x, y, width, height, sourceID, atlasX, atlasY int) (TilemapCellResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.cell-set-rect",
		Params:    map[string]any{"node": nodePath, "layer": layer, "x": x, "y": y, "width": width, "height": height, "source_id": sourceID, "atlas_x": atlasX, "atlas_y": atlasY},
	}
	return callPost[TilemapCellResult](ctx, c, "/tilemap/cell-set-rect", env)
}

func (c *Client) TilemapCellClear(ctx context.Context, requestID, nodePath string, layer, x, y int) (TilemapCellResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.cell-clear",
		Params:    map[string]any{"node": nodePath, "layer": layer, "x": x, "y": y},
	}
	return callPost[TilemapCellResult](ctx, c, "/tilemap/cell-clear", env)
}

func (c *Client) AudioBusAdd(ctx context.Context, requestID, name string, ifMissing bool) (AudioBusResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "audio.bus-add", Params: map[string]any{"name": name, "if_missing": ifMissing}}
	return callPost[AudioBusResult](ctx, c, "/audio/bus-add", env)
}

func (c *Client) AudioBusVolumeSet(ctx context.Context, requestID, name string, volumeDB float64) (AudioBusResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "audio.bus-volume-set", Params: map[string]any{"name": name, "volume_db": volumeDB}}
	return callPost[AudioBusResult](ctx, c, "/audio/bus-volume-set", env)
}

func (c *Client) AudioBusEffectAdd(ctx context.Context, requestID, busName, effectType string) (AudioBusResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "audio.bus-effect-add", Params: map[string]any{"name": busName, "effect_type": effectType}}
	return callPost[AudioBusResult](ctx, c, "/audio/bus-effect-add", env)
}

func (c *Client) ViewportSetSize(ctx context.Context, requestID string, width, height int, path string) (ViewportSetSizeResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.set-size",
		Params:    map[string]any{"width": width, "height": height, "path": path},
	}
	return callPost[ViewportSetSizeResult](ctx, c, "/viewport/set-size", env)
}

func (c *Client) ViewportCameraAssign(ctx context.Context, requestID, viewportPath, cameraPath string) (ViewportCameraAssignResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.camera-assign",
		Params:    map[string]any{"viewport": viewportPath, "camera": cameraPath},
	}
	return callPost[ViewportCameraAssignResult](ctx, c, "/viewport/camera-assign", env)
}

func (c *Client) AudioListenerMakeCurrent(ctx context.Context, requestID, path string) (AudioListenerResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "audio.listener-make-current", Params: map[string]any{"path": path}}
	return callPost[AudioListenerResult](ctx, c, "/audio/listener-make-current", env)
}

func (c *Client) ViewportAdd(ctx context.Context, requestID, parent string, width, height int, addCamera bool) (ViewportAddResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.add",
		Params:    map[string]any{"parent": parent, "width": width, "height": height, "add_camera": addCamera},
	}
	return callPost[ViewportAddResult](ctx, c, "/viewport/add", env)
}
