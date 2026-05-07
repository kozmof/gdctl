package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	embeddedaddons "gdctl/addons"
	"gdctl/internal/addon"
	"gdctl/internal/bridge"
)

var newAddonManager = func() addon.Manager {
	return addon.NewManager(embeddedaddons.FS)
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg, rest, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}
	cfg, err = cfg.WithProjectToken()
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		printUsage(stderr)
		return fmt.Errorf("missing command")
	}

	client := bridge.NewClient(cfg)
	addonManager := newAddonManager()
	switch rest[0] {
	case "ping":
		return runPing(ctx, client, stdout)
	case "doctor":
		return runDoctor(ctx, cfg, client, addonManager, rest[1:], stdout)
	case "addon":
		return runAddon(ctx, cfg, client, addonManager, rest[1:], stdout)
	case "bridge":
		return runBridge(ctx, client, addonManager, rest[1:], stdout)
	case "scene":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runSceneCreate(ctx, client, rest[2:], stdout)
			case "open":
				return runSceneOpen(ctx, client, rest[2:], stdout)
			case "instance":
				return runSceneInstance(ctx, client, rest[2:], stdout)
			case "tree":
				return runSceneTree(ctx, client, stdout)
			case "save":
				return runSceneSave(ctx, client, rest[2:], stdout)
			}
		}
	case "node":
		if len(rest) >= 2 {
			switch rest[1] {
			case "add":
				return runNodeAdd(ctx, client, rest[2:], stdout)
			case "remove":
				return runNodeRemove(ctx, client, rest[2:], stdout)
			case "rename":
				return runNodeRename(ctx, client, rest[2:], stdout)
			case "move":
				return runNodeMove(ctx, client, rest[2:], stdout)
			case "get":
				return runNodeGet(ctx, client, rest[2:], stdout)
			case "set":
				return runNodeSet(ctx, client, rest[2:], stdout)
			case "attach-script":
				return runNodeAttachScript(ctx, client, rest[2:], stdout)
			}
		}
	case "script":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runScriptCreate(ctx, client, rest[2:], stdout)
			case "write":
				return runScriptWrite(ctx, client, rest[2:], stdout)
			case "check":
				return runScriptCheck(ctx, client, rest[2:], stdout)
			}
		}
	case "viewport":
		if len(rest) >= 2 {
			switch rest[1] {
			case "screenshot":
				return runViewportScreenshot(ctx, client, rest[2:], stdout)
			}
		}
	}

	printUsage(stderr)
	return fmt.Errorf("unknown command: %s", strings.Join(rest, " "))
}

func runBridge(ctx context.Context, client *bridge.Client, manager addon.Manager, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("bridge requires a subcommand")
	}
	switch args[0] {
	case "info":
		ping, err := client.Ping(ctx)
		if err != nil {
			return err
		}
		if !ping.OK {
			return fmt.Errorf("godot bridge returned not ok")
		}
		printBridgeInfo(stdout, ping)
		return nil
	case "logs":
		return runBridgeLogs(ctx, client, args[1:], stdout)
	case "addon-update":
		return runBridgeAddonUpdate(ctx, client, manager, stdout)
	default:
		return fmt.Errorf("unknown bridge command: %s", strings.Join(args, " "))
	}
}

func runBridgeLogs(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("bridge logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "write logs as JSON")
	clear := fs.Bool("clear", false, "clear logs after reading")
	if err := fs.Parse(args); err != nil {
		return err
	}
	entries, err := client.Logs(ctx)
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{"entries": entries}); err != nil {
			return err
		}
	} else {
		for _, entry := range entries {
			fmt.Fprintf(stdout, "%s [%s] %s: %s", entry.Time, entry.Level, entry.Source, entry.Message)
			if len(entry.Detail) > 0 {
				encoded, err := json.Marshal(entry.Detail)
				if err != nil {
					return err
				}
				fmt.Fprintf(stdout, " %s", encoded)
			}
			fmt.Fprintln(stdout)
		}
	}
	if *clear {
		if err := client.ClearLogs(ctx, requestID()); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "Logs cleared")
	}
	return nil
}

func runBridgeAddonUpdate(ctx context.Context, client *bridge.Client, manager addon.Manager, stdout io.Writer) error {
	manifest, files, err := addon.PackageEmbeddedUpdate(manager.Source)
	if err != nil {
		return err
	}
	result, err := client.UpdateAddon(ctx, requestID(), manifest, files)
	if err != nil {
		return err
	}
	if result.Updated {
		fmt.Fprintf(stdout, "Addon updated over bridge: %d files written\n", result.FilesWritten)
	} else {
		fmt.Fprintln(stdout, "Addon already up to date")
	}
	if result.Backup != "" {
		fmt.Fprintf(stdout, "Backup: %s\n", result.Backup)
	}
	if result.ReloadRequired {
		fmt.Fprintln(stdout, "Reload required: disable/enable the Godot plugin or restart Godot")
	}
	return nil
}

func parseGlobalFlags(args []string) (bridge.Config, []string, error) {
	cfg := bridge.ConfigFromEnv()
	fs := flag.NewFlagSet("gdctl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	host := fs.String("host", cfg.Host, "bridge host")
	port := fs.Int("port", cfg.Port, "bridge port")
	token := fs.String("token", cfg.Token, "bridge bearer token")
	project := fs.String("project", cfg.Project, "Godot project path")
	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	cfg.Host = *host
	cfg.Port = *port
	cfg.Token = *token
	cfg.Project = *project
	return cfg, fs.Args(), nil
}

func runPing(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	ping, err := client.Ping(ctx)
	if err != nil {
		return err
	}
	if !ping.OK {
		return fmt.Errorf("godot bridge returned not ok")
	}
	fmt.Fprintln(stdout, "Godot bridge: ok")
	fmt.Fprintf(stdout, "Engine: %s %s\n", ping.Engine, ping.EngineVersion)
	fmt.Fprintf(stdout, "Project: %s\n", ping.ProjectName)
	fmt.Fprintf(stdout, "Plugin: %s\n", ping.PluginVersion)
	return nil
}

func printBridgeInfo(stdout io.Writer, ping bridge.PingResponse) {
	fmt.Fprintln(stdout, "Godot bridge")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  reachable: yes")
	fmt.Fprintf(stdout, "  service: %s\n", valueOrDash(ping.Service))
	fmt.Fprintf(stdout, "  engine: %s %s\n", valueOrDash(ping.Engine), valueOrDash(ping.EngineVersion))
	fmt.Fprintf(stdout, "  project: %s\n", valueOrDash(ping.ProjectName))
	fmt.Fprintf(stdout, "  project_path: %s\n", valueOrDash(ping.ProjectPath))
	fmt.Fprintf(stdout, "  plugin_version: %s\n", valueOrDash(ping.PluginVersion))
	fmt.Fprintf(stdout, "  protocol: %s\n", valueOrDash(ping.ProtocolVersion))
	fmt.Fprintf(stdout, "  auth_enabled: %s\n", yesNo(ping.AuthEnabled))
	fmt.Fprintf(stdout, "  listen: %s:%d\n", valueOrDash(ping.Host), ping.Port)
	fmt.Fprintf(stdout, "  scene_open: %s\n", yesNo(ping.SceneOpen))
	if len(ping.Capabilities) > 0 {
		fmt.Fprintf(stdout, "  capabilities: %s\n", strings.Join(ping.Capabilities, ", "))
	}
}

func runAddon(ctx context.Context, cfg bridge.Config, client *bridge.Client, manager addon.Manager, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("addon requires a subcommand")
	}
	switch args[0] {
	case "install":
		fs := flag.NewFlagSet("addon install", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		force := fs.Bool("force", false, "overwrite conflicting addon files")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Install(addon.InstallOptions{ProjectPath: *project, Force: *force})
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "update":
		fs := flag.NewFlagSet("addon update", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			return runBridgeAddonUpdate(ctx, client, manager, stdout)
		}
		result, err := manager.Update(addon.UpdateOptions{ProjectPath: *project})
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "enable":
		fs := flag.NewFlagSet("addon enable", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Enable(*project)
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "disable":
		fs := flag.NewFlagSet("addon disable", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Disable(*project)
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "remove":
		fs := flag.NewFlagSet("addon remove", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Remove(addon.RemoveOptions{ProjectPath: *project})
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "status":
		fs := flag.NewFlagSet("addon status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		jsonOut := fs.Bool("json", false, "write status as JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			ping, err := client.Ping(ctx)
			if err != nil {
				return err
			}
			return printRuntimeAddonStatus(stdout, ping, *jsonOut)
		}
		status, err := manager.Status(ctx, addon.StatusOptions{ProjectPath: *project, BridgeConfig: cfg, CheckRuntime: true})
		if err != nil {
			return err
		}
		return printAddonStatus(stdout, status, *jsonOut)
	case "doctor":
		fs := flag.NewFlagSet("addon doctor", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		fix := fs.Bool("fix", false, "install and enable the addon when needed")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			ping, err := client.Ping(ctx)
			printRuntimeAddonDoctor(stdout, ping, err)
			return err
		}
		status, actions, err := manager.Doctor(ctx, addon.DoctorOptions{ProjectPath: *project, BridgeConfig: cfg, Fix: *fix})
		printAddonDoctor(stdout, status, actions)
		return err
	default:
		return fmt.Errorf("unknown addon command: %s", strings.Join(args, " "))
	}
}

func printAddonResult(stdout io.Writer, result addon.Result) {
	fmt.Fprintln(stdout, result.Message)
	if result.Backup != "" {
		fmt.Fprintf(stdout, "Backup: %s\n", result.Backup)
	}
}

func printAddonStatus(stdout io.Writer, status addon.Status, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}
	fmt.Fprintln(stdout, "gdctl bridge addon")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "  installed: %s\n", yesNo(status.Installed))
	fmt.Fprintf(stdout, "  path: %s\n", status.AddonPath)
	fmt.Fprintf(stdout, "  enabled: %s\n", yesNo(status.Enabled))
	fmt.Fprintf(stdout, "  version: %s\n", valueOrDash(status.DeclaredVersion))
	fmt.Fprintf(stdout, "  manifest: %s\n", valueOrDash(status.ManifestVersion))
	fmt.Fprintf(stdout, "  embedded: %s\n", status.EmbeddedVersion)
	fmt.Fprintf(stdout, "  reachable: %s\n", yesNo(status.Reachable))
	fmt.Fprintf(stdout, "  runtime: %s\n", valueOrDash(status.RuntimeVersion))
	fmt.Fprintf(stdout, "  protocol: %s\n", valueOrDash(status.ProtocolVersion))
	fmt.Fprintf(stdout, "  compatible: %s (%s)\n", yesNo(status.Compatible), status.Compatibility)
	if status.RuntimeError != "" {
		fmt.Fprintf(stdout, "  runtime_error: %s\n", status.RuntimeError)
	}
	return nil
}

func printRuntimeAddonStatus(stdout io.Writer, ping bridge.PingResponse, jsonOut bool) error {
	compatible := ping.PluginVersion == "" || ping.PluginVersion == "0.1.0"
	if jsonOut {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"installed":        ping.OK,
			"enabled":          ping.OK,
			"reachable":        ping.OK,
			"runtime_version":  ping.PluginVersion,
			"protocol_version": ping.ProtocolVersion,
			"compatible":       compatible,
			"capabilities":     ping.Capabilities,
			"project_path":     ping.ProjectPath,
			"mode":             "runtime",
		})
	}
	fmt.Fprintln(stdout, "gdctl bridge addon")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  mode: runtime")
	fmt.Fprintf(stdout, "  installed: %s\n", yesNo(ping.OK))
	fmt.Fprintf(stdout, "  enabled: %s\n", yesNo(ping.OK))
	fmt.Fprintf(stdout, "  reachable: %s\n", yesNo(ping.OK))
	fmt.Fprintf(stdout, "  runtime: %s\n", valueOrDash(ping.PluginVersion))
	fmt.Fprintf(stdout, "  protocol: %s\n", valueOrDash(ping.ProtocolVersion))
	fmt.Fprintf(stdout, "  compatible: %s\n", yesNo(compatible))
	return nil
}

func printAddonDoctor(stdout io.Writer, status addon.Status, actions []string) {
	fmt.Fprintln(stdout, "gdctl Addon Doctor")
	for _, action := range actions {
		fmt.Fprintf(stdout, "[fix] %s\n", action)
	}
	if status.Installed {
		fmt.Fprintln(stdout, "[ok] addon installed")
	} else {
		fmt.Fprintln(stdout, "[fail] addon not installed")
	}
	if status.Enabled {
		fmt.Fprintln(stdout, "[ok] addon enabled")
	} else {
		fmt.Fprintln(stdout, "[fail] addon not enabled")
	}
	if status.Compatible {
		fmt.Fprintln(stdout, "[ok] addon compatible")
	} else {
		fmt.Fprintf(stdout, "[fail] addon compatibility: %s\n", status.Compatibility)
	}
	if status.Reachable {
		fmt.Fprintln(stdout, "[ok] runtime bridge reachable")
	} else {
		fmt.Fprintln(stdout, "[warn] runtime bridge not reachable")
	}
}

func printRuntimeAddonDoctor(stdout io.Writer, ping bridge.PingResponse, err error) {
	fmt.Fprintln(stdout, "gdctl Addon Doctor")
	fmt.Fprintln(stdout, "[info] projectless runtime mode")
	if err != nil {
		fmt.Fprintf(stdout, "[fail] runtime bridge reachable: %v\n", err)
		return
	}
	if ping.OK {
		fmt.Fprintln(stdout, "[ok] runtime bridge reachable")
		fmt.Fprintf(stdout, "[ok] runtime addon version %s\n", valueOrDash(ping.PluginVersion))
	} else {
		fmt.Fprintln(stdout, "[fail] ping returned not ok")
	}
}

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
	result, err := client.OpenScene(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("scene open did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "scene open")
	if err != nil {
		return err
	}
	pathValue, _ := job.Result["path"].(string)
	if pathValue == "" {
		pathValue = result.Path
	}
	root, _ := job.Result["root"].(string)
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
	result, err := client.SaveScene(ctx, requestID(), "")
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("scene save did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "scene save")
	if err != nil {
		return err
	}
	pathValue, _ := job.Result["path"].(string)
	if pathValue == "" {
		pathValue = result.Path
	}
	fmt.Fprintf(stdout, "Scene saved: %s\n", pathValue)
	return nil
}

func waitForJob(ctx context.Context, client *bridge.Client, jobID string, timeout time.Duration, label string) (bridge.Job, error) {
	deadline := time.Now().Add(timeout)
	for {
		job, err := client.Job(ctx, jobID)
		if err != nil {
			return bridge.Job{}, err
		}
		switch job.Status {
		case "succeeded":
			return job, nil
		case "failed":
			if job.Error != nil {
				return bridge.Job{}, job.Error
			}
			return bridge.Job{}, fmt.Errorf("%s job failed", label)
		}
		if time.Now().After(deadline) {
			return bridge.Job{}, fmt.Errorf("%s timed out waiting for job %s", label, jobID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func runNodeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "parent node path")
	nodeType := fs.String("type", "", "Godot node type")
	name := fs.String("name", "", "new node name")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" || *nodeType == "" || *name == "" {
		return fmt.Errorf("node add requires --parent, --type, and --name")
	}
	result, err := client.AddNode(ctx, requestID(), *parent, *nodeType, *name, *dryRun)
	if err != nil {
		return err
	}
	path, _ := result["path"].(string)
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", path)
		return nil
	}
	fmt.Fprintf(stdout, "Added node: %s\n", path)
	return nil
}

func runNodeRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node remove requires --path")
	}
	result, err := client.RemoveNode(ctx, requestID(), *path, *dryRun)
	if err != nil {
		return err
	}
	removed, _ := result["removed"].(string)
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", removed)
		return nil
	}
	fmt.Fprintf(stdout, "Removed node: %s\n", removed)
	return nil
}

func runNodeRename(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node rename", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	name := fs.String("name", "", "new node name")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("node rename requires --path and --name")
	}
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
}

func runNodeMove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node move", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	parent := fs.String("parent", "", "new parent node path")
	index := fs.Int("index", -1, "optional child index under new parent")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *parent == "" {
		return fmt.Errorf("node move requires --path and --parent")
	}
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
	valueText := fs.String("value", "", "typed JSON value, for example {\"kind\":\"Vector2\",\"value\":[200,400]}")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *property == "" || *valueText == "" {
		return fmt.Errorf("node set requires --path, --property, and --value")
	}
	var value any
	if err := json.Unmarshal([]byte(*valueText), &value); err != nil {
		return fmt.Errorf("node set --value must be typed JSON: %w", err)
	}
	result, err := client.SetNodeProperty(ctx, requestID(), *path, *property, value)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Set %s on %s\n", result.Property, result.Path)
	return nil
}

func runNodeAttachScript(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node attach-script", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	scriptPath := fs.String("script", "", "script resource path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *scriptPath == "" {
		return fmt.Errorf("node attach-script requires --path and --script")
	}
	result, err := client.AttachScript(ctx, requestID(), *path, *scriptPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Attached script: %s -> %s\n", result.Script, result.Path)
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
	result, err := client.WriteScript(ctx, requestID(), *path, bodyText)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("script write produced invalid script: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Script written: %s\n", result.Path)
	return nil
}

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
	content, _ := job.Result["content_base64"].(string)
	if content == "" {
		return fmt.Errorf("viewport screenshot job did not return PNG data")
	}
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return fmt.Errorf("decode screenshot PNG: %w", err)
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
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

func runDoctor(ctx context.Context, cfg bridge.Config, client *bridge.Client, manager addon.Manager, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	project := fs.String("project", cfg.Project, "Godot project path")
	fix := fs.Bool("fix", false, "install and enable the addon when needed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg.Project = *project

	fmt.Fprintln(stdout, "Godot TCP Bridge Doctor")
	fmt.Fprintln(stdout)

	usable := true
	if cfg.Project == "" && *fix {
		if err := runBridgeAddonUpdate(ctx, client, manager, stdout); err != nil {
			usable = false
			fmt.Fprintf(stdout, "[fail] projectless addon update: %v\n", err)
		} else {
			fmt.Fprintln(stdout, "[ok] projectless addon update requested")
		}
	}
	if cfg.Project != "" {
		status, actions, err := manager.Doctor(ctx, addon.DoctorOptions{ProjectPath: cfg.Project, BridgeConfig: cfg, Fix: *fix})
		for _, action := range actions {
			fmt.Fprintf(stdout, "[fix] %s\n", action)
		}
		if err != nil {
			usable = false
			fmt.Fprintf(stdout, "[fail] addon doctor: %v\n", err)
		} else {
			if status.Installed {
				fmt.Fprintln(stdout, "[ok] addon installed")
			}
			if status.Enabled {
				fmt.Fprintln(stdout, "[ok] addon enabled")
			}
			if status.Compatible {
				fmt.Fprintln(stdout, "[ok] addon compatible")
			}
		}
	}

	resolved := false
	if addrs, err := net.LookupHost(cfg.Host); err == nil && len(addrs) > 0 {
		resolved = true
		fmt.Fprintf(stdout, "[ok] %s resolved\n", cfg.Host)
	} else {
		usable = false
		fmt.Fprintf(stdout, "[fail] %s could not be resolved\n", cfg.Host)
	}

	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if resolved {
		if err := client.Dial(dialCtx); err == nil {
			fmt.Fprintf(stdout, "[ok] bridge reachable at %s\n", cfg.Address())
		} else {
			usable = false
			fmt.Fprintf(stdout, "[fail] bridge unreachable at %s: %v\n", cfg.Address(), err)
		}
	}

	ping, err := client.Ping(ctx)
	if err == nil && ping.OK {
		fmt.Fprintln(stdout, "[ok] ping returned ok")
		if ping.ProjectName != "" {
			fmt.Fprintln(stdout, "[ok] Godot editor project is open")
		} else {
			fmt.Fprintln(stdout, "[warn] Godot editor project name is empty")
		}
		if ping.PluginVersion != "" {
			fmt.Fprintf(stdout, "[ok] plugin version %s\n", ping.PluginVersion)
		} else {
			fmt.Fprintln(stdout, "[warn] plugin version is empty")
		}
	} else {
		usable = false
		if err != nil {
			fmt.Fprintf(stdout, "[fail] ping failed: %v\n", err)
		} else {
			fmt.Fprintln(stdout, "[fail] ping returned not ok")
		}
	}

	if cfg.Token == "" {
		fmt.Fprintln(stdout, "[warn] no mutation token configured")
	} else {
		fmt.Fprintln(stdout, "[ok] mutation token configured")
	}
	fmt.Fprintln(stdout, "[warn] no Linux headless Godot configured")
	fmt.Fprintln(stdout)
	if usable {
		fmt.Fprintln(stdout, "Result: usable")
		return nil
	}
	fmt.Fprintln(stdout, "Result: unusable")
	return fmt.Errorf("doctor found bridge problems")
}

func requestID() string {
	return "cli-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] [--project path] ping")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] [--project path] doctor [--project PATH] [--fix]")
	fmt.Fprintln(w, "  gdctl addon install --project PATH [--force]")
	fmt.Fprintln(w, "  gdctl addon enable --project PATH")
	fmt.Fprintln(w, "  gdctl addon disable --project PATH")
	fmt.Fprintln(w, "  gdctl addon status [--project PATH] [--json]")
	fmt.Fprintln(w, "  gdctl addon update [--project PATH]")
	fmt.Fprintln(w, "  gdctl addon remove --project PATH")
	fmt.Fprintln(w, "  gdctl addon doctor [--project PATH] [--fix]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] bridge info")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] bridge logs [--json] [--clear]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] bridge addon-update")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] scene create --path PATH --root TYPE --name NAME [--force]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] scene open --path PATH")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] scene instance --parent PATH --scene SCENE --name NAME")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] scene tree")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] scene save")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node add --parent PATH --type TYPE --name NAME [--dry-run]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node remove --path PATH [--dry-run]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node rename --path PATH --name NAME [--dry-run]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node move --path PATH --parent PARENT [--index N] [--dry-run]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node get --path PATH --property PROPERTY")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node set --path PATH --property PROPERTY --value TYPED_JSON")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node attach-script --path PATH --script SCRIPT")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] script create --path PATH --extends CLASS [--force]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] script write --path PATH (--body TEXT | --body-file FILE)")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] script check --path PATH")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] viewport screenshot --out FILE [--kind 2d|3d] [--index N]")
}
