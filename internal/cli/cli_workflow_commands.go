package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"gdctl/internal/bridge"
)

// runApply is the top-level desired-state command.
// gdctl apply <file> --scene <path> [--dry-run] [--json]
func runApply(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("apply")
	scene := fs.String("scene", "", "scene path to open and mutate")
	dryRun := fs.Bool("dry-run", false, "validate without mutating or saving")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for open/save jobs")
	jsonOut := fs.Bool("json", false, "print result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	file := fs.Arg(0)
	if *scene == "" || file == "" {
		return fmt.Errorf("apply requires <file> and --scene <path>")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return fmt.Errorf("apply: file must be JSON: %w", err)
	}
	openedPath, root, err := openSceneAndWait(ctx, client, *scene, *timeout)
	if err != nil {
		return err
	}
	result, err := client.ApplyScene(ctx, requestID(), tree, *dryRun)
	if err != nil {
		return err
	}
	if *dryRun {
		if *jsonOut {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{
				"scene":      openedPath,
				"dry_run":    true,
				"created":    result.Created,
				"properties": result.Updated,
			})
		}
		fmt.Fprintf(stdout, "Plan: %s (created: %d, properties: %d) — dry run\n", openedPath, result.Created, result.Updated)
		return nil
	}
	savedPath, err := saveSceneAndWait(ctx, client, *timeout)
	if err != nil {
		return err
	}
	if root == "" {
		root = result.Root
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"scene":      savedPath,
			"root":       root,
			"created":    result.Created,
			"properties": result.Updated,
		})
	}
	fmt.Fprintf(stdout, "Applied: %s\n", savedPath)
	if root != "" {
		fmt.Fprintf(stdout, "Root: %s\n", root)
	}
	fmt.Fprintf(stdout, "Created: %d\n", result.Created)
	fmt.Fprintf(stdout, "Properties: %d\n", result.Updated)
	return nil
}

// runPlan is a dry-run apply that shows what would change.
// gdctl plan <file> --scene <path> [--json]
func runPlan(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("plan")
	scene := fs.String("scene", "", "scene path to preview")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open job")
	jsonOut := fs.Bool("json", false, "print result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	file := fs.Arg(0)
	if *scene == "" || file == "" {
		return fmt.Errorf("plan requires <file> and --scene <path>")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return fmt.Errorf("plan: file must be JSON: %w", err)
	}
	openedPath, _, err := openSceneAndWait(ctx, client, *scene, *timeout)
	if err != nil {
		return err
	}
	result, err := client.ApplyScene(ctx, requestID(), tree, true)
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"scene":      openedPath,
			"dry_run":    true,
			"created":    result.Created,
			"properties": result.Updated,
		})
	}
	fmt.Fprintf(stdout, "Plan: %s\n", openedPath)
	fmt.Fprintf(stdout, "  Nodes to create:     %d\n", result.Created)
	fmt.Fprintf(stdout, "  Properties to update: %d\n", result.Updated)
	fmt.Fprintln(stdout, "  (dry run — no changes written)")
	return nil
}

// gateCheckResult holds the outcome of a single gate check.
type gateCheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// runGate is the unified validation entrypoint.
// gdctl gate run <profile> [--json]
func runGate(ctx context.Context, client *bridge.Client, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("gate requires a subcommand: run")
	}
	switch args[0] {
	case "run":
		return runGateRun(ctx, client, args[1:], stdout, stderr)
	}
	return fmt.Errorf("unknown gate subcommand: %s", args[0])
}

func runGateRun(ctx context.Context, client *bridge.Client, args []string, stdout, stderr io.Writer) error {
	// Pull the profile name out before flag parsing so that
	// `gate run quick --json` and `gate run --json quick` both work.
	profile := "quick"
	flagArgs := args
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		profile = args[0]
		flagArgs = args[1:]
	}
	fs := newFlagSet("gate run")
	jsonOut := fs.Bool("json", false, "print results as JSON")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	checks := gateChecksForProfile(profile)
	if len(checks) == 0 {
		return fmt.Errorf("unknown gate profile %q; use: production, ci, quick", profile)
	}

	results := make([]gateCheckResult, 0, len(checks))
	allPassed := true

	for _, check := range checks {
		res := runGateCheck(ctx, client, check, stderr)
		results = append(results, res)
		if !res.Passed {
			allPassed = false
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"profile": profile,
			"passed":  allPassed,
			"checks":  results,
		})
	}

	fmt.Fprintf(stdout, "Gate: %s\n\n", profile)
	for _, r := range results {
		mark := "PASS"
		if !r.Passed {
			mark = "FAIL"
		}
		if r.Detail != "" {
			fmt.Fprintf(stdout, "  %s  %s — %s\n", mark, r.Name, r.Detail)
		} else {
			fmt.Fprintf(stdout, "  %s  %s\n", mark, r.Name)
		}
	}
	fmt.Fprintln(stdout)
	if allPassed {
		fmt.Fprintln(stdout, "All checks passed.")
	} else {
		fmt.Fprintln(stdout, "Gate failed.")
		return fmt.Errorf("gate %s: checks failed", profile)
	}
	return nil
}

type gateCheck struct {
	name string
	run  func(ctx context.Context, client *bridge.Client, stderr io.Writer) (passed bool, detail string)
}

func gateChecksForProfile(profile string) []gateCheck {
	ping := gateCheck{
		name: "bridge",
		run: func(ctx context.Context, client *bridge.Client, _ io.Writer) (bool, string) {
			ping, err := client.Ping(ctx)
			if err != nil || !ping.OK {
				return false, "bridge unreachable"
			}
			return true, fmt.Sprintf("%s %s", ping.Engine, ping.EngineVersion)
		},
	}
	scripts := gateCheck{
		name: "scripts",
		run: func(ctx context.Context, client *bridge.Client, _ io.Writer) (bool, string) {
			var buf bytes.Buffer
			err := runTest(ctx, client, []string{"gdscript"}, &buf, &buf)
			if err != nil {
				return false, err.Error()
			}
			return true, buf.String()
		},
	}

	switch profile {
	case "production":
		return []gateCheck{ping, scripts}
	case "ci":
		return []gateCheck{ping, scripts}
	case "quick":
		return []gateCheck{ping}
	}
	return nil
}

func runGateCheck(ctx context.Context, client *bridge.Client, check gateCheck, stderr io.Writer) gateCheckResult {
	passed, detail := check.run(ctx, client, stderr)
	return gateCheckResult{Name: check.name, Passed: passed, Detail: detail}
}
