package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gdctl/internal/bridge"
)

func runEnvironment(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("environment requires a subcommand: set-background")
	}
	switch args[0] {
	case "set-background":
		return runEnvironmentSetBackground(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown environment subcommand: %s", args[0])
	}
}

func runEnvironmentSetBackground(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("environment set-background")
	path := fs.String("path", "", "WorldEnvironment node path in the open scene")
	mode := fs.String("mode", "color", "background mode: color, sky, or clear")
	colorStr := fs.String("color", "", "background color R,G,B or R,G,B,A in 0.0–1.0 (required for --mode color)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("environment set-background requires --path")
	}
	modeInt, err := environmentBackgroundModeInt(*mode)
	if err != nil {
		return err
	}
	// Set background_mode via the sub-resource property path syntax node.set("environment:background_mode", N)
	modeValue := map[string]any{"kind": "int", "value": modeInt}
	if _, err := client.SetNodeProperty(ctx, requestID(), *path, "environment:background_mode", modeValue); err != nil {
		return fmt.Errorf("environment set-background: set background_mode: %w", err)
	}
	if strings.ToLower(*mode) == "color" {
		if *colorStr == "" {
			return fmt.Errorf("environment set-background --mode color requires --color R,G,B[,A]")
		}
		parts := strings.Split(*colorStr, ",")
		if len(parts) != 3 && len(parts) != 4 {
			return fmt.Errorf("environment set-background --color must be R,G,B or R,G,B,A")
		}
		floats, err := parseFloatComponents(*colorStr, len(parts))
		if err != nil {
			return fmt.Errorf("environment set-background --color: %w", err)
		}
		colorValue := map[string]any{"kind": "Color", "value": floats}
		if _, err := client.SetNodeProperty(ctx, requestID(), *path, "environment:background_color", colorValue); err != nil {
			return fmt.Errorf("environment set-background: set background_color: %w", err)
		}
		fmt.Fprintf(stdout, "Environment background set: %s (mode: %s, color: %s)\n", *path, *mode, *colorStr)
	} else {
		fmt.Fprintf(stdout, "Environment background set: %s (mode: %s)\n", *path, *mode)
	}
	return nil
}

// environmentBackgroundModeInt maps a mode name to Godot's Environment.BackgroundMode enum value.
// BG_CLEAR_COLOR=0, BG_COLOR=1, BG_SKY=2, BG_CANVAS=3, BG_KEEP=4, BG_CAMERA_FEED=5
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
