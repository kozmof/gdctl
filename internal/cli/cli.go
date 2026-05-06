package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
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
		return runAddon(ctx, cfg, addonManager, rest[1:], stdout)
	case "scene":
		if len(rest) >= 2 && rest[1] == "tree" {
			return runSceneTree(ctx, client, stdout)
		}
	case "node":
		if len(rest) >= 2 {
			switch rest[1] {
			case "add":
				return runNodeAdd(ctx, client, rest[2:], stdout)
			case "remove":
				return runNodeRemove(ctx, client, rest[2:], stdout)
			}
		}
	}

	printUsage(stderr)
	return fmt.Errorf("unknown command: %s", strings.Join(rest, " "))
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

func runAddon(ctx context.Context, cfg bridge.Config, manager addon.Manager, args []string, stdout io.Writer) error {
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

func runSceneTree(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	root, err := client.SceneTree(ctx)
	if err != nil {
		return err
	}
	bridge.RenderTree(stdout, root)
	return nil
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
	if cfg.Project != "" || *fix {
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
	fmt.Fprintln(w, "  gdctl addon status --project PATH [--json]")
	fmt.Fprintln(w, "  gdctl addon update --project PATH")
	fmt.Fprintln(w, "  gdctl addon remove --project PATH")
	fmt.Fprintln(w, "  gdctl addon doctor --project PATH [--fix]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] scene tree")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node add --parent PATH --type TYPE --name NAME [--dry-run]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node remove --path PATH [--dry-run]")
}
