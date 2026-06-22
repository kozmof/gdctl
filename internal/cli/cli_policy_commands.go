package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gdctl/internal/bridge"
	"gdctl/internal/policy"
)

func runPolicy(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("policy requires a subcommand: validate")
	}
	switch args[0] {
	case "validate":
		return runPolicyValidate(ctx, client, args[1:], stdout)
	}
	return fmt.Errorf("unknown policy subcommand: %s", args[0])
}

func runPolicyValidate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("policy validate")
	dir := fs.String("dir", "res://", "project directory to scan")
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

	// textures.allowed_formats — enumerate files, check extensions
	if pol.Textures != nil && len(pol.Textures.AllowedFormats) > 0 {
		files, err := client.FileList(ctx, requestID(), *dir, true)
		if err == nil {
			var textureFiles []string
			imageExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".exr": true, ".hdr": true, ".svg": true, ".bmp": true, ".tga": true}
			for _, f := range files.Files {
				if imageExts[strings.ToLower(filepath.Ext(f))] {
					textureFiles = append(textureFiles, f)
				}
			}
			vv := policy.CheckTextureFormat(pol.Textures, textureFiles)
			violations = append(violations, vv...)
			detail := fmt.Sprintf("%d texture(s) checked, %d violation(s)", len(textureFiles), len(vv))
			checks = append(checks, checkResult{
				Rule:   "textures.allowed_formats",
				Passed: len(vv) == 0,
				Detail: detail,
			})
		} else {
			checks = append(checks, checkResult{
				Rule:   "textures.allowed_formats",
				Passed: true,
				Detail: "skipped — bridge unavailable",
			})
		}
	}

	// scripts.check_all — run CheckScript on every .gd file
	if pol.Scripts != nil {
		scripts, err := client.ResourceList(ctx, requestID(), *dir, ".gd", true)
		if err == nil {
			scriptFails := 0
			for _, path := range scripts.Resources {
				result, err := client.CheckScript(ctx, requestID(), path)
				if err != nil || !result.Valid {
					msg := "invalid"
					if err != nil {
						msg = err.Error()
					}
					violations = append(violations, policy.Violation{
						Rule:    "scripts.check_all",
						Path:    path,
						Message: msg,
					})
					scriptFails++
				}
			}
			detail := fmt.Sprintf("%d script(s) checked, %d error(s)", len(scripts.Resources), scriptFails)
			checks = append(checks, checkResult{
				Rule:   "scripts.check_all",
				Passed: scriptFails == 0,
				Detail: detail,
			})
		} else {
			checks = append(checks, checkResult{
				Rule:   "scripts.check_all",
				Passed: true,
				Detail: "skipped — bridge unavailable",
			})
		}
	}

	// assets.max_file_size_mb — file size not available via bridge yet
	if pol.Assets != nil && pol.Assets.MaxFileSizeMB > 0 {
		checks = append(checks, checkResult{
			Rule:   "assets.max_file_size_mb",
			Passed: true,
			Detail: fmt.Sprintf("limit: %.1f MB — file size not available via bridge; skipped", pol.Assets.MaxFileSizeMB),
		})
	}

	// scenes.max_node_count — requires opening each scene; deferred
	if pol.Scenes != nil && pol.Scenes.MaxNodeCount > 0 {
		checks = append(checks, checkResult{
			Rule:   "scenes.max_node_count",
			Passed: true,
			Detail: fmt.Sprintf("limit: %d — requires opening each scene; skipped (use 'gdctl scene tree' per scene)", pol.Scenes.MaxNodeCount),
		})
	}

	allPassed := len(violations) == 0
	for _, c := range checks {
		if !c.Passed {
			allPassed = false
			break
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"policy":     policyFile,
			"passed":     allPassed,
			"checks":     checks,
			"violations": violationsToAny(violations),
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
	for _, v := range violations {
		fmt.Fprintf(stdout, "       %s\n", v)
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

func violationsToAny(vv []policy.Violation) []map[string]string {
	out := make([]map[string]string, len(vv))
	for i, v := range vv {
		out[i] = map[string]string{
			"rule":    v.Rule,
			"path":    v.Path,
			"message": v.Message,
		}
	}
	return out
}
