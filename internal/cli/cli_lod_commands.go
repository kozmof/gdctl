package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gdctl/internal/bridge"
)

func runLOD(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("lod requires a subcommand: set, set-many")
	}
	switch args[0] {
	case "set":
		return runLODSet(ctx, client, args[1:], stdout)
	case "set-many":
		return runLODSetMany(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown lod subcommand: %s", args[0])
	}
}

func runLODSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("lod set")
	path := fs.String("path", "", "GeometryInstance3D node path")
	begin := fs.Float64("begin", 0, "distance at which LOD begins fading out (0 = disabled)")
	end := fs.Float64("end", 0, "distance at which node disappears (0 = disabled)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("lod set requires --path")
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

func runLODSetMany(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("lod set-many")
	file := fs.String("file", "", "JSON file with LOD configuration array")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("lod set-many requires --file")
	}
	content, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("lod set-many: could not read file: %w", err)
	}
	var entries []bridge.LodEntry
	if err := json.Unmarshal(content, &entries); err != nil {
		return fmt.Errorf("lod set-many: could not parse JSON: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("lod set-many: file contains no entries")
	}
	result, err := client.LodSetMany(ctx, requestID(), entries)
	if err != nil {
		return err
	}
	updated, _ := result["updated"].(float64)
	fmt.Fprintf(stdout, "LOD set-many: %d nodes updated from %s\n", int(updated), *file)
	return nil
}
