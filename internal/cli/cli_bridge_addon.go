package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"gdctl/internal/addon"
	"gdctl/internal/bridge"
	"gdctl/internal/version"
)

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
	// Load the currently installed manifest so the bridge can remove stale files.
	// This is best-effort: if we can't read or decode it, we proceed without a
	// stale-file list rather than failing the update.
	var oldManifestMap map[string]any
	if ping, err := client.Ping(ctx); err == nil && ping.ProjectPath != "" {
		if installed, err := addon.LoadInstalledManifest(ping.ProjectPath); err == nil && len(installed.Files) > 0 {
			if data, err := json.Marshal(installed); err == nil {
				if err := json.Unmarshal(data, &oldManifestMap); err != nil {
					oldManifestMap = nil
				}
			}
		}
	}
	result, err := client.UpdateAddon(ctx, requestID(), manifest, oldManifestMap, files)
	if err != nil {
		return err
	}
	if result.Updated {
		fmt.Fprintf(stdout, "Addon updated over bridge: %d files written\n", result.FilesWritten)
	} else {
		fmt.Fprintln(stdout, "Addon already up to date")
	}
	if result.FilesRemoved > 0 {
		fmt.Fprintf(stdout, "Stale files removed: %d\n", result.FilesRemoved)
	}
	if result.Backup != "" {
		fmt.Fprintf(stdout, "Backup: %s\n", result.Backup)
	}
	if result.ReloadRequired {
		fmt.Fprintln(stdout, "Reload required: disable/enable the Godot plugin or restart Godot")
	}
	return nil
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
	case "rollback":
		fs := flag.NewFlagSet("addon rollback", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		backup := fs.String("backup", "", "backup directory to restore; defaults to latest")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			return fmt.Errorf("addon rollback requires --project")
		}
		result, err := manager.Rollback(addon.RollbackOptions{ProjectPath: *project, BackupPath: *backup})
		if err != nil {
			return err
		}
		printAddonRollbackResult(stdout, result)
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

func printAddonRollbackResult(stdout io.Writer, result addon.Result) {
	fmt.Fprintln(stdout, result.Message)
	if result.Backup != "" {
		fmt.Fprintf(stdout, "Restored: %s\n", result.Backup)
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
	compatible := ping.PluginVersion == "" || ping.PluginVersion == version.EmbeddedBridgeVersion
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
	if cfg.GodotPath == "" {
		fmt.Fprintln(stdout, "[warn] no headless Godot configured (set GDCTL_GODOT_PATH or --godot)")
	} else {
		fmt.Fprintf(stdout, "[ok] headless Godot: %s\n", cfg.GodotPath)
	}
	fmt.Fprintln(stdout)
	if usable {
		fmt.Fprintln(stdout, "Result: usable")
		return nil
	}
	fmt.Fprintln(stdout, "Result: unusable")
	return fmt.Errorf("doctor found bridge problems")
}
