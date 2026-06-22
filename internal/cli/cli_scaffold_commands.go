package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gdctl/internal/bridge"
)

// runScaffold creates project structure from built-in templates.
// gdctl scaffold <template> --out res://path [--name NAME]
func runScaffold(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("scaffold requires a template: player, scene, autoload, test")
	}
	template := args[0]
	rest := args[1:]

	switch template {
	case "player":
		return runScaffoldPlayer(ctx, client, rest, stdout)
	case "scene":
		return runScaffoldScene(ctx, client, rest, stdout)
	case "autoload":
		return runScaffoldAutoload(ctx, client, rest, stdout)
	case "test":
		return runScaffoldTest(ctx, client, rest, stdout)
	}
	return fmt.Errorf("unknown scaffold template %q; use: player, scene, autoload, test", template)
}

func runScaffoldPlayer(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("scaffold player")
	out := fs.String("out", "", "output scene path (e.g. res://scenes/player.tscn)")
	name := fs.String("name", "Player", "root node name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("scaffold player requires --out")
	}
	scriptPath := strings.TrimSuffix(*out, ".tscn") + ".gd"

	scene, err := client.CreateScene(ctx, requestID(), *out, "CharacterBody3D", *name, false)
	if err != nil {
		return fmt.Errorf("scaffold player: create scene: %w", err)
	}

	scriptBody := playerScriptTemplate(*name)
	_, err = client.CreateScript(ctx, requestID(), scriptPath, "CharacterBody3D", false)
	if err != nil {
		return fmt.Errorf("scaffold player: create script: %w", err)
	}
	_, err = client.WriteScript(ctx, requestID(), scriptPath, scriptBody, true, false)
	if err != nil {
		return fmt.Errorf("scaffold player: write script: %w", err)
	}
	_, err = client.AttachScript(ctx, requestID(), scene.RootPath, scriptPath)
	if err != nil {
		return fmt.Errorf("scaffold player: attach script: %w", err)
	}

	fmt.Fprintf(stdout, "Scaffolded: player\n")
	fmt.Fprintf(stdout, "  Scene:  %s\n", scene.Path)
	fmt.Fprintf(stdout, "  Script: %s\n", scriptPath)
	fmt.Fprintf(stdout, "  Root:   %s (%s)\n", scene.RootPath, scene.RootType)
	return nil
}

func runScaffoldScene(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("scaffold scene")
	out := fs.String("out", "", "output scene path (e.g. res://scenes/world.tscn)")
	root := fs.String("root", "Node3D", "root node type")
	name := fs.String("name", "World", "root node name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("scaffold scene requires --out")
	}
	scene, err := client.CreateScene(ctx, requestID(), *out, *root, *name, false)
	if err != nil {
		return fmt.Errorf("scaffold scene: %w", err)
	}
	fmt.Fprintf(stdout, "Scaffolded: scene\n")
	fmt.Fprintf(stdout, "  Scene: %s\n", scene.Path)
	fmt.Fprintf(stdout, "  Root:  %s (%s)\n", scene.RootPath, scene.RootType)
	return nil
}

func runScaffoldAutoload(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("scaffold autoload")
	out := fs.String("out", "", "output script path (e.g. res://autoloads/game_state.gd)")
	name := fs.String("name", "GameState", "autoload singleton name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("scaffold autoload requires --out")
	}

	body := autoloadScriptTemplate(*name)
	_, err := client.CreateScript(ctx, requestID(), *out, "Node", false)
	if err != nil {
		return fmt.Errorf("scaffold autoload: create script: %w", err)
	}
	_, err = client.WriteScript(ctx, requestID(), *out, body, true, false)
	if err != nil {
		return fmt.Errorf("scaffold autoload: write script: %w", err)
	}

	fmt.Fprintf(stdout, "Scaffolded: autoload\n")
	fmt.Fprintf(stdout, "  Script: %s\n", *out)
	fmt.Fprintf(stdout, "  Next:   gdctl autoload add --name %s --path %s\n", *name, *out)
	return nil
}

func runScaffoldTest(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("scaffold test")
	out := fs.String("out", "", "output test script path (e.g. res://tests/test_player.gd)")
	subject := fs.String("subject", "Subject", "name of the class under test")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("scaffold test requires --out")
	}

	body := testScriptTemplate(*subject)
	_, err := client.CreateScript(ctx, requestID(), *out, "GdUnitTestSuite", false)
	if err != nil {
		// GdUnitTestSuite may not be available — fall back to Node
		_, err = client.CreateScript(ctx, requestID(), *out, "Node", false)
		if err != nil {
			return fmt.Errorf("scaffold test: create script: %w", err)
		}
	}
	_, err = client.WriteScript(ctx, requestID(), *out, body, true, false)
	if err != nil {
		return fmt.Errorf("scaffold test: write script: %w", err)
	}

	fmt.Fprintf(stdout, "Scaffolded: test\n")
	fmt.Fprintf(stdout, "  Script: %s\n", *out)
	fmt.Fprintf(stdout, "  Next:   gdctl test gdscript --path %s\n", *out)
	return nil
}

func playerScriptTemplate(name string) string {
	return fmt.Sprintf(`## %s — generated by gdctl scaffold player
extends CharacterBody3D

const SPEED := 5.0
const JUMP_VELOCITY := 4.5

func _physics_process(delta: float) -> void:
	if not is_on_floor():
		velocity += get_gravity() * delta
	if Input.is_action_just_pressed("ui_accept") and is_on_floor():
		velocity.y = JUMP_VELOCITY
	var direction := Input.get_vector("ui_left", "ui_right", "ui_up", "ui_down")
	if direction:
		velocity.x = direction.x * SPEED
		velocity.z = direction.y * SPEED
	else:
		velocity.x = move_toward(velocity.x, 0, SPEED)
		velocity.z = move_toward(velocity.z, 0, SPEED)
	move_and_slide()
`, name)
}

func autoloadScriptTemplate(name string) string {
	return fmt.Sprintf(`extends Node
## %s — autoload singleton

signal state_changed

func _ready() -> void:
	pass
`, name)
}

func testScriptTemplate(subject string) string {
	return fmt.Sprintf(`extends Node
## Tests for %s

func test_%s_exists() -> void:
	assert(true, "%s exists")
`, subject, strings.ToLower(subject), subject)
}
