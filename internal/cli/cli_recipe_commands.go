package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"gdctl/internal/bridge"
)

func runRecipe(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe requires a name: fog-volume, decal, occluder, voxelgi, reflection-probe, csg, lod, softbody, terrain, graph-edit, environment, character-body, area-trigger, camera-3d, camera-2d, light-3d, light-2d, ui-button")
	}
	switch args[0] {
	case "fog-volume":
		return runRecipeFogVolume(ctx, client, args[1:], stdout)
	case "decal":
		return runRecipeDecal(ctx, client, args[1:], stdout)
	case "occluder":
		return runRecipeOccluder(ctx, client, args[1:], stdout)
	case "voxelgi":
		return runRecipeVoxelGI(ctx, client, args[1:], stdout)
	case "reflection-probe":
		return runRecipeReflectionProbe(ctx, client, args[1:], stdout)
	case "csg":
		return runRecipeCSG(ctx, client, args[1:], stdout)
	case "lod":
		return runRecipeLOD(ctx, client, args[1:], stdout)
	case "softbody":
		return runRecipeSoftBody(ctx, client, args[1:], stdout)
	case "terrain":
		return runRecipeTerrain(ctx, client, args[1:], stdout)
	case "graph-edit":
		return runRecipeGraphEdit(ctx, client, args[1:], stdout)
	case "environment":
		return runRecipeEnvironment(ctx, client, args[1:], stdout)
	case "character-body":
		return runRecipeCharacterBody(ctx, client, args[1:], stdout)
	case "area-trigger":
		return runRecipeAreaTrigger(ctx, client, args[1:], stdout)
	case "camera-3d":
		return runRecipeCamera3D(ctx, client, args[1:], stdout)
	case "camera-2d":
		return runRecipeCamera2D(ctx, client, args[1:], stdout)
	case "light-3d":
		return runRecipeLight3D(ctx, client, args[1:], stdout)
	case "light-2d":
		return runRecipeLight2D(ctx, client, args[1:], stdout)
	case "ui-button":
		return runRecipeUIButton(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown recipe: %s", args[0])
}

// recipe fog-volume

func runRecipeFogVolume(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe fog-volume requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeFogVolumeAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown fog-volume verb: %s", args[0])
}

func runRecipeFogVolumeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe fog-volume add")
	parent := fs.String("parent", "", "parent node path")
	shape := fs.String("shape", "box", "shape: box, ellipsoid, cone, cylinder, world")
	sizeStr := fs.String("size", "2,2,2", "size as X,Y,Z")
	density := fs.Float64("density", 1.0, "fog density")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe fog-volume add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type FogVolume --name FogVolume\n", *parent)
		fmt.Fprintf(stdout, "gdctl node set --path %s/FogVolume --property shape --int 0\n", *parent)
		fmt.Fprintf(stdout, "gdctl node set --path %s/FogVolume --property density --float %.2f\n", *parent, *density)
		return nil
	}
	size, err := parseVec3(*sizeStr)
	if err != nil {
		return fmt.Errorf("recipe fog-volume add --size: %w", err)
	}
	result, err := client.FogVolumeAdd(ctx, requestID(), *parent, *shape, size, *density)
	if err != nil {
		return err
	}
	path, _ := result["path"].(string)
	fmt.Fprintf(stdout, "FogVolume added: %s (shape: %s, density: %.2f)\n", path, *shape, *density)
	return nil
}

// recipe decal

func runRecipeDecal(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe decal requires a verb: add, set-normal-fade")
	}
	switch args[0] {
	case "add":
		return runRecipeDecalAdd(ctx, client, args[1:], stdout)
	case "set-normal-fade":
		return runRecipeDecalSetNormalFade(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown decal verb: %s", args[0])
}

func runRecipeDecalAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe decal add")
	parent := fs.String("parent", "", "parent node path")
	texture := fs.String("texture", "", "albedo texture path (res://...)")
	sizeStr := fs.String("size", "1,1,1", "size as X,Y,Z")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe decal add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type Decal --name Decal\n", *parent)
		if *texture != "" {
			fmt.Fprintf(stdout, "gdctl node set-resource --path %s/Decal --property texture_albedo --resource %s\n", *parent, *texture)
		}
		fmt.Fprintf(stdout, "gdctl node set --path %s/Decal --property size --vector3 %s\n", *parent, *sizeStr)
		return nil
	}
	size, err := parseVec3(*sizeStr)
	if err != nil {
		return fmt.Errorf("recipe decal add --size: %w", err)
	}
	result, err := client.DecalAdd(ctx, requestID(), *parent, *texture, size)
	if err != nil {
		return err
	}
	path, _ := result["path"].(string)
	fmt.Fprintf(stdout, "Decal added: %s\n", path)
	return nil
}

func runRecipeDecalSetNormalFade(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe decal set-normal-fade")
	path := fs.String("path", "", "Decal node path")
	fade := fs.Float64("fade", 0.0, "normal fade (0.0–1.0)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("recipe decal set-normal-fade requires --path")
	}
	result, err := client.DecalSetNormalFade(ctx, requestID(), *path, *fade)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "Decal normal fade set: %s (%.3f)\n", *path, *fade)
	return nil
}

// recipe occluder

func runRecipeOccluder(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe occluder requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeOccluderAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown occluder verb: %s", args[0])
}

func runRecipeOccluderAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe occluder add")
	parent := fs.String("parent", "", "parent node path")
	shape := fs.String("shape", "box", "shape: box, sphere, quad")
	sizeStr := fs.String("size", "1,1,1", "size as X,Y,Z")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe occluder add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type OccluderInstance3D --name OccluderInstance3D\n", *parent)
		return nil
	}
	size, err := parseVec3(*sizeStr)
	if err != nil {
		return fmt.Errorf("recipe occluder add --size: %w", err)
	}
	result, err := client.OccluderAdd(ctx, requestID(), *parent, *shape, size)
	if err != nil {
		return err
	}
	path, _ := result["path"].(string)
	fmt.Fprintf(stdout, "OccluderInstance3D added: %s (shape: %s)\n", path, *shape)
	return nil
}

// recipe voxelgi

func runRecipeVoxelGI(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe voxelgi requires a verb: bake")
	}
	switch args[0] {
	case "bake":
		return runRecipeVoxelGIBake(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown voxelgi verb: %s", args[0])
}

func runRecipeVoxelGIBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe voxelgi bake")
	path := fs.String("path", "", "VoxelGI node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("recipe voxelgi bake requires --path")
	}
	result, err := client.VoxelGIBake(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	status, _ := result["status"].(string)
	fmt.Fprintf(stdout, "VoxelGI bake: %s (%s)\n", *path, status)
	return nil
}

// recipe reflection-probe

func runRecipeReflectionProbe(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe reflection-probe requires a verb: bake")
	}
	switch args[0] {
	case "bake":
		return runRecipeReflectionProbeBake(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown reflection-probe verb: %s", args[0])
}

func runRecipeReflectionProbeBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe reflection-probe bake")
	path := fs.String("path", "", "ReflectionProbe node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("recipe reflection-probe bake requires --path")
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

// recipe csg

func runRecipeCSG(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe csg requires a verb: node-add, operation-set, size-set")
	}
	switch args[0] {
	case "node-add":
		return runRecipeCSGNodeAdd(ctx, client, args[1:], stdout)
	case "operation-set":
		return runRecipeCSGOperationSet(ctx, client, args[1:], stdout)
	case "size-set":
		return runRecipeCSGSizeSet(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown csg verb: %s", args[0])
}

func runRecipeCSGNodeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe csg node-add")
	parent := fs.String("parent", "", "parent node path")
	csgType := fs.String("type", "CSGBox3D", "CSG node type: CSGBox3D, CSGSphere3D, CSGCylinder3D, CSGCombiner3D")
	name := fs.String("name", "", "node name")
	noCollision := fs.Bool("no-collision", false, "skip setting use_collision=true")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe csg node-add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type %s --name %s\n", *parent, *csgType, *name)
		if !*noCollision {
			fmt.Fprintf(stdout, "gdctl node set --path %s/%s --property use_collision --bool true\n", *parent, *name)
		}
		return nil
	}
	props := map[string]any{}
	if !*noCollision {
		props["use_collision"] = map[string]any{"kind": "bool", "value": true}
	}
	result, err := client.AddNode(ctx, requestID(), *parent, *csgType, *name, props, false)
	if err != nil {
		return err
	}
	nodePath, _ := result["path"].(string)
	fmt.Fprintf(stdout, "Added node: %s\n", nodePath)
	return nil
}

func runRecipeCSGOperationSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe csg operation-set")
	path := fs.String("path", "", "CSG node path")
	operation := fs.String("operation", "union", "CSG operation: union, intersection, subtraction")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("recipe csg operation-set requires --path")
	}
	opInt := csgOperationInt(*operation)
	setArgs := []string{"--path", *path, "--property", "operation", "--int", fmt.Sprintf("%d", opInt)}
	if err := runNodeSet(ctx, client, setArgs, stdout); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "CSG operation set: %s = %s\n", *path, *operation)
	return nil
}

func runRecipeCSGSizeSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe csg size-set")
	path := fs.String("path", "", "CSGBox3D node path")
	sizeStr := fs.String("size", "1,1,1", "size as X,Y,Z")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("recipe csg size-set requires --path")
	}
	setArgs := []string{"--path", *path, "--property", "size", "--vector3", *sizeStr}
	if err := runNodeSet(ctx, client, setArgs, stdout); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "CSG size set: %s = %s\n", *path, *sizeStr)
	return nil
}

func csgOperationInt(op string) int {
	switch strings.ToLower(op) {
	case "union":
		return 0
	case "intersection":
		return 1
	case "subtraction", "subtract":
		return 2
	}
	return 0
}

// recipe lod

func runRecipeLOD(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe lod requires a verb: set, set-many")
	}
	switch args[0] {
	case "set":
		return runRecipeLODSet(ctx, client, args[1:], stdout)
	case "set-many":
		return runRecipeLODSetMany(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown lod verb: %s", args[0])
}

func runRecipeLODSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe lod set")
	path := fs.String("path", "", "GeometryInstance3D node path")
	begin := fs.Float64("begin", 0, "distance at which LOD begins fading out (0 = disabled)")
	end := fs.Float64("end", 0, "distance at which node disappears (0 = disabled)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("recipe lod set requires --path")
	}
	result, err := client.LodSet(ctx, requestID(), *path, *begin, *end)
	if err != nil {
		return err
	}
	beginOut, _ := result["begin"].(float64)
	endOut, _ := result["end"].(float64)
	fmt.Fprintf(stdout, "LOD set: %s (begin: %.1f, end: %.1f)\n", *path, beginOut, endOut)
	return nil
}

func runRecipeLODSetMany(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe lod set-many")
	file := fs.String("file", "", "JSON file with LOD configuration array")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("recipe lod set-many requires --file")
	}
	content, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("recipe lod set-many: could not read file: %w", err)
	}
	var entries []bridge.LodEntry
	if err := json.Unmarshal(content, &entries); err != nil {
		return fmt.Errorf("recipe lod set-many: could not parse JSON: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("recipe lod set-many: file contains no entries")
	}
	result, err := client.LodSetMany(ctx, requestID(), entries)
	if err != nil {
		return err
	}
	updated, _ := result["updated"].(float64)
	fmt.Fprintf(stdout, "LOD set-many: %d nodes updated from %s\n", int(updated), *file)
	return nil
}

// recipe softbody

func runRecipeSoftBody(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe softbody requires a verb: pin-point, unpin-point")
	}
	switch args[0] {
	case "pin-point":
		return runRecipeSoftBodyPin(ctx, client, args[1:], stdout, true)
	case "unpin-point":
		return runRecipeSoftBodyPin(ctx, client, args[1:], stdout, false)
	}
	return fmt.Errorf("unknown softbody verb: %s", args[0])
}

func runRecipeSoftBodyPin(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer, pin bool) error {
	op := "pin-point"
	if !pin {
		op = "unpin-point"
	}
	fs := newFlagSet("recipe softbody " + op)
	path := fs.String("path", "", "SoftBody3D node path")
	point := fs.Int("point", -1, "vertex index to pin/unpin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *point < 0 {
		return fmt.Errorf("recipe softbody %s requires --path and --point >= 0", op)
	}
	var result map[string]any
	var err error
	if pin {
		result, err = client.SoftBodyPinPoint(ctx, requestID(), *path, *point)
	} else {
		result, err = client.SoftBodyUnpinPoint(ctx, requestID(), *path, *point)
	}
	if err != nil {
		return err
	}
	_ = result
	if pin {
		fmt.Fprintf(stdout, "SoftBody3D point pinned: %s[%d]\n", *path, *point)
	} else {
		fmt.Fprintf(stdout, "SoftBody3D point unpinned: %s[%d]\n", *path, *point)
	}
	return nil
}

// recipe terrain

func runRecipeTerrain(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe terrain requires a verb: heightmap-import")
	}
	switch args[0] {
	case "heightmap-import":
		return runRecipeTerrainHeightmapImport(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown terrain verb: %s", args[0])
}

func runRecipeTerrainHeightmapImport(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe terrain heightmap-import")
	path := fs.String("path", "", "HeightMapShape3D node path")
	texture := fs.String("texture", "", "heightmap texture path (res://...)")
	minHeight := fs.Float64("min-height", -10.0, "minimum terrain height")
	maxHeight := fs.Float64("max-height", 50.0, "maximum terrain height")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *texture == "" {
		return fmt.Errorf("recipe terrain heightmap-import requires --path and --texture")
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

// recipe graph-edit

func runRecipeGraphEdit(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe graph-edit requires a verb: node-add, connection-add, node-remove")
	}
	switch args[0] {
	case "node-add":
		return runRecipeGraphEditNodeAdd(ctx, client, args[1:], stdout)
	case "connection-add":
		return runRecipeGraphEditConnectionAdd(ctx, client, args[1:], stdout)
	case "node-remove":
		return runRecipeGraphEditNodeRemove(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown graph-edit verb: %s", args[0])
}

func runRecipeGraphEditNodeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe graph-edit node-add")
	path := fs.String("path", "", "GraphEdit node path")
	name := fs.String("name", "", "GraphNode name/title")
	posStr := fs.String("position", "0,0", "position as X,Y")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("recipe graph-edit node-add requires --path and --name")
	}
	posX, posY, err := parseVec2(*posStr)
	if err != nil {
		return fmt.Errorf("recipe graph-edit node-add --position: %w", err)
	}
	result, err := client.GraphEditNodeAdd(ctx, requestID(), *path, *name, posX, posY)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "GraphNode added: %s (graph: %s)\n", *name, *path)
	return nil
}

func runRecipeGraphEditConnectionAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe graph-edit connection-add")
	graph := fs.String("graph", "", "GraphEdit node path")
	from := fs.String("from", "", "source GraphNode name")
	fromPort := fs.Int("from-port", 0, "source port index")
	to := fs.String("to", "", "target GraphNode name")
	toPort := fs.Int("to-port", 0, "target port index")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *graph == "" || *from == "" || *to == "" {
		return fmt.Errorf("recipe graph-edit connection-add requires --graph, --from, and --to")
	}
	result, err := client.GraphEditConnectionAdd(ctx, requestID(), *graph, *from, *fromPort, *to, *toPort)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "GraphEdit connection added: %s:%d -> %s:%d (graph: %s)\n", *from, *fromPort, *to, *toPort, *graph)
	return nil
}

func runRecipeGraphEditNodeRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe graph-edit node-remove")
	path := fs.String("path", "", "GraphEdit node path")
	name := fs.String("name", "", "GraphNode name to remove")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("recipe graph-edit node-remove requires --path and --name")
	}
	result, err := client.GraphEditNodeRemove(ctx, requestID(), *path, *name)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "GraphNode removed: %s (graph: %s)\n", *name, *path)
	return nil
}

// recipe environment

func runRecipeEnvironment(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe environment requires a verb: set-background")
	}
	switch args[0] {
	case "set-background":
		return runRecipeEnvironmentSetBackground(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown environment verb: %s", args[0])
}

func runRecipeEnvironmentSetBackground(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe environment set-background")
	path := fs.String("path", "", "WorldEnvironment node path in the open scene")
	mode := fs.String("mode", "color", "background mode: color, sky, or clear")
	colorStr := fs.String("color", "", "background color R,G,B or R,G,B,A in 0.0–1.0 (required for --mode color)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("recipe environment set-background requires --path")
	}
	modeInt, err := environmentBackgroundModeInt(*mode)
	if err != nil {
		return err
	}
	modeValue := map[string]any{"kind": "int", "value": modeInt}
	if _, err := client.SetNodeProperty(ctx, requestID(), *path, "environment:background_mode", modeValue); err != nil {
		return fmt.Errorf("recipe environment set-background: set background_mode: %w", err)
	}
	if strings.ToLower(*mode) == "color" {
		if *colorStr == "" {
			return fmt.Errorf("recipe environment set-background --mode color requires --color R,G,B[,A]")
		}
		parts := strings.Split(*colorStr, ",")
		if len(parts) != 3 && len(parts) != 4 {
			return fmt.Errorf("recipe environment set-background --color must be R,G,B or R,G,B,A")
		}
		floats, err := parseFloatComponents(*colorStr, len(parts))
		if err != nil {
			return fmt.Errorf("recipe environment set-background --color: %w", err)
		}
		colorValue := map[string]any{"kind": "Color", "value": floats}
		if _, err := client.SetNodeProperty(ctx, requestID(), *path, "environment:background_color", colorValue); err != nil {
			return fmt.Errorf("recipe environment set-background: set background_color: %w", err)
		}
		fmt.Fprintf(stdout, "Environment background set: %s (mode: %s, color: %s)\n", *path, *mode, *colorStr)
	} else {
		fmt.Fprintf(stdout, "Environment background set: %s (mode: %s)\n", *path, *mode)
	}
	return nil
}

func environmentBackgroundModeInt(mode string) (int, error) {
	switch strings.ToLower(mode) {
	case "clear", "clear_color":
		return 0, nil
	case "color":
		return 1, nil
	case "sky":
		return 2, nil
	case "canvas":
		return 3, nil
	case "keep":
		return 4, nil
	case "camera_feed":
		return 5, nil
	}
	return 0, fmt.Errorf("unknown background mode %q; use color, sky, clear, canvas, keep, or camera_feed", mode)
}

// recipe character-body

func runRecipeCharacterBody(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe character-body requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeCharacterBodyAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown character-body verb: %s", args[0])
}

func runRecipeCharacterBodyAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe character-body add")
	parent := fs.String("parent", "", "parent node path")
	name := fs.String("name", "CharacterBody3D", "node name")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe character-body add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type CharacterBody3D --name %s\n", *parent, *name)
		fmt.Fprintf(stdout, "gdctl node add --parent %s/%s --type CollisionShape3D --name CollisionShape3D\n", *parent, *name)
		return nil
	}
	result, err := client.AddNode(ctx, requestID(), *parent, "CharacterBody3D", *name, map[string]any{}, false)
	if err != nil {
		return err
	}
	bodyPath, _ := result["path"].(string)
	_, err = client.AddNode(ctx, requestID(), bodyPath, "CollisionShape3D", "CollisionShape3D", map[string]any{}, false)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "CharacterBody3D added: %s (with CollisionShape3D)\n", bodyPath)
	return nil
}

// recipe area-trigger

func runRecipeAreaTrigger(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe area-trigger requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeAreaTriggerAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown area-trigger verb: %s", args[0])
}

func runRecipeAreaTriggerAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe area-trigger add")
	parent := fs.String("parent", "", "parent node path")
	name := fs.String("name", "Area3D", "node name")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe area-trigger add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type Area3D --name %s\n", *parent, *name)
		fmt.Fprintf(stdout, "gdctl node add --parent %s/%s --type CollisionShape3D --name CollisionShape3D\n", *parent, *name)
		return nil
	}
	result, err := client.AddNode(ctx, requestID(), *parent, "Area3D", *name, map[string]any{}, false)
	if err != nil {
		return err
	}
	areaPath, _ := result["path"].(string)
	_, err = client.AddNode(ctx, requestID(), areaPath, "CollisionShape3D", "CollisionShape3D", map[string]any{}, false)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Area3D trigger added: %s (with CollisionShape3D)\n", areaPath)
	return nil
}

// recipe camera-3d

func runRecipeCamera3D(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe camera-3d requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeCamera3DAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown camera-3d verb: %s", args[0])
}

func runRecipeCamera3DAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe camera-3d add")
	parent := fs.String("parent", "", "parent node path")
	name := fs.String("name", "Camera3D", "node name")
	current := fs.Bool("current", false, "set as the current camera")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe camera-3d add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type Camera3D --name %s\n", *parent, *name)
		if *current {
			fmt.Fprintf(stdout, "gdctl node set --path %s/%s --property current --bool true\n", *parent, *name)
		}
		return nil
	}
	props := map[string]any{}
	if *current {
		props["current"] = map[string]any{"kind": "bool", "value": true}
	}
	result, err := client.AddNode(ctx, requestID(), *parent, "Camera3D", *name, props, false)
	if err != nil {
		return err
	}
	nodePath, _ := result["path"].(string)
	fmt.Fprintf(stdout, "Camera3D added: %s\n", nodePath)
	return nil
}

// recipe camera-2d

func runRecipeCamera2D(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe camera-2d requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeCamera2DAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown camera-2d verb: %s", args[0])
}

func runRecipeCamera2DAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe camera-2d add")
	parent := fs.String("parent", "", "parent node path")
	name := fs.String("name", "Camera2D", "node name")
	enabled := fs.Bool("enabled", true, "enable the camera")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe camera-2d add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type Camera2D --name %s\n", *parent, *name)
		if !*enabled {
			fmt.Fprintf(stdout, "gdctl node set --path %s/%s --property enabled --bool false\n", *parent, *name)
		}
		return nil
	}
	props := map[string]any{}
	if !*enabled {
		props["enabled"] = map[string]any{"kind": "bool", "value": false}
	}
	result, err := client.AddNode(ctx, requestID(), *parent, "Camera2D", *name, props, false)
	if err != nil {
		return err
	}
	nodePath, _ := result["path"].(string)
	fmt.Fprintf(stdout, "Camera2D added: %s\n", nodePath)
	return nil
}

// recipe light-3d

func runRecipeLight3D(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe light-3d requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeLight3DAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown light-3d verb: %s", args[0])
}

func runRecipeLight3DAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe light-3d add")
	parent := fs.String("parent", "", "parent node path")
	name := fs.String("name", "DirectionalLight3D", "node name")
	lightType := fs.String("type", "directional", "light type: directional, omni, spot")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe light-3d add requires --parent")
	}
	nodeType := light3DNodeType(*lightType)
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type %s --name %s\n", *parent, nodeType, *name)
		return nil
	}
	result, err := client.AddNode(ctx, requestID(), *parent, nodeType, *name, map[string]any{}, false)
	if err != nil {
		return err
	}
	nodePath, _ := result["path"].(string)
	fmt.Fprintf(stdout, "%s added: %s\n", nodeType, nodePath)
	return nil
}

func light3DNodeType(t string) string {
	switch strings.ToLower(t) {
	case "omni":
		return "OmniLight3D"
	case "spot":
		return "SpotLight3D"
	}
	return "DirectionalLight3D"
}

// recipe light-2d

func runRecipeLight2D(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe light-2d requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeLight2DAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown light-2d verb: %s", args[0])
}

func runRecipeLight2DAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe light-2d add")
	parent := fs.String("parent", "", "parent node path")
	name := fs.String("name", "DirectionalLight2D", "node name")
	lightType := fs.String("type", "directional", "light type: directional, point")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe light-2d add requires --parent")
	}
	nodeType := light2DNodeType(*lightType)
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type %s --name %s\n", *parent, nodeType, *name)
		return nil
	}
	result, err := client.AddNode(ctx, requestID(), *parent, nodeType, *name, map[string]any{}, false)
	if err != nil {
		return err
	}
	nodePath, _ := result["path"].(string)
	fmt.Fprintf(stdout, "%s added: %s\n", nodeType, nodePath)
	return nil
}

func light2DNodeType(t string) string {
	switch strings.ToLower(t) {
	case "point":
		return "PointLight2D"
	}
	return "DirectionalLight2D"
}

// recipe ui-button

func runRecipeUIButton(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("recipe ui-button requires a verb: add")
	}
	switch args[0] {
	case "add":
		return runRecipeUIButtonAdd(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown ui-button verb: %s", args[0])
}

func runRecipeUIButtonAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("recipe ui-button add")
	parent := fs.String("parent", "", "parent node path")
	name := fs.String("name", "Button", "node name")
	text := fs.String("text", "", "button label text")
	printCore := fs.Bool("print-core", false, "print underlying core commands instead of executing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("recipe ui-button add requires --parent")
	}
	if *printCore {
		fmt.Fprintf(stdout, "gdctl node add --parent %s --type Button --name %s\n", *parent, *name)
		if *text != "" {
			fmt.Fprintf(stdout, "gdctl node set --path %s/%s --property text --string %q\n", *parent, *name, *text)
		}
		return nil
	}
	props := map[string]any{}
	if *text != "" {
		props["text"] = map[string]any{"kind": "String", "value": *text}
	}
	result, err := client.AddNode(ctx, requestID(), *parent, "Button", *name, props, false)
	if err != nil {
		return err
	}
	nodePath, _ := result["path"].(string)
	fmt.Fprintf(stdout, "Button added: %s\n", nodePath)
	return nil
}

// Shared helpers

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

func parseVec2(s string) (float64, float64, error) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected X,Y but got %q", s)
	}
	x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("X: %w", err)
	}
	y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("Y: %w", err)
	}
	return x, y, nil
}

func parseFloatComponents(s string, count int) ([]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != count {
		return nil, fmt.Errorf("expected %d comma-separated values, got %d", count, len(parts))
	}
	out := make([]float64, count)
	for i, p := range parts {
		p = strings.TrimSpace(p)
		var f float64
		if _, err := fmt.Sscanf(p, "%f", &f); err != nil {
			return nil, fmt.Errorf("invalid float value %q at position %d", p, i+1)
		}
		out[i] = f
	}
	return out, nil
}
