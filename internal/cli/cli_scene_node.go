package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

func runSceneTree(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	root, err := client.SceneTree(ctx)
	if err != nil {
		return err
	}
	bridge.RenderTree(stdout, root)
	return nil
}

func runSceneCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path, for example res://scenes/Main.tscn")
	rootType := fs.String("root", "", "root node type")
	rootName := fs.String("name", "", "root node name")
	force := fs.Bool("force", false, "overwrite an existing scene file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *rootType == "" || *rootName == "" {
		return fmt.Errorf("scene create requires --path, --root, and --name")
	}
	result, err := client.CreateScene(ctx, requestID(), *path, *rootType, *rootName, *force)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene created: %s\n", result.Path)
	fmt.Fprintf(stdout, "Root: %s %s\n", result.RootPath, result.RootType)
	return nil
}

func runSceneOpen(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene open", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path, for example res://scenes/Main.tscn")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for open job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("scene open requires --path")
	}
	pathValue, root, err := openSceneAndWait(ctx, client, *path, *timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene opened: %s\n", pathValue)
	if root != "" {
		fmt.Fprintf(stdout, "Root: %s\n", root)
	}
	return nil
}

func runSceneInstance(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene instance", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "parent node path")
	scenePath := fs.String("scene", "", "scene resource path")
	name := fs.String("name", "", "instance node name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" || *scenePath == "" || *name == "" {
		return fmt.Errorf("scene instance requires --parent, --scene, and --name")
	}
	result, err := client.InstanceScene(ctx, requestID(), *parent, *scenePath, *name)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene instanced: %s\n", result.Path)
	return nil
}

func runSceneSave(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene save", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "unsupported placeholder for future save-as support")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for save job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path != "" {
		return fmt.Errorf("scene save --path is temporarily unsupported; save the scene once in Godot, then run scene save")
	}
	pathValue, err := saveSceneAndWait(ctx, client, *timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene saved: %s\n", pathValue)
	return nil
}

func runSceneApply(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path to open and mutate")
	filePath := fs.String("file", "", "JSON scene tree file")
	dryRun := fs.Bool("dry-run", false, "validate without mutating or saving")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *filePath == "" {
		return fmt.Errorf("scene apply requires --path and --file")
	}
	data, err := os.ReadFile(*filePath)
	if err != nil {
		return err
	}
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return fmt.Errorf("scene apply --file must be JSON: %w", err)
	}
	openedPath, root, err := openSceneAndWait(ctx, client, *path, *timeout)
	if err != nil {
		return err
	}
	result, err := client.ApplyScene(ctx, requestID(), tree, *dryRun)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s (created: %d, properties: %d)\n", openedPath, result.Created, result.Updated)
		return nil
	}
	savedPath, err := saveSceneAndWait(ctx, client, *timeout)
	if err != nil {
		return err
	}
	if root == "" {
		root = result.Root
	}
	fmt.Fprintf(stdout, "Scene applied: %s\n", savedPath)
	if root != "" {
		fmt.Fprintf(stdout, "Root: %s\n", root)
	}
	fmt.Fprintf(stdout, "Created: %d\n", result.Created)
	fmt.Fprintf(stdout, "Properties: %d\n", result.Updated)
	return nil
}

func runSceneBatch(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene batch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path to open, mutate, and save")
	filePath := fs.String("file", "", "JSON file containing batch operations")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *filePath == "" {
		return fmt.Errorf("scene batch requires --path and --file")
	}
	data, err := os.ReadFile(*filePath)
	if err != nil {
		return err
	}
	var payload struct {
		Operations []map[string]any `json:"operations"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("scene batch --file must be JSON: %w", err)
	}
	if len(payload.Operations) == 0 {
		return fmt.Errorf("scene batch requires at least one operation")
	}

	sceneMu.Lock()
	defer sceneMu.Unlock()
	openedPath, _, err := openSceneAndWait(ctx, client, *path, *timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	for idx, op := range payload.Operations {
		if err := runSceneBatchOperation(ctx, client, idx, op, stdout); err != nil {
			return err
		}
	}
	savedPath, err := saveSceneAndWait(ctx, client, *timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene batch saved: %s (%d operations)\n", savedPath, len(payload.Operations))
	return nil
}

func runSceneBatchOperation(ctx context.Context, client *bridge.Client, idx int, op map[string]any, stdout io.Writer) error {
	kind, _ := op["op"].(string)
	if kind == "" {
		return fmt.Errorf("scene batch operation %d requires op", idx)
	}
	switch kind {
	case "node.add":
		parent, _ := op["parent"].(string)
		nodeType, _ := op["type"].(string)
		name, _ := op["name"].(string)
		if parent == "" || nodeType == "" || name == "" {
			return fmt.Errorf("scene batch operation %d node.add requires parent, type, and name", idx)
		}
		props := map[string]any{}
		if rawProps, ok := op["props"].(map[string]any); ok {
			props = rawProps
		}
		result, err := client.AddNode(ctx, requestID(), parent, nodeType, name, props, false)
		if err != nil {
			return err
		}
		path, _ := result["path"].(string)
		fmt.Fprintf(stdout, "Batch node.add: %s\n", valueOrDash(path))
	case "node.set":
		path, _ := op["path"].(string)
		property, _ := op["property"].(string)
		value, ok := op["value"]
		if path == "" || property == "" || !ok {
			return fmt.Errorf("scene batch operation %d node.set requires path, property, and value", idx)
		}
		if _, err := client.SetNodeProperty(ctx, requestID(), path, property, value); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Batch node.set: %s.%s\n", path, property)
	case "node.set-many":
		path, _ := op["path"].(string)
		if path == "" {
			return fmt.Errorf("scene batch operation %d node.set-many requires path", idx)
		}
		properties, err := propertiesMapFromValue(op["properties"])
		if err != nil {
			return fmt.Errorf("scene batch operation %d node.set-many: %w", idx, err)
		}
		result, err := client.SetNodeProperties(ctx, requestID(), path, properties)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Batch node.set-many: %s (%d properties)\n", path, result.Updated)
	case "node.attach-script":
		path, _ := op["path"].(string)
		script, _ := op["script"].(string)
		if path == "" || script == "" {
			return fmt.Errorf("scene batch operation %d node.attach-script requires path and script", idx)
		}
		if _, err := client.AttachScript(ctx, requestID(), path, script); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Batch node.attach-script: %s -> %s\n", path, script)
	case "node.set-resource":
		path, _ := op["path"].(string)
		property, _ := op["property"].(string)
		resource, _ := op["resource"].(string)
		if path == "" || property == "" || resource == "" {
			return fmt.Errorf("scene batch operation %d node.set-resource requires path, property, and resource", idx)
		}
		if _, err := client.SetNodeResource(ctx, requestID(), path, property, resource); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Batch node.set-resource: %s.%s -> %s\n", path, property, resource)
	default:
		return fmt.Errorf("scene batch operation %d has unsupported op %q", idx, kind)
	}
	return nil
}

// withScene handles the open→fn→save lifecycle for commands that accept --scene.
// When scenePath is empty fn runs directly with no scene management.
// When dryRun is true the save step is skipped (the scene was opened but not mutated).
func withScene(ctx context.Context, client *bridge.Client, scenePath string, dryRun bool, timeout time.Duration, stdout io.Writer, fn func() error) error {
	if scenePath == "" {
		return fn()
	}
	sceneMu.Lock()
	defer sceneMu.Unlock()
	openedPath, _, err := openSceneAndWait(ctx, client, scenePath, timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	if err := fn(); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	savedPath, err := saveSceneAndWait(ctx, client, timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene saved: %s\n", savedPath)
	return nil
}

func openSceneAndWait(ctx context.Context, client *bridge.Client, path string, timeout time.Duration) (string, string, error) {
	result, err := client.OpenScene(ctx, requestID(), path)
	if err != nil {
		return "", "", err
	}
	if result.JobID == "" {
		return "", "", fmt.Errorf("scene open did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, timeout, "scene open")
	if err != nil {
		return "", "", err
	}
	pathValue, _ := job.Result["path"].(string)
	if pathValue == "" {
		pathValue = result.Path
	}
	root, _ := job.Result["root"].(string)
	return pathValue, root, nil
}

func saveSceneAndWait(ctx context.Context, client *bridge.Client, timeout time.Duration) (string, error) {
	result, err := client.SaveScene(ctx, requestID(), "")
	if err != nil {
		return "", err
	}
	if result.JobID == "" {
		return "", fmt.Errorf("scene save did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, timeout, "scene save")
	if err != nil {
		return "", err
	}
	pathValue, _ := job.Result["path"].(string)
	if pathValue == "" {
		pathValue = result.Path
	}
	return pathValue, nil
}

func runNodeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "parent node path")
	nodeType := fs.String("type", "", "Godot node type")
	name := fs.String("name", "", "new node name")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before adding and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	propFlags := stringListFlag{}
	fs.Var(&propFlags, "prop", "initial property in name=TYPED_JSON form")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" || *nodeType == "" || *name == "" {
		return fmt.Errorf("node add requires --parent, --type, and --name")
	}
	props, err := parseNameJSONPairs(propFlags)
	if err != nil {
		return err
	}
	return withScene(ctx, client, *scenePath, *dryRun, *timeout, stdout, func() error {
		result, err := client.AddNode(ctx, requestID(), *parent, *nodeType, *name, props, *dryRun)
		if err != nil {
			return err
		}
		nodePath, _ := result["path"].(string)
		if *dryRun {
			fmt.Fprintf(stdout, "Dry run ok: %s\n", nodePath)
			return nil
		}
		fmt.Fprintf(stdout, "Added node: %s\n", nodePath)
		return nil
	})
}

func runNodeRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before removing and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node remove requires --path")
	}
	return withScene(ctx, client, *scenePath, *dryRun, *timeout, stdout, func() error {
		result, err := client.RemoveNode(ctx, requestID(), *path, *dryRun)
		if err != nil {
			return err
		}
		removed, _ := result["path"].(string)
		if *dryRun {
			fmt.Fprintf(stdout, "Dry run ok: %s\n", removed)
			return nil
		}
		fmt.Fprintf(stdout, "Removed node: %s\n", removed)
		return nil
	})
}

func runNodeRename(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node rename", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	name := fs.String("name", "", "new node name")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before renaming and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("node rename requires --path and --name")
	}
	return withScene(ctx, client, *scenePath, *dryRun, *timeout, stdout, func() error {
		result, err := client.RenameNode(ctx, requestID(), *path, *name, *dryRun)
		if err != nil {
			return err
		}
		newPath, _ := result["path"].(string)
		if *dryRun {
			fmt.Fprintf(stdout, "Dry run ok: %s\n", newPath)
			return nil
		}
		fmt.Fprintf(stdout, "Renamed node: %s\n", newPath)
		return nil
	})
}

func runNodeMove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node move", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	parent := fs.String("parent", "", "new parent node path")
	index := fs.Int("index", -1, "optional child index under new parent")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before moving and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *parent == "" {
		return fmt.Errorf("node move requires --path and --parent")
	}
	return withScene(ctx, client, *scenePath, *dryRun, *timeout, stdout, func() error {
		result, err := client.MoveNode(ctx, requestID(), *path, *parent, *index, *dryRun)
		if err != nil {
			return err
		}
		newPath, _ := result["path"].(string)
		if *dryRun {
			fmt.Fprintf(stdout, "Dry run ok: %s\n", newPath)
			return nil
		}
		fmt.Fprintf(stdout, "Moved node: %s\n", newPath)
		return nil
	})
}

func runNodeGet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	property := fs.String("property", "", "property name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *property == "" {
		return fmt.Errorf("node get requires --path and --property")
	}
	result, err := client.GetNodeProperty(ctx, requestID(), *path, *property)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runNodeSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	property := fs.String("property", "", "property name")
	position := fs.String("position", "", "shorthand for --property position --vector3 X,Y,Z")
	rotationDeg := fs.String("rotation-degrees", "", "shorthand for --property rotation_degrees --vector3 X,Y,Z")
	scale := fs.String("scale", "", "shorthand for --property scale --vector3 X,Y,Z")
	scenePath := fs.String("scene", "", "scene path to open before setting and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	valueFlags := newTypedValueFlags(fs, "node set")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node set requires --path")
	}
	// Resolve transform shorthands into property + value
	var resolvedProp string
	var resolvedValue any
	shorthandCount := 0
	for _, sh := range []struct{ flag, prop string }{
		{*position, "position"},
		{*rotationDeg, "rotation_degrees"},
		{*scale, "scale"},
	} {
		if sh.flag != "" {
			shorthandCount++
			resolvedProp = sh.prop
			vals, err := parseFloatList(sh.flag, 3, "--"+sh.prop)
			if err != nil {
				return err
			}
			resolvedValue = map[string]any{"kind": "Vector3", "value": vals}
		}
	}
	if shorthandCount > 1 {
		return fmt.Errorf("node set: only one of --position, --rotation-degrees, --scale may be used at a time")
	}
	if shorthandCount > 0 && *property != "" {
		return fmt.Errorf("node set: --property cannot be combined with --position, --rotation-degrees, or --scale")
	}
	if shorthandCount == 0 {
		if *property == "" {
			return fmt.Errorf("node set requires --property (or a transform shorthand)")
		}
		resolvedProp = *property
		var err error
		resolvedValue, err = valueFlags.Value()
		if err != nil {
			return err
		}
	}
	return withScene(ctx, client, *scenePath, false, *timeout, stdout, func() error {
		result, err := client.SetNodeProperty(ctx, requestID(), *path, resolvedProp, resolvedValue)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Set %s on %s\n", result.Property, result.Path)
		return nil
	})
}

func runNodeSetMany(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node set-many", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	filePath := fs.String("file", "", "JSON file containing properties")
	scenePath := fs.String("scene", "", "scene path to open before setting and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *filePath == "" {
		return fmt.Errorf("node set-many requires --path and --file")
	}
	properties, err := readSetManyPropertiesFile(*filePath)
	if err != nil {
		return err
	}
	return withScene(ctx, client, *scenePath, false, *timeout, stdout, func() error {
		result, err := client.SetNodeProperties(ctx, requestID(), *path, properties)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Set %d properties on %s\n", result.Updated, result.Path)
		return nil
	})
}

func readSetManyPropertiesFile(filePath string) (map[string]any, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("node set-many --file must be JSON: %w", err)
	}
	return propertiesMapFromValue(payload["properties"])
}

func propertiesMapFromValue(value any) (map[string]any, error) {
	raw, ok := value.(map[string]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("requires non-empty properties object")
	}
	return raw, nil
}

func runNodeSetResource(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node set-resource", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	property := fs.String("property", "", "property name")
	resourcePath := fs.String("resource", "", "resource path")
	scenePath := fs.String("scene", "", "scene path to open before setting and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *property == "" || *resourcePath == "" {
		return fmt.Errorf("node set-resource requires --path, --property, and --resource")
	}
	return withScene(ctx, client, *scenePath, false, *timeout, stdout, func() error {
		result, err := client.SetNodeResource(ctx, requestID(), *path, *property, *resourcePath)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Set %s on %s to %s\n", result.Property, result.Path, result.Resource)
		return nil
	})
}

func runNodeAttachScript(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node attach-script", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	scriptPath := fs.String("script", "", "script resource path")
	scenePath := fs.String("scene", "", "scene path to open before attaching and save after attaching")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *scriptPath == "" {
		return fmt.Errorf("node attach-script requires --path and --script")
	}
	return withScene(ctx, client, *scenePath, false, *timeout, stdout, func() error {
		result, err := client.AttachScript(ctx, requestID(), *path, *scriptPath)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Attached script: %s -> %s\n", result.Script, result.Path)
		return nil
	})
}

func runNodeGroupAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node group add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	group := fs.String("group", "", "group name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *group == "" {
		return fmt.Errorf("node group add requires --path and --group")
	}
	result, err := client.NodeGroupAdd(ctx, requestID(), *path, *group)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Added to group: %s on %s\n", result.Group, result.Path)
	return nil
}

func runNodeGroupRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node group remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	group := fs.String("group", "", "group name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *group == "" {
		return fmt.Errorf("node group remove requires --path and --group")
	}
	result, err := client.NodeGroupRemove(ctx, requestID(), *path, *group)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Removed from group: %s on %s\n", result.Group, result.Path)
	return nil
}

func runNodeGroupList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node group list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node group list requires --path")
	}
	result, err := client.NodeGroupList(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Groups on %s: %s\n", result.Path, strings.Join(result.Groups, ", "))
	return nil
}

func runNodeDuplicate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node duplicate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "source node path")
	name := fs.String("name", "", "name for the duplicate")
	parent := fs.String("parent", "", "parent node path (defaults to source's parent)")
	dryRun := fs.Bool("dry-run", false, "preview without modifying scene")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("node duplicate requires --path and --name")
	}
	result, err := client.NodeDuplicate(ctx, requestID(), *path, *name, *parent, *dryRun)
	if err != nil {
		return err
	}
	if result.DryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", result.Path)
		return nil
	}
	fmt.Fprintf(stdout, "Duplicated: %s (source: %s)\n", result.Path, result.SourcePath)
	return nil
}

func runNodeListProperties(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node list-properties", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node list-properties requires --path")
	}
	result, err := client.NodeListProperties(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result.Properties)
}
