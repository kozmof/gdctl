package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

func runSignalConnect(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("signal connect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "", "source node path")
	sig := fs.String("signal", "", "signal name")
	to := fs.String("to", "", "target node path")
	method := fs.String("method", "", "method name on target node")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *sig == "" || *to == "" || *method == "" {
		return fmt.Errorf("signal connect requires --from, --signal, --to, and --method")
	}
	result, err := client.SignalConnect(ctx, requestID(), *from, *sig, *to, *method)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Connected: %s::%s -> %s::%s\n", result.From, result.Signal, result.To, result.Method)
	return nil
}

func runSignalDisconnect(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("signal disconnect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "", "source node path")
	sig := fs.String("signal", "", "signal name")
	to := fs.String("to", "", "target node path")
	method := fs.String("method", "", "method name on target node")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *sig == "" || *to == "" || *method == "" {
		return fmt.Errorf("signal disconnect requires --from, --signal, --to, and --method")
	}
	result, err := client.SignalDisconnect(ctx, requestID(), *from, *sig, *to, *method)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Disconnected: %s::%s -> %s::%s\n", result.From, result.Signal, result.To, result.Method)
	return nil
}

func runProjectSettingGet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("project setting get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	key := fs.String("key", "", "project setting key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" {
		return fmt.Errorf("project setting get requires --key")
	}
	result, err := client.ProjectSettingGet(ctx, requestID(), *key)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runProjectSettingSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("project setting set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	key := fs.String("key", "", "project setting key")
	valueFlags := newTypedValueFlags(fs, "project setting set")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" {
		return fmt.Errorf("project setting set requires --key")
	}
	value, err := valueFlags.Value()
	if err != nil {
		return err
	}
	result, err := client.ProjectSettingSet(ctx, requestID(), *key, value)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Set %s\n", result.Key)
	return nil
}

func runScriptCheck(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("script check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "script path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("script check requires --path")
	}
	result, err := client.CheckScript(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("script check failed: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Script OK: %s\n", result.Path)
	return nil
}

func runScriptCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("script create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "script path")
	extends := fs.String("extends", "", "base Godot class name")
	force := fs.Bool("force", false, "overwrite an existing script")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *extends == "" {
		return fmt.Errorf("script create requires --path and --extends")
	}
	result, err := client.CreateScript(ctx, requestID(), *path, *extends, *force)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("script create produced invalid script: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Script created: %s\n", result.Path)
	return nil
}

func runScriptWrite(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("script write", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "script path")
	body := fs.String("body", "", "script body")
	bodyFile := fs.String("body-file", "", "local file containing script body")
	allowMissingPreloads := fs.Bool("allow-missing-preloads", false, "write script even if preloaded scenes/resources do not exist yet")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("script write requires --path")
	}
	if (*body == "" && *bodyFile == "") || (*body != "" && *bodyFile != "") {
		return fmt.Errorf("script write requires exactly one of --body or --body-file")
	}
	bodyText := *body
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile)
		if err != nil {
			return err
		}
		bodyText = string(data)
	}
	result, err := client.WriteScript(ctx, requestID(), *path, bodyText, *allowMissingPreloads)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("script write produced invalid script: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Script written: %s\n", result.Path)
	return nil
}

func runShaderCheck(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("shader check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "shader path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("shader check requires --path")
	}
	result, err := client.CheckShader(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("shader check failed: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Shader OK: %s\n", result.Path)
	return nil
}

func runShaderWrite(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("shader write", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "shader path")
	body := fs.String("body", "", "shader body")
	bodyFile := fs.String("body-file", "", "local file containing shader body")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("shader write requires --path")
	}
	if (*body == "" && *bodyFile == "") || (*body != "" && *bodyFile != "") {
		return fmt.Errorf("shader write requires exactly one of --body or --body-file")
	}
	bodyText := *body
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile)
		if err != nil {
			return err
		}
		bodyText = string(data)
	}
	result, err := client.WriteShader(ctx, requestID(), *path, bodyText)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("shader write produced invalid shader: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Shader written: %s\n", result.Path)
	return nil
}

func runResourceCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("resource create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "resource file path (res:// .tres)")
	resourceType := fs.String("type", "", "Godot resource class name (e.g. StandardMaterial3D)")
	scriptPath := fs.String("script", "", "GDScript resource script to instantiate")
	propFlags := stringListFlag{}
	fs.Var(&propFlags, "prop", "property in name=TYPED_JSON form, e.g. --prop albedo_color='{\"kind\":\"Color\",\"value\":[1,0,0,1]}'")
	shaderParamFlags := stringListFlag{}
	fs.Var(&shaderParamFlags, "shader-param", "ShaderMaterial parameter in name=res://path form")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("resource create requires --path")
	}
	if *resourceType == "" && *scriptPath == "" {
		return fmt.Errorf("resource create requires --type or --script")
	}
	props, err := parseNameJSONPairs(propFlags)
	if err != nil {
		return err
	}
	shaderParams, err := parseNameResourcePairs(shaderParamFlags)
	if err != nil {
		return err
	}
	result, err := client.CreateResource(ctx, requestID(), *path, *resourceType, *scriptPath, props, shaderParams)
	if err != nil {
		return err
	}
	if result.Script != "" {
		fmt.Fprintf(stdout, "Resource created: %s (%s via %s)\n", result.Path, result.Type, result.Script)
	} else {
		fmt.Fprintf(stdout, "Resource created: %s (%s)\n", result.Path, result.Type)
	}
	return nil
}

func runAutoload(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("autoload requires a subcommand (add, remove, list)")
	}
	switch args[0] {
	case "add":
		return runAutoloadAdd(ctx, client, args[1:], stdout)
	case "remove":
		return runAutoloadRemove(ctx, client, args[1:], stdout)
	case "list":
		return runAutoloadList(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown autoload subcommand: %s", args[0])
	}
}

func runAutoloadAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("autoload add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "autoload singleton name")
	path := fs.String("path", "", "script or scene path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *path == "" {
		return fmt.Errorf("autoload add requires --name and --path")
	}
	result, err := client.AutoloadAdd(ctx, requestID(), *name, *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Autoload added: %s -> %s\n", result.Name, result.Path)
	return nil
}

func runAutoloadRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("autoload remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "autoload singleton name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("autoload remove requires --name")
	}
	result, err := client.AutoloadRemove(ctx, requestID(), *name)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Autoload removed: %s\n", result.Name)
	return nil
}

func runAutoloadList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("autoload list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.AutoloadList(ctx, requestID())
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	if len(result.Autoloads) == 0 {
		fmt.Fprintln(stdout, "No autoloads")
		return nil
	}
	for _, item := range result.Autoloads {
		fmt.Fprintf(stdout, "%s -> %s\n", item.Name, item.Path)
	}
	return nil
}

func runInputMap(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("input requires a subcommand (action, event)")
	}
	switch args[0] {
	case "action":
		if len(args) < 2 {
			return fmt.Errorf("input action requires a subcommand (add, remove, list)")
		}
		switch args[1] {
		case "add":
			return runInputActionAdd(ctx, client, args[2:], stdout)
		case "remove":
			return runInputActionRemove(ctx, client, args[2:], stdout)
		case "list":
			return runInputActionList(ctx, client, args[2:], stdout)
		default:
			return fmt.Errorf("unknown input action subcommand: %s", args[1])
		}
	case "event":
		if len(args) < 2 {
			return fmt.Errorf("input event requires a subcommand (add-key)")
		}
		switch args[1] {
		case "add-key":
			return runInputEventAddKey(ctx, client, args[2:], stdout)
		default:
			return fmt.Errorf("unknown input event subcommand: %s", args[1])
		}
	default:
		return fmt.Errorf("unknown input subcommand: %s", args[0])
	}
}

func runInputActionAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("input action add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	action := fs.String("name", "", "input action name")
	deadzone := fs.Float64("deadzone", 0.5, "input action deadzone")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *action == "" {
		return fmt.Errorf("input action add requires --name")
	}
	result, err := client.InputActionAdd(ctx, requestID(), *action, *deadzone)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Input action added: %s (deadzone %.2f)\n", result.Action, result.Deadzone)
	return nil
}

func runInputActionRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("input action remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	action := fs.String("name", "", "input action name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *action == "" {
		return fmt.Errorf("input action remove requires --name")
	}
	result, err := client.InputActionRemove(ctx, requestID(), *action)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Input action removed: %s\n", result.Action)
	return nil
}

func runInputActionList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("input action list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "print JSON")
	all := fs.Bool("all", false, "include built-in engine actions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.InputActionList(ctx, requestID(), *all)
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	if len(result.Actions) == 0 {
		fmt.Fprintln(stdout, "No input actions")
		return nil
	}
	for _, action := range result.Actions {
		fmt.Fprintf(stdout, "%s", action.Action)
		if len(action.Events) > 0 {
			var eventLabels []string
			for _, event := range action.Events {
				if event.Key != "" {
					eventLabels = append(eventLabels, event.Key)
				} else if event.Text != "" {
					eventLabels = append(eventLabels, event.Text)
				} else {
					eventLabels = append(eventLabels, event.Type)
				}
			}
			fmt.Fprintf(stdout, " [%s]", strings.Join(eventLabels, ", "))
		}
		fmt.Fprintln(stdout)
	}
	return nil
}

func runInputEventAddKey(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("input event add-key", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	action := fs.String("action", "", "input action name")
	key := fs.String("key", "", "key name, e.g. W, Space, Up")
	physical := fs.Bool("physical", true, "use physical keycode instead of layout keycode")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *action == "" || *key == "" {
		return fmt.Errorf("input event add-key requires --action and --key")
	}
	result, err := client.InputEventAddKey(ctx, requestID(), *action, *key, *physical)
	if err != nil {
		return err
	}
	if result.EventAdded {
		fmt.Fprintf(stdout, "Input key added: %s -> %s\n", result.Action, result.Key)
	} else {
		fmt.Fprintf(stdout, "Input key already present: %s -> %s\n", result.Action, result.Key)
	}
	return nil
}

func runFileWriteBytes(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file write-bytes", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "resource file path")
	inPath := fs.String("in", "", "local input file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *inPath == "" {
		return fmt.Errorf("file write-bytes requires --path and --in")
	}
	data, err := os.ReadFile(*inPath)
	if err != nil {
		return err
	}
	result, err := client.WriteFileBytes(ctx, requestID(), *path, base64.StdEncoding.EncodeToString(data))
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "File written: %s (%d bytes)\n", result.Path, result.Bytes)
	return nil
}

func runFileList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// directory path")
	recursive := fs.Bool("recursive", false, "list recursively")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file list requires --path")
	}
	result, err := client.FileList(ctx, requestID(), *path, *recursive)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runFileMkdir(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file mkdir", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// directory path to create")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file mkdir requires --path")
	}
	result, err := client.FileMkdir(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Created: %s\n", result.Path)
	return nil
}

func runFileDelete(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// path to delete")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file delete requires --path")
	}
	result, err := client.FileDelete(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Deleted: %s\n", result.Path)
	return nil
}

func runFileExists(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file exists", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// path to check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file exists requires --path")
	}
	result, err := client.FileExists(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runNavigationBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("navigation bake", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "NavigationRegion node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("navigation bake requires --path")
	}
	result, err := client.NavigationBake(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Baked: %s (%s)\n", result.Path, result.Kind)
	return nil
}

func runImportSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("import set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "asset path (res://textures/player.png)")
	paramFlags := stringListFlag{}
	fs.Var(&paramFlags, "param", "import param in name=VALUE form where VALUE is raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("import set requires --path")
	}
	params, err := parseNameRawJSONPairs(paramFlags)
	if err != nil {
		return err
	}
	result, err := client.ImportSet(ctx, requestID(), *path, params)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Import set: %s (%d params)\n", result.Path, result.Params)
	return nil
}

func runSceneList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "res://", "res:// directory to search")
	recursive := fs.Bool("recursive", true, "search recursively")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.SceneList(ctx, requestID(), *dir, *recursive)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runResourceList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("resource list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "res://", "res:// directory to search")
	recursive := fs.Bool("recursive", true, "search recursively")
	ext := fs.String("ext", "", "file extension filter (e.g. .tres)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.ResourceList(ctx, requestID(), *dir, *ext, *recursive)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runProjectRun(ctx context.Context, cfg bridge.Config, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("project run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scene := fs.String("scene", "", "scene to run (res://main.tscn); omit to use the project main scene")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum time to wait for Godot to exit")
	godot := fs.String("godot", cfg.GodotPath, "headless Godot binary path")
	project := fs.String("project", cfg.Project, "Godot project path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *godot == "" {
		return fmt.Errorf("project run requires a headless Godot binary: set GDCTL_GODOT_PATH, --godot, or pass --godot PATH")
	}
	if *project == "" {
		return fmt.Errorf("project run requires --project or GDCTL_PROJECT pointing to the Godot project directory")
	}
	return execGodotHeadless(ctx, *godot, *project, *scene, *timeout, stdout)
}

func runSceneRun(ctx context.Context, cfg bridge.Config, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path (res://main.tscn)")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum time to wait for Godot to exit")
	godot := fs.String("godot", cfg.GodotPath, "headless Godot binary path")
	project := fs.String("project", cfg.Project, "Godot project path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("scene run requires --path")
	}
	if *godot == "" {
		return fmt.Errorf("scene run requires a headless Godot binary: set GDCTL_GODOT_PATH, --godot, or pass --godot PATH")
	}
	if *project == "" {
		return fmt.Errorf("scene run requires --project or GDCTL_PROJECT pointing to the Godot project directory")
	}
	return execGodotHeadless(ctx, *godot, *project, *path, *timeout, stdout)
}

func execGodotHeadless(ctx context.Context, godotBin, projectPath, scene string, timeout time.Duration, stdout io.Writer) error {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	argv := []string{"--headless", "--path", projectPath}
	if scene != "" {
		argv = append(argv, scene)
	}
	cmd := exec.CommandContext(runCtx, godotBin, argv...)
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return fmt.Errorf("project run timed out after %s", timeout)
		}
		return fmt.Errorf("project run failed: %w", err)
	}
	return nil
}

func intFromJobResult(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
