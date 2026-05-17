package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

func runRun(ctx context.Context, client *bridge.Client, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires a subcommand")
	}
	switch args[0] {
	case "start":
		return runRunStart(ctx, client, args[1:], stdout)
	case "status":
		return runRunStatus(ctx, client, args[1:], stdout)
	case "helper-status":
		return runRunHelperStatus(ctx, client, args[1:], stdout)
	case "stop":
		return runRunStop(ctx, client, stdout)
	case "logs":
		return runRunLogs(ctx, client, args[1:], stdout)
	case "screenshot":
		return runRunScreenshot(ctx, client, args[1:], stdout, stderr)
	case "input":
		return runRunInput(ctx, client, args[1:], stdout)
	case "wait-probe":
		return runRunWaitProbe(ctx, client, args[1:], stdout)
	case "smoke":
		return runRunSmoke(ctx, client, args[1:], stdout)
	case "probe":
		return runRunProbe(ctx, client, args[1:], stdout)
	case "instantiate":
		return runRunInstantiate(ctx, client, args[1:], stdout)
	case "scene-reload":
		return runRunSceneReload(ctx, client, args[1:], stdout)
	case "profile":
		return runRunProfile(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown run command: %s", strings.Join(args, " "))
	}
}

func runRunStart(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scene := fs.String("scene", "", "scene path to run")
	main := fs.Bool("main", false, "run the project main scene")
	clearLogs := fs.Bool("clear-logs", true, "clear runtime logs before starting")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scene != "" && *main {
		return fmt.Errorf("run start requires at most one of --scene or --main")
	}
	result, err := client.RunStart(ctx, requestID(), *scene, *main, *clearLogs)
	if err != nil {
		return err
	}
	if result.Scene != "" {
		fmt.Fprintf(stdout, "Run started: %s\n", result.Scene)
	} else if result.PlayingScene != "" {
		fmt.Fprintf(stdout, "Run started: %s\n", result.PlayingScene)
	} else {
		fmt.Fprintln(stdout, "Run started")
	}
	return nil
}

func runRunStatus(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "print status as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.RunStatus(ctx, requestID())
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	if result.Running {
		if result.Debugger.Paused {
			location := result.Debugger.File
			if result.Debugger.Line > 0 {
				location = fmt.Sprintf("%s:%d", location, result.Debugger.Line)
			}
			if location != "" && result.Debugger.Message != "" {
				fmt.Fprintf(stdout, "Run status: paused (%s) %s %s\n", result.PlayingScene, location, result.Debugger.Message)
			} else if result.Debugger.Message != "" {
				fmt.Fprintf(stdout, "Run status: paused (%s) %s\n", result.PlayingScene, result.Debugger.Message)
			} else {
				fmt.Fprintf(stdout, "Run status: paused (%s)\n", result.PlayingScene)
			}
			// Print typed stack frames if available (from bridge_server debugger capture).
			for i, frame := range result.Debugger.StackFrames {
				loc := frame.File
				if frame.Line > 0 {
					loc = fmt.Sprintf("%s:%d", frame.File, frame.Line)
				}
				if frame.Function != "" {
					fmt.Fprintf(stdout, "  #%d %s in %s\n", i, frame.Function, loc)
				} else {
					fmt.Fprintf(stdout, "  #%d %s\n", i, loc)
				}
			}
			// Fallback: raw stack entries from older bridge versions.
			if len(result.Debugger.StackFrames) == 0 {
				for i, raw := range result.Debugger.Stack {
					file, _ := raw["file"].(string)
					fn, _ := raw["function"].(string)
					line := 0
					if v, ok := raw["line"]; ok {
						switch lv := v.(type) {
						case float64:
							line = int(lv)
						case int:
							line = lv
						}
					}
					loc := file
					if line > 0 {
						loc = fmt.Sprintf("%s:%d", file, line)
					}
					if fn != "" {
						fmt.Fprintf(stdout, "  #%d %s in %s\n", i, fn, loc)
					} else if loc != "" {
						fmt.Fprintf(stdout, "  #%d %s\n", i, loc)
					}
				}
			}
			return nil
		}
		if !result.RuntimeHelper.Present {
			if result.PlayingScene != "" {
				fmt.Fprintf(stdout, "Run status: running without runtime helper (%s)\n", result.PlayingScene)
			} else {
				fmt.Fprintln(stdout, "Run status: running without runtime helper")
			}
		} else if result.PlayingScene != "" {
			fmt.Fprintf(stdout, "Run status: running (%s)\n", result.PlayingScene)
		} else {
			fmt.Fprintln(stdout, "Run status: running")
		}
		printRuntimeHelperSummary(stdout, result.RuntimeHelper)
		return nil
	}
	fmt.Fprintln(stdout, "Run status: stopped")
	printRuntimeHelperSummary(stdout, result.RuntimeHelper)
	return nil
}

func runRunHelperStatus(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run helper-status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "print helper status as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.RunStatus(ctx, requestID())
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.RuntimeHelper)
	}
	if result.RuntimeHelper.Present {
		fmt.Fprintln(stdout, "Runtime helper: present")
	} else {
		fmt.Fprintln(stdout, "Runtime helper: not present")
	}
	if result.RuntimeHelper.AutoloadConfigured {
		fmt.Fprintf(stdout, "Autoload: configured (%s)\n", result.RuntimeHelper.Path)
	} else {
		fmt.Fprintf(stdout, "Autoload: not configured (%s)\n", result.RuntimeHelper.Path)
	}
	if result.RuntimeHelper.LastSeen != "" {
		fmt.Fprintf(stdout, "Last seen: %s (%s)\n", result.RuntimeHelper.LastSeen, result.RuntimeHelper.LastMessage)
	}
	if result.RuntimeHelper.Error != "" {
		fmt.Fprintf(stdout, "Issue: %s\n", result.RuntimeHelper.Error)
	}
	return nil
}

func printRuntimeHelperSummary(stdout io.Writer, helper bridge.RuntimeHelperStatus) {
	if helper.Path == "" && helper.LastSeen == "" && helper.Error == "" && !helper.Present && !helper.AutoloadConfigured {
		return
	}
	if helper.Present {
		if helper.LastSeen != "" {
			fmt.Fprintf(stdout, "Runtime helper: present (last seen %s)\n", helper.LastSeen)
		} else {
			fmt.Fprintln(stdout, "Runtime helper: present")
		}
		return
	}
	if helper.Error != "" {
		fmt.Fprintf(stdout, "Runtime helper: not present (%s)\n", helper.Error)
	} else {
		fmt.Fprintln(stdout, "Runtime helper: not present")
	}
}

func requireRuntimeHelper(ctx context.Context, client *bridge.Client, commandName string) error {
	status, err := client.RunStatus(ctx, requestID())
	if err != nil {
		return fmt.Errorf("%s runtime helper preflight: %w", commandName, err)
	}
	if status.RuntimeHelper.Present {
		return nil
	}
	diagnostic := runtimeHelperDiagnostic(status)
	if diagnostic == "" {
		diagnostic = "runtime helper is not present"
	}
	return fmt.Errorf("%s requires the gdctl runtime helper; %s", commandName, diagnostic)
}

func runtimeHelperDiagnostic(status bridge.RunStatusResult) string {
	parts := []string{}
	if status.Running {
		if status.PlayingScene != "" {
			parts = append(parts, fmt.Sprintf("run is running (%s)", status.PlayingScene))
		} else {
			parts = append(parts, "run is running")
		}
	} else {
		parts = append(parts, "run is stopped")
	}
	if status.RuntimeHelper.Error != "" {
		parts = append(parts, "runtime helper not present: "+status.RuntimeHelper.Error)
	} else {
		parts = append(parts, "runtime helper not present")
	}
	if status.RuntimeHelper.AutoloadConfigured {
		parts = append(parts, "try restarting the scene with gdctl run start --scene <scene>, or restart/reload Godot if the autoload was just configured")
	} else {
		parts = append(parts, "try gdctl run start --scene <scene> to configure and start the runtime helper")
	}
	return strings.Join(parts, "; ")
}

func runRunInput(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run input", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "input JSON file path")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for input job")
	summaryProbe := fs.String("summary-probe", "", "after completion print latest probe from this source")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *filePath == "" {
		return fmt.Errorf("run input requires --file")
	}
	content, err := os.ReadFile(*filePath)
	if err != nil {
		return fmt.Errorf("read input file: %w", err)
	}
	var payload struct {
		Steps []any `json:"steps"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return fmt.Errorf("run input --file must be JSON: %w", err)
	}
	if len(payload.Steps) == 0 {
		return fmt.Errorf("run input file requires at least one step")
	}
	if err := validateRunInputSteps(payload.Steps); err != nil {
		return err
	}
	if err := requireRuntimeHelper(ctx, client, "run input"); err != nil {
		return err
	}
	result, err := client.RunInput(ctx, requestID(), payload.Steps)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("run input did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "run input")
	if err != nil {
		return err
	}
	steps := intFromJobResult(job.Result["steps"])
	duration := intFromJobResult(job.Result["duration_ms"])
	if duration > 0 {
		fmt.Fprintf(stdout, "Run input completed: %d steps (%dms)\n", steps, duration)
	} else {
		fmt.Fprintf(stdout, "Run input completed: %d steps\n", steps)
	}
	if *summaryProbe != "" {
		entries, err := client.RunLogs(ctx)
		if err == nil {
			entries = filterLogEntries(entries, *summaryProbe, true, false)
			if len(entries) > 0 {
				detail := entries[len(entries)-1].Detail
				if encoded, err := json.Marshal(detail); err == nil {
					fmt.Fprintf(stdout, "Probe [%s]: %s\n", *summaryProbe, encoded)
				}
			}
		}
	}
	return nil
}

func validateRunInputSteps(steps []any) error {
	for i, raw := range steps {
		step, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("run input step %d must be an object", i)
		}
		typ, ok := runInputString(step, "type")
		if !ok || typ == "" {
			return fmt.Errorf("run input step %d requires type", i)
		}
		switch typ {
		case "wait":
			if _, err := runInputNumber(step, "ms", true); err != nil {
				return fmt.Errorf("run input step %d wait %w", i, err)
			}
		case "key":
			key, ok := runInputString(step, "key")
			if !ok || key == "" {
				return fmt.Errorf("run input step %d key requires key", i)
			}
			if err := validateRunInputMode(step, "action", fmt.Sprintf("run input step %d key action", i)); err != nil {
				return err
			}
			if _, err := runInputNumber(step, "duration_ms", false); err != nil {
				return fmt.Errorf("run input step %d key %w", i, err)
			}
		case "mouse_button":
			button, ok := runInputString(step, "button")
			if !ok || button == "" {
				return fmt.Errorf("run input step %d mouse_button requires button", i)
			}
			switch strings.ToLower(button) {
			case "left", "right", "middle", "1", "2", "3":
			default:
				return fmt.Errorf("run input step %d mouse_button button must be left, right, middle, 1, 2, or 3", i)
			}
			if err := validateRunInputMode(step, "action", fmt.Sprintf("run input step %d mouse_button action", i)); err != nil {
				return err
			}
			if _, err := runInputNumber(step, "duration_ms", false); err != nil {
				return fmt.Errorf("run input step %d mouse_button %w", i, err)
			}
		case "mouse_motion":
			if _, hasDX := step["dx"]; hasDX {
				if _, hasDY := step["dy"]; hasDY {
					return fmt.Errorf("run input step %d mouse_motion requires relative: [x, y]; got dx/dy", i)
				}
			}
			relative, ok := step["relative"]
			if !ok {
				return fmt.Errorf("run input step %d mouse_motion requires relative: [x, y]", i)
			}
			items, ok := relative.([]any)
			if !ok || len(items) != 2 {
				return fmt.Errorf("run input step %d mouse_motion relative must be [x, y]", i)
			}
			for j, item := range items {
				if _, ok := item.(float64); !ok {
					return fmt.Errorf("run input step %d mouse_motion relative[%d] must be numeric", i, j)
				}
			}
		case "action":
			action, ok := runInputString(step, "action")
			if !ok || action == "" {
				return fmt.Errorf("run input step %d action requires action", i)
			}
			if err := validateRunInputMode(step, "mode", fmt.Sprintf("run input step %d action mode", i)); err != nil {
				return err
			}
			if _, err := runInputNumber(step, "duration_ms", false); err != nil {
				return fmt.Errorf("run input step %d action %w", i, err)
			}
		default:
			return fmt.Errorf("run input step %d unsupported type: %s", i, typ)
		}
	}
	return nil
}

func runInputString(step map[string]any, key string) (string, bool) {
	val, ok := step[key]
	if !ok {
		return "", false
	}
	text, ok := val.(string)
	return text, ok
}

func runInputNumber(step map[string]any, key string, required bool) (float64, error) {
	val, ok := step[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("requires %s", key)
		}
		return 0, nil
	}
	number, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be numeric", key)
	}
	if number < 0 {
		return 0, fmt.Errorf("%s must be non-negative", key)
	}
	return number, nil
}

func validateRunInputMode(step map[string]any, key, label string) error {
	mode, ok := runInputString(step, key)
	if !ok {
		return nil
	}
	switch mode {
	case "tap", "press", "release":
		return nil
	default:
		return fmt.Errorf("%s must be tap, press, or release", label)
	}
}

func runRunWaitProbe(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run wait-probe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	source := fs.String("source", "", "log source to watch (e.g. runtime.echo_unit)")
	assertExpr := fs.String("assert", "", "predicate in KEY>=VALUE form")
	assertKey := fs.String("assert-key", "", "predicate key")
	assertOp := fs.String("assert-op", "", "predicate operator: >= <= == != > <")
	assertValue := fs.String("assert-value", "", "predicate value")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum time to wait")
	jsonOut := fs.Bool("json", false, "print matching entry as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *source == "" {
		return fmt.Errorf("run wait-probe requires --source")
	}
	predicate, err := resolveAssertPredicate(*assertExpr, *assertKey, *assertOp, *assertValue)
	if err != nil {
		return err
	}
	if predicate == "" {
		return fmt.Errorf("run wait-probe requires --assert KEY>=VALUE or --assert-key/--assert-op/--assert-value")
	}
	key, op, rawVal, err := parseAssertExpr(predicate)
	if err != nil {
		return err
	}
	if err := requireRuntimeHelper(ctx, client, "run wait-probe"); err != nil {
		return err
	}

	deadline := time.Now().Add(*timeout)
	var lastEntry *bridge.LogEntry
	for {
		entries, err := client.RunLogs(ctx)
		if err != nil {
			return err
		}
		filtered := filterLogEntries(entries, *source, true, false)
		if len(filtered) > 0 {
			e := filtered[len(filtered)-1]
			lastEntry = &e
			if evalPredicate(e.Detail, key, op, rawVal) {
				if *jsonOut {
					enc := json.NewEncoder(stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(e)
				}
				encoded, _ := json.Marshal(e.Detail)
				fmt.Fprintf(stdout, "Probe [%s] matched %s: %s\n", *source, predicate, encoded)
				return nil
			}
		}
		if time.Now().After(deadline) {
			if lastEntry != nil {
				encoded, _ := json.Marshal(lastEntry.Detail)
				return fmt.Errorf("run wait-probe timed out after %s; last probe [%s]: %s", *timeout, *source, encoded)
			}
			return fmt.Errorf("run wait-probe timed out after %s; no probe entries from %s", *timeout, *source)
		}
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func parseAssertExpr(expr string) (key, op, val string, err error) {
	for _, candidate := range []string{">=", "<=", "==", "!=", ">", "<"} {
		if idx := strings.Index(expr, candidate); idx > 0 {
			return strings.TrimSpace(expr[:idx]), candidate, strings.TrimSpace(expr[idx+len(candidate):]), nil
		}
	}
	return "", "", "", fmt.Errorf("--assert must be in KEY>=VALUE form (operators: >= <= == != > <): %q", expr)
}

func resolveAssertPredicate(assertExpr, key, op, value string) (string, error) {
	splitCount := 0
	for _, item := range []string{key, op, value} {
		if item != "" {
			splitCount++
		}
	}
	if assertExpr != "" && splitCount > 0 {
		return "", fmt.Errorf("--assert cannot be combined with --assert-key/--assert-op/--assert-value")
	}
	if splitCount == 0 {
		return assertExpr, nil
	}
	if splitCount != 3 {
		return "", fmt.Errorf("--assert-key, --assert-op, and --assert-value must be provided together")
	}
	switch op {
	case ">=", "<=", "==", "!=", ">", "<":
		return key + op + value, nil
	default:
		return "", fmt.Errorf("--assert-op must be one of >= <= == != > <")
	}
}

func evalPredicate(detail map[string]any, key, op, rawVal string) bool {
	val, ok := detail[key]
	if !ok {
		return false
	}
	// try numeric comparison
	wantF, wantErr := strconv.ParseFloat(rawVal, 64)
	if wantErr == nil {
		var gotF float64
		switch v := val.(type) {
		case float64:
			gotF = v
		case int:
			gotF = float64(v)
		case int64:
			gotF = float64(v)
		case bool:
			if v {
				gotF = 1
			}
		default:
			return false
		}
		switch op {
		case ">=":
			return gotF >= wantF
		case "<=":
			return gotF <= wantF
		case ">":
			return gotF > wantF
		case "<":
			return gotF < wantF
		case "==":
			return gotF == wantF
		case "!=":
			return gotF != wantF
		}
	}
	// string comparison
	gotS := fmt.Sprintf("%v", val)
	switch op {
	case "==":
		return gotS == rawVal
	case "!=":
		return gotS != rawVal
	}
	return false
}

func runRunSmoke(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run smoke", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scenePath := fs.String("scene", "", "scene path to run")
	main := fs.Bool("main", false, "run the project main scene")
	inputFile := fs.String("input", "", "input JSON file for automated steps")
	assertExpr := fs.String("assert", "", "probe predicate in SOURCE:KEY>=VALUE form")
	assertSource := fs.String("assert-source", "", "probe source for split assertion form")
	assertKey := fs.String("assert-key", "", "probe detail key for split assertion form")
	assertOp := fs.String("assert-op", "", "predicate operator for split assertion form: >= <= == != > <")
	assertValue := fs.String("assert-value", "", "predicate value for split assertion form")
	screenshotOut := fs.String("screenshot", "", "save game viewport screenshot to this path")
	screenshotViewport := fs.String("screenshot-viewport", "", "SubViewport path for screenshot (empty = main viewport)")
	timeout := fs.Duration("timeout", 30*time.Second, "overall smoke timeout")
	keepRunning := fs.Bool("keep-running", false, "do not stop the run after smoke completes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scenePath != "" && *main {
		return fmt.Errorf("run smoke requires at most one of --scene or --main")
	}

	// stop any in-progress run before starting smoke to prevent Godot crash
	_, _ = client.RunStop(ctx, requestID())
	time.Sleep(300 * time.Millisecond)

	// start
	startResult, err := client.RunStart(ctx, requestID(), *scenePath, *main, true)
	if err != nil {
		return fmt.Errorf("smoke start: %w", err)
	}
	scene := startResult.Scene
	if scene == "" {
		scene = startResult.PlayingScene
	}
	fmt.Fprintf(stdout, "Smoke started: %s\n", scene)

	stop := func() {
		if !*keepRunning {
			_, _ = client.RunStop(ctx, requestID())
		}
	}
	defer stop()

	// input
	if *inputFile != "" {
		content, err := os.ReadFile(*inputFile)
		if err != nil {
			return fmt.Errorf("smoke input read: %w", err)
		}
		var payload struct {
			Steps []any `json:"steps"`
		}
		if err := json.Unmarshal(content, &payload); err != nil {
			return fmt.Errorf("smoke input parse: %w", err)
		}
		if len(payload.Steps) > 0 {
			if err := validateRunInputSteps(payload.Steps); err != nil {
				return fmt.Errorf("smoke input: %w", err)
			}
			res, err := client.RunInput(ctx, requestID(), payload.Steps)
			if err != nil {
				return fmt.Errorf("smoke input: %w", err)
			}
			if res.JobID != "" {
				if _, err := waitForJob(ctx, client, res.JobID, *timeout, "smoke input"); err != nil {
					return fmt.Errorf("smoke input: %w", err)
				}
			}
			fmt.Fprintf(stdout, "Smoke input: %d steps\n", len(payload.Steps))
		}
	}

	// assert
	resolvedAssert := *assertExpr
	if *assertSource != "" || *assertKey != "" || *assertOp != "" || *assertValue != "" {
		if *assertExpr != "" {
			return fmt.Errorf("--assert cannot be combined with --assert-source/--assert-key/--assert-op/--assert-value")
		}
		predicate, err := resolveAssertPredicate("", *assertKey, *assertOp, *assertValue)
		if err != nil {
			return err
		}
		if *assertSource == "" {
			return fmt.Errorf("--assert-source is required with split smoke assertions")
		}
		resolvedAssert = *assertSource + ":" + predicate
	}
	if resolvedAssert != "" {
		// parse SOURCE:KEY>=VALUE
		probeSource, predicate, found := strings.Cut(resolvedAssert, ":")
		if !found {
			return fmt.Errorf("smoke --assert must be SOURCE:KEY>=VALUE")
		}
		key, op, rawVal, err := parseAssertExpr(predicate)
		if err != nil {
			return err
		}
		deadline := time.Now().Add(*timeout)
		var lastDetail map[string]any
		matched := false
		for !matched {
			entries, err := client.RunLogs(ctx)
			if err != nil {
				return fmt.Errorf("smoke assert: %w", err)
			}
			filtered := filterLogEntries(entries, probeSource, true, false)
			if len(filtered) > 0 {
				lastDetail = filtered[len(filtered)-1].Detail
				if evalPredicate(lastDetail, key, op, rawVal) {
					matched = true
				}
			}
			if !matched {
				if time.Now().After(deadline) {
					encoded, _ := json.Marshal(lastDetail)
					helperSummary := smokeHelperFailureSummary(ctx, client)
					if helperSummary != "" {
						return fmt.Errorf("Smoke: FAIL — assert %s timed out; last probe: %s; %s", resolvedAssert, encoded, helperSummary)
					}
					return fmt.Errorf("Smoke: FAIL — assert %s timed out; last probe: %s", resolvedAssert, encoded)
				}
				select {
				case <-time.After(500 * time.Millisecond):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		fmt.Fprintf(stdout, "Smoke assert: %s ok\n", resolvedAssert)
	}

	// screenshot
	if *screenshotOut != "" {
		vpPath := ""
		if *screenshotViewport != "" {
			vpPath = *screenshotViewport
		}
		ssResult, err := client.RunScreenshot(ctx, requestID(), "game", 0, vpPath)
		if err != nil {
			return fmt.Errorf("smoke screenshot: %w", err)
		}
		if ssResult.JobID != "" {
			job, err := waitForJob(ctx, client, ssResult.JobID, *timeout, "smoke screenshot")
			if err != nil {
				return fmt.Errorf("smoke screenshot: %w", err)
			}
			if err := writeScreenshotJob(*screenshotOut, job); err != nil {
				return fmt.Errorf("smoke screenshot write: %w", err)
			}
			w := intFromJobResult(job.Result["width"])
			h := intFromJobResult(job.Result["height"])
			fmt.Fprintf(stdout, "Smoke screenshot: %s (%dx%d)\n", *screenshotOut, w, h)
		}
	}

	fmt.Fprintln(stdout, "Smoke: PASS")
	return nil
}

func smokeHelperFailureSummary(ctx context.Context, client *bridge.Client) string {
	status, err := client.RunStatus(ctx, requestID())
	if err != nil {
		return ""
	}
	if status.RuntimeHelper.Present {
		return "runtime helper present"
	}
	if status.RuntimeHelper.Error != "" {
		return "runtime helper not present: " + status.RuntimeHelper.Error
	}
	return "runtime helper not present"
}

func runRunProbe(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("run probe requires a subcommand (raycast, node)")
	}
	switch args[0] {
	case "raycast":
		return runRunProbeRaycast(ctx, client, args[1:], stdout)
	case "node":
		return runRunProbeNode(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown run probe subcommand: %s", args[0])
	}
}

func runRunProbeNode(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run probe node", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "runtime node path")
	var properties stringListFlag
	fs.Var(&properties, "property", "property to read; repeat for multiple properties")
	jsonOut := fs.Bool("json", false, "print probe result as JSON")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for probe job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("run probe node requires --path")
	}
	if len(properties) == 0 {
		return fmt.Errorf("run probe node requires at least one --property")
	}
	result, err := client.RunProbeNode(ctx, requestID(), *path, []string(properties))
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("run probe node did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "run probe node")
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(job.Result)
	}
	probePath, _ := job.Result["path"].(string)
	if probePath == "" {
		probePath = *path
	}
	nodeType, _ := job.Result["type"].(string)
	if nodeType != "" {
		fmt.Fprintf(stdout, "Node probe: %s (%s)\n", probePath, nodeType)
	} else {
		fmt.Fprintf(stdout, "Node probe: %s\n", probePath)
	}
	if props, ok := job.Result["properties"].(map[string]any); ok {
		for _, property := range properties {
			encoded, _ := json.Marshal(props[string(property)])
			fmt.Fprintf(stdout, "  %s: %s\n", property, encoded)
		}
	}
	return nil
}

func runRunStop(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	result, err := client.RunStop(ctx, requestID())
	if err != nil {
		return err
	}
	if result.Stopped {
		fmt.Fprintln(stdout, "Run stopped")
	} else {
		fmt.Fprintln(stdout, "Run was not running")
	}
	return nil
}

func runRunLogs(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "write logs as JSON")
	clear := fs.Bool("clear", false, "clear run logs after reading")
	source := fs.String("source", "", "keep only entries from this source")
	latest := fs.Bool("latest", false, "keep only the last entry per source")
	sinceStart := fs.Bool("since-start", false, "drop entries before the most recent run.start entry")
	if err := fs.Parse(args); err != nil {
		return err
	}
	entries, err := client.RunLogs(ctx)
	if err != nil {
		return err
	}
	entries = filterLogEntries(entries, *source, *latest, *sinceStart)
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{"entries": entries}); err != nil {
			return err
		}
	} else if len(entries) == 0 {
		fmt.Fprintln(stdout, "No run logs")
	} else {
		for _, entry := range entries {
			fmt.Fprintf(stdout, "%s [%s] %s: %s", entry.Time, entry.Level, entry.Source, entry.Message)
			if len(entry.Detail) > 0 {
				encoded, err := json.Marshal(entry.Detail)
				if err != nil {
					return err
				}
				fmt.Fprintf(stdout, " %s", encoded)
			}
			fmt.Fprintln(stdout)
		}
	}
	if *clear {
		return client.ClearRunLogs(ctx, requestID())
	}
	return nil
}

func filterLogEntries(entries []bridge.LogEntry, source string, latest, sinceStart bool) []bridge.LogEntry {
	if sinceStart {
		startTime := ""
		for _, e := range entries {
			if e.Source == "run.start" && e.Time > startTime {
				startTime = e.Time
			}
		}
		if startTime != "" {
			filtered := entries[:0]
			for _, e := range entries {
				if e.Source != "run.start" && e.Time > startTime {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}
	}
	if source != "" {
		filtered := entries[:0]
		for _, e := range entries {
			if e.Source == source {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
	if latest {
		seen := make(map[string]bool)
		out := make([]bridge.LogEntry, 0, len(entries))
		for i := len(entries) - 1; i >= 0; i-- {
			e := entries[i]
			if !seen[e.Source] {
				seen[e.Source] = true
				out = append(out, e)
			}
		}
		slices.Reverse(out)
		entries = out
	}
	return entries
}

func runRunScreenshot(ctx context.Context, client *bridge.Client, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("run screenshot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outPath := fs.String("out", "", "local PNG output path")
	source := fs.String("source", "game", "screenshot source: game or screen")
	screen := fs.Int("screen", 0, "host display screen index")
	viewport := fs.String("viewport", "", "SubViewport node path within the running scene")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for screenshot job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := *outPath
	if path == "" {
		path = defaultScreenshotPath()
	}
	if *screen < 0 {
		return fmt.Errorf("run screenshot --screen must be 0 or greater")
	}
	if *source != "game" && *source != "screen" {
		return fmt.Errorf("run screenshot --source must be game or screen")
	}
	if *source == "game" {
		if err := requireRuntimeHelper(ctx, client, "run screenshot"); err != nil {
			return err
		}
	}
	result, err := client.RunScreenshot(ctx, requestID(), *source, *screen, *viewport)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("run screenshot did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "run screenshot")
	if err != nil {
		return err
	}
	if err := writeScreenshotJob(path, job); err != nil {
		return err
	}
	width := intFromJobResult(job.Result["width"])
	height := intFromJobResult(job.Result["height"])
	resultSource, _ := job.Result["source"].(string)
	if resultSource == "" {
		resultSource = *source
	}
	sourceLabel := resultSource
	if resultSource == "game" {
		sourceLabel = "game viewport"
	} else if resultSource == "screen" {
		sourceLabel = "host screen"
	}
	if width > 0 && height > 0 {
		fmt.Fprintf(stdout, "Run screenshot written: %s (%dx%d, %s)\n", path, width, height, sourceLabel)
	} else {
		fmt.Fprintf(stdout, "Run screenshot written: %s (%s)\n", path, sourceLabel)
	}
	if pngData, err := os.ReadFile(path); err == nil {
		if isSuspectedEditorCapture(pngData) {
			fmt.Fprintln(stderr, "warning: screenshot may be the desktop or editor background (low pixel variance)")
		}
	}
	return nil
}

func runRunInstantiate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run instantiate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scene := fs.String("scene", "", "packed scene path to instantiate (res://...)")
	parent := fs.String("parent", "", "parent node path in the running scene")
	name := fs.String("name", "", "name for the new node (optional)")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for instantiate job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scene == "" || *parent == "" {
		return fmt.Errorf("run instantiate requires --scene and --parent")
	}
	result, err := client.RunInstantiate(ctx, requestID(), *scene, *parent, *name)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("run instantiate did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "run instantiate")
	if err != nil {
		return err
	}
	nodePath, _ := job.Result["path"].(string)
	if nodePath != "" {
		fmt.Fprintf(stdout, "Instantiated: %s at %s\n", *scene, nodePath)
	} else {
		fmt.Fprintf(stdout, "Instantiated: %s\n", *scene)
	}
	return nil
}

func runRunSceneReload(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run scene-reload", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene reload")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.RunSceneReload(ctx, requestID())
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("run scene-reload did not return a job id")
	}
	_, err = waitForJob(ctx, client, result.JobID, *timeout, "run scene-reload")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Scene reloaded")
	return nil
}

func waitForJob(ctx context.Context, client *bridge.Client, jobID string, timeout time.Duration, label string) (bridge.Job, error) {
	deadline := time.Now().Add(timeout)
	for {
		job, err := client.Job(ctx, jobID)
		if err != nil {
			return bridge.Job{}, err
		}
		switch job.Status {
		case "succeeded":
			return job, nil
		case "failed":
			if job.Error != nil {
				return bridge.Job{}, job.Error
			}
			return bridge.Job{}, fmt.Errorf("%s job failed", label)
		}
		if time.Now().After(deadline) {
			return bridge.Job{}, fmt.Errorf("%s timed out waiting for job %s", label, jobID)
		}
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return bridge.Job{}, ctx.Err()
		}
	}
}
