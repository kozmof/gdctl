package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gdctl/internal/bridge"
)

func runAnimationTree(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("animation tree requires a subcommand: add-state, add-transition, blend-space-2d-add, set-param")
	}
	switch args[0] {
	case "add-state":
		return runAnimationTreeAddState(ctx, client, args[1:], stdout)
	case "add-transition":
		return runAnimationTreeAddTransition(ctx, client, args[1:], stdout)
	case "blend-space-2d-add":
		return runAnimationTreeBlendSpace2DAdd(ctx, client, args[1:], stdout)
	case "set-param":
		return runAnimationTreeSetParam(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown animation tree subcommand: %s", args[0])
	}
}

func runAnimationTreeAddState(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("animation tree add-state")
	tree := fs.String("tree", "", "AnimationTree node path")
	name := fs.String("name", "", "state name")
	animation := fs.String("animation", "", "animation name from AnimationPlayer library")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tree == "" || *name == "" {
		return fmt.Errorf("animation tree add-state requires --tree and --name")
	}
	result, err := client.AnimationTreeAddState(ctx, requestID(), *tree, *name, *animation)
	if err != nil {
		return err
	}
	created, _ := result["created"].(bool)
	if created {
		fmt.Fprintf(stdout, "AnimationTree state added: %s (tree: %s)\n", *name, *tree)
	} else {
		fmt.Fprintf(stdout, "AnimationTree state already exists: %s (tree: %s)\n", *name, *tree)
	}
	return nil
}

func runAnimationTreeAddTransition(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("animation tree add-transition")
	tree := fs.String("tree", "", "AnimationTree node path")
	from := fs.String("from", "", "source state name")
	to := fs.String("to", "", "target state name")
	condition := fs.String("condition", "", "advance condition name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tree == "" || *from == "" || *to == "" {
		return fmt.Errorf("animation tree add-transition requires --tree, --from, and --to")
	}
	result, err := client.AnimationTreeAddTransition(ctx, requestID(), *tree, *from, *to, *condition)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "AnimationTree transition added: %s -> %s (tree: %s)\n", *from, *to, *tree)
	return nil
}

func runAnimationTreeBlendSpace2DAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("animation tree blend-space-2d-add")
	tree := fs.String("tree", "", "AnimationTree node path")
	state := fs.String("state", "", "state name to replace with BlendSpace2D")
	blendX := fs.String("blend-x", "", "X axis parameter name")
	blendY := fs.String("blend-y", "", "Y axis parameter name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tree == "" || *state == "" {
		return fmt.Errorf("animation tree blend-space-2d-add requires --tree and --state")
	}
	result, err := client.AnimationTreeBlendSpace2DAdd(ctx, requestID(), *tree, *state, *blendX, *blendY)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "AnimationTree BlendSpace2D added: %s (tree: %s)\n", *state, *tree)
	return nil
}

func runAnimationTreeSetParam(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("animation tree set-param")
	tree := fs.String("tree", "", "AnimationTree node path")
	param := fs.String("param", "", "parameter path (e.g. parameters/playback)")
	vector2 := fs.String("vector2", "", "Vector2 value as X,Y")
	floatVal := fs.Float64("float", 0, "float value")
	boolVal := fs.Bool("bool", false, "bool value")
	intVal := fs.Int("int", 0, "int value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tree == "" || *param == "" {
		return fmt.Errorf("animation tree set-param requires --tree and --param")
	}
	var valueKey string
	var value any
	if *vector2 != "" {
		parts := strings.SplitN(*vector2, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--vector2 must be X,Y")
		}
		x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return fmt.Errorf("--vector2 X: %w", err)
		}
		y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return fmt.Errorf("--vector2 Y: %w", err)
		}
		valueKey = "vector2"
		value = []float64{x, y}
	} else if fs.Lookup("float").Value.String() != "0" {
		valueKey = "float"
		value = *floatVal
	} else if fs.Lookup("bool").Value.String() != "false" {
		valueKey = "bool"
		value = *boolVal
	} else {
		valueKey = "int"
		value = *intVal
	}
	result, err := client.AnimationTreeSetParam(ctx, requestID(), *tree, *param, valueKey, value)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "AnimationTree param set: %s (tree: %s)\n", *param, *tree)
	return nil
}
