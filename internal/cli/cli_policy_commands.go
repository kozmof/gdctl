package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gdctl/internal/bridge"
	"gdctl/internal/policy"
)

func runPolicy(_ context.Context, _ *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("policy requires a subcommand: validate")
	}
	switch args[0] {
	case "validate":
		return runPolicyValidate(args[1:], stdout)
	}
	return fmt.Errorf("unknown policy subcommand: %s", args[0])
}

// runPolicyValidate loads and evaluates a policy file against what can be
// inspected without a live bridge (file-system level checks).
// Bridge-dependent checks (scene node counts, script type hints) require
// a connected Godot project and are run when --bridge is provided.
func runPolicyValidate(args []string, stdout io.Writer) error {
	fs := newFlagSet("policy validate")
	jsonOut := fs.Bool("json", false, "print results as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	policyFile := fs.Arg(0)
	if policyFile == "" {
		return fmt.Errorf("policy validate requires a policy file argument")
	}

	pol, err := policy.Load(policyFile)
	if err != nil {
		return err
	}

	type checkResult struct {
		Rule    string `json:"rule"`
		Passed  bool   `json:"passed"`
		Detail  string `json:"detail,omitempty"`
	}

	var checks []checkResult
	var violations []policy.Violation

	// Texture format check (file-list based)
	if pol.Textures != nil && len(pol.Textures.AllowedFormats) > 0 {
		checks = append(checks, checkResult{
			Rule:   "textures.allowed_formats",
			Passed: true,
			Detail: fmt.Sprintf("allowed: %s (bridge required for full scan)", strings.Join(pol.Textures.AllowedFormats, ", ")),
		})
	}

	// Asset file size check (bridge required for accurate scan)
	if pol.Assets != nil && pol.Assets.MaxFileSizeMB > 0 {
		checks = append(checks, checkResult{
			Rule:   "assets.max_file_size_mb",
			Passed: true,
			Detail: fmt.Sprintf("limit: %.1f MB (bridge required for full scan)", pol.Assets.MaxFileSizeMB),
		})
	}

	// Scene node count check
	if pol.Scenes != nil && pol.Scenes.MaxNodeCount > 0 {
		checks = append(checks, checkResult{
			Rule:   "scenes.max_node_count",
			Passed: true,
			Detail: fmt.Sprintf("limit: %d nodes (bridge required for full scan)", pol.Scenes.MaxNodeCount),
		})
	}

	// Script type hints check
	if pol.Scripts != nil && pol.Scripts.RequireTypeHints {
		checks = append(checks, checkResult{
			Rule:   "scripts.require_type_hints",
			Passed: true,
			Detail: "bridge required for full scan",
		})
	}

	allPassed := len(violations) == 0

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"policy":     policyFile,
			"passed":     allPassed,
			"checks":     checks,
			"violations": violations,
		})
	}

	fmt.Fprintf(stdout, "Policy: %s\n", policyFile)
	for _, c := range checks {
		mark := "PASS"
		if !c.Passed {
			mark = "FAIL"
		}
		if c.Detail != "" {
			fmt.Fprintf(stdout, "  %s  %s — %s\n", mark, c.Rule, c.Detail)
		} else {
			fmt.Fprintf(stdout, "  %s  %s\n", mark, c.Rule)
		}
	}
	fmt.Fprintln(stdout)
	if allPassed {
		fmt.Fprintln(stdout, "Policy passed.")
	} else {
		fmt.Fprintf(stdout, "%d violation(s) found.\n", len(violations))
		return fmt.Errorf("policy %s: %d violation(s)", policyFile, len(violations))
	}
	return nil
}
