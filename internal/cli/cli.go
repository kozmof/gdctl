package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

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
	switch rest[0] {
	case "ping":
		return runPing(ctx, client, stdout)
	case "doctor":
		return runDoctor(ctx, cfg, client, stdout)
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
	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	cfg.Host = *host
	cfg.Port = *port
	cfg.Token = *token
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

func runDoctor(ctx context.Context, cfg bridge.Config, client *bridge.Client, stdout io.Writer) error {
	fmt.Fprintln(stdout, "Godot TCP Bridge Doctor")
	fmt.Fprintln(stdout)

	usable := true
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

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] ping")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] doctor")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] scene tree")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node add --parent PATH --type TYPE --name NAME [--dry-run]")
	fmt.Fprintln(w, "  gdctl [--host host] [--port port] [--token token] node remove --path PATH [--dry-run]")
}
