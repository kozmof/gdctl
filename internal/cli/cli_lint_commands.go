package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"gdctl/internal/bridge"
)

// runLint checks all scripts and shaders in the project for errors.
// gdctl lint [--dir res://] [--json]
func runLint(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("lint")
	dir := fs.String("dir", "res://", "project directory to lint")
	jsonOut := fs.Bool("json", false, "print results as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	type lintError struct {
		Path    string `json:"path"`
		Kind    string `json:"kind"`
		Message string `json:"message"`
	}
	var errors []lintError

	// Check GDScript files
	scripts, err := client.ResourceList(ctx, requestID(), *dir, ".gd", true)
	if err != nil {
		return fmt.Errorf("lint: could not list scripts: %w", err)
	}
	scriptErrors := 0
	for _, path := range scripts.Resources {
		result, err := client.CheckScript(ctx, requestID(), path)
		if err != nil {
			errors = append(errors, lintError{Path: path, Kind: "script", Message: err.Error()})
			scriptErrors++
			continue
		}
		if !result.Valid {
			errors = append(errors, lintError{Path: path, Kind: "script", Message: "invalid"})
			scriptErrors++
		}
	}

	// Check shader files
	shaders, err := client.ResourceList(ctx, requestID(), *dir, ".gdshader", true)
	if err != nil {
		return fmt.Errorf("lint: could not list shaders: %w", err)
	}
	shaderErrors := 0
	for _, path := range shaders.Resources {
		result, err := client.CheckShader(ctx, requestID(), path)
		if err != nil {
			errors = append(errors, lintError{Path: path, Kind: "shader", Message: err.Error()})
			shaderErrors++
			continue
		}
		if !result.Valid {
			errors = append(errors, lintError{Path: path, Kind: "shader", Message: "invalid"})
			shaderErrors++
		}
	}

	allPassed := len(errors) == 0

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"dir":            *dir,
			"passed":         allPassed,
			"scripts_total":  len(scripts.Resources),
			"scripts_errors": scriptErrors,
			"shaders_total":  len(shaders.Resources),
			"shaders_errors": shaderErrors,
			"errors":         errors,
		})
	}

	fmt.Fprintf(stdout, "Lint: %s\n", *dir)
	fmt.Fprintf(stdout, "  scripts: %d checked, %d error(s)\n", len(scripts.Resources), scriptErrors)
	fmt.Fprintf(stdout, "  shaders: %d checked, %d error(s)\n", len(shaders.Resources), shaderErrors)
	if len(errors) > 0 {
		fmt.Fprintln(stdout)
		for _, e := range errors {
			fmt.Fprintf(stdout, "  FAIL  %s — %s\n", e.Path, e.Message)
		}
		fmt.Fprintf(stdout, "\n%d error(s) found.\n", len(errors))
		return fmt.Errorf("lint: %d error(s)", len(errors))
	}
	fmt.Fprintln(stdout, "\nAll checks passed.")
	return nil
}
