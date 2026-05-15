package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

// Terrain commands

func runTerrain(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("terrain requires a subcommand: heightmap-import")
	}
	switch args[0] {
	case "heightmap-import":
		return runTerrainHeightmapImport(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown terrain subcommand: %s", args[0])
	}
}

func runTerrainHeightmapImport(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("terrain heightmap-import")
	path := fs.String("path", "", "HeightMapShape3D node path (or CollisionShape3D with HeightMapShape3D)")
	texture := fs.String("texture", "", "heightmap texture path (res://...)")
	minHeight := fs.Float64("min-height", -10.0, "minimum terrain height")
	maxHeight := fs.Float64("max-height", 50.0, "maximum terrain height")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *texture == "" {
		return fmt.Errorf("terrain heightmap-import requires --path and --texture")
	}
	result, err := client.TerrainHeightmapImport(ctx, requestID(), *path, *texture, *minHeight, *maxHeight)
	if err != nil {
		return err
	}
	w, _ := result["width"].(float64)
	h, _ := result["height"].(float64)
	fmt.Fprintf(stdout, "Terrain heightmap imported: %s (%dx%d, height %.1f..%.1f)\n", *path, int(w), int(h), *minHeight, *maxHeight)
	return nil
}

// Lightmap / VoxelGI / ReflectionProbe bake commands

func runLightmap(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("lightmap requires a subcommand: bake")
	}
	switch args[0] {
	case "bake":
		return runLightmapBake(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown lightmap subcommand: %s", args[0])
	}
}

func runLightmapBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("lightmap bake")
	path := fs.String("path", "", "LightmapGI node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("lightmap bake requires --path")
	}
	result, err := client.LightmapBake(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	status, _ := result["status"].(string)
	note, _ := result["note"].(string)
	fmt.Fprintf(stdout, "LightmapGI bake: %s (%s)\n", *path, status)
	if note != "" {
		fmt.Fprintf(stdout, "  Note: %s\n", note)
	}
	return nil
}

func runVoxelGI(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("voxelgi requires a subcommand: bake")
	}
	switch args[0] {
	case "bake":
		return runVoxelGIBake(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown voxelgi subcommand: %s", args[0])
	}
}

func runVoxelGIBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("voxelgi bake")
	path := fs.String("path", "", "VoxelGI node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("voxelgi bake requires --path")
	}
	result, err := client.VoxelGIBake(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	status, _ := result["status"].(string)
	fmt.Fprintf(stdout, "VoxelGI bake: %s (%s)\n", *path, status)
	return nil
}

func runReflectionProbe(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("reflection-probe requires a subcommand: bake")
	}
	switch args[0] {
	case "bake":
		return runReflectionProbeBake(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown reflection-probe subcommand: %s", args[0])
	}
}

func runReflectionProbeBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("reflection-probe bake")
	path := fs.String("path", "", "ReflectionProbe node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("reflection-probe bake requires --path")
	}
	result, err := client.ReflectionProbeBake(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	status, _ := result["status"].(string)
	note, _ := result["note"].(string)
	fmt.Fprintf(stdout, "ReflectionProbe bake: %s (%s)\n", *path, status)
	if note != "" {
		fmt.Fprintf(stdout, "  Note: %s\n", note)
	}
	return nil
}

// Decal commands

func runDecal(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("decal requires a subcommand: add, set-normal-fade")
	}
	switch args[0] {
	case "add":
		return runDecalAdd(ctx, client, args[1:], stdout)
	case "set-normal-fade":
		return runDecalSetNormalFade(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown decal subcommand: %s", args[0])
	}
}

func runDecalAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("decal add")
	parent := fs.String("parent", "", "parent node path")
	texture := fs.String("texture", "", "albedo texture path (res://...)")
	sizeStr := fs.String("size", "1,1,1", "size as X,Y,Z")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("decal add requires --parent")
	}
	size, err := parseVec3(*sizeStr)
	if err != nil {
		return fmt.Errorf("decal add --size: %w", err)
	}
	result, err := client.DecalAdd(ctx, requestID(), *parent, *texture, size)
	if err != nil {
		return err
	}
	path, _ := result["path"].(string)
	fmt.Fprintf(stdout, "Decal added: %s\n", path)
	return nil
}

func runDecalSetNormalFade(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("decal set-normal-fade")
	path := fs.String("path", "", "Decal node path")
	fade := fs.Float64("fade", 0.0, "normal fade (0.0–1.0)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("decal set-normal-fade requires --path")
	}
	result, err := client.DecalSetNormalFade(ctx, requestID(), *path, *fade)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "Decal normal fade set: %s (%.3f)\n", *path, *fade)
	return nil
}

// FogVolume commands

func runFogVolume(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("fog-volume requires a subcommand: add")
	}
	switch args[0] {
	case "add":
		return runFogVolumeAdd(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown fog-volume subcommand: %s", args[0])
	}
}

func runFogVolumeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("fog-volume add")
	parent := fs.String("parent", "", "parent node path")
	shape := fs.String("shape", "box", "shape: box, ellipsoid, cone, cylinder, world")
	sizeStr := fs.String("size", "2,2,2", "size as X,Y,Z")
	density := fs.Float64("density", 1.0, "fog density")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("fog-volume add requires --parent")
	}
	size, err := parseVec3(*sizeStr)
	if err != nil {
		return fmt.Errorf("fog-volume add --size: %w", err)
	}
	result, err := client.FogVolumeAdd(ctx, requestID(), *parent, *shape, size, *density)
	if err != nil {
		return err
	}
	path, _ := result["path"].(string)
	fmt.Fprintf(stdout, "FogVolume added: %s (shape: %s, density: %.2f)\n", path, *shape, *density)
	return nil
}

// Occluder commands

func runOccluder(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("occluder requires a subcommand: add")
	}
	switch args[0] {
	case "add":
		return runOccluderAdd(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown occluder subcommand: %s", args[0])
	}
}

func runOccluderAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("occluder add")
	parent := fs.String("parent", "", "parent node path")
	shape := fs.String("shape", "box", "shape: box, sphere, quad")
	sizeStr := fs.String("size", "1,1,1", "size as X,Y,Z")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("occluder add requires --parent")
	}
	size, err := parseVec3(*sizeStr)
	if err != nil {
		return fmt.Errorf("occluder add --size: %w", err)
	}
	result, err := client.OccluderAdd(ctx, requestID(), *parent, *shape, size)
	if err != nil {
		return err
	}
	path, _ := result["path"].(string)
	fmt.Fprintf(stdout, "OccluderInstance3D added: %s (shape: %s)\n", path, *shape)
	return nil
}

// Run profile command

func runRunProfile(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("run profile")
	metric := fs.String("metric", "fps", "comma-separated metrics: fps,draw_calls,physics_time,memory_usage")
	duration := fs.Duration("duration", 5*time.Second, "sampling duration (e.g. 5s, 30s)")
	timeout := fs.Duration("timeout", 120*time.Second, "maximum time to wait for profile result")
	if err := fs.Parse(args); err != nil {
		return err
	}
	metrics := strings.Split(*metric, ",")
	for i, m := range metrics {
		metrics[i] = strings.TrimSpace(m)
	}
	durationMS := float64(duration.Milliseconds())
	result, err := client.RunProfile(ctx, requestID(), metrics, durationMS)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("run profile did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "run profile")
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Profile result (%.0fms, %d samples):\n", durationMS, int(floatFromResult(job.Result["sample_count"])))
	for _, m := range metrics {
		avgKey := m + "_avg"
		minKey := m + "_min"
		avg := floatFromResult(job.Result[avgKey])
		min := floatFromResult(job.Result[minKey])
		if avg > 0 || min > 0 {
			fmt.Fprintf(stdout, "  %s: avg=%.2f min=%.2f\n", m, avg, min)
		}
	}
	return nil
}

func floatFromResult(v any) float64 {
	switch f := v.(type) {
	case float64:
		return f
	case float32:
		return float64(f)
	}
	return 0
}

// parseVec3 parses "X,Y,Z" into [3]float64
func parseVec3(s string) ([3]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return [3]float64{}, fmt.Errorf("expected X,Y,Z but got %q", s)
	}
	var out [3]float64
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return [3]float64{}, fmt.Errorf("component %d: %w", i, err)
		}
		out[i] = v
	}
	return out, nil
}
