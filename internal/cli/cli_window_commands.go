package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gdctl/internal/bridge"
)

func runWindow(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("window requires a subcommand: create, assign-viewport")
	}
	switch args[0] {
	case "create":
		return runWindowCreate(ctx, client, args[1:], stdout)
	case "assign-viewport":
		return runWindowAssignViewport(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown window subcommand: %s", args[0])
	}
}

func runWindowCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("window create")
	title := fs.String("title", "Window", "window title")
	width := fs.Int("width", 640, "window width")
	height := fs.Int("height", 480, "window height")
	posStr := fs.String("position", "0,0", "window position as X,Y")
	if err := fs.Parse(args); err != nil {
		return err
	}
	posX, posY, err := parseWindowPos(*posStr)
	if err != nil {
		return fmt.Errorf("window create --position: %w", err)
	}
	result, err := client.WindowCreate(ctx, requestID(), *title, *width, *height, posX, posY)
	if err != nil {
		return err
	}
	windowID, _ := result["window_id"].(float64)
	path, _ := result["path"].(string)
	fmt.Fprintf(stdout, "Window created: %s (id: %d, %dx%d)\n", path, int(windowID), *width, *height)
	return nil
}

func runWindowAssignViewport(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("window assign-viewport")
	windowID := fs.Int("window-id", 0, "window ID from window create")
	viewport := fs.String("viewport", "", "SubViewport node path to assign")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *viewport == "" {
		return fmt.Errorf("window assign-viewport requires --viewport")
	}
	result, err := client.WindowAssignViewport(ctx, requestID(), *windowID, *viewport)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "Viewport assigned: %s -> window %d\n", *viewport, *windowID)
	return nil
}

func parseWindowPos(s string) (int, int, error) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected X,Y but got %q", s)
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("X: %w", err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("Y: %w", err)
	}
	return x, y, nil
}
