package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"gdctl/internal/bridge"
)

func runTest(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("test requires a subcommand")
	}
	switch args[0] {
	case "gdscript":
		return runTestGDScript(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown test command: %s", args[0])
	}
}

func runTestGDScript(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("test gdscript")
	path := fs.String("path", "", "GDScript test file path")
	dir := fs.String("dir", "", "GDScript test directory")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum time to wait for test job")
	jsonOut := fs.Bool("json", false, "print result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*path == "" && *dir == "") || (*path != "" && *dir != "") {
		return fmt.Errorf("test gdscript requires exactly one of --path or --dir")
	}
	if *path != "" && (!isResPath(*path) || !hasSuffix(*path, ".gd")) {
		return fmt.Errorf("test gdscript --path must be a res:// .gd path")
	}
	if *dir != "" && !isResPath(*dir) {
		return fmt.Errorf("test gdscript --dir must be a res:// path")
	}
	queued, err := client.TestGDScript(ctx, requestID(), *path, *dir)
	if err != nil {
		return err
	}
	if queued.JobID == "" {
		return fmt.Errorf("test gdscript did not return a job id")
	}
	job, err := waitForJob(ctx, client, queued.JobID, *timeout, "test gdscript")
	if err != nil {
		return err
	}
	var result bridge.GDScriptTestResult
	if err := mapJobResult(job.Result, &result); err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return err
		}
	} else {
		printGDScriptTestResult(stdout, result)
	}
	if !result.Passed {
		return fmt.Errorf("gdscript tests failed: %d/%d failed", result.FailedCount, result.Total)
	}
	return nil
}

func printGDScriptTestResult(stdout io.Writer, result bridge.GDScriptTestResult) {
	status := "PASS"
	if !result.Passed {
		status = "FAIL"
	}
	fmt.Fprintf(stdout, "%s gdscript tests: %d passed, %d failed, %d total (%dms)\n", status, result.PassedCount, result.FailedCount, result.Total, result.DurationMS)
	for _, file := range result.Files {
		if file.Passed {
			continue
		}
		fmt.Fprintf(stdout, "\n%s\n", file.Path)
		for _, test := range file.Tests {
			if test.Status == "passed" {
				continue
			}
			fmt.Fprintf(stdout, "  %s: %s\n", test.Name, test.Status)
			for _, failure := range test.Failures {
				if failure.Code != "" {
					fmt.Fprintf(stdout, "    [%s] %s\n", failure.Code, failure.Message)
				} else {
					fmt.Fprintf(stdout, "    %s\n", failure.Message)
				}
			}
		}
	}
}

func mapJobResult(result map[string]any, target any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func isResPath(path string) bool {
	return len(path) >= len("res://") && path[:len("res://")] == "res://"
}

func hasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
