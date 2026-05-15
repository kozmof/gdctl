package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gdctl/internal/bridge"
)

func runCSG(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("csg requires a subcommand: node-add, operation-set, size-set")
	}
	switch args[0] {
	case "node-add":
		return runCSGNodeAdd(ctx, client, args[1:], stdout)
	case "operation-set":
		return runCSGOperationSet(ctx, client, args[1:], stdout)
	case "size-set":
		return runCSGSizeSet(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown csg subcommand: %s", args[0])
	}
}

func runCSGNodeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("csg node-add")
	parent := fs.String("parent", "", "parent node path")
	csgType := fs.String("type", "CSGBox3D", "CSG node type: CSGBox3D, CSGSphere3D, CSGCylinder3D, CSGCombiner3D")
	name := fs.String("name", "", "node name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" {
		return fmt.Errorf("csg node-add requires --parent")
	}
	nodeArgs := []string{"--parent", *parent, "--type", *csgType}
	if *name != "" {
		nodeArgs = append(nodeArgs, "--name", *name)
	}
	return runNodeAdd(ctx, client, nodeArgs, stdout)
}

func runCSGOperationSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("csg operation-set")
	path := fs.String("path", "", "CSG node path")
	operation := fs.String("operation", "union", "CSG operation: union, intersection, subtraction")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("csg operation-set requires --path")
	}
	opInt := csgOperationInt(*operation)
	setArgs := []string{"--path", *path, "--property", "operation", "--int", fmt.Sprintf("%d", opInt)}
	if err := runNodeSet(ctx, client, setArgs, stdout); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "CSG operation set: %s = %s\n", *path, *operation)
	return nil
}

func runCSGSizeSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("csg size-set")
	path := fs.String("path", "", "CSGBox3D node path")
	sizeStr := fs.String("size", "1,1,1", "size as X,Y,Z")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("csg size-set requires --path")
	}
	setArgs := []string{"--path", *path, "--property", "size", "--vector3", *sizeStr}
	if err := runNodeSet(ctx, client, setArgs, stdout); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "CSG size set: %s = %s\n", *path, *sizeStr)
	return nil
}

func csgOperationInt(op string) int {
	switch strings.ToLower(op) {
	case "union":
		return 0
	case "intersection":
		return 1
	case "subtraction", "subtract":
		return 2
	}
	return 0
}
