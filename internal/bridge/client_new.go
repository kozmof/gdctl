package bridge

import (
	"context"
	"strings"
)

// AnimationTree commands

func (c *Client) AnimationTreeAddState(ctx context.Context, requestID, treePath, name, animation string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation-tree.add-state",
		Params:    map[string]any{"tree_path": treePath, "name": name, "animation": animation},
	}
	return c.postEnvelope(ctx, "/animation-tree/add-state", env)
}

func (c *Client) AnimationTreeAddTransition(ctx context.Context, requestID, treePath, from, to, condition string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation-tree.add-transition",
		Params:    map[string]any{"tree_path": treePath, "from": from, "to": to, "condition": condition},
	}
	return c.postEnvelope(ctx, "/animation-tree/add-transition", env)
}

func (c *Client) AnimationTreeBlendSpace2DAdd(ctx context.Context, requestID, treePath, state, blendX, blendY string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "animation-tree.blend-space-2d-add",
		Params:    map[string]any{"tree_path": treePath, "state": state, "blend_x": blendX, "blend_y": blendY},
	}
	return c.postEnvelope(ctx, "/animation-tree/blend-space-2d-add", env)
}

func (c *Client) AnimationTreeSetParam(ctx context.Context, requestID, treePath, param string, valueKey string, value any) (map[string]any, error) {
	params := map[string]any{"tree_path": treePath, "param": param, valueKey: value}
	env := RequestEnvelope{RequestID: requestID, Op: "animation-tree.set-param", Params: params}
	return c.postEnvelope(ctx, "/animation-tree/set-param", env)
}

// SoftBody commands

func (c *Client) SoftBodyPinPoint(ctx context.Context, requestID, path string, point int) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "softbody.pin-point",
		Params:    map[string]any{"path": path, "point": point},
	}
	return c.postEnvelope(ctx, "/softbody/pin-point", env)
}

func (c *Client) SoftBodyUnpinPoint(ctx context.Context, requestID, path string, point int) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "softbody.unpin-point",
		Params:    map[string]any{"path": path, "point": point},
	}
	return c.postEnvelope(ctx, "/softbody/unpin-point", env)
}

// LOD commands

type LodEntry struct {
	Path  string  `json:"path"`
	Begin float64 `json:"begin"`
	End   float64 `json:"end"`
}

func (c *Client) LodSet(ctx context.Context, requestID, path string, begin, end float64) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "lod.set",
		Params:    map[string]any{"path": path, "begin": begin, "end": end},
	}
	return c.postEnvelope(ctx, "/lod/set", env)
}

func (c *Client) LodSetMany(ctx context.Context, requestID string, entries []LodEntry) (map[string]any, error) {
	entriesRaw := make([]map[string]any, len(entries))
	for i, e := range entries {
		entriesRaw[i] = map[string]any{"path": e.Path, "begin": e.Begin, "end": e.End}
	}
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "lod.set-many",
		Params:    map[string]any{"entries": entriesRaw},
	}
	return c.postEnvelope(ctx, "/lod/set-many", env)
}

// Run profile command

type RunProfileResult struct {
	JobID string `json:"job_id"`
}

func (c *Client) RunProfile(ctx context.Context, requestID string, metrics []string, durationMS float64) (RunProfileResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.profile",
		Params:    map[string]any{"metrics": strings.Join(metrics, ","), "duration_ms": durationMS},
	}
	return callPost[RunProfileResult](ctx, c, "/run/profile", env)
}

// Terrain command

func (c *Client) TerrainHeightmapImport(ctx context.Context, requestID, nodePath, texturePath string, minHeight, maxHeight float64) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "terrain.heightmap-import",
		Params:    map[string]any{"path": nodePath, "texture": texturePath, "min_height": minHeight, "max_height": maxHeight},
	}
	return c.postEnvelope(ctx, "/terrain/heightmap-import", env)
}

// Lighting commands

func (c *Client) LightmapBake(ctx context.Context, requestID, path string) (map[string]any, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "lightmap.bake", Params: map[string]any{"path": path}}
	return c.postEnvelope(ctx, "/lightmap/bake", env)
}

func (c *Client) VoxelGIBake(ctx context.Context, requestID, path string) (map[string]any, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "voxelgi.bake", Params: map[string]any{"path": path}}
	return c.postEnvelope(ctx, "/voxelgi/bake", env)
}

func (c *Client) ReflectionProbeBake(ctx context.Context, requestID, path string) (map[string]any, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "reflection-probe.bake", Params: map[string]any{"path": path}}
	return c.postEnvelope(ctx, "/reflection-probe/bake", env)
}

// Window commands

func (c *Client) WindowCreate(ctx context.Context, requestID, title string, width, height, posX, posY int) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "window.create",
		Params:    map[string]any{"title": title, "width": width, "height": height, "position_x": posX, "position_y": posY},
	}
	return c.postEnvelope(ctx, "/window/create", env)
}

func (c *Client) WindowAssignViewport(ctx context.Context, requestID string, windowID int, viewportPath string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "window.assign-viewport",
		Params:    map[string]any{"window_id": windowID, "viewport_path": viewportPath},
	}
	return c.postEnvelope(ctx, "/window/assign-viewport", env)
}

// GraphEdit commands

func (c *Client) GraphEditNodeAdd(ctx context.Context, requestID, graphPath, name string, posX, posY float64) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "graph-edit.node-add",
		Params:    map[string]any{"path": graphPath, "name": name, "position": []float64{posX, posY}},
	}
	return c.postEnvelope(ctx, "/graph-edit/node-add", env)
}

func (c *Client) GraphEditConnectionAdd(ctx context.Context, requestID, graphPath, from string, fromPort int, to string, toPort int) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "graph-edit.connection-add",
		Params:    map[string]any{"graph": graphPath, "from": from, "from_port": fromPort, "to": to, "to_port": toPort},
	}
	return c.postEnvelope(ctx, "/graph-edit/connection-add", env)
}

func (c *Client) GraphEditNodeRemove(ctx context.Context, requestID, graphPath, name string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "graph-edit.node-remove",
		Params:    map[string]any{"path": graphPath, "name": name},
	}
	return c.postEnvelope(ctx, "/graph-edit/node-remove", env)
}

// Accessibility commands

func (c *Client) AccessibilityTTSSpeak(ctx context.Context, requestID, text string, interrupt bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "accessibility.tts-speak",
		Params:    map[string]any{"text": text, "interrupt": interrupt},
	}
	return c.postEnvelope(ctx, "/accessibility/tts-speak", env)
}

func (c *Client) AccessibilityTTSConfigure(ctx context.Context, requestID string, pitch, rate float64, voice string) (map[string]any, error) {
	params := map[string]any{}
	if pitch != 0 {
		params["pitch"] = pitch
	}
	if rate != 0 {
		params["rate"] = rate
	}
	if voice != "" {
		params["voice"] = voice
	}
	env := RequestEnvelope{RequestID: requestID, Op: "accessibility.tts-configure", Params: params}
	return c.postEnvelope(ctx, "/accessibility/tts-configure", env)
}

func (c *Client) AccessibilityTTSStop(ctx context.Context, requestID string) (map[string]any, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "accessibility.tts-stop", Params: map[string]any{}}
	return c.postEnvelope(ctx, "/accessibility/tts-stop", env)
}

// i18n commands

func (c *Client) I18nLocaleSet(ctx context.Context, requestID, locale string) (map[string]any, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "i18n.locale-set", Params: map[string]any{"locale": locale}}
	return c.postEnvelope(ctx, "/i18n/locale-set", env)
}

func (c *Client) I18nStringAdd(ctx context.Context, requestID, key, locale, text string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "i18n.string-add",
		Params:    map[string]any{"key": key, "locale": locale, "text": text},
	}
	return c.postEnvelope(ctx, "/i18n/string-add", env)
}

// Audio playlist commands

func (c *Client) AudioPlaylistAdd(ctx context.Context, requestID, bus, stream string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "audio.playlist-add",
		Params:    map[string]any{"bus": bus, "stream": stream},
	}
	return c.postEnvelope(ctx, "/audio/playlist-add", env)
}

func (c *Client) AudioPlaylistAutoplay(ctx context.Context, requestID, bus, mode string) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "audio.playlist-autoplay",
		Params:    map[string]any{"bus": bus, "mode": mode},
	}
	return c.postEnvelope(ctx, "/audio/playlist-autoplay", env)
}

// Decal commands

func (c *Client) DecalAdd(ctx context.Context, requestID, parent, texture string, size [3]float64) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "decal.add",
		Params:    map[string]any{"parent": parent, "texture": texture, "size": []float64{size[0], size[1], size[2]}},
	}
	return c.postEnvelope(ctx, "/decal/add", env)
}

func (c *Client) DecalSetNormalFade(ctx context.Context, requestID, path string, fade float64) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "decal.set-normal-fade",
		Params:    map[string]any{"path": path, "fade": fade},
	}
	return c.postEnvelope(ctx, "/decal/set-normal-fade", env)
}

// FogVolume command

func (c *Client) FogVolumeAdd(ctx context.Context, requestID, parent, shape string, size [3]float64, density float64) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "fog-volume.add",
		Params:    map[string]any{"parent": parent, "shape": shape, "size": []float64{size[0], size[1], size[2]}, "density": density},
	}
	return c.postEnvelope(ctx, "/fog-volume/add", env)
}

// Occluder command

func (c *Client) OccluderAdd(ctx context.Context, requestID, parent, shape string, size [3]float64) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "occluder.add",
		Params:    map[string]any{"parent": parent, "shape": shape, "size": []float64{size[0], size[1], size[2]}},
	}
	return c.postEnvelope(ctx, "/occluder/add", env)
}
