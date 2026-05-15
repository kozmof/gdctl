package cli

import (
	"context"
	"fmt"
	"io"

	"gdctl/internal/bridge"
)

func runSoftBody(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("softbody requires a subcommand: pin-point, unpin-point")
	}
	switch args[0] {
	case "pin-point":
		return runSoftBodyPin(ctx, client, args[1:], stdout, true)
	case "unpin-point":
		return runSoftBodyPin(ctx, client, args[1:], stdout, false)
	default:
		return fmt.Errorf("unknown softbody subcommand: %s", args[0])
	}
}

func runSoftBodyPin(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer, pin bool) error {
	op := "pin-point"
	if !pin {
		op = "unpin-point"
	}
	fs := newFlagSet("softbody " + op)
	path := fs.String("path", "", "SoftBody3D node path")
	point := fs.Int("point", -1, "vertex index to pin/unpin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *point < 0 {
		return fmt.Errorf("softbody %s requires --path and --point >= 0", op)
	}
	var result map[string]any
	var err error
	if pin {
		result, err = client.SoftBodyPinPoint(ctx, requestID(), *path, *point)
	} else {
		result, err = client.SoftBodyUnpinPoint(ctx, requestID(), *path, *point)
	}
	if err != nil {
		return err
	}
	_ = result
	if pin {
		fmt.Fprintf(stdout, "SoftBody3D point pinned: %s[%d]\n", *path, *point)
	} else {
		fmt.Fprintf(stdout, "SoftBody3D point unpinned: %s[%d]\n", *path, *point)
	}
	return nil
}
