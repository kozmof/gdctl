package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gdctl/internal/bridge"
)

func runGraphEdit(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("graph-edit requires a subcommand: node-add, connection-add, node-remove")
	}
	switch args[0] {
	case "node-add":
		return runGraphEditNodeAdd(ctx, client, args[1:], stdout)
	case "connection-add":
		return runGraphEditConnectionAdd(ctx, client, args[1:], stdout)
	case "node-remove":
		return runGraphEditNodeRemove(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown graph-edit subcommand: %s", args[0])
	}
}

func runGraphEditNodeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("graph-edit node-add")
	path := fs.String("path", "", "GraphEdit node path")
	name := fs.String("name", "", "GraphNode name/title")
	posStr := fs.String("position", "0,0", "position as X,Y")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("graph-edit node-add requires --path and --name")
	}
	posX, posY, err := parseVec2(*posStr)
	if err != nil {
		return fmt.Errorf("graph-edit node-add --position: %w", err)
	}
	result, err := client.GraphEditNodeAdd(ctx, requestID(), *path, *name, posX, posY)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "GraphNode added: %s (graph: %s)\n", *name, *path)
	return nil
}

func runGraphEditConnectionAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("graph-edit connection-add")
	graph := fs.String("graph", "", "GraphEdit node path")
	from := fs.String("from", "", "source GraphNode name")
	fromPort := fs.Int("from-port", 0, "source port index")
	to := fs.String("to", "", "target GraphNode name")
	toPort := fs.Int("to-port", 0, "target port index")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *graph == "" || *from == "" || *to == "" {
		return fmt.Errorf("graph-edit connection-add requires --graph, --from, and --to")
	}
	result, err := client.GraphEditConnectionAdd(ctx, requestID(), *graph, *from, *fromPort, *to, *toPort)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "GraphEdit connection added: %s:%d -> %s:%d (graph: %s)\n", *from, *fromPort, *to, *toPort, *graph)
	return nil
}

func runGraphEditNodeRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("graph-edit node-remove")
	path := fs.String("path", "", "GraphEdit node path")
	name := fs.String("name", "", "GraphNode name to remove")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("graph-edit node-remove requires --path and --name")
	}
	result, err := client.GraphEditNodeRemove(ctx, requestID(), *path, *name)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "GraphNode removed: %s (graph: %s)\n", *name, *path)
	return nil
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
