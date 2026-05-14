package bridge

import (
	"context"
	"encoding/json"
)

func (c *Client) ScreenshotViewport(ctx context.Context, requestID, kind string, index int) (ViewportScreenshotResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.screenshot",
		Params: map[string]any{
			"kind":  kind,
			"index": index,
		},
	}
	result, err := c.postEnvelope(ctx, "/viewport/screenshot", env)
	if err != nil {
		return ViewportScreenshotResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ViewportScreenshotResult{}, err
	}
	var out ViewportScreenshotResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return ViewportScreenshotResult{}, err
	}
	return out, nil
}

func (c *Client) RunRaycast(ctx context.Context, requestID string) (RunRaycastResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "run.probe.raycast", Params: map[string]any{}}
	result, err := c.postEnvelope(ctx, "/run/probe/raycast", env)
	if err != nil {
		return RunRaycastResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return RunRaycastResult{}, err
	}
	var out RunRaycastResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ThemeCreate(ctx context.Context, requestID, path string, force bool) (ThemeCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "theme.create",
		Params:    map[string]any{"path": path, "force": force},
	}
	result, err := c.postEnvelope(ctx, "/theme/create", env)
	if err != nil {
		return ThemeCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ThemeCreateResult{}, err
	}
	var out ThemeCreateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ThemeSetColor(ctx context.Context, requestID, path, nodeType, name string, rgba [4]float64) (ThemeSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "theme.set-color",
		Params:    map[string]any{"path": path, "node_type": nodeType, "name": name, "value": rgba},
	}
	result, err := c.postEnvelope(ctx, "/theme/set-color", env)
	if err != nil {
		return ThemeSetResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ThemeSetResult{}, err
	}
	var out ThemeSetResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ThemeSetFontSize(ctx context.Context, requestID, path, nodeType, name string, size int) (ThemeSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "theme.set-font-size",
		Params:    map[string]any{"path": path, "node_type": nodeType, "name": name, "value": size},
	}
	result, err := c.postEnvelope(ctx, "/theme/set-font-size", env)
	if err != nil {
		return ThemeSetResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ThemeSetResult{}, err
	}
	var out ThemeSetResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ThemeSetConstant(ctx context.Context, requestID, path, nodeType, name string, value int) (ThemeSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "theme.set-constant",
		Params:    map[string]any{"path": path, "node_type": nodeType, "name": name, "value": value},
	}
	result, err := c.postEnvelope(ctx, "/theme/set-constant", env)
	if err != nil {
		return ThemeSetResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ThemeSetResult{}, err
	}
	var out ThemeSetResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AnimationCreate(ctx context.Context, requestID, libraryPath, name string, length float64, loop bool) (AnimationCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.create",
		Params:    map[string]any{"path": libraryPath, "name": name, "length": length, "loop": loop},
	}
	result, err := c.postEnvelope(ctx, "/animation/create", env)
	if err != nil {
		return AnimationCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AnimationCreateResult{}, err
	}
	var out AnimationCreateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AnimationTrackAdd(ctx context.Context, requestID, libraryPath, animName, nodePath, property string) (AnimationTrackResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.track-add",
		Params:    map[string]any{"path": libraryPath, "animation": animName, "node_path": nodePath, "property": property},
	}
	result, err := c.postEnvelope(ctx, "/animation/track-add", env)
	if err != nil {
		return AnimationTrackResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AnimationTrackResult{}, err
	}
	var out AnimationTrackResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AnimationKeyframeAdd(ctx context.Context, requestID, libraryPath, animName string, trackIdx int, timePos float64, value any) (AnimationKeyframeResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.keyframe-add",
		Params:    map[string]any{"path": libraryPath, "animation": animName, "track_idx": trackIdx, "time": timePos, "value": value},
	}
	result, err := c.postEnvelope(ctx, "/animation/keyframe-add", env)
	if err != nil {
		return AnimationKeyframeResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AnimationKeyframeResult{}, err
	}
	var out AnimationKeyframeResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AnimationLengthSet(ctx context.Context, requestID, libraryPath, animName string, length float64) (AnimationCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.length-set",
		Params:    map[string]any{"path": libraryPath, "animation": animName, "length": length},
	}
	result, err := c.postEnvelope(ctx, "/animation/length-set", env)
	if err != nil {
		return AnimationCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AnimationCreateResult{}, err
	}
	var out AnimationCreateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AnimationPlayerPlay(ctx context.Context, requestID, nodePath, animName string) (AnimationCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation.player-play",
		Params:    map[string]any{"node_path": nodePath, "animation": animName},
	}
	result, err := c.postEnvelope(ctx, "/animation/player-play", env)
	if err != nil {
		return AnimationCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AnimationCreateResult{}, err
	}
	var out AnimationCreateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) TilesetCreate(ctx context.Context, requestID, path string, tileWidth, tileHeight int) (TilesetCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.tileset-create",
		Params:    map[string]any{"path": path, "tile_width": tileWidth, "tile_height": tileHeight},
	}
	result, err := c.postEnvelope(ctx, "/tilemap/tileset-create", env)
	if err != nil {
		return TilesetCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return TilesetCreateResult{}, err
	}
	var out TilesetCreateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) TilesetSourceAdd(ctx context.Context, requestID, tilesetPath, texturePath string, tileWidth, tileHeight int) (TilesetCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.source-add",
		Params:    map[string]any{"path": tilesetPath, "texture": texturePath, "tile_width": tileWidth, "tile_height": tileHeight},
	}
	result, err := c.postEnvelope(ctx, "/tilemap/source-add", env)
	if err != nil {
		return TilesetCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return TilesetCreateResult{}, err
	}
	var out TilesetCreateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) TilemapCellSet(ctx context.Context, requestID, nodePath string, layer, x, y, sourceID, atlasX, atlasY int) (TilemapCellResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.cell-set",
		Params:    map[string]any{"node": nodePath, "layer": layer, "x": x, "y": y, "source_id": sourceID, "atlas_x": atlasX, "atlas_y": atlasY},
	}
	result, err := c.postEnvelope(ctx, "/tilemap/cell-set", env)
	if err != nil {
		return TilemapCellResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return TilemapCellResult{}, err
	}
	var out TilemapCellResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) TilemapCellSetRect(ctx context.Context, requestID, nodePath string, layer, x, y, width, height, sourceID, atlasX, atlasY int) (TilemapCellResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.cell-set-rect",
		Params:    map[string]any{"node": nodePath, "layer": layer, "x": x, "y": y, "width": width, "height": height, "source_id": sourceID, "atlas_x": atlasX, "atlas_y": atlasY},
	}
	result, err := c.postEnvelope(ctx, "/tilemap/cell-set-rect", env)
	if err != nil {
		return TilemapCellResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return TilemapCellResult{}, err
	}
	var out TilemapCellResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) TilemapCellClear(ctx context.Context, requestID, nodePath string, layer, x, y int) (TilemapCellResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "tilemap.cell-clear",
		Params:    map[string]any{"node": nodePath, "layer": layer, "x": x, "y": y},
	}
	result, err := c.postEnvelope(ctx, "/tilemap/cell-clear", env)
	if err != nil {
		return TilemapCellResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return TilemapCellResult{}, err
	}
	var out TilemapCellResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AudioBusAdd(ctx context.Context, requestID, name string, ifMissing bool) (AudioBusResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "audio.bus-add",
		Params:    map[string]any{"name": name, "if_missing": ifMissing},
	}
	result, err := c.postEnvelope(ctx, "/audio/bus-add", env)
	if err != nil {
		return AudioBusResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AudioBusResult{}, err
	}
	var out AudioBusResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AudioBusVolumeSet(ctx context.Context, requestID, name string, volumeDB float64) (AudioBusResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "audio.bus-volume-set",
		Params:    map[string]any{"name": name, "volume_db": volumeDB},
	}
	result, err := c.postEnvelope(ctx, "/audio/bus-volume-set", env)
	if err != nil {
		return AudioBusResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AudioBusResult{}, err
	}
	var out AudioBusResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AudioBusEffectAdd(ctx context.Context, requestID, busName, effectType string) (AudioBusResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "audio.bus-effect-add",
		Params:    map[string]any{"name": busName, "effect_type": effectType},
	}
	result, err := c.postEnvelope(ctx, "/audio/bus-effect-add", env)
	if err != nil {
		return AudioBusResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AudioBusResult{}, err
	}
	var out AudioBusResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ViewportSetSize(ctx context.Context, requestID string, width, height int, path string) (ViewportSetSizeResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.set-size",
		Params:    map[string]any{"width": width, "height": height, "path": path},
	}
	result, err := c.postEnvelope(ctx, "/viewport/set-size", env)
	if err != nil {
		return ViewportSetSizeResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ViewportSetSizeResult{}, err
	}
	var out ViewportSetSizeResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ViewportCameraAssign(ctx context.Context, requestID, viewportPath, cameraPath string) (ViewportCameraAssignResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.camera-assign",
		Params:    map[string]any{"viewport": viewportPath, "camera": cameraPath},
	}
	result, err := c.postEnvelope(ctx, "/viewport/camera-assign", env)
	if err != nil {
		return ViewportCameraAssignResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ViewportCameraAssignResult{}, err
	}
	var out ViewportCameraAssignResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AudioListenerMakeCurrent(ctx context.Context, requestID, path string) (AudioListenerResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "audio.listener-make-current",
		Params:    map[string]any{"path": path},
	}
	result, err := c.postEnvelope(ctx, "/audio/listener-make-current", env)
	if err != nil {
		return AudioListenerResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AudioListenerResult{}, err
	}
	var out AudioListenerResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ViewportAdd(ctx context.Context, requestID, parent string, width, height int, addCamera bool) (ViewportAddResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.add",
		Params:    map[string]any{"parent": parent, "width": width, "height": height, "add_camera": addCamera},
	}
	result, err := c.postEnvelope(ctx, "/viewport/add", env)
	if err != nil {
		return ViewportAddResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ViewportAddResult{}, err
	}
	var out ViewportAddResult
	return out, json.Unmarshal(encoded, &out)
}
