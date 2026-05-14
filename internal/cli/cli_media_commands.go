package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

func runViewportScreenshot(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viewport screenshot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outPath := fs.String("out", "", "local PNG output path")
	kind := fs.String("kind", "2d", "editor viewport kind: 2d or 3d")
	index := fs.Int("index", 0, "3D viewport index")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for screenshot job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outPath == "" {
		return fmt.Errorf("viewport screenshot requires --out")
	}
	if *kind != "2d" && *kind != "3d" {
		return fmt.Errorf("viewport screenshot --kind must be 2d or 3d")
	}
	result, err := client.ScreenshotViewport(ctx, requestID(), *kind, *index)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("viewport screenshot did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "viewport screenshot")
	if err != nil {
		return err
	}
	if err := writeScreenshotJob(*outPath, job); err != nil {
		return err
	}
	width := intFromJobResult(job.Result["width"])
	height := intFromJobResult(job.Result["height"])
	if width > 0 && height > 0 {
		fmt.Fprintf(stdout, "Screenshot written: %s (%dx%d)\n", *outPath, width, height)
	} else {
		fmt.Fprintf(stdout, "Screenshot written: %s\n", *outPath)
	}
	return nil
}

func runViewportSetSize(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viewport set-size", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	width := fs.Int("width", 0, "viewport width in pixels")
	height := fs.Int("height", 0, "viewport height in pixels")
	path := fs.String("path", "", "SubViewport node path (empty = main viewport)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *width <= 0 || *height <= 0 {
		return fmt.Errorf("viewport set-size requires --width and --height > 0")
	}
	result, err := client.ViewportSetSize(ctx, requestID(), *width, *height, *path)
	if err != nil {
		return err
	}
	if result.Path != "" {
		fmt.Fprintf(stdout, "Viewport size set: %s (%dx%d)\n", result.Path, result.Width, result.Height)
	} else {
		fmt.Fprintf(stdout, "Viewport size set: %dx%d\n", result.Width, result.Height)
	}
	return nil
}

func runViewportAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viewport add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "parent node path (optional)")
	width := fs.Int("width", 320, "SubViewport width")
	height := fs.Int("height", 0, "SubViewport height")
	addCamera := fs.Bool("camera", true, "add a Camera3D child")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *width <= 0 || *height <= 0 {
		return fmt.Errorf("viewport add requires --width and --height > 0")
	}
	result, err := client.ViewportAdd(ctx, requestID(), *parent, *width, *height, *addCamera)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "SubViewport added: %s (%dx%d)\n", result.Path, result.Width, result.Height)
	return nil
}

func runSceneApplyBlueprint(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene apply-blueprint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path")
	blueprint := fs.String("blueprint", "", "blueprint name: player3d, spotlight, trigger_area, hud_label, world_environment, directional_light, gpu_particles")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	propFlags := stringListFlag{}
	fs.Var(&propFlags, "prop", "override property in name=TYPED_JSON form")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *blueprint == "" {
		return fmt.Errorf("scene apply-blueprint requires --path and --blueprint")
	}
	props, err := parseNameJSONPairs(propFlags)
	if err != nil {
		return err
	}
	result, err := client.ApplyBlueprint(ctx, requestID(), *path, *blueprint, props, *dryRun)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s (%s, %d nodes)\n", result.Path, result.Blueprint, result.Created)
		return nil
	}
	fmt.Fprintf(stdout, "Blueprint applied: %s (%s, %d nodes)\n", result.Path, result.Blueprint, result.Created)
	return nil
}

func runRunProbeRaycast(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run probe raycast", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "output as JSON")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for raycast job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	queued, err := client.RunRaycast(ctx, requestID())
	if err != nil {
		return err
	}
	var result bridge.RunRaycastResult
	if queued.JobID != "" {
		job, err := waitForJob(ctx, client, queued.JobID, *timeout, "run probe raycast")
		if err != nil {
			return err
		}
		encoded, _ := json.Marshal(job.Result)
		_ = json.Unmarshal(encoded, &result)
	} else {
		result = queued
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	if result.Hit {
		fmt.Fprintf(stdout, "Raycast hit: %s at distance %.3f\n", result.HitCollider, result.HitDistance)
	} else {
		fmt.Fprintln(stdout, "Raycast: no hit")
	}
	if result.CameraPath != "" {
		fmt.Fprintf(stdout, "Camera: %s\n", result.CameraPath)
	}
	return nil
}

func runThemeCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path (res://ui/main.tres)")
	force := fs.Bool("force", false, "overwrite existing theme")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("theme create requires --path")
	}
	result, err := client.ThemeCreate(ctx, requestID(), *path, *force)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme created: %s\n", result.Path)
	return nil
}

func runThemeSetColor(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme set-color", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path")
	nodeType := fs.String("node-type", "", "Godot node type (e.g. Label)")
	name := fs.String("name", "", "color override name (e.g. font_color)")
	value := fs.String("value", "", "RGBA as r,g,b,a with 0-1 floats or rrggbbaa hex")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *nodeType == "" || *name == "" || *value == "" {
		return fmt.Errorf("theme set-color requires --path, --node-type, --name, --value")
	}
	rgba, err := parseColorValue(*value)
	if err != nil {
		return err
	}
	result, err := client.ThemeSetColor(ctx, requestID(), *path, *nodeType, *name, rgba)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme color set: %s/%s on %s\n", result.NodeType, result.Name, result.Path)
	return nil
}

func runThemeSetFontSize(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme set-font-size", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path")
	nodeType := fs.String("node-type", "", "Godot node type (e.g. Label)")
	name := fs.String("name", "", "font size name (e.g. font_size)")
	size := fs.Int("value", 0, "font size in pixels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *nodeType == "" || *name == "" || *size <= 0 {
		return fmt.Errorf("theme set-font-size requires --path, --node-type, --name, --value > 0")
	}
	result, err := client.ThemeSetFontSize(ctx, requestID(), *path, *nodeType, *name, *size)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme font size set: %s/%s=%d on %s\n", result.NodeType, result.Name, *size, result.Path)
	return nil
}

func runThemeSetConstant(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme set-constant", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path")
	nodeType := fs.String("node-type", "", "Godot node type (e.g. MarginContainer)")
	name := fs.String("name", "", "constant name (e.g. margin_top)")
	value := fs.Int("value", 0, "integer constant value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *nodeType == "" || *name == "" {
		return fmt.Errorf("theme set-constant requires --path, --node-type, --name, --value")
	}
	result, err := client.ThemeSetConstant(ctx, requestID(), *path, *nodeType, *name, *value)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme constant set: %s/%s=%d on %s\n", result.NodeType, result.Name, *value, result.Path)
	return nil
}

func parseColorValue(s string) ([4]float64, error) {
	// Support r,g,b,a (0-1 floats) or rrggbbaa / rrggbb hex
	if strings.ContainsRune(s, ',') {
		parts := strings.Split(s, ",")
		if len(parts) < 3 || len(parts) > 4 {
			return [4]float64{}, fmt.Errorf("color must be r,g,b or r,g,b,a: %q", s)
		}
		var out [4]float64
		out[3] = 1.0
		for i, p := range parts {
			f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return out, fmt.Errorf("invalid color component %q: %w", p, err)
			}
			out[i] = f
		}
		return out, nil
	}
	// hex
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 {
		s += "ff"
	}
	if len(s) != 8 {
		return [4]float64{}, fmt.Errorf("hex color must be rrggbb or rrggbbaa: %q", s)
	}
	var out [4]float64
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return out, fmt.Errorf("invalid hex color %q: %w", s, err)
		}
		out[i] = float64(v) / 255.0
	}
	return out, nil
}

func runAnimationCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path (res://anims/player.tres)")
	name := fs.String("name", "", "animation name")
	length := fs.Float64("length", 1.0, "animation length in seconds")
	loop := fs.Bool("loop", false, "enable looping")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("animation create requires --path and --name")
	}
	result, err := client.AnimationCreate(ctx, requestID(), *path, *name, *length, *loop)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Animation created: %s in %s\n", result.Name, result.Path)
	return nil
}

func runAnimationTrackAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation track-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path")
	animation := fs.String("animation", "", "animation name")
	nodePath := fs.String("node-path", "", "node path for the track")
	property := fs.String("property", "", "property name for the track")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *animation == "" || *nodePath == "" || *property == "" {
		return fmt.Errorf("animation track-add requires --path, --animation, --node-path, --property")
	}
	result, err := client.AnimationTrackAdd(ctx, requestID(), *path, *animation, *nodePath, *property)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Track added: index %d in %s/%s\n", result.TrackIdx, result.Path, result.Animation)
	return nil
}

func runAnimationKeyframeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation keyframe-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path")
	animation := fs.String("animation", "", "animation name")
	trackIdx := fs.Int("track-idx", 0, "track index")
	timePos := fs.Float64("time", 0, "keyframe time in seconds")
	valueStr := fs.String("value", "", "keyframe value as TYPED_JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *animation == "" || *valueStr == "" {
		return fmt.Errorf("animation keyframe-add requires --path, --animation, --value")
	}
	var value any
	if err := json.Unmarshal([]byte(*valueStr), &value); err != nil {
		return fmt.Errorf("animation keyframe-add --value must be JSON: %w", err)
	}
	result, err := client.AnimationKeyframeAdd(ctx, requestID(), *path, *animation, *trackIdx, *timePos, value)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Keyframe added: track %d at t=%.3f in %s/%s\n", result.TrackIdx, *timePos, result.Path, result.Animation)
	return nil
}

func runAnimationLengthSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation length-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path")
	animation := fs.String("animation", "", "animation name")
	length := fs.Float64("length", 0, "animation length in seconds")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *animation == "" || *length <= 0 {
		return fmt.Errorf("animation length-set requires --path, --animation, --length > 0")
	}
	result, err := client.AnimationLengthSet(ctx, requestID(), *path, *animation, *length)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Animation length set: %s in %s to %.3fs\n", result.Name, result.Path, *length)
	return nil
}

func runAnimationPlayerPlay(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation player-play", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	nodePath := fs.String("node-path", "", "AnimationPlayer node path in running scene")
	animation := fs.String("animation", "", "animation name to play")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *nodePath == "" || *animation == "" {
		return fmt.Errorf("animation player-play requires --node-path and --animation")
	}
	_, err := client.AnimationPlayerPlay(ctx, requestID(), *nodePath, *animation)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Playing animation: %s on %s\n", *animation, *nodePath)
	return nil
}

func runTilesetCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap tileset-create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "TileSet resource path (res://tilesets/world.tres)")
	tileW := fs.Int("tile-width", 16, "tile width in pixels")
	tileH := fs.Int("tile-height", 16, "tile height in pixels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("tilemap tileset-create requires --path")
	}
	result, err := client.TilesetCreate(ctx, requestID(), *path, *tileW, *tileH)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "TileSet created: %s\n", result.Path)
	return nil
}

func runTilesetSourceAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap source-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "TileSet resource path")
	texture := fs.String("texture", "", "texture resource path (res://textures/tileset.png)")
	tileW := fs.Int("tile-width", 16, "tile width in pixels")
	tileH := fs.Int("tile-height", 16, "tile height in pixels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *texture == "" {
		return fmt.Errorf("tilemap source-add requires --path and --texture")
	}
	result, err := client.TilesetSourceAdd(ctx, requestID(), *path, *texture, *tileW, *tileH)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "TileSet source added: %s\n", result.Path)
	return nil
}

func runTilemapCellSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap cell-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	node := fs.String("node", "", "TileMap node path")
	layer := fs.Int("layer", 0, "tile layer index")
	x := fs.Int("x", 0, "cell x coordinate")
	y := fs.Int("y", 0, "cell y coordinate")
	sourceID := fs.Int("source-id", 0, "TileSet source id")
	atlasX := fs.Int("atlas-x", 0, "atlas x coordinate")
	atlasY := fs.Int("atlas-y", 0, "atlas y coordinate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *node == "" {
		return fmt.Errorf("tilemap cell-set requires --node")
	}
	result, err := client.TilemapCellSet(ctx, requestID(), *node, *layer, *x, *y, *sourceID, *atlasX, *atlasY)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Cell set: %s [%d,%d] layer %d\n", result.Node, *x, *y, *layer)
	return nil
}

func runTilemapCellSetRect(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap cell-set-rect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	node := fs.String("node", "", "TileMap node path")
	layer := fs.Int("layer", 0, "tile layer index")
	x := fs.Int("x", 0, "rectangle x coordinate")
	y := fs.Int("y", 0, "rectangle y coordinate")
	width := fs.Int("width", 0, "rectangle width in cells")
	height := fs.Int("height", 0, "rectangle height in cells")
	sourceID := fs.Int("source-id", 0, "TileSet source id")
	atlasX := fs.Int("atlas-x", 0, "atlas x coordinate")
	atlasY := fs.Int("atlas-y", 0, "atlas y coordinate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *node == "" {
		return fmt.Errorf("tilemap cell-set-rect requires --node")
	}
	if *width <= 0 || *height <= 0 {
		return fmt.Errorf("tilemap cell-set-rect requires --width and --height greater than 0")
	}
	result, err := client.TilemapCellSetRect(ctx, requestID(), *node, *layer, *x, *y, *width, *height, *sourceID, *atlasX, *atlasY)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Cell rect set: %s [%d,%d] %dx%d layer %d (%d cells)\n", result.Node, *x, *y, *width, *height, *layer, result.Cells)
	return nil
}

func runTilemapCellClear(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap cell-clear", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	node := fs.String("node", "", "TileMap node path")
	layer := fs.Int("layer", 0, "tile layer index")
	x := fs.Int("x", 0, "cell x coordinate")
	y := fs.Int("y", 0, "cell y coordinate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *node == "" {
		return fmt.Errorf("tilemap cell-clear requires --node")
	}
	result, err := client.TilemapCellClear(ctx, requestID(), *node, *layer, *x, *y)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Cell cleared: %s [%d,%d] layer %d\n", result.Node, *x, *y, *layer)
	return nil
}

func runAudioBusAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("audio bus-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "audio bus name")
	ifMissing := fs.Bool("if-missing", false, "succeed when the audio bus already exists")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("audio bus-add requires --name")
	}
	result, err := client.AudioBusAdd(ctx, requestID(), *name, *ifMissing)
	if err != nil {
		return err
	}
	if *ifMissing && !result.Created {
		fmt.Fprintf(stdout, "Audio bus already exists: %s\n", result.Bus)
		return nil
	}
	fmt.Fprintf(stdout, "Audio bus added: %s\n", result.Bus)
	return nil
}

func runAudioBusVolumeSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("audio bus-volume-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "audio bus name")
	volumeDB := fs.Float64("volume-db", 0, "volume in decibels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("audio bus-volume-set requires --name")
	}
	result, err := client.AudioBusVolumeSet(ctx, requestID(), *name, *volumeDB)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Audio bus volume set: %s = %.1f dB\n", result.Bus, *volumeDB)
	return nil
}

func runViewportCameraAssign(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viewport camera-assign", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	viewport := fs.String("viewport", "", "SubViewport node path")
	camera := fs.String("camera", "", "Camera3D or Camera2D node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *viewport == "" || *camera == "" {
		return fmt.Errorf("viewport camera-assign requires --viewport and --camera")
	}
	result, err := client.ViewportCameraAssign(ctx, requestID(), *viewport, *camera)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Camera assigned: %s -> %s\n", result.Camera, result.Viewport)
	return nil
}

func runAudioListenerMakeCurrent(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("audio listener-make-current", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AudioListener3D or AudioListener2D node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("audio listener-make-current requires --path")
	}
	result, err := client.AudioListenerMakeCurrent(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Audio listener active: %s (%s)\n", result.Path, result.Type)
	return nil
}

func runAudioBusEffectAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("audio bus-effect-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "audio bus name")
	effectType := fs.String("effect-type", "", "AudioEffect subclass (e.g. AudioEffectReverb)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *effectType == "" {
		return fmt.Errorf("audio bus-effect-add requires --name and --effect-type")
	}
	result, err := client.AudioBusEffectAdd(ctx, requestID(), *name, *effectType)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Audio effect added: %s on %s\n", *effectType, result.Bus)
	return nil
}
