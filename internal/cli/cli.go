package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	embeddedaddons "gdctl/addons"
	"gdctl/internal/addon"
	"gdctl/internal/bridge"
	"gdctl/internal/version"
)

var newAddonManager = func() addon.Manager {
	return addon.NewManager(embeddedaddons.FS)
}

// sceneMu serializes all commands that open, mutate, and save a scene via --scene,
// preventing cross-wire races when multiple gdctl invocations run concurrently in one process.
var sceneMu sync.Mutex

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg, rest, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}
	cfg, err = cfg.WithProjectToken()
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		printUsage(stderr)
		return fmt.Errorf("missing command")
	}
	rest = normalizeCommandArgs(rest)

	client := bridge.NewClient(cfg)
	addonManager := newAddonManager()
	switch rest[0] {
	case "ping":
		return runPing(ctx, client, stdout)
	case "doctor":
		return runDoctor(ctx, cfg, client, addonManager, rest[1:], stdout)
	case "addon":
		return runAddon(ctx, cfg, client, addonManager, rest[1:], stdout)
	case "bridge":
		return runBridge(ctx, client, addonManager, rest[1:], stdout)
	case "run":
		return runRun(ctx, client, rest[1:], stdout, stderr)
	case "scene":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runSceneCreate(ctx, client, rest[2:], stdout)
			case "open":
				return runSceneOpen(ctx, client, rest[2:], stdout)
			case "instance":
				return runSceneInstance(ctx, client, rest[2:], stdout)
			case "tree":
				return runSceneTree(ctx, client, stdout)
			case "save":
				return runSceneSave(ctx, client, rest[2:], stdout)
			case "apply":
				return runSceneApply(ctx, client, rest[2:], stdout)
			case "apply-blueprint":
				return runSceneApplyBlueprint(ctx, client, rest[2:], stdout)
			case "list":
				return runSceneList(ctx, client, rest[2:], stdout)
			case "run":
				return runSceneRun(ctx, cfg, rest[2:], stdout)
			}
		}
	case "node":
		if len(rest) >= 2 {
			switch rest[1] {
			case "add":
				return runNodeAdd(ctx, client, rest[2:], stdout)
			case "remove":
				return runNodeRemove(ctx, client, rest[2:], stdout)
			case "rename":
				return runNodeRename(ctx, client, rest[2:], stdout)
			case "move":
				return runNodeMove(ctx, client, rest[2:], stdout)
			case "get":
				return runNodeGet(ctx, client, rest[2:], stdout)
			case "set":
				return runNodeSet(ctx, client, rest[2:], stdout)
			case "set-resource":
				return runNodeSetResource(ctx, client, rest[2:], stdout)
			case "attach-script":
				return runNodeAttachScript(ctx, client, rest[2:], stdout)
			case "group":
				if len(rest) >= 3 {
					switch rest[2] {
					case "add":
						return runNodeGroupAdd(ctx, client, rest[3:], stdout)
					case "remove":
						return runNodeGroupRemove(ctx, client, rest[3:], stdout)
					case "list":
						return runNodeGroupList(ctx, client, rest[3:], stdout)
					}
				}
			case "duplicate":
				return runNodeDuplicate(ctx, client, rest[2:], stdout)
			case "list-properties":
				return runNodeListProperties(ctx, client, rest[2:], stdout)
			}
		}
	case "script":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runScriptCreate(ctx, client, rest[2:], stdout)
			case "write":
				return runScriptWrite(ctx, client, rest[2:], stdout)
			case "check":
				return runScriptCheck(ctx, client, rest[2:], stdout)
			}
		}
	case "shader":
		if len(rest) >= 2 {
			switch rest[1] {
			case "write":
				return runShaderWrite(ctx, client, rest[2:], stdout)
			case "check":
				return runShaderCheck(ctx, client, rest[2:], stdout)
			}
		}
	case "resource":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runResourceCreate(ctx, client, rest[2:], stdout)
			case "list":
				return runResourceList(ctx, client, rest[2:], stdout)
			}
		}
	case "import":
		if len(rest) >= 2 {
			switch rest[1] {
			case "set":
				return runImportSet(ctx, client, rest[2:], stdout)
			}
		}
	case "file":
		if len(rest) >= 2 {
			switch rest[1] {
			case "write-bytes":
				return runFileWriteBytes(ctx, client, rest[2:], stdout)
			case "lut-write":
				return runLUTWrite(ctx, client, rest[2:], stdout)
			case "list":
				return runFileList(ctx, client, rest[2:], stdout)
			case "mkdir":
				return runFileMkdir(ctx, client, rest[2:], stdout)
			case "delete":
				return runFileDelete(ctx, client, rest[2:], stdout)
			case "exists":
				return runFileExists(ctx, client, rest[2:], stdout)
			}
		}
	case "navigation":
		if len(rest) >= 2 {
			switch rest[1] {
			case "bake":
				return runNavigationBake(ctx, client, rest[2:], stdout)
			}
		}
	case "signal":
		if len(rest) >= 2 {
			switch rest[1] {
			case "connect":
				return runSignalConnect(ctx, client, rest[2:], stdout)
			case "disconnect":
				return runSignalDisconnect(ctx, client, rest[2:], stdout)
			}
		}
	case "project":
		if len(rest) >= 2 {
			switch rest[1] {
			case "setting":
				if len(rest) >= 3 {
					switch rest[2] {
					case "get":
						return runProjectSettingGet(ctx, client, rest[3:], stdout)
					case "set":
						return runProjectSettingSet(ctx, client, rest[3:], stdout)
					}
				}
			case "run":
				return runProjectRun(ctx, cfg, rest[2:], stdout)
			}
		}
	case "viewport":
		if len(rest) >= 2 {
			switch rest[1] {
			case "screenshot":
				return runViewportScreenshot(ctx, client, rest[2:], stdout)
			case "set-size":
				return runViewportSetSize(ctx, client, rest[2:], stdout)
			case "add":
				return runViewportAdd(ctx, client, rest[2:], stdout)
			}
		}
	case "theme":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runThemeCreate(ctx, client, rest[2:], stdout)
			case "set-color":
				return runThemeSetColor(ctx, client, rest[2:], stdout)
			case "set-font-size":
				return runThemeSetFontSize(ctx, client, rest[2:], stdout)
			case "set-constant":
				return runThemeSetConstant(ctx, client, rest[2:], stdout)
			}
		}
	case "animation":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runAnimationCreate(ctx, client, rest[2:], stdout)
			case "track-add":
				return runAnimationTrackAdd(ctx, client, rest[2:], stdout)
			case "keyframe-add":
				return runAnimationKeyframeAdd(ctx, client, rest[2:], stdout)
			case "length-set":
				return runAnimationLengthSet(ctx, client, rest[2:], stdout)
			case "player-play":
				return runAnimationPlayerPlay(ctx, client, rest[2:], stdout)
			}
		}
	case "tilemap":
		if len(rest) >= 2 {
			switch rest[1] {
			case "tileset-create":
				return runTilesetCreate(ctx, client, rest[2:], stdout)
			case "source-add":
				return runTilesetSourceAdd(ctx, client, rest[2:], stdout)
			case "cell-set":
				return runTilemapCellSet(ctx, client, rest[2:], stdout)
			case "cell-clear":
				return runTilemapCellClear(ctx, client, rest[2:], stdout)
			}
		}
	case "audio":
		if len(rest) >= 2 {
			switch rest[1] {
			case "bus-add":
				return runAudioBusAdd(ctx, client, rest[2:], stdout)
			case "bus-volume-set":
				return runAudioBusVolumeSet(ctx, client, rest[2:], stdout)
			case "bus-effect-add":
				return runAudioBusEffectAdd(ctx, client, rest[2:], stdout)
			}
		}
	case "help":
		return runHelp(rest[1:], stdout)
	}

	printUsage(stderr)
	return fmt.Errorf("unknown command: %s", strings.Join(rest, " "))
}

func normalizeCommandArgs(args []string) []string {
	if len(args) >= 2 && args[0] == "help" && strings.Contains(args[1], ".") {
		parts := strings.Split(args[1], ".")
		normalized := []string{"help"}
		for _, part := range parts {
			if part != "" {
				normalized = append(normalized, part)
			}
		}
		if len(normalized) > 1 {
			return append(normalized, args[2:]...)
		}
	}
	if len(args) == 0 || !strings.Contains(args[0], ".") {
		return args
	}
	parts := strings.Split(args[0], ".")
	normalized := make([]string, 0, len(parts)+len(args)-1)
	for _, part := range parts {
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	if len(normalized) == 0 {
		return args
	}
	return append(normalized, args[1:]...)
}

func runRun(ctx context.Context, client *bridge.Client, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires a subcommand")
	}
	switch args[0] {
	case "start":
		return runRunStart(ctx, client, args[1:], stdout)
	case "status":
		return runRunStatus(ctx, client, stdout)
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

func runRunStatus(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	result, err := client.RunStatus(ctx, requestID())
	if err != nil {
		return err
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
		if result.PlayingScene != "" {
			fmt.Fprintf(stdout, "Run status: running (%s)\n", result.PlayingScene)
		} else {
			fmt.Fprintln(stdout, "Run status: running")
		}
		return nil
	}
	fmt.Fprintln(stdout, "Run status: stopped")
	return nil
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

func runRunWaitProbe(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run wait-probe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	source := fs.String("source", "", "log source to watch (e.g. runtime.echo_unit)")
	assertExpr := fs.String("assert", "", "predicate in KEY>=VALUE form")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum time to wait")
	jsonOut := fs.Bool("json", false, "print matching entry as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *source == "" {
		return fmt.Errorf("run wait-probe requires --source")
	}
	if *assertExpr == "" {
		return fmt.Errorf("run wait-probe requires --assert KEY>=VALUE")
	}
	key, op, rawVal, err := parseAssertExpr(*assertExpr)
	if err != nil {
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
				fmt.Fprintf(stdout, "Probe [%s] matched %s: %s\n", *source, *assertExpr, encoded)
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
		time.Sleep(500 * time.Millisecond)
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
	screenshotOut := fs.String("screenshot", "", "save game viewport screenshot to this path")
	timeout := fs.Duration("timeout", 30*time.Second, "overall smoke timeout")
	keepRunning := fs.Bool("keep-running", false, "do not stop the run after smoke completes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scenePath != "" && *main {
		return fmt.Errorf("run smoke requires at most one of --scene or --main")
	}

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

	// input
	if *inputFile != "" {
		content, err := os.ReadFile(*inputFile)
		if err != nil {
			stop()
			return fmt.Errorf("smoke input read: %w", err)
		}
		var payload struct {
			Steps []any `json:"steps"`
		}
		if err := json.Unmarshal(content, &payload); err != nil {
			stop()
			return fmt.Errorf("smoke input parse: %w", err)
		}
		if len(payload.Steps) > 0 {
			res, err := client.RunInput(ctx, requestID(), payload.Steps)
			if err != nil {
				stop()
				return fmt.Errorf("smoke input: %w", err)
			}
			if res.JobID != "" {
				if _, err := waitForJob(ctx, client, res.JobID, *timeout, "smoke input"); err != nil {
					stop()
					return fmt.Errorf("smoke input: %w", err)
				}
			}
			fmt.Fprintf(stdout, "Smoke input: %d steps\n", len(payload.Steps))
		}
	}

	// assert
	if *assertExpr != "" {
		// parse SOURCE:KEY>=VALUE
		probeSource, predicate, found := strings.Cut(*assertExpr, ":")
		if !found {
			stop()
			return fmt.Errorf("smoke --assert must be SOURCE:KEY>=VALUE")
		}
		key, op, rawVal, err := parseAssertExpr(predicate)
		if err != nil {
			stop()
			return err
		}
		deadline := time.Now().Add(*timeout)
		var lastDetail map[string]any
		matched := false
		for !matched {
			entries, err := client.RunLogs(ctx)
			if err != nil {
				stop()
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
					stop()
					encoded, _ := json.Marshal(lastDetail)
					return fmt.Errorf("Smoke: FAIL — assert %s timed out; last probe: %s", *assertExpr, encoded)
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
		fmt.Fprintf(stdout, "Smoke assert: %s ok\n", *assertExpr)
	}

	// screenshot
	if *screenshotOut != "" {
		ssResult, err := client.RunScreenshot(ctx, requestID(), "game", 0)
		if err != nil {
			stop()
			return fmt.Errorf("smoke screenshot: %w", err)
		}
		if ssResult.JobID != "" {
			job, err := waitForJob(ctx, client, ssResult.JobID, *timeout, "smoke screenshot")
			if err != nil {
				stop()
				return fmt.Errorf("smoke screenshot: %w", err)
			}
			if err := writeScreenshotJob(*screenshotOut, job); err != nil {
				stop()
				return fmt.Errorf("smoke screenshot write: %w", err)
			}
			w := intFromJobResult(job.Result["width"])
			h := intFromJobResult(job.Result["height"])
			fmt.Fprintf(stdout, "Smoke screenshot: %s (%dx%d)\n", *screenshotOut, w, h)
		}
	}

	stop()
	fmt.Fprintln(stdout, "Smoke: PASS")
	return nil
}

func runRunProbe(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("run probe requires a subcommand (raycast)")
	}
	switch args[0] {
	case "raycast":
		return runRunProbeRaycast(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown run probe subcommand: %s", args[0])
	}
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
		seen := make(map[string]int)
		for i, e := range entries {
			seen[e.Source] = i
		}
		order := make([]int, 0, len(seen))
		for _, idx := range seen {
			order = append(order, idx)
		}
		// sort by original position
		for i := 1; i < len(order); i++ {
			for j := i; j > 0 && order[j] < order[j-1]; j-- {
				order[j], order[j-1] = order[j-1], order[j]
			}
		}
		out := make([]bridge.LogEntry, len(order))
		for i, idx := range order {
			out[i] = entries[idx]
		}
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
	result, err := client.RunScreenshot(ctx, requestID(), *source, *screen)
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

// isSuspectedEditorCapture returns true when the image looks like a uniform
// desktop/editor background: samples a 10x10 grid and considers it uniform if
// ≥90% of sampled pixels are within ±10 of the dominant color per channel.
func isSuspectedEditorCapture(pngData []byte) bool {
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return false
	}
	b := img.Bounds()
	w, h := b.Max.X-b.Min.X, b.Max.Y-b.Min.Y
	if w < 10 || h < 10 {
		return false
	}
	const grid = 10
	var samples []color.RGBA
	for gy := 0; gy < grid; gy++ {
		for gx := 0; gx < grid; gx++ {
			x := b.Min.X + (gx*w)/grid + w/(grid*2)
			y := b.Min.Y + (gy*h)/grid + h/(grid*2)
			c := img.At(x, y)
			r, g, bv, a := c.RGBA()
			samples = append(samples, color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bv >> 8), A: uint8(a >> 8)})
		}
	}
	if len(samples) == 0 {
		return false
	}
	dom := samples[0]
	match := 0
	const delta = 10
	for _, s := range samples {
		dr := int(s.R) - int(dom.R)
		dg := int(s.G) - int(dom.G)
		db := int(s.B) - int(dom.B)
		if dr < 0 {
			dr = -dr
		}
		if dg < 0 {
			dg = -dg
		}
		if db < 0 {
			db = -db
		}
		if dr <= delta && dg <= delta && db <= delta {
			match++
		}
	}
	return match*10 >= len(samples)*9
}

func runBridge(ctx context.Context, client *bridge.Client, manager addon.Manager, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("bridge requires a subcommand")
	}
	switch args[0] {
	case "info":
		ping, err := client.Ping(ctx)
		if err != nil {
			return err
		}
		if !ping.OK {
			return fmt.Errorf("godot bridge returned not ok")
		}
		printBridgeInfo(stdout, ping)
		return nil
	case "logs":
		return runBridgeLogs(ctx, client, args[1:], stdout)
	case "addon-update":
		return runBridgeAddonUpdate(ctx, client, manager, stdout)
	default:
		return fmt.Errorf("unknown bridge command: %s", strings.Join(args, " "))
	}
}

func runBridgeLogs(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("bridge logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "write logs as JSON")
	clear := fs.Bool("clear", false, "clear logs after reading")
	if err := fs.Parse(args); err != nil {
		return err
	}
	entries, err := client.Logs(ctx)
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{"entries": entries}); err != nil {
			return err
		}
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
		if err := client.ClearLogs(ctx, requestID()); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "Logs cleared")
	}
	return nil
}

func runBridgeAddonUpdate(ctx context.Context, client *bridge.Client, manager addon.Manager, stdout io.Writer) error {
	manifest, files, err := addon.PackageEmbeddedUpdate(manager.Source)
	if err != nil {
		return err
	}
	// Load the currently installed manifest so the bridge can remove stale files.
	var oldManifestMap map[string]any
	cfg := bridge.Config{}
	if ping, err := client.Ping(ctx); err == nil && ping.ProjectPath != "" {
		if installed, err := addon.LoadInstalledManifest(ping.ProjectPath); err == nil && len(installed.Files) > 0 {
			if data, err := json.Marshal(installed); err == nil {
				_ = json.Unmarshal(data, &oldManifestMap)
			}
		}
	}
	_ = cfg
	result, err := client.UpdateAddon(ctx, requestID(), manifest, oldManifestMap, files)
	if err != nil {
		return err
	}
	if result.Updated {
		fmt.Fprintf(stdout, "Addon updated over bridge: %d files written\n", result.FilesWritten)
	} else {
		fmt.Fprintln(stdout, "Addon already up to date")
	}
	if result.FilesRemoved > 0 {
		fmt.Fprintf(stdout, "Stale files removed: %d\n", result.FilesRemoved)
	}
	if result.Backup != "" {
		fmt.Fprintf(stdout, "Backup: %s\n", result.Backup)
	}
	if result.ReloadRequired {
		fmt.Fprintln(stdout, "Reload required: disable/enable the Godot plugin or restart Godot")
	}
	return nil
}

func parseGlobalFlags(args []string) (bridge.Config, []string, error) {
	cfg := bridge.ConfigFromEnv()
	fs := flag.NewFlagSet("gdctl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	host := fs.String("host", cfg.Host, "bridge host")
	port := fs.Int("port", cfg.Port, "bridge port")
	token := fs.String("token", cfg.Token, "bridge bearer token")
	project := fs.String("project", cfg.Project, "Godot project path")
	godot := fs.String("godot", cfg.GodotPath, "headless Godot binary path")
	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	cfg.Host = *host
	cfg.Port = *port
	cfg.Token = *token
	cfg.Project = *project
	cfg.GodotPath = *godot
	return cfg, fs.Args(), nil
}

func runPing(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	ping, err := client.Ping(ctx)
	if err != nil {
		return err
	}
	if !ping.OK {
		return fmt.Errorf("godot bridge returned not ok")
	}
	fmt.Fprintln(stdout, "Godot bridge: ok")
	fmt.Fprintf(stdout, "Engine: %s %s\n", ping.Engine, ping.EngineVersion)
	fmt.Fprintf(stdout, "Project: %s\n", ping.ProjectName)
	fmt.Fprintf(stdout, "Plugin: %s\n", ping.PluginVersion)
	return nil
}

func printBridgeInfo(stdout io.Writer, ping bridge.PingResponse) {
	fmt.Fprintln(stdout, "Godot bridge")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  reachable: yes")
	fmt.Fprintf(stdout, "  service: %s\n", valueOrDash(ping.Service))
	fmt.Fprintf(stdout, "  engine: %s %s\n", valueOrDash(ping.Engine), valueOrDash(ping.EngineVersion))
	fmt.Fprintf(stdout, "  project: %s\n", valueOrDash(ping.ProjectName))
	fmt.Fprintf(stdout, "  project_path: %s\n", valueOrDash(ping.ProjectPath))
	fmt.Fprintf(stdout, "  plugin_version: %s\n", valueOrDash(ping.PluginVersion))
	fmt.Fprintf(stdout, "  protocol: %s\n", valueOrDash(ping.ProtocolVersion))
	fmt.Fprintf(stdout, "  auth_enabled: %s\n", yesNo(ping.AuthEnabled))
	fmt.Fprintf(stdout, "  listen: %s:%d\n", valueOrDash(ping.Host), ping.Port)
	fmt.Fprintf(stdout, "  scene_open: %s\n", yesNo(ping.SceneOpen))
	if len(ping.Capabilities) > 0 {
		fmt.Fprintf(stdout, "  capabilities: %s\n", strings.Join(ping.Capabilities, ", "))
	}
}

func runAddon(ctx context.Context, cfg bridge.Config, client *bridge.Client, manager addon.Manager, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("addon requires a subcommand")
	}
	switch args[0] {
	case "install":
		fs := flag.NewFlagSet("addon install", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		force := fs.Bool("force", false, "overwrite conflicting addon files")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Install(addon.InstallOptions{ProjectPath: *project, Force: *force})
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "update":
		fs := flag.NewFlagSet("addon update", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			return runBridgeAddonUpdate(ctx, client, manager, stdout)
		}
		result, err := manager.Update(addon.UpdateOptions{ProjectPath: *project})
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "enable":
		fs := flag.NewFlagSet("addon enable", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Enable(*project)
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "disable":
		fs := flag.NewFlagSet("addon disable", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Disable(*project)
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "remove":
		fs := flag.NewFlagSet("addon remove", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result, err := manager.Remove(addon.RemoveOptions{ProjectPath: *project})
		if err != nil {
			return err
		}
		printAddonResult(stdout, result)
		return nil
	case "rollback":
		fs := flag.NewFlagSet("addon rollback", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		backup := fs.String("backup", "", "backup directory to restore; defaults to latest")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			return fmt.Errorf("addon rollback requires --project")
		}
		result, err := manager.Rollback(addon.RollbackOptions{ProjectPath: *project, BackupPath: *backup})
		if err != nil {
			return err
		}
		printAddonRollbackResult(stdout, result)
		return nil
	case "status":
		fs := flag.NewFlagSet("addon status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		jsonOut := fs.Bool("json", false, "write status as JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			ping, err := client.Ping(ctx)
			if err != nil {
				return err
			}
			return printRuntimeAddonStatus(stdout, ping, *jsonOut)
		}
		status, err := manager.Status(ctx, addon.StatusOptions{ProjectPath: *project, BridgeConfig: cfg, CheckRuntime: true})
		if err != nil {
			return err
		}
		return printAddonStatus(stdout, status, *jsonOut)
	case "doctor":
		fs := flag.NewFlagSet("addon doctor", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		project := fs.String("project", cfg.Project, "Godot project path")
		fix := fs.Bool("fix", false, "install and enable the addon when needed")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *project == "" {
			ping, err := client.Ping(ctx)
			printRuntimeAddonDoctor(stdout, ping, err)
			return err
		}
		status, actions, err := manager.Doctor(ctx, addon.DoctorOptions{ProjectPath: *project, BridgeConfig: cfg, Fix: *fix})
		printAddonDoctor(stdout, status, actions)
		return err
	default:
		return fmt.Errorf("unknown addon command: %s", strings.Join(args, " "))
	}
}

func printAddonResult(stdout io.Writer, result addon.Result) {
	fmt.Fprintln(stdout, result.Message)
	if result.Backup != "" {
		fmt.Fprintf(stdout, "Backup: %s\n", result.Backup)
	}
}

func printAddonRollbackResult(stdout io.Writer, result addon.Result) {
	fmt.Fprintln(stdout, result.Message)
	if result.Backup != "" {
		fmt.Fprintf(stdout, "Restored: %s\n", result.Backup)
	}
}

func printAddonStatus(stdout io.Writer, status addon.Status, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}
	fmt.Fprintln(stdout, "gdctl bridge addon")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "  installed: %s\n", yesNo(status.Installed))
	fmt.Fprintf(stdout, "  path: %s\n", status.AddonPath)
	fmt.Fprintf(stdout, "  enabled: %s\n", yesNo(status.Enabled))
	fmt.Fprintf(stdout, "  version: %s\n", valueOrDash(status.DeclaredVersion))
	fmt.Fprintf(stdout, "  manifest: %s\n", valueOrDash(status.ManifestVersion))
	fmt.Fprintf(stdout, "  embedded: %s\n", status.EmbeddedVersion)
	fmt.Fprintf(stdout, "  reachable: %s\n", yesNo(status.Reachable))
	fmt.Fprintf(stdout, "  runtime: %s\n", valueOrDash(status.RuntimeVersion))
	fmt.Fprintf(stdout, "  protocol: %s\n", valueOrDash(status.ProtocolVersion))
	fmt.Fprintf(stdout, "  compatible: %s (%s)\n", yesNo(status.Compatible), status.Compatibility)
	if status.RuntimeError != "" {
		fmt.Fprintf(stdout, "  runtime_error: %s\n", status.RuntimeError)
	}
	return nil
}

func printRuntimeAddonStatus(stdout io.Writer, ping bridge.PingResponse, jsonOut bool) error {
	compatible := ping.PluginVersion == "" || ping.PluginVersion == version.EmbeddedBridgeVersion
	if jsonOut {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"installed":        ping.OK,
			"enabled":          ping.OK,
			"reachable":        ping.OK,
			"runtime_version":  ping.PluginVersion,
			"protocol_version": ping.ProtocolVersion,
			"compatible":       compatible,
			"capabilities":     ping.Capabilities,
			"project_path":     ping.ProjectPath,
			"mode":             "runtime",
		})
	}
	fmt.Fprintln(stdout, "gdctl bridge addon")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  mode: runtime")
	fmt.Fprintf(stdout, "  installed: %s\n", yesNo(ping.OK))
	fmt.Fprintf(stdout, "  enabled: %s\n", yesNo(ping.OK))
	fmt.Fprintf(stdout, "  reachable: %s\n", yesNo(ping.OK))
	fmt.Fprintf(stdout, "  runtime: %s\n", valueOrDash(ping.PluginVersion))
	fmt.Fprintf(stdout, "  protocol: %s\n", valueOrDash(ping.ProtocolVersion))
	fmt.Fprintf(stdout, "  compatible: %s\n", yesNo(compatible))
	return nil
}

func printAddonDoctor(stdout io.Writer, status addon.Status, actions []string) {
	fmt.Fprintln(stdout, "gdctl Addon Doctor")
	for _, action := range actions {
		fmt.Fprintf(stdout, "[fix] %s\n", action)
	}
	if status.Installed {
		fmt.Fprintln(stdout, "[ok] addon installed")
	} else {
		fmt.Fprintln(stdout, "[fail] addon not installed")
	}
	if status.Enabled {
		fmt.Fprintln(stdout, "[ok] addon enabled")
	} else {
		fmt.Fprintln(stdout, "[fail] addon not enabled")
	}
	if status.Compatible {
		fmt.Fprintln(stdout, "[ok] addon compatible")
	} else {
		fmt.Fprintf(stdout, "[fail] addon compatibility: %s\n", status.Compatibility)
	}
	if status.Reachable {
		fmt.Fprintln(stdout, "[ok] runtime bridge reachable")
	} else {
		fmt.Fprintln(stdout, "[warn] runtime bridge not reachable")
	}
}

func printRuntimeAddonDoctor(stdout io.Writer, ping bridge.PingResponse, err error) {
	fmt.Fprintln(stdout, "gdctl Addon Doctor")
	fmt.Fprintln(stdout, "[info] projectless runtime mode")
	if err != nil {
		fmt.Fprintf(stdout, "[fail] runtime bridge reachable: %v\n", err)
		return
	}
	if ping.OK {
		fmt.Fprintln(stdout, "[ok] runtime bridge reachable")
		fmt.Fprintf(stdout, "[ok] runtime addon version %s\n", valueOrDash(ping.PluginVersion))
	} else {
		fmt.Fprintln(stdout, "[fail] ping returned not ok")
	}
}

func runSceneTree(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	root, err := client.SceneTree(ctx)
	if err != nil {
		return err
	}
	bridge.RenderTree(stdout, root)
	return nil
}

func runSceneCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path, for example res://scenes/Main.tscn")
	rootType := fs.String("root", "", "root node type")
	rootName := fs.String("name", "", "root node name")
	force := fs.Bool("force", false, "overwrite an existing scene file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *rootType == "" || *rootName == "" {
		return fmt.Errorf("scene create requires --path, --root, and --name")
	}
	result, err := client.CreateScene(ctx, requestID(), *path, *rootType, *rootName, *force)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene created: %s\n", result.Path)
	fmt.Fprintf(stdout, "Root: %s %s\n", result.RootPath, result.RootType)
	return nil
}

func runSceneOpen(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene open", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path, for example res://scenes/Main.tscn")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for open job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("scene open requires --path")
	}
	pathValue, root, err := openSceneAndWait(ctx, client, *path, *timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene opened: %s\n", pathValue)
	if root != "" {
		fmt.Fprintf(stdout, "Root: %s\n", root)
	}
	return nil
}

func runSceneInstance(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene instance", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "parent node path")
	scenePath := fs.String("scene", "", "scene resource path")
	name := fs.String("name", "", "instance node name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" || *scenePath == "" || *name == "" {
		return fmt.Errorf("scene instance requires --parent, --scene, and --name")
	}
	result, err := client.InstanceScene(ctx, requestID(), *parent, *scenePath, *name)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene instanced: %s\n", result.Path)
	return nil
}

func runSceneSave(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene save", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "unsupported placeholder for future save-as support")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for save job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path != "" {
		return fmt.Errorf("scene save --path is temporarily unsupported; save the scene once in Godot, then run scene save")
	}
	pathValue, err := saveSceneAndWait(ctx, client, *timeout)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Scene saved: %s\n", pathValue)
	return nil
}

func runSceneApply(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path to open and mutate")
	filePath := fs.String("file", "", "JSON scene tree file")
	dryRun := fs.Bool("dry-run", false, "validate without mutating or saving")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *filePath == "" {
		return fmt.Errorf("scene apply requires --path and --file")
	}
	data, err := os.ReadFile(*filePath)
	if err != nil {
		return err
	}
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return fmt.Errorf("scene apply --file must be JSON: %w", err)
	}
	openedPath, root, err := openSceneAndWait(ctx, client, *path, *timeout)
	if err != nil {
		return err
	}
	result, err := client.ApplyScene(ctx, requestID(), tree, *dryRun)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s (created: %d, properties: %d)\n", openedPath, result.Created, result.Updated)
		return nil
	}
	savedPath, err := saveSceneAndWait(ctx, client, *timeout)
	if err != nil {
		return err
	}
	if root == "" {
		root = result.Root
	}
	fmt.Fprintf(stdout, "Scene applied: %s\n", savedPath)
	if root != "" {
		fmt.Fprintf(stdout, "Root: %s\n", root)
	}
	fmt.Fprintf(stdout, "Created: %d\n", result.Created)
	fmt.Fprintf(stdout, "Properties: %d\n", result.Updated)
	return nil
}

func openSceneAndWait(ctx context.Context, client *bridge.Client, path string, timeout time.Duration) (string, string, error) {
	result, err := client.OpenScene(ctx, requestID(), path)
	if err != nil {
		return "", "", err
	}
	if result.JobID == "" {
		return "", "", fmt.Errorf("scene open did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, timeout, "scene open")
	if err != nil {
		return "", "", err
	}
	pathValue, _ := job.Result["path"].(string)
	if pathValue == "" {
		pathValue = result.Path
	}
	root, _ := job.Result["root"].(string)
	return pathValue, root, nil
}

func saveSceneAndWait(ctx context.Context, client *bridge.Client, timeout time.Duration) (string, error) {
	result, err := client.SaveScene(ctx, requestID(), "")
	if err != nil {
		return "", err
	}
	if result.JobID == "" {
		return "", fmt.Errorf("scene save did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, timeout, "scene save")
	if err != nil {
		return "", err
	}
	pathValue, _ := job.Result["path"].(string)
	if pathValue == "" {
		pathValue = result.Path
	}
	return pathValue, nil
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
		time.Sleep(100 * time.Millisecond)
	}
}

func runNodeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "parent node path")
	nodeType := fs.String("type", "", "Godot node type")
	name := fs.String("name", "", "new node name")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before adding and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	propFlags := stringListFlag{}
	fs.Var(&propFlags, "prop", "initial property in name=TYPED_JSON form")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parent == "" || *nodeType == "" || *name == "" {
		return fmt.Errorf("node add requires --parent, --type, and --name")
	}
	props, err := parseNameJSONPairs(propFlags)
	if err != nil {
		return err
	}
	if *scenePath != "" {
		sceneMu.Lock()
		defer sceneMu.Unlock()
		openedPath, _, err := openSceneAndWait(ctx, client, *scenePath, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	}
	result, err := client.AddNode(ctx, requestID(), *parent, *nodeType, *name, props, *dryRun)
	if err != nil {
		return err
	}
	nodePath, _ := result["path"].(string)
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", nodePath)
		return nil
	}
	fmt.Fprintf(stdout, "Added node: %s\n", nodePath)
	if *scenePath != "" {
		savedPath, err := saveSceneAndWait(ctx, client, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene saved: %s\n", savedPath)
	}
	return nil
}

func runNodeRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before removing and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node remove requires --path")
	}
	if *scenePath != "" {
		sceneMu.Lock()
		defer sceneMu.Unlock()
		openedPath, _, err := openSceneAndWait(ctx, client, *scenePath, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	}
	result, err := client.RemoveNode(ctx, requestID(), *path, *dryRun)
	if err != nil {
		return err
	}
	removed, _ := result["path"].(string)
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", removed)
		return nil
	}
	fmt.Fprintf(stdout, "Removed node: %s\n", removed)
	if *scenePath != "" {
		savedPath, err := saveSceneAndWait(ctx, client, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene saved: %s\n", savedPath)
	}
	return nil
}

func runNodeRename(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node rename", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	name := fs.String("name", "", "new node name")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before renaming and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("node rename requires --path and --name")
	}
	if *scenePath != "" {
		sceneMu.Lock()
		defer sceneMu.Unlock()
		openedPath, _, err := openSceneAndWait(ctx, client, *scenePath, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	}
	result, err := client.RenameNode(ctx, requestID(), *path, *name, *dryRun)
	if err != nil {
		return err
	}
	newPath, _ := result["path"].(string)
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", newPath)
		return nil
	}
	fmt.Fprintf(stdout, "Renamed node: %s\n", newPath)
	if *scenePath != "" {
		savedPath, err := saveSceneAndWait(ctx, client, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene saved: %s\n", savedPath)
	}
	return nil
}

func runNodeMove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node move", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	parent := fs.String("parent", "", "new parent node path")
	index := fs.Int("index", -1, "optional child index under new parent")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	scenePath := fs.String("scene", "", "scene path to open before moving and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *parent == "" {
		return fmt.Errorf("node move requires --path and --parent")
	}
	if *scenePath != "" {
		sceneMu.Lock()
		defer sceneMu.Unlock()
		openedPath, _, err := openSceneAndWait(ctx, client, *scenePath, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	}
	result, err := client.MoveNode(ctx, requestID(), *path, *parent, *index, *dryRun)
	if err != nil {
		return err
	}
	newPath, _ := result["path"].(string)
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", newPath)
		return nil
	}
	fmt.Fprintf(stdout, "Moved node: %s\n", newPath)
	if *scenePath != "" {
		savedPath, err := saveSceneAndWait(ctx, client, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene saved: %s\n", savedPath)
	}
	return nil
}

func runNodeGet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	property := fs.String("property", "", "property name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *property == "" {
		return fmt.Errorf("node get requires --path and --property")
	}
	result, err := client.GetNodeProperty(ctx, requestID(), *path, *property)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runNodeSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	property := fs.String("property", "", "property name")
	position := fs.String("position", "", "shorthand for --property position --vector3 X,Y,Z")
	rotationDeg := fs.String("rotation-degrees", "", "shorthand for --property rotation_degrees --vector3 X,Y,Z")
	scale := fs.String("scale", "", "shorthand for --property scale --vector3 X,Y,Z")
	scenePath := fs.String("scene", "", "scene path to open before setting and save after")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	valueFlags := newTypedValueFlags(fs, "node set")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node set requires --path")
	}
	// Resolve transform shorthands into property + value
	var resolvedProp string
	var resolvedValue any
	shorthandCount := 0
	for _, sh := range []struct{ flag, prop string }{
		{*position, "position"},
		{*rotationDeg, "rotation_degrees"},
		{*scale, "scale"},
	} {
		if sh.flag != "" {
			shorthandCount++
			resolvedProp = sh.prop
			vals, err := parseFloatList(sh.flag, 3, "--"+sh.prop)
			if err != nil {
				return err
			}
			resolvedValue = map[string]any{"kind": "Vector3", "value": vals}
		}
	}
	if shorthandCount > 1 {
		return fmt.Errorf("node set: only one of --position, --rotation-degrees, --scale may be used at a time")
	}
	if shorthandCount > 0 && *property != "" {
		return fmt.Errorf("node set: --property cannot be combined with --position, --rotation-degrees, or --scale")
	}
	if shorthandCount == 0 {
		if *property == "" {
			return fmt.Errorf("node set requires --property (or a transform shorthand)")
		}
		resolvedProp = *property
		var err error
		resolvedValue, err = valueFlags.Value()
		if err != nil {
			return err
		}
	}
	if *scenePath != "" {
		sceneMu.Lock()
		defer sceneMu.Unlock()
		openedPath, _, err := openSceneAndWait(ctx, client, *scenePath, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	}
	result, err := client.SetNodeProperty(ctx, requestID(), *path, resolvedProp, resolvedValue)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Set %s on %s\n", result.Property, result.Path)
	if *scenePath != "" {
		savedPath, err := saveSceneAndWait(ctx, client, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene saved: %s\n", savedPath)
	}
	return nil
}

func runNodeSetResource(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node set-resource", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	property := fs.String("property", "", "property name")
	resourcePath := fs.String("resource", "", "resource path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *property == "" || *resourcePath == "" {
		return fmt.Errorf("node set-resource requires --path, --property, and --resource")
	}
	result, err := client.SetNodeResource(ctx, requestID(), *path, *property, *resourcePath)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Set %s on %s to %s\n", result.Property, result.Path, result.Resource)
	return nil
}

func runNodeAttachScript(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node attach-script", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	scriptPath := fs.String("script", "", "script resource path")
	scenePath := fs.String("scene", "", "scene path to open before attaching and save after attaching")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open/save jobs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *scriptPath == "" {
		return fmt.Errorf("node attach-script requires --path and --script")
	}
	if *scenePath != "" {
		sceneMu.Lock()
		defer sceneMu.Unlock()
		openedPath, _, err := openSceneAndWait(ctx, client, *scenePath, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene opened: %s\n", openedPath)
	}
	result, err := client.AttachScript(ctx, requestID(), *path, *scriptPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Attached script: %s -> %s\n", result.Script, result.Path)
	if *scenePath != "" {
		savedPath, err := saveSceneAndWait(ctx, client, *timeout)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Scene saved: %s\n", savedPath)
	}
	return nil
}

func runNodeGroupAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node group add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	group := fs.String("group", "", "group name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *group == "" {
		return fmt.Errorf("node group add requires --path and --group")
	}
	result, err := client.NodeGroupAdd(ctx, requestID(), *path, *group)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Added to group: %s on %s\n", result.Group, result.Path)
	return nil
}

func runNodeGroupRemove(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node group remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	group := fs.String("group", "", "group name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *group == "" {
		return fmt.Errorf("node group remove requires --path and --group")
	}
	result, err := client.NodeGroupRemove(ctx, requestID(), *path, *group)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Removed from group: %s on %s\n", result.Group, result.Path)
	return nil
}

func runNodeGroupList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node group list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node group list requires --path")
	}
	result, err := client.NodeGroupList(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Groups on %s: %s\n", result.Path, strings.Join(result.Groups, ", "))
	return nil
}

func runNodeDuplicate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node duplicate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "source node path")
	name := fs.String("name", "", "name for the duplicate")
	parent := fs.String("parent", "", "parent node path (defaults to source's parent)")
	dryRun := fs.Bool("dry-run", false, "preview without modifying scene")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("node duplicate requires --path and --name")
	}
	result, err := client.NodeDuplicate(ctx, requestID(), *path, *name, *parent, *dryRun)
	if err != nil {
		return err
	}
	if result.DryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s\n", result.Path)
		return nil
	}
	fmt.Fprintf(stdout, "Duplicated: %s (source: %s)\n", result.Path, result.SourcePath)
	return nil
}

func runNodeListProperties(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node list-properties", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("node list-properties requires --path")
	}
	result, err := client.NodeListProperties(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result.Properties)
}

func runSignalConnect(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("signal connect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "", "source node path")
	sig := fs.String("signal", "", "signal name")
	to := fs.String("to", "", "target node path")
	method := fs.String("method", "", "method name on target node")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *sig == "" || *to == "" || *method == "" {
		return fmt.Errorf("signal connect requires --from, --signal, --to, and --method")
	}
	result, err := client.SignalConnect(ctx, requestID(), *from, *sig, *to, *method)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Connected: %s::%s -> %s::%s\n", result.From, result.Signal, result.To, result.Method)
	return nil
}

func runSignalDisconnect(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("signal disconnect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "", "source node path")
	sig := fs.String("signal", "", "signal name")
	to := fs.String("to", "", "target node path")
	method := fs.String("method", "", "method name on target node")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *sig == "" || *to == "" || *method == "" {
		return fmt.Errorf("signal disconnect requires --from, --signal, --to, and --method")
	}
	result, err := client.SignalDisconnect(ctx, requestID(), *from, *sig, *to, *method)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Disconnected: %s::%s -> %s::%s\n", result.From, result.Signal, result.To, result.Method)
	return nil
}

func runProjectSettingGet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("project setting get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	key := fs.String("key", "", "project setting key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" {
		return fmt.Errorf("project setting get requires --key")
	}
	result, err := client.ProjectSettingGet(ctx, requestID(), *key)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runProjectSettingSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("project setting set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	key := fs.String("key", "", "project setting key")
	valueFlags := newTypedValueFlags(fs, "project setting set")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" {
		return fmt.Errorf("project setting set requires --key")
	}
	value, err := valueFlags.Value()
	if err != nil {
		return err
	}
	result, err := client.ProjectSettingSet(ctx, requestID(), *key, value)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Set %s\n", result.Key)
	return nil
}

func runScriptCheck(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("script check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "script path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("script check requires --path")
	}
	result, err := client.CheckScript(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("script check failed: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Script OK: %s\n", result.Path)
	return nil
}

func runScriptCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("script create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "script path")
	extends := fs.String("extends", "", "base Godot class name")
	force := fs.Bool("force", false, "overwrite an existing script")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *extends == "" {
		return fmt.Errorf("script create requires --path and --extends")
	}
	result, err := client.CreateScript(ctx, requestID(), *path, *extends, *force)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("script create produced invalid script: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Script created: %s\n", result.Path)
	return nil
}

func runScriptWrite(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("script write", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "script path")
	body := fs.String("body", "", "script body")
	bodyFile := fs.String("body-file", "", "local file containing script body")
	allowMissingPreloads := fs.Bool("allow-missing-preloads", false, "write script even if preloaded scenes/resources do not exist yet")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("script write requires --path")
	}
	if (*body == "" && *bodyFile == "") || (*body != "" && *bodyFile != "") {
		return fmt.Errorf("script write requires exactly one of --body or --body-file")
	}
	bodyText := *body
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile)
		if err != nil {
			return err
		}
		bodyText = string(data)
	}
	result, err := client.WriteScript(ctx, requestID(), *path, bodyText, *allowMissingPreloads)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("script write produced invalid script: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Script written: %s\n", result.Path)
	return nil
}

func runShaderCheck(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("shader check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "shader path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("shader check requires --path")
	}
	result, err := client.CheckShader(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("shader check failed: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Shader OK: %s\n", result.Path)
	return nil
}

func runShaderWrite(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("shader write", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "shader path")
	body := fs.String("body", "", "shader body")
	bodyFile := fs.String("body-file", "", "local file containing shader body")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("shader write requires --path")
	}
	if (*body == "" && *bodyFile == "") || (*body != "" && *bodyFile != "") {
		return fmt.Errorf("shader write requires exactly one of --body or --body-file")
	}
	bodyText := *body
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile)
		if err != nil {
			return err
		}
		bodyText = string(data)
	}
	result, err := client.WriteShader(ctx, requestID(), *path, bodyText)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("shader write produced invalid shader: %s", result.Path)
	}
	fmt.Fprintf(stdout, "Shader written: %s\n", result.Path)
	return nil
}

func runResourceCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("resource create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "resource file path (res:// .tres)")
	resourceType := fs.String("type", "", "Godot resource class name (e.g. StandardMaterial3D)")
	propFlags := stringListFlag{}
	fs.Var(&propFlags, "prop", "property in name=TYPED_JSON form, e.g. --prop albedo_color='{\"kind\":\"Color\",\"value\":[1,0,0,1]}'")
	shaderParamFlags := stringListFlag{}
	fs.Var(&shaderParamFlags, "shader-param", "ShaderMaterial parameter in name=res://path form")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *resourceType == "" {
		return fmt.Errorf("resource create requires --path and --type")
	}
	props, err := parseNameJSONPairs(propFlags)
	if err != nil {
		return err
	}
	shaderParams, err := parseNameResourcePairs(shaderParamFlags)
	if err != nil {
		return err
	}
	result, err := client.CreateResource(ctx, requestID(), *path, *resourceType, props, shaderParams)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Resource created: %s (%s)\n", result.Path, result.Type)
	return nil
}

type stringListFlag []string

func (s *stringListFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type typedValueFlags struct {
	label        string
	valueText    *string
	stringValue  *string
	intValue     *string
	floatValue   *string
	boolValue    *string
	vector2      *string
	vector3      *string
	color        *string
	resource     *string
	arrayVector2 *string
	arrayVector3 *string
	arrayString  *string
	arrayInt     *string
	arrayFloat   *string
	arrayBool    *string
}

func newTypedValueFlags(fs *flag.FlagSet, label string) *typedValueFlags {
	flags := &typedValueFlags{label: label}
	flags.valueText = fs.String("value", "", `typed JSON value, for example {"kind":"Vector2","value":[200,400]}`)
	flags.stringValue = fs.String("string", "", "string value shorthand")
	flags.intValue = fs.String("int", "", "integer value shorthand")
	flags.floatValue = fs.String("float", "", "float value shorthand")
	flags.boolValue = fs.String("bool", "", "boolean value shorthand")
	flags.vector2 = fs.String("vector2", "", "Vector2 shorthand as x,y")
	flags.vector3 = fs.String("vector3", "", "Vector3 shorthand as x,y,z")
	flags.color = fs.String("color", "", "Color shorthand as r,g,b[,a]")
	flags.resource = fs.String("resource", "", "Resource shorthand as res://path")
	flags.arrayVector2 = fs.String("array-vector2", "", "Array[Vector2] shorthand as x,y;x,y")
	flags.arrayVector3 = fs.String("array-vector3", "", "Array[Vector3] shorthand as x,y,z;x,y,z")
	flags.arrayString = fs.String("array-string", "", "Array[String] shorthand as a;b;c")
	flags.arrayInt = fs.String("array-int", "", "Array[int] shorthand as 1;2;3")
	flags.arrayFloat = fs.String("array-float", "", "Array[float] shorthand as 1.2;3.4")
	flags.arrayBool = fs.String("array-bool", "", "Array[bool] shorthand as true;false")
	return flags
}

func (f *typedValueFlags) Value() (any, error) {
	values := []struct {
		name  string
		value string
	}{
		{"value", *f.valueText},
		{"string", *f.stringValue},
		{"int", *f.intValue},
		{"float", *f.floatValue},
		{"bool", *f.boolValue},
		{"vector2", *f.vector2},
		{"vector3", *f.vector3},
		{"color", *f.color},
		{"resource", *f.resource},
		{"array-vector2", *f.arrayVector2},
		{"array-vector3", *f.arrayVector3},
		{"array-string", *f.arrayString},
		{"array-int", *f.arrayInt},
		{"array-float", *f.arrayFloat},
		{"array-bool", *f.arrayBool},
	}
	count := 0
	for _, item := range values {
		if item.value != "" {
			count++
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("%s requires a value flag: --value, --string, --int, --float, --bool, --vector2, --vector3, --color, --resource, or --array-*", f.label)
	}
	if count > 1 {
		return nil, fmt.Errorf("%s requires exactly one value flag", f.label)
	}
	if *f.valueText != "" {
		var value any
		if err := json.Unmarshal([]byte(*f.valueText), &value); err != nil {
			return nil, fmt.Errorf("%s --value must be typed JSON: %w", f.label, err)
		}
		return value, nil
	}
	if *f.stringValue != "" {
		return map[string]any{"kind": "String", "value": *f.stringValue}, nil
	}
	if *f.intValue != "" {
		v, err := strconv.Atoi(*f.intValue)
		if err != nil {
			return nil, fmt.Errorf("%s --int must be an integer: %w", f.label, err)
		}
		return map[string]any{"kind": "int", "value": v}, nil
	}
	if *f.floatValue != "" {
		v, err := strconv.ParseFloat(*f.floatValue, 64)
		if err != nil {
			return nil, fmt.Errorf("%s --float must be a number: %w", f.label, err)
		}
		return map[string]any{"kind": "float", "value": v}, nil
	}
	if *f.boolValue != "" {
		v, err := strconv.ParseBool(*f.boolValue)
		if err != nil {
			return nil, fmt.Errorf("%s --bool must be true or false: %w", f.label, err)
		}
		return map[string]any{"kind": "bool", "value": v}, nil
	}
	if *f.vector2 != "" {
		values, err := parseFloatList(*f.vector2, 2, "vector2")
		if err != nil {
			return nil, fmt.Errorf("%s --vector2 %w", f.label, err)
		}
		return map[string]any{"kind": "Vector2", "value": values}, nil
	}
	if *f.vector3 != "" {
		values, err := parseFloatList(*f.vector3, 3, "vector3")
		if err != nil {
			return nil, fmt.Errorf("%s --vector3 %w", f.label, err)
		}
		return map[string]any{"kind": "Vector3", "value": values}, nil
	}
	if *f.color != "" {
		parts := strings.Split(*f.color, ",")
		if len(parts) != 3 && len(parts) != 4 {
			return nil, fmt.Errorf("%s --color must be r,g,b or r,g,b,a", f.label)
		}
		values, err := parseFloatList(*f.color, len(parts), "color")
		if err != nil {
			return nil, fmt.Errorf("%s --color %w", f.label, err)
		}
		return map[string]any{"kind": "Color", "value": values}, nil
	}
	if *f.arrayVector2 != "" {
		values, err := parseVectorArray(*f.arrayVector2, 2, "array-vector2")
		if err != nil {
			return nil, fmt.Errorf("%s --array-vector2 %w", f.label, err)
		}
		return map[string]any{"kind": "Array[Vector2]", "value": values}, nil
	}
	if *f.arrayVector3 != "" {
		values, err := parseVectorArray(*f.arrayVector3, 3, "array-vector3")
		if err != nil {
			return nil, fmt.Errorf("%s --array-vector3 %w", f.label, err)
		}
		return map[string]any{"kind": "Array[Vector3]", "value": values}, nil
	}
	if *f.arrayString != "" {
		return map[string]any{"kind": "Array[String]", "value": parseStringArray(*f.arrayString)}, nil
	}
	if *f.arrayInt != "" {
		values, err := parseIntArray(*f.arrayInt)
		if err != nil {
			return nil, fmt.Errorf("%s --array-int %w", f.label, err)
		}
		return map[string]any{"kind": "Array[int]", "value": values}, nil
	}
	if *f.arrayFloat != "" {
		values, err := parseFloatArray(*f.arrayFloat)
		if err != nil {
			return nil, fmt.Errorf("%s --array-float %w", f.label, err)
		}
		return map[string]any{"kind": "Array[float]", "value": values}, nil
	}
	if *f.arrayBool != "" {
		values, err := parseBoolArray(*f.arrayBool)
		if err != nil {
			return nil, fmt.Errorf("%s --array-bool %w", f.label, err)
		}
		return map[string]any{"kind": "Array[bool]", "value": values}, nil
	}
	return map[string]any{"kind": "Resource", "value": *f.resource}, nil
}

func parseVectorArray(value string, want int, label string) ([][]float64, error) {
	groups := strings.Split(value, ";")
	out := make([][]float64, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		parsed, err := parseFloatList(group, want, label)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("must contain at least one vector")
	}
	return out, nil
}

func parseStringArray(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func parseIntArray(value string) ([]int, error) {
	parts := strings.Split(value, ";")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("component %q must be an integer: %w", part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseFloatArray(value string) ([]float64, error) {
	parts := strings.Split(value, ";")
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, fmt.Errorf("component %q must be a number: %w", part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseBoolArray(value string) ([]bool, error) {
	parts := strings.Split(value, ";")
	out := make([]bool, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseBool(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("component %q must be true or false: %w", part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseFloatList(value string, want int, label string) ([]float64, error) {
	parts := strings.Split(value, ",")
	if len(parts) != want {
		return nil, fmt.Errorf("must be %d comma-separated numbers", want)
	}
	out := make([]float64, 0, want)
	for _, part := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, fmt.Errorf("%s component %q must be a number: %w", label, part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseNameJSONPairs(values []string) (map[string]any, error) {
	out := map[string]any{}
	for _, value := range values {
		name, jsonVal, ok := strings.Cut(value, "=")
		if !ok || name == "" || jsonVal == "" {
			return nil, fmt.Errorf("--prop must use name=JSON_VALUE")
		}
		var decoded any
		if err := json.Unmarshal([]byte(jsonVal), &decoded); err != nil {
			return nil, fmt.Errorf("--prop %s value must be typed JSON: %w", name, err)
		}
		out[name] = decoded
	}
	return out, nil
}

func parseNameRawJSONPairs(values []string) (map[string]any, error) {
	out := map[string]any{}
	for _, value := range values {
		name, jsonVal, ok := strings.Cut(value, "=")
		if !ok || name == "" || jsonVal == "" {
			return nil, fmt.Errorf("--param must use name=JSON_VALUE")
		}
		var decoded any
		if err := json.Unmarshal([]byte(jsonVal), &decoded); err != nil {
			return nil, fmt.Errorf("--param %s value must be JSON: %w", name, err)
		}
		out[name] = decoded
	}
	return out, nil
}

func parseNameResourcePairs(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, value := range values {
		name, resourcePath, ok := strings.Cut(value, "=")
		if !ok || name == "" || resourcePath == "" {
			return nil, fmt.Errorf("--texture-param must use name=res://path")
		}
		if !strings.HasPrefix(resourcePath, "res://") {
			return nil, fmt.Errorf("--texture-param resource must be a res:// path: %s", resourcePath)
		}
		out[name] = resourcePath
	}
	return out, nil
}

func runFileWriteBytes(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file write-bytes", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "resource file path")
	inPath := fs.String("in", "", "local input file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *inPath == "" {
		return fmt.Errorf("file write-bytes requires --path and --in")
	}
	data, err := os.ReadFile(*inPath)
	if err != nil {
		return err
	}
	result, err := client.WriteFileBytes(ctx, requestID(), *path, base64.StdEncoding.EncodeToString(data))
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "File written: %s (%d bytes)\n", result.Path, result.Bytes)
	return nil
}

func runFileList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// directory path")
	recursive := fs.Bool("recursive", false, "list recursively")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file list requires --path")
	}
	result, err := client.FileList(ctx, requestID(), *path, *recursive)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runFileMkdir(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file mkdir", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// directory path to create")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file mkdir requires --path")
	}
	result, err := client.FileMkdir(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Created: %s\n", result.Path)
	return nil
}

func runFileDelete(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// path to delete")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file delete requires --path")
	}
	result, err := client.FileDelete(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Deleted: %s\n", result.Path)
	return nil
}

func runFileExists(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file exists", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "res:// path to check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("file exists requires --path")
	}
	result, err := client.FileExists(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runNavigationBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("navigation bake", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "NavigationRegion node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("navigation bake requires --path")
	}
	result, err := client.NavigationBake(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Baked: %s (%s)\n", result.Path, result.Kind)
	return nil
}

type edgeProfile struct {
	ID    int     `json:"id"`
	Mode  string  `json:"mode"`
	Mix   float64 `json:"mix"`
	Blur  float64 `json:"blur"`
	Width float64 `json:"width"`
}

func runLUTWrite(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file lut-write", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "resource PNG path")
	profilesPath := fs.String("profiles", "", "local edge profile JSON path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *profilesPath == "" {
		return fmt.Errorf("file lut-write requires --path and --profiles")
	}
	data, err := os.ReadFile(*profilesPath)
	if err != nil {
		return err
	}
	var profiles []edgeProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("parse edge profiles: %w", err)
	}
	pngData, err := buildEdgeLUTPNG(profiles)
	if err != nil {
		return err
	}
	result, err := client.WriteFileBytes(ctx, requestID(), *path, base64.StdEncoding.EncodeToString(pngData))
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "LUT written: %s (%d bytes)\n", result.Path, result.Bytes)
	fmt.Fprintf(stdout, "Profiles: %d\n", len(profiles))
	return nil
}

func buildEdgeLUTPNG(profiles []edgeProfile) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, 256, 1))
	for _, profile := range profiles {
		if profile.ID < 0 || profile.ID > 255 {
			return nil, fmt.Errorf("edge profile id must be between 0 and 255: %d", profile.ID)
		}
		img.SetNRGBA(profile.ID, 0, color.NRGBA{
			R: byteFromUnit(profile.Mix),
			G: byteFromUnit(profile.Blur),
			B: byteFromUnit(profile.Width),
			A: byteFromUnit(modeValue(profile.Mode)),
		})
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func byteFromUnit(value float64) uint8 {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return uint8(value*255 + 0.5)
}

func modeValue(mode string) float64 {
	switch strings.ToLower(mode) {
	case "", "off":
		return 0.0
	case "blur":
		return 0.33
	case "bleed":
		return 0.66
	case "preserve", "ink-preserve", "crisp":
		return 1.0
	default:
		return 0.0
	}
}

func runViewportScreenshot(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viewport screenshot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outPath := fs.String("out", "", "local PNG output path")
	kind := fs.String("kind", "2d", "editor viewport kind: 2d or 3d")
	index := fs.Int("index", 0, "3D viewport index")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for screenshot job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outPath == "" {
		return fmt.Errorf("viewport screenshot requires --out")
	}
	if *kind != "2d" && *kind != "3d" {
		return fmt.Errorf("viewport screenshot --kind must be 2d or 3d")
	}
	result, err := client.ScreenshotViewport(ctx, requestID(), *kind, *index)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("viewport screenshot did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "viewport screenshot")
	if err != nil {
		return err
	}
	if err := writeScreenshotJob(*outPath, job); err != nil {
		return err
	}
	width := intFromJobResult(job.Result["width"])
	height := intFromJobResult(job.Result["height"])
	if width > 0 && height > 0 {
		fmt.Fprintf(stdout, "Screenshot written: %s (%dx%d)\n", *outPath, width, height)
	} else {
		fmt.Fprintf(stdout, "Screenshot written: %s\n", *outPath)
	}
	return nil
}

func runViewportSetSize(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viewport set-size", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	width := fs.Int("width", 0, "viewport width in pixels")
	height := fs.Int("height", 0, "viewport height in pixels")
	path := fs.String("path", "", "SubViewport node path (empty = main viewport)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *width <= 0 || *height <= 0 {
		return fmt.Errorf("viewport set-size requires --width and --height > 0")
	}
	result, err := client.ViewportSetSize(ctx, requestID(), *width, *height, *path)
	if err != nil {
		return err
	}
	if result.Path != "" {
		fmt.Fprintf(stdout, "Viewport size set: %s (%dx%d)\n", result.Path, result.Width, result.Height)
	} else {
		fmt.Fprintf(stdout, "Viewport size set: %dx%d\n", result.Width, result.Height)
	}
	return nil
}

func runViewportAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viewport add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "parent node path (optional)")
	width := fs.Int("width", 320, "SubViewport width")
	height := fs.Int("height", 0, "SubViewport height")
	addCamera := fs.Bool("camera", true, "add a Camera3D child")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *width <= 0 || *height <= 0 {
		return fmt.Errorf("viewport add requires --width and --height > 0")
	}
	result, err := client.ViewportAdd(ctx, requestID(), *parent, *width, *height, *addCamera)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "SubViewport added: %s (%dx%d)\n", result.Path, result.Width, result.Height)
	return nil
}

func runSceneApplyBlueprint(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene apply-blueprint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path")
	blueprint := fs.String("blueprint", "", "blueprint name: player3d, spotlight, trigger_area, hud_label")
	dryRun := fs.Bool("dry-run", false, "validate without mutating")
	propFlags := stringListFlag{}
	fs.Var(&propFlags, "prop", "override property in name=TYPED_JSON form")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *blueprint == "" {
		return fmt.Errorf("scene apply-blueprint requires --path and --blueprint")
	}
	props, err := parseNameJSONPairs(propFlags)
	if err != nil {
		return err
	}
	result, err := client.ApplyBlueprint(ctx, requestID(), *path, *blueprint, props, *dryRun)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run ok: %s (%s, %d nodes)\n", result.Path, result.Blueprint, result.Created)
		return nil
	}
	fmt.Fprintf(stdout, "Blueprint applied: %s (%s, %d nodes)\n", result.Path, result.Blueprint, result.Created)
	return nil
}

func runRunProbeRaycast(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run probe raycast", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "output as JSON")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for raycast job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	queued, err := client.RunRaycast(ctx, requestID())
	if err != nil {
		return err
	}
	var result bridge.RunRaycastResult
	if queued.JobID != "" {
		job, err := waitForJob(ctx, client, queued.JobID, *timeout, "run probe raycast")
		if err != nil {
			return err
		}
		encoded, _ := json.Marshal(job.Result)
		_ = json.Unmarshal(encoded, &result)
	} else {
		result = queued
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	if result.Hit {
		fmt.Fprintf(stdout, "Raycast hit: %s at distance %.3f\n", result.HitCollider, result.HitDistance)
	} else {
		fmt.Fprintln(stdout, "Raycast: no hit")
	}
	if result.CameraPath != "" {
		fmt.Fprintf(stdout, "Camera: %s\n", result.CameraPath)
	}
	return nil
}

func runThemeCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path (res://ui/main.tres)")
	force := fs.Bool("force", false, "overwrite existing theme")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("theme create requires --path")
	}
	result, err := client.ThemeCreate(ctx, requestID(), *path, *force)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme created: %s\n", result.Path)
	return nil
}

func runThemeSetColor(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme set-color", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path")
	nodeType := fs.String("node-type", "", "Godot node type (e.g. Label)")
	name := fs.String("name", "", "color override name (e.g. font_color)")
	value := fs.String("value", "", "RGBA as r,g,b,a with 0-1 floats or rrggbbaa hex")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *nodeType == "" || *name == "" || *value == "" {
		return fmt.Errorf("theme set-color requires --path, --node-type, --name, --value")
	}
	rgba, err := parseColorValue(*value)
	if err != nil {
		return err
	}
	result, err := client.ThemeSetColor(ctx, requestID(), *path, *nodeType, *name, rgba)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme color set: %s/%s on %s\n", result.NodeType, result.Name, result.Path)
	return nil
}

func runThemeSetFontSize(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme set-font-size", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path")
	nodeType := fs.String("node-type", "", "Godot node type (e.g. Label)")
	name := fs.String("name", "", "font size name (e.g. font_size)")
	size := fs.Int("value", 0, "font size in pixels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *nodeType == "" || *name == "" || *size <= 0 {
		return fmt.Errorf("theme set-font-size requires --path, --node-type, --name, --value > 0")
	}
	result, err := client.ThemeSetFontSize(ctx, requestID(), *path, *nodeType, *name, *size)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme font size set: %s/%s=%d on %s\n", result.NodeType, result.Name, *size, result.Path)
	return nil
}

func runThemeSetConstant(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("theme set-constant", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "theme resource path")
	nodeType := fs.String("node-type", "", "Godot node type (e.g. MarginContainer)")
	name := fs.String("name", "", "constant name (e.g. margin_top)")
	value := fs.Int("value", 0, "integer constant value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *nodeType == "" || *name == "" {
		return fmt.Errorf("theme set-constant requires --path, --node-type, --name, --value")
	}
	result, err := client.ThemeSetConstant(ctx, requestID(), *path, *nodeType, *name, *value)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Theme constant set: %s/%s=%d on %s\n", result.NodeType, result.Name, *value, result.Path)
	return nil
}

func parseColorValue(s string) ([4]float64, error) {
	// Support r,g,b,a (0-1 floats) or rrggbbaa / rrggbb hex
	if strings.ContainsRune(s, ',') {
		parts := strings.Split(s, ",")
		if len(parts) < 3 || len(parts) > 4 {
			return [4]float64{}, fmt.Errorf("color must be r,g,b or r,g,b,a: %q", s)
		}
		var out [4]float64
		out[3] = 1.0
		for i, p := range parts {
			f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return out, fmt.Errorf("invalid color component %q: %w", p, err)
			}
			out[i] = f
		}
		return out, nil
	}
	// hex
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 {
		s += "ff"
	}
	if len(s) != 8 {
		return [4]float64{}, fmt.Errorf("hex color must be rrggbb or rrggbbaa: %q", s)
	}
	var out [4]float64
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return out, fmt.Errorf("invalid hex color %q: %w", s, err)
		}
		out[i] = float64(v) / 255.0
	}
	return out, nil
}

func runAnimationCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path (res://anims/player.tres)")
	name := fs.String("name", "", "animation name")
	length := fs.Float64("length", 1.0, "animation length in seconds")
	loop := fs.Bool("loop", false, "enable looping")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *name == "" {
		return fmt.Errorf("animation create requires --path and --name")
	}
	result, err := client.AnimationCreate(ctx, requestID(), *path, *name, *length, *loop)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Animation created: %s in %s\n", result.Name, result.Path)
	return nil
}

func runAnimationTrackAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation track-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path")
	animation := fs.String("animation", "", "animation name")
	nodePath := fs.String("node-path", "", "node path for the track")
	property := fs.String("property", "", "property name for the track")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *animation == "" || *nodePath == "" || *property == "" {
		return fmt.Errorf("animation track-add requires --path, --animation, --node-path, --property")
	}
	result, err := client.AnimationTrackAdd(ctx, requestID(), *path, *animation, *nodePath, *property)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Track added: index %d in %s/%s\n", result.TrackIdx, result.Path, result.Animation)
	return nil
}

func runAnimationKeyframeAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation keyframe-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path")
	animation := fs.String("animation", "", "animation name")
	trackIdx := fs.Int("track-idx", 0, "track index")
	timePos := fs.Float64("time", 0, "keyframe time in seconds")
	valueStr := fs.String("value", "", "keyframe value as TYPED_JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *animation == "" || *valueStr == "" {
		return fmt.Errorf("animation keyframe-add requires --path, --animation, --value")
	}
	var value any
	if err := json.Unmarshal([]byte(*valueStr), &value); err != nil {
		return fmt.Errorf("animation keyframe-add --value must be JSON: %w", err)
	}
	result, err := client.AnimationKeyframeAdd(ctx, requestID(), *path, *animation, *trackIdx, *timePos, value)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Keyframe added: track %d at t=%.3f in %s/%s\n", result.TrackIdx, *timePos, result.Path, result.Animation)
	return nil
}

func runAnimationLengthSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation length-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "AnimationLibrary resource path")
	animation := fs.String("animation", "", "animation name")
	length := fs.Float64("length", 0, "animation length in seconds")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *animation == "" || *length <= 0 {
		return fmt.Errorf("animation length-set requires --path, --animation, --length > 0")
	}
	result, err := client.AnimationLengthSet(ctx, requestID(), *path, *animation, *length)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Animation length set: %s in %s to %.3fs\n", result.Name, result.Path, *length)
	return nil
}

func runAnimationPlayerPlay(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("animation player-play", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	nodePath := fs.String("node-path", "", "AnimationPlayer node path in running scene")
	animation := fs.String("animation", "", "animation name to play")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *nodePath == "" || *animation == "" {
		return fmt.Errorf("animation player-play requires --node-path and --animation")
	}
	_, err := client.AnimationPlayerPlay(ctx, requestID(), *nodePath, *animation)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Playing animation: %s on %s\n", *animation, *nodePath)
	return nil
}

func runTilesetCreate(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap tileset-create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "TileSet resource path (res://tilesets/world.tres)")
	tileW := fs.Int("tile-width", 16, "tile width in pixels")
	tileH := fs.Int("tile-height", 16, "tile height in pixels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("tilemap tileset-create requires --path")
	}
	result, err := client.TilesetCreate(ctx, requestID(), *path, *tileW, *tileH)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "TileSet created: %s\n", result.Path)
	return nil
}

func runTilesetSourceAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap source-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "TileSet resource path")
	texture := fs.String("texture", "", "texture resource path (res://textures/tileset.png)")
	tileW := fs.Int("tile-width", 16, "tile width in pixels")
	tileH := fs.Int("tile-height", 16, "tile height in pixels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *texture == "" {
		return fmt.Errorf("tilemap source-add requires --path and --texture")
	}
	result, err := client.TilesetSourceAdd(ctx, requestID(), *path, *texture, *tileW, *tileH)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "TileSet source added: %s\n", result.Path)
	return nil
}

func runTilemapCellSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap cell-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	node := fs.String("node", "", "TileMap node path")
	layer := fs.Int("layer", 0, "tile layer index")
	x := fs.Int("x", 0, "cell x coordinate")
	y := fs.Int("y", 0, "cell y coordinate")
	sourceID := fs.Int("source-id", 0, "TileSet source id")
	atlasX := fs.Int("atlas-x", 0, "atlas x coordinate")
	atlasY := fs.Int("atlas-y", 0, "atlas y coordinate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *node == "" {
		return fmt.Errorf("tilemap cell-set requires --node")
	}
	result, err := client.TilemapCellSet(ctx, requestID(), *node, *layer, *x, *y, *sourceID, *atlasX, *atlasY)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Cell set: %s [%d,%d] layer %d\n", result.Node, *x, *y, *layer)
	return nil
}

func runTilemapCellClear(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("tilemap cell-clear", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	node := fs.String("node", "", "TileMap node path")
	layer := fs.Int("layer", 0, "tile layer index")
	x := fs.Int("x", 0, "cell x coordinate")
	y := fs.Int("y", 0, "cell y coordinate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *node == "" {
		return fmt.Errorf("tilemap cell-clear requires --node")
	}
	result, err := client.TilemapCellClear(ctx, requestID(), *node, *layer, *x, *y)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Cell cleared: %s [%d,%d] layer %d\n", result.Node, *x, *y, *layer)
	return nil
}

func runAudioBusAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("audio bus-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "audio bus name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("audio bus-add requires --name")
	}
	result, err := client.AudioBusAdd(ctx, requestID(), *name)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Audio bus added: %s\n", result.Bus)
	return nil
}

func runAudioBusVolumeSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("audio bus-volume-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "audio bus name")
	volumeDB := fs.Float64("volume-db", 0, "volume in decibels")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("audio bus-volume-set requires --name")
	}
	result, err := client.AudioBusVolumeSet(ctx, requestID(), *name, *volumeDB)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Audio bus volume set: %s = %.1f dB\n", result.Bus, *volumeDB)
	return nil
}

func runAudioBusEffectAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("audio bus-effect-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "audio bus name")
	effectType := fs.String("effect-type", "", "AudioEffect subclass (e.g. AudioEffectReverb)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *effectType == "" {
		return fmt.Errorf("audio bus-effect-add requires --name and --effect-type")
	}
	result, err := client.AudioBusEffectAdd(ctx, requestID(), *name, *effectType)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Audio effect added: %s on %s\n", *effectType, result.Bus)
	return nil
}

func writeScreenshotJob(outPath string, job bridge.Job) error {
	content, _ := job.Result["content_base64"].(string)
	if content == "" {
		return fmt.Errorf("%s job did not return PNG data", strings.ReplaceAll(job.Kind, ".", " "))
	}
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return fmt.Errorf("decode screenshot PNG: %w", err)
	}
	if dir := filepath.Dir(outPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return err
	}
	return nil
}

func defaultScreenshotPath() string {
	return filepath.Join("screenshots", time.Now().UTC().Format("20060102-150405")+".png")
}

func runImportSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("import set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "asset path (res://textures/player.png)")
	paramFlags := stringListFlag{}
	fs.Var(&paramFlags, "param", "import param in name=VALUE form where VALUE is raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("import set requires --path")
	}
	params, err := parseNameRawJSONPairs(paramFlags)
	if err != nil {
		return err
	}
	result, err := client.ImportSet(ctx, requestID(), *path, params)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Import set: %s (%d params)\n", result.Path, result.Params)
	return nil
}

func runSceneList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "res://", "res:// directory to search")
	recursive := fs.Bool("recursive", true, "search recursively")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.SceneList(ctx, requestID(), *dir, *recursive)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runResourceList(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("resource list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "res://", "res:// directory to search")
	recursive := fs.Bool("recursive", true, "search recursively")
	ext := fs.String("ext", "", "file extension filter (e.g. .tres)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.ResourceList(ctx, requestID(), *dir, *ext, *recursive)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runProjectRun(ctx context.Context, cfg bridge.Config, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("project run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scene := fs.String("scene", "", "scene to run (res://main.tscn); omit to use the project main scene")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum time to wait for Godot to exit")
	godot := fs.String("godot", cfg.GodotPath, "headless Godot binary path")
	project := fs.String("project", cfg.Project, "Godot project path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *godot == "" {
		return fmt.Errorf("project run requires a headless Godot binary: set GDCTL_GODOT_PATH, --godot, or pass --godot PATH")
	}
	if *project == "" {
		return fmt.Errorf("project run requires --project or GDCTL_PROJECT pointing to the Godot project directory")
	}
	return execGodotHeadless(ctx, *godot, *project, *scene, *timeout, stdout)
}

func runSceneRun(ctx context.Context, cfg bridge.Config, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scene run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "scene path (res://main.tscn)")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum time to wait for Godot to exit")
	godot := fs.String("godot", cfg.GodotPath, "headless Godot binary path")
	project := fs.String("project", cfg.Project, "Godot project path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("scene run requires --path")
	}
	if *godot == "" {
		return fmt.Errorf("scene run requires a headless Godot binary: set GDCTL_GODOT_PATH, --godot, or pass --godot PATH")
	}
	if *project == "" {
		return fmt.Errorf("scene run requires --project or GDCTL_PROJECT pointing to the Godot project directory")
	}
	return execGodotHeadless(ctx, *godot, *project, *path, *timeout, stdout)
}

func execGodotHeadless(ctx context.Context, godotBin, projectPath, scene string, timeout time.Duration, stdout io.Writer) error {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	argv := []string{"--headless", "--path", projectPath}
	if scene != "" {
		argv = append(argv, scene)
	}
	cmd := exec.CommandContext(runCtx, godotBin, argv...)
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return fmt.Errorf("project run timed out after %s", timeout)
		}
		return fmt.Errorf("project run failed: %w", err)
	}
	return nil
}

func intFromJobResult(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func runDoctor(ctx context.Context, cfg bridge.Config, client *bridge.Client, manager addon.Manager, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	project := fs.String("project", cfg.Project, "Godot project path")
	fix := fs.Bool("fix", false, "install and enable the addon when needed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg.Project = *project

	fmt.Fprintln(stdout, "Godot TCP Bridge Doctor")
	fmt.Fprintln(stdout)

	usable := true
	if cfg.Project == "" && *fix {
		if err := runBridgeAddonUpdate(ctx, client, manager, stdout); err != nil {
			usable = false
			fmt.Fprintf(stdout, "[fail] projectless addon update: %v\n", err)
		} else {
			fmt.Fprintln(stdout, "[ok] projectless addon update requested")
		}
	}
	if cfg.Project != "" {
		status, actions, err := manager.Doctor(ctx, addon.DoctorOptions{ProjectPath: cfg.Project, BridgeConfig: cfg, Fix: *fix})
		for _, action := range actions {
			fmt.Fprintf(stdout, "[fix] %s\n", action)
		}
		if err != nil {
			usable = false
			fmt.Fprintf(stdout, "[fail] addon doctor: %v\n", err)
		} else {
			if status.Installed {
				fmt.Fprintln(stdout, "[ok] addon installed")
			}
			if status.Enabled {
				fmt.Fprintln(stdout, "[ok] addon enabled")
			}
			if status.Compatible {
				fmt.Fprintln(stdout, "[ok] addon compatible")
			}
		}
	}

	resolved := false
	if addrs, err := net.LookupHost(cfg.Host); err == nil && len(addrs) > 0 {
		resolved = true
		fmt.Fprintf(stdout, "[ok] %s resolved\n", cfg.Host)
	} else {
		usable = false
		fmt.Fprintf(stdout, "[fail] %s could not be resolved\n", cfg.Host)
	}

	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if resolved {
		if err := client.Dial(dialCtx); err == nil {
			fmt.Fprintf(stdout, "[ok] bridge reachable at %s\n", cfg.Address())
		} else {
			usable = false
			fmt.Fprintf(stdout, "[fail] bridge unreachable at %s: %v\n", cfg.Address(), err)
		}
	}

	ping, err := client.Ping(ctx)
	if err == nil && ping.OK {
		fmt.Fprintln(stdout, "[ok] ping returned ok")
		if ping.ProjectName != "" {
			fmt.Fprintln(stdout, "[ok] Godot editor project is open")
		} else {
			fmt.Fprintln(stdout, "[warn] Godot editor project name is empty")
		}
		if ping.PluginVersion != "" {
			fmt.Fprintf(stdout, "[ok] plugin version %s\n", ping.PluginVersion)
		} else {
			fmt.Fprintln(stdout, "[warn] plugin version is empty")
		}
	} else {
		usable = false
		if err != nil {
			fmt.Fprintf(stdout, "[fail] ping failed: %v\n", err)
		} else {
			fmt.Fprintln(stdout, "[fail] ping returned not ok")
		}
	}

	if cfg.Token == "" {
		fmt.Fprintln(stdout, "[warn] no mutation token configured")
	} else {
		fmt.Fprintln(stdout, "[ok] mutation token configured")
	}
	if cfg.GodotPath == "" {
		fmt.Fprintln(stdout, "[warn] no headless Godot configured (set GDCTL_GODOT_PATH or --godot)")
	} else {
		fmt.Fprintf(stdout, "[ok] headless Godot: %s\n", cfg.GodotPath)
	}
	fmt.Fprintln(stdout)
	if usable {
		fmt.Fprintln(stdout, "Result: usable")
		return nil
	}
	fmt.Fprintln(stdout, "Result: unusable")
	return fmt.Errorf("doctor found bridge problems")
}

func requestID() string {
	return "cli-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

type helpFlag struct {
	name  string
	meta  string
	usage string
}

type helpCmd struct {
	sub   string
	line  string
	desc  string
	flags []helpFlag
	notes []string
}

type helpGroup struct {
	name string
	cmds []helpCmd
}

var helpGroups = []helpGroup{
	{name: "global", cmds: []helpCmd{
		{
			sub:  "ping",
			line: "  gdctl [--host host] [--port port] [--token token] [--project path] ping",
			desc: "check bridge connectivity",
		},
		{
			sub:  "doctor",
			line: "  gdctl [--host host] [--port port] [--token token] [--project path] doctor [--project PATH] [--fix]",
			desc: "diagnose addon and bridge setup",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
				{name: "fix", usage: "install and enable the addon when needed"},
			},
		},
		{
			sub:  "help",
			line: "  gdctl help [topic]",
			desc: "show usage information",
			flags: []helpFlag{
				{name: "topic", meta: "TOPIC", usage: "command group or specific command (e.g. scene, scene create, scene.create)"},
			},
		},
	}},
	{name: "addon", cmds: []helpCmd{
		{
			sub:  "install",
			line: "  gdctl addon install --project PATH [--force]",
			desc: "install the addon into a Godot project",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
				{name: "force", usage: "overwrite conflicting addon files"},
			},
		},
		{
			sub:  "enable",
			line: "  gdctl addon enable --project PATH",
			desc: "enable the addon in project.godot",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
			},
		},
		{
			sub:  "disable",
			line: "  gdctl addon disable --project PATH",
			desc: "disable the addon in project.godot",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
			},
		},
		{
			sub:  "status",
			line: "  gdctl addon status [--project PATH] [--json]",
			desc: "show addon installation and runtime status",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path (omit to query the live bridge)"},
				{name: "json", usage: "write status as JSON"},
			},
		},
		{
			sub:  "update",
			line: "  gdctl addon update [--project PATH]",
			desc: "update the addon files",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path (omit to update over the bridge)"},
			},
		},
		{
			sub:  "rollback",
			line: "  gdctl addon rollback --project PATH [--backup PATH]",
			desc: "restore addon files from a filesystem backup",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
				{name: "backup", meta: "PATH", usage: "backup directory to restore (defaults to latest)"},
			},
			notes: []string{
				"Use this when a bad addon update prevents the bridge from starting.",
			},
		},
		{
			sub:  "remove",
			line: "  gdctl addon remove --project PATH",
			desc: "remove the addon from a Godot project",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
			},
		},
		{
			sub:  "doctor",
			line: "  gdctl addon doctor [--project PATH] [--fix]",
			desc: "diagnose addon setup and optionally fix issues",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path (omit to use runtime mode)"},
				{name: "fix", usage: "install and enable the addon when needed"},
			},
		},
	}},
	{name: "bridge", cmds: []helpCmd{
		{
			sub:  "info",
			line: "  gdctl [--host host] [--port port] bridge info",
			desc: "show bridge connection details",
		},
		{
			sub:  "logs",
			line: "  gdctl [--host host] [--port port] [--token token] bridge logs [--json] [--clear]",
			desc: "read bridge log entries",
			flags: []helpFlag{
				{name: "json", usage: "write logs as JSON"},
				{name: "clear", usage: "clear logs after reading"},
			},
		},
		{
			sub:  "addon-update",
			line: "  gdctl [--host host] [--port port] [--token token] bridge addon-update",
			desc: "update the addon over the bridge",
		},
	}},
	{name: "run", cmds: []helpCmd{
		{
			sub:  "start",
			line: "  gdctl [--host host] [--port port] [--token token] run start [--scene SCENE | --main] [--clear-logs=false]",
			desc: "start a scene from the already-open Godot editor",
			flags: []helpFlag{
				{name: "scene", meta: "SCENE", usage: "scene to run with the editor (res://main.tscn)"},
				{name: "main", usage: "run the project main scene"},
				{name: "clear-logs", meta: "BOOL", usage: "clear run logs before starting (default true)"},
			},
			notes: []string{
				"Omit --scene and --main to run the current editor scene.",
				"This uses the host editor through the bridge, so it does not require GDCTL_GODOT_PATH.",
			},
		},
		{
			sub:  "status",
			line: "  gdctl [--host host] [--port port] [--token token] run status",
			desc: "show whether the editor is currently running a scene",
		},
		{
			sub:  "stop",
			line: "  gdctl [--host host] [--port port] [--token token] run stop",
			desc: "stop the scene currently running from the editor",
		},
		{
			sub:  "logs",
			line: "  gdctl [--host host] [--port port] [--token token] run logs [--json] [--clear] [--source SOURCE] [--latest] [--since-start]",
			desc: "read run/debug logs captured by the bridge",
			flags: []helpFlag{
				{name: "json", usage: "write logs as JSON"},
				{name: "clear", usage: "clear run logs after reading"},
				{name: "source", meta: "SOURCE", usage: "filter by log source (e.g. runtime.game)"},
				{name: "latest", usage: "keep only the most-recent entry per distinct source"},
				{name: "since-start", usage: "exclude entries logged before the current run start"},
			},
		},
		{
			sub:  "screenshot",
			line: "  gdctl [--host host] [--port port] [--token token] run screenshot [--out FILE] [--source game|screen] [--screen N]",
			desc: "capture the running game viewport or host screen",
			flags: []helpFlag{
				{name: "out", meta: "FILE", usage: "local PNG output path (default screenshots/YYYYMMDD-HHMMSS.png)"},
				{name: "source", meta: "SOURCE", usage: "screenshot source: game or screen (default game)"},
				{name: "screen", meta: "N", usage: "host display screen index when --source screen is used (default 0)"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for screenshot job (default 5s)"},
			},
			notes: []string{
				"Game screenshots require the gdctl runtime helper autoload installed by run start.",
				"Use --source screen for the legacy whole-host-screen capture.",
			},
		},
		{
			sub:  "input",
			line: "  gdctl [--host host] [--port port] [--token token] run input --file input.json [--timeout DURATION] [--summary-probe SOURCE]",
			desc: "play a short input sequence into the running game",
			flags: []helpFlag{
				{name: "file", meta: "FILE", usage: "input JSON file containing steps"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for input job (default 5s)"},
				{name: "summary-probe", meta: "SOURCE", usage: "after input completes, print the latest log entry for this source"},
			},
		},
		{
			sub:  "wait-probe",
			line: "  gdctl [--host host] [--port port] [--token token] run wait-probe --source SOURCE --assert KEY OP VALUE [--timeout DURATION] [--json]",
			desc: "poll run logs until a probe field satisfies a predicate or timeout fires",
			flags: []helpFlag{
				{name: "source", meta: "SOURCE", usage: "log source to watch (e.g. runtime.game)"},
				{name: "assert", meta: "EXPR", usage: "predicate expression, e.g. targets_disabled>=1 (ops: >= <= > < == !=)"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait (default 30s)"},
				{name: "json", usage: "print matching entry as JSON"},
			},
		},
		{
			sub:  "probe raycast",
			line: "  gdctl [--host host] [--port port] [--token token] run probe raycast [--json] [--timeout DURATION]",
			desc: "fire a center-screen ray in the running 3D game and report the hit",
			flags: []helpFlag{
				{name: "json", usage: "print result as JSON"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for raycast result (default 5s)"},
			},
			notes: []string{
				"Requires GdctlRuntimeBridge autoload and an active Camera3D in the running scene.",
			},
		},
		{
			sub:  "smoke",
			line: "  gdctl [--host host] [--port port] [--token token] run smoke [--scene SCENE | --main] [--input FILE] [--assert SOURCE:KEY>=VALUE] [--screenshot OUT] [--timeout DURATION] [--keep-running]",
			desc: "one-shot automated test: start, optionally inject input, probe, screenshot, then stop",
			flags: []helpFlag{
				{name: "scene", meta: "SCENE", usage: "scene to run (res://)"},
				{name: "main", usage: "run the project main scene"},
				{name: "input", meta: "FILE", usage: "input JSON file to inject after start"},
				{name: "assert", meta: "SOURCE:KEY>=VALUE", usage: "wait for a probe predicate before proceeding"},
				{name: "screenshot", meta: "FILE", usage: "capture game viewport to this PNG path"},
				{name: "timeout", meta: "DURATION", usage: "total time limit for the smoke run (default 30s)"},
				{name: "keep-running", usage: "do not stop the scene after the test"},
			},
			notes: []string{
				"Exits with code 0 on pass, 1 on failure.",
				"Prints 'Smoke: PASS' or 'Smoke: FAIL — <reason>'.",
			},
		},
	}},
	{name: "scene", cmds: []helpCmd{
		{
			sub:  "create",
			line: "  gdctl [--host host] [--port port] [--token token] scene create --path PATH --root TYPE --name NAME [--force]",
			desc: "create a new scene file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "scene file path (e.g. res://scenes/Main.tscn)"},
				{name: "root", meta: "TYPE", usage: "root node type (e.g. Node2D, Node3D)"},
				{name: "name", meta: "NAME", usage: "root node name"},
				{name: "force", usage: "overwrite an existing scene file"},
			},
		},
		{
			sub:  "open",
			line: "  gdctl [--host host] [--port port] [--token token] scene open --path PATH",
			desc: "open a scene in the editor",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "scene file path (e.g. res://scenes/Main.tscn)"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for open job (default 5s)"},
			},
		},
		{
			sub:  "instance",
			line: "  gdctl [--host host] [--port port] [--token token] scene instance --parent PATH --scene SCENE --name NAME",
			desc: "instance a scene under a parent node",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "scene", meta: "SCENE", usage: "scene resource path (res://)"},
				{name: "name", meta: "NAME", usage: "instance node name"},
			},
		},
		{
			sub:  "tree",
			line: "  gdctl [--host host] [--port port] [--token token] scene tree",
			desc: "print the current scene node tree",
		},
		{
			sub:  "save",
			line: "  gdctl [--host host] [--port port] [--token token] scene save",
			desc: "save the current scene",
			flags: []helpFlag{
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for save job (default 5s)"},
			},
		},
		{
			sub:  "apply",
			line: "  gdctl [--host host] [--port port] [--token token] scene apply --path SCENE --file TREE.json [--dry-run]",
			desc: "apply a JSON node tree to a scene and save it",
			flags: []helpFlag{
				{name: "path", meta: "SCENE", usage: "scene to open and mutate (res://main.tscn)"},
				{name: "file", meta: "FILE", usage: "JSON scene tree file"},
				{name: "dry-run", usage: "validate without mutating or saving"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for open/save jobs (default 5s)"},
			},
			notes: []string{
				"Tree nodes use name, type, properties, and children fields.",
				"Properties use the same typed JSON values as node set, including inline Resource values.",
			},
		},
		{
			sub:  "list",
			line: "  gdctl [--host host] [--port port] [--token token] scene list [--dir res://] [--recursive]",
			desc: "list .tscn files in the project",
			flags: []helpFlag{
				{name: "dir", meta: "PATH", usage: "res:// directory to search (default res://)"},
				{name: "recursive", usage: "search recursively (default true)"},
			},
		},
		{
			sub:  "apply-blueprint",
			line: "  gdctl [--host host] [--port port] [--token token] scene apply-blueprint --path SCENE --blueprint NAME [--dry-run] [--timeout DURATION]",
			desc: "apply a named node-tree blueprint to the current scene",
			flags: []helpFlag{
				{name: "path", meta: "SCENE", usage: "scene to open and mutate (res://main.tscn)"},
				{name: "blueprint", meta: "NAME", usage: "blueprint name: player3d, spotlight, trigger_area, hud_label"},
				{name: "dry-run", usage: "validate without mutating or saving"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for open/save jobs (default 5s)"},
			},
		},
		{
			sub:  "run",
			line: "  gdctl [--project PATH] [--godot PATH] scene run --path SCENE [--timeout DURATION]",
			desc: "run a scene with headless Godot",
			flags: []helpFlag{
				{name: "path", meta: "SCENE", usage: "scene path (res://main.tscn)"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for Godot to exit (default 30s)"},
				{name: "godot", meta: "PATH", usage: "headless Godot binary path (or set GDCTL_GODOT_PATH)"},
				{name: "project", meta: "PATH", usage: "Godot project path (or set GDCTL_PROJECT)"},
			},
		},
	}},
	{name: "node", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] node add --parent PATH --type TYPE --name NAME [--prop NAME=TYPED_JSON] [--dry-run] [--scene SCENE] [--timeout DURATION]",
			desc: "add a node to the scene",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "type", meta: "TYPE", usage: "Godot node type (e.g. Node2D, CharacterBody3D)"},
				{name: "name", meta: "NAME", usage: "new node name"},
				{name: "prop", meta: "NAME=TYPED_JSON", usage: "initial property value (repeatable)"},
				{name: "dry-run", usage: "validate without mutating"},
				{name: "scene", meta: "SCENE", usage: "open this scene before mutating and save it after"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene open/save jobs (default 5s)"},
			},
		},
		{
			sub:  "remove",
			line: "  gdctl [--host host] [--port port] [--token token] node remove --path PATH [--dry-run] [--scene SCENE] [--timeout DURATION]",
			desc: "remove a node from the scene",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "dry-run", usage: "validate without mutating"},
				{name: "scene", meta: "SCENE", usage: "open this scene before mutating and save it after"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene open/save jobs (default 5s)"},
			},
		},
		{
			sub:  "rename",
			line: "  gdctl [--host host] [--port port] [--token token] node rename --path PATH --name NAME [--dry-run] [--scene SCENE] [--timeout DURATION]",
			desc: "rename a node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "name", meta: "NAME", usage: "new node name"},
				{name: "dry-run", usage: "validate without mutating"},
				{name: "scene", meta: "SCENE", usage: "open this scene before mutating and save it after"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene open/save jobs (default 5s)"},
			},
		},
		{
			sub:  "move",
			line: "  gdctl [--host host] [--port port] [--token token] node move --path PATH --parent PARENT [--index N] [--dry-run] [--scene SCENE] [--timeout DURATION]",
			desc: "move a node to a new parent",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "parent", meta: "PATH", usage: "new parent node path"},
				{name: "index", meta: "N", usage: "child index under new parent (-1 = append)"},
				{name: "dry-run", usage: "validate without mutating"},
				{name: "scene", meta: "SCENE", usage: "open this scene before mutating and save it after"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene open/save jobs (default 5s)"},
			},
		},
		{
			sub:  "get",
			line: "  gdctl [--host host] [--port port] [--token token] node get --path PATH --property PROPERTY",
			desc: "get a node property value",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "property", meta: "NAME", usage: "property name"},
			},
		},
		{
			sub:  "set",
			line: "  gdctl [--host host] [--port port] [--token token] node set --path PATH (--property PROPERTY VALUE | --position X,Y,Z | --rotation-degrees X,Y,Z | --scale X,Y,Z) [--scene SCENE] [--timeout DURATION]",
			desc: "set a node property value",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "property", meta: "NAME", usage: "property name"},
				{name: "value", meta: "TYPED_JSON", usage: `typed JSON value (e.g. {"kind":"Vector2","value":[200,400]})`},
				{name: "string", meta: "S", usage: "string shorthand"},
				{name: "int", meta: "N", usage: "integer shorthand"},
				{name: "float", meta: "N", usage: "float shorthand"},
				{name: "bool", meta: "BOOL", usage: "boolean shorthand"},
				{name: "vector2", meta: "X,Y", usage: "Vector2 shorthand"},
				{name: "vector3", meta: "X,Y,Z", usage: "Vector3 shorthand"},
				{name: "color", meta: "R,G,B[,A]", usage: "Color shorthand"},
				{name: "resource", meta: "PATH", usage: "Resource shorthand (res:// path)"},
				{name: "array-vector2", meta: "X,Y;X,Y", usage: "Array[Vector2] shorthand"},
				{name: "array-vector3", meta: "X,Y,Z;X,Y,Z", usage: "Array[Vector3] shorthand"},
				{name: "array-string", meta: "A;B", usage: "Array[String] shorthand"},
				{name: "array-int", meta: "N;N", usage: "Array[int] shorthand"},
				{name: "array-float", meta: "N;N", usage: "Array[float] shorthand"},
				{name: "array-bool", meta: "BOOL;BOOL", usage: "Array[bool] shorthand"},
				{name: "position", meta: "X,Y,Z", usage: "transform shorthand: sets position property as Vector3"},
				{name: "rotation-degrees", meta: "X,Y,Z", usage: "transform shorthand: sets rotation_degrees property as Vector3"},
				{name: "scale", meta: "X,Y,Z", usage: "transform shorthand: sets scale property as Vector3"},
				{name: "scene", meta: "SCENE", usage: "open this scene before mutating and save it after"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene open/save jobs (default 5s)"},
			},
			notes: []string{
				"Transform shorthands (--position, --rotation-degrees, --scale) cannot be combined with --property.",
			},
		},
		{
			sub:  "set-resource",
			line: "  gdctl [--host host] [--port port] [--token token] node set-resource --path PATH --property PROPERTY --resource RESOURCE",
			desc: "assign a resource to a node property",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "property", meta: "NAME", usage: "property name"},
				{name: "resource", meta: "PATH", usage: "resource path (res://)"},
			},
		},
		{
			sub:  "attach-script",
			line: "  gdctl [--host host] [--port port] [--token token] node attach-script --path PATH --script SCRIPT [--scene SCENE] [--timeout DURATION]",
			desc: "attach a script to a node after syntax-checking it",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "script", meta: "PATH", usage: "script resource path (res://)"},
				{name: "scene", meta: "SCENE", usage: "open this scene before attaching and save it after attaching"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene open/save jobs (default 5s)"},
			},
			notes: []string{
				"Without --scene, the command mutates the currently open editor scene.",
				"With --scene, the command opens that scene, attaches the script, and saves it.",
				"Invalid GDScript reports Godot's diagnostic, line number, and nearby source context when available.",
			},
		},
		{
			sub:  "group add",
			line: "  gdctl [--host host] [--port port] [--token token] node group add --path PATH --group GROUP",
			desc: "add a node to a group",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "group", meta: "NAME", usage: "group name"},
			},
		},
		{
			sub:  "group remove",
			line: "  gdctl [--host host] [--port port] [--token token] node group remove --path PATH --group GROUP",
			desc: "remove a node from a group",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "group", meta: "NAME", usage: "group name"},
			},
		},
		{
			sub:  "group list",
			line: "  gdctl [--host host] [--port port] [--token token] node group list --path PATH",
			desc: "list groups on a node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
			},
		},
		{
			sub:  "duplicate",
			line: "  gdctl [--host host] [--port port] [--token token] node duplicate --path PATH --name NAME [--parent PARENT] [--dry-run]",
			desc: "duplicate a node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "source node path"},
				{name: "name", meta: "NAME", usage: "name for the duplicate"},
				{name: "parent", meta: "PATH", usage: "parent node path (defaults to source's parent)"},
				{name: "dry-run", usage: "preview without modifying scene"},
			},
		},
		{
			sub:  "list-properties",
			line: "  gdctl [--host host] [--port port] [--token token] node list-properties --path PATH",
			desc: "list all exported properties on a node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
			},
		},
	}},
	{name: "script", cmds: []helpCmd{
		{
			sub:  "create",
			line: "  gdctl [--host host] [--port port] [--token token] script create --path PATH --extends CLASS [--force]",
			desc: "create a new GDScript file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "script path (res://)"},
				{name: "extends", meta: "CLASS", usage: "base Godot class name (e.g. CharacterBody2D)"},
				{name: "force", usage: "overwrite an existing script"},
			},
		},
		{
			sub:  "write",
			line: "  gdctl [--host host] [--port port] [--token token] script write --path PATH (--body TEXT | --body-file FILE) [--allow-missing-preloads]",
			desc: "syntax-check and write a GDScript file body",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "script path (res://)"},
				{name: "body", meta: "TEXT", usage: "script body as a string"},
				{name: "body-file", meta: "FILE", usage: "local file containing the script body"},
				{name: "allow-missing-preloads", usage: "suppress preload/resource-not-found errors and write anyway"},
			},
			notes: []string{
				"Invalid GDScript is not written and reports Godot's diagnostic, line number, and nearby source context when available.",
				"--allow-missing-preloads is useful during iterative authoring when preloaded scenes will be created shortly after.",
			},
		},
		{
			sub:  "check",
			line: "  gdctl [--host host] [--port port] [--token token] script check --path PATH",
			desc: "syntax-check a GDScript file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "script path (res://)"},
			},
			notes: []string{
				"Invalid GDScript reports Godot's diagnostic, line number, and nearby source context when available.",
			},
		},
	}},
	{name: "shader", cmds: []helpCmd{
		{
			sub:  "write",
			line: "  gdctl [--host host] [--port port] [--token token] shader write --path PATH (--body TEXT | --body-file FILE)",
			desc: "write a shader file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "shader path (res://)"},
				{name: "body", meta: "TEXT", usage: "shader body as a string"},
				{name: "body-file", meta: "FILE", usage: "local file containing the shader body"},
			},
		},
		{
			sub:  "check",
			line: "  gdctl [--host host] [--port port] [--token token] shader check --path PATH",
			desc: "syntax-check a shader file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "shader path (res://)"},
			},
		},
	}},
	{name: "resource", cmds: []helpCmd{
		{
			sub:  "create",
			line: "  gdctl [--host host] [--port port] [--token token] resource create --path PATH --type TYPE [--prop NAME=TYPED_JSON] [--shader-param NAME=RESOURCE]",
			desc: "create a Godot resource file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "resource file path (res://, .tres)"},
				{name: "type", meta: "TYPE", usage: "Godot resource class name (e.g. StandardMaterial3D)"},
				{name: "prop", meta: "NAME=TYPED_JSON", usage: "property value in name=TYPED_JSON form (repeatable)"},
				{name: "shader-param", meta: "NAME=PATH", usage: "ShaderMaterial param in name=res://path form (repeatable)"},
			},
		},
		{
			sub:  "list",
			line: "  gdctl [--host host] [--port port] [--token token] resource list [--dir res://] [--recursive] [--ext EXT]",
			desc: "list resource files in the project",
			flags: []helpFlag{
				{name: "dir", meta: "PATH", usage: "res:// directory to search (default res://)"},
				{name: "recursive", usage: "search recursively (default true)"},
				{name: "ext", meta: "EXT", usage: "file extension filter (e.g. .tres)"},
			},
		},
	}},
	{name: "import", cmds: []helpCmd{
		{
			sub:  "set",
			line: "  gdctl [--host host] [--port port] [--token token] import set --path PATH [--param NAME=VALUE]",
			desc: "set import parameters for an asset",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "asset path (e.g. res://textures/player.png)"},
				{name: "param", meta: "NAME=VALUE", usage: "import param in name=VALUE form where VALUE is raw JSON (repeatable)"},
			},
		},
	}},
	{name: "file", cmds: []helpCmd{
		{
			sub:  "write-bytes",
			line: "  gdctl [--host host] [--port port] [--token token] file write-bytes --path PATH --in FILE",
			desc: "upload binary data to a resource path",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "resource file path (res://)"},
				{name: "in", meta: "FILE", usage: "local input file path"},
			},
		},
		{
			sub:  "lut-write",
			line: "  gdctl [--host host] [--port port] [--token token] file lut-write --path PATH --profiles FILE",
			desc: "generate and upload a 256x1 edge LUT PNG",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "resource PNG path (res://)"},
				{name: "profiles", meta: "FILE", usage: "local edge profile JSON path"},
			},
		},
		{
			sub:  "list",
			line: "  gdctl [--host host] [--port port] [--token token] file list --path PATH [--recursive]",
			desc: "list files in a res:// directory",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "res:// directory path"},
				{name: "recursive", usage: "list recursively"},
			},
		},
		{
			sub:  "mkdir",
			line: "  gdctl [--host host] [--port port] [--token token] file mkdir --path PATH",
			desc: "create a res:// directory",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "res:// directory path to create"},
			},
		},
		{
			sub:  "delete",
			line: "  gdctl [--host host] [--port port] [--token token] file delete --path PATH",
			desc: "delete a res:// file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "res:// path to delete"},
			},
		},
		{
			sub:  "exists",
			line: "  gdctl [--host host] [--port port] [--token token] file exists --path PATH",
			desc: "check whether a res:// path exists",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "res:// path to check"},
			},
		},
	}},
	{name: "navigation", cmds: []helpCmd{
		{
			sub:  "bake",
			line: "  gdctl [--host host] [--port port] [--token token] navigation bake --path PATH",
			desc: "bake a navigation mesh",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "NavigationRegion node path"},
			},
		},
	}},
	{name: "signal", cmds: []helpCmd{
		{
			sub:  "connect",
			line: "  gdctl [--host host] [--port port] [--token token] signal connect --from PATH --signal NAME --to PATH --method METHOD",
			desc: "connect a signal between two nodes",
			flags: []helpFlag{
				{name: "from", meta: "PATH", usage: "source node path"},
				{name: "signal", meta: "NAME", usage: "signal name"},
				{name: "to", meta: "PATH", usage: "target node path"},
				{name: "method", meta: "NAME", usage: "method name on target node"},
			},
		},
		{
			sub:  "disconnect",
			line: "  gdctl [--host host] [--port port] [--token token] signal disconnect --from PATH --signal NAME --to PATH --method METHOD",
			desc: "disconnect a signal between two nodes",
			flags: []helpFlag{
				{name: "from", meta: "PATH", usage: "source node path"},
				{name: "signal", meta: "NAME", usage: "signal name"},
				{name: "to", meta: "PATH", usage: "target node path"},
				{name: "method", meta: "NAME", usage: "method name on target node"},
			},
		},
	}},
	{name: "project", cmds: []helpCmd{
		{
			sub:  "setting get",
			line: "  gdctl [--host host] [--port port] [--token token] project setting get --key KEY",
			desc: "get a project setting value",
			flags: []helpFlag{
				{name: "key", meta: "KEY", usage: "project setting key"},
			},
		},
		{
			sub:  "setting set",
			line: "  gdctl [--host host] [--port port] [--token token] project setting set --key KEY (--value TYPED_JSON | --string S | --int N | --float N | --bool BOOL | --vector2 X,Y | --vector3 X,Y,Z | --color R,G,B[,A] | --resource PATH | --array-vector3 A;B)",
			desc: "set a project setting value",
			flags: []helpFlag{
				{name: "key", meta: "KEY", usage: "project setting key"},
				{name: "value", meta: "TYPED_JSON", usage: `typed JSON value (e.g. {"kind":"int","value":1920})`},
				{name: "string", meta: "S", usage: "string shorthand"},
				{name: "int", meta: "N", usage: "integer shorthand"},
				{name: "float", meta: "N", usage: "float shorthand"},
				{name: "bool", meta: "BOOL", usage: "boolean shorthand"},
				{name: "vector2", meta: "X,Y", usage: "Vector2 shorthand"},
				{name: "vector3", meta: "X,Y,Z", usage: "Vector3 shorthand"},
				{name: "color", meta: "R,G,B[,A]", usage: "Color shorthand"},
				{name: "resource", meta: "PATH", usage: "Resource shorthand (res:// path)"},
				{name: "array-vector2", meta: "X,Y;X,Y", usage: "Array[Vector2] shorthand"},
				{name: "array-vector3", meta: "X,Y,Z;X,Y,Z", usage: "Array[Vector3] shorthand"},
				{name: "array-string", meta: "A;B", usage: "Array[String] shorthand"},
				{name: "array-int", meta: "N;N", usage: "Array[int] shorthand"},
				{name: "array-float", meta: "N;N", usage: "Array[float] shorthand"},
				{name: "array-bool", meta: "BOOL;BOOL", usage: "Array[bool] shorthand"},
			},
		},
		{
			sub:  "run",
			line: "  gdctl [--project PATH] [--godot PATH] project run [--scene SCENE] [--timeout DURATION]",
			desc: "run the project with headless Godot",
			flags: []helpFlag{
				{name: "scene", meta: "SCENE", usage: "scene to run (res://main.tscn); omit to use project main scene"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for Godot to exit (default 30s)"},
				{name: "godot", meta: "PATH", usage: "headless Godot binary path (or set GDCTL_GODOT_PATH)"},
				{name: "project", meta: "PATH", usage: "Godot project path (or set GDCTL_PROJECT)"},
			},
		},
	}},
	{name: "viewport", cmds: []helpCmd{
		{
			sub:  "screenshot",
			line: "  gdctl [--host host] [--port port] [--token token] viewport screenshot --out FILE [--kind 2d|3d] [--index N]",
			desc: "capture the editor viewport as a PNG",
			flags: []helpFlag{
				{name: "out", meta: "FILE", usage: "local PNG output path"},
				{name: "kind", meta: "KIND", usage: "editor viewport kind: 2d or 3d (default 2d)"},
				{name: "index", meta: "N", usage: "3D viewport index (default 0)"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for screenshot job (default 5s)"},
			},
		},
		{
			sub:  "set-size",
			line: "  gdctl [--host host] [--port port] [--token token] viewport set-size --width W --height H [--path NODE_PATH]",
			desc: "resize the main window or a SubViewport node",
			flags: []helpFlag{
				{name: "width", meta: "W", usage: "viewport width in pixels"},
				{name: "height", meta: "H", usage: "viewport height in pixels"},
				{name: "path", meta: "NODE_PATH", usage: "SubViewport node path; omit to resize the main window"},
			},
		},
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] viewport add --width W --height H [--parent PATH] [--add-camera]",
			desc: "add a SubViewport node to the current scene",
			flags: []helpFlag{
				{name: "width", meta: "W", usage: "SubViewport width in pixels (default 320)"},
				{name: "height", meta: "H", usage: "SubViewport height in pixels (default 240)"},
				{name: "parent", meta: "PATH", usage: "parent node path (defaults to scene root)"},
				{name: "add-camera", usage: "add a Camera3D child inside the SubViewport"},
			},
		},
	}},
	{name: "theme", cmds: []helpCmd{
		{
			sub:  "create",
			line: "  gdctl [--host host] [--port port] [--token token] theme create --path PATH [--force]",
			desc: "create a Theme resource file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "theme path (res://*.tres)"},
				{name: "force", usage: "overwrite an existing theme file"},
			},
		},
		{
			sub:  "set-color",
			line: "  gdctl [--host host] [--port port] [--token token] theme set-color --path PATH --node-type TYPE --name NAME --value COLOR",
			desc: "set a named color override on a theme",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "theme file path (res://*.tres)"},
				{name: "node-type", meta: "TYPE", usage: "Godot control type (e.g. Label, Button)"},
				{name: "name", meta: "NAME", usage: "color override name (e.g. font_color)"},
				{name: "value", meta: "COLOR", usage: "color as R,G,B,A floats or HTML hex (e.g. ff8800ff)"},
			},
		},
		{
			sub:  "set-font-size",
			line: "  gdctl [--host host] [--port port] [--token token] theme set-font-size --path PATH --node-type TYPE --name NAME --value N",
			desc: "set a named font size override on a theme",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "theme file path"},
				{name: "node-type", meta: "TYPE", usage: "Godot control type"},
				{name: "name", meta: "NAME", usage: "font size override name (e.g. font_size)"},
				{name: "value", meta: "N", usage: "font size in pixels"},
			},
		},
		{
			sub:  "set-constant",
			line: "  gdctl [--host host] [--port port] [--token token] theme set-constant --path PATH --node-type TYPE --name NAME --value N",
			desc: "set a named integer constant override on a theme",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "theme file path"},
				{name: "node-type", meta: "TYPE", usage: "Godot control type"},
				{name: "name", meta: "NAME", usage: "constant name (e.g. margin_top)"},
				{name: "value", meta: "N", usage: "integer constant value"},
			},
		},
	}},
	{name: "animation", cmds: []helpCmd{
		{
			sub:  "create",
			line: "  gdctl [--host host] [--port port] [--token token] animation create --path LIBRARY --name NAME [--length N] [--loop]",
			desc: "create an animation in an AnimationLibrary resource",
			flags: []helpFlag{
				{name: "path", meta: "LIBRARY", usage: "AnimationLibrary .tres path (created if absent)"},
				{name: "name", meta: "NAME", usage: "animation name (valid GDScript identifier)"},
				{name: "length", meta: "N", usage: "animation length in seconds (default 1.0)"},
				{name: "loop", usage: "enable linear loop mode"},
			},
		},
		{
			sub:  "track-add",
			line: "  gdctl [--host host] [--port port] [--token token] animation track-add --path LIBRARY --animation NAME --node-path NODE --property PROP",
			desc: "add a value track to an animation",
			flags: []helpFlag{
				{name: "path", meta: "LIBRARY", usage: "AnimationLibrary .tres path"},
				{name: "animation", meta: "NAME", usage: "animation name"},
				{name: "node-path", meta: "NODE", usage: "node path relative to AnimationPlayer root (e.g. Player)"},
				{name: "property", meta: "PROP", usage: "property name on that node (e.g. position)"},
			},
		},
		{
			sub:  "keyframe-add",
			line: "  gdctl [--host host] [--port port] [--token token] animation keyframe-add --path LIBRARY --animation NAME --track-idx N --time T --value TYPED_JSON",
			desc: "insert a keyframe on a track at a given time",
			flags: []helpFlag{
				{name: "path", meta: "LIBRARY", usage: "AnimationLibrary .tres path"},
				{name: "animation", meta: "NAME", usage: "animation name"},
				{name: "track-idx", meta: "N", usage: "track index (from track-add output)"},
				{name: "time", meta: "T", usage: "time position in seconds"},
				{name: "value", meta: "TYPED_JSON", usage: "keyframe value as typed JSON"},
			},
		},
		{
			sub:  "length-set",
			line: "  gdctl [--host host] [--port port] [--token token] animation length-set --path LIBRARY --animation NAME --length N",
			desc: "set the duration of an animation",
			flags: []helpFlag{
				{name: "path", meta: "LIBRARY", usage: "AnimationLibrary .tres path"},
				{name: "animation", meta: "NAME", usage: "animation name"},
				{name: "length", meta: "N", usage: "new duration in seconds"},
			},
		},
		{
			sub:  "player-play",
			line: "  gdctl [--host host] [--port port] [--token token] animation player-play --node-path PATH [--animation NAME]",
			desc: "trigger playback on an AnimationPlayer node in the open scene",
			flags: []helpFlag{
				{name: "node-path", meta: "PATH", usage: "path to the AnimationPlayer node"},
				{name: "animation", meta: "NAME", usage: "animation name to play (defaults to current)"},
			},
		},
	}},
	{name: "tilemap", cmds: []helpCmd{
		{
			sub:  "tileset-create",
			line: "  gdctl [--host host] [--port port] [--token token] tilemap tileset-create --path PATH [--tile-width W] [--tile-height H] [--force]",
			desc: "create a TileSet resource file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "TileSet resource path (res://)"},
				{name: "tile-width", meta: "W", usage: "tile width in pixels (default 16)"},
				{name: "tile-height", meta: "H", usage: "tile height in pixels (default 16)"},
				{name: "force", usage: "overwrite an existing TileSet"},
			},
		},
		{
			sub:  "source-add",
			line: "  gdctl [--host host] [--port port] [--token token] tilemap source-add --path TILESET --texture TEX [--tile-width W] [--tile-height H]",
			desc: "add an atlas texture source to a TileSet",
			flags: []helpFlag{
				{name: "path", meta: "TILESET", usage: "TileSet resource path"},
				{name: "texture", meta: "TEX", usage: "atlas texture resource path (res://)"},
				{name: "tile-width", meta: "W", usage: "tile width in pixels (default 16)"},
				{name: "tile-height", meta: "H", usage: "tile height in pixels (default 16)"},
			},
		},
		{
			sub:  "cell-set",
			line: "  gdctl [--host host] [--port port] [--token token] tilemap cell-set --node PATH --layer N --x X --y Y --source-id ID [--atlas-x AX] [--atlas-y AY]",
			desc: "paint a cell on a TileMap node",
			flags: []helpFlag{
				{name: "node", meta: "PATH", usage: "TileMap node path in the open scene"},
				{name: "layer", meta: "N", usage: "layer index (default 0)"},
				{name: "x", meta: "X", usage: "cell column"},
				{name: "y", meta: "Y", usage: "cell row"},
				{name: "source-id", meta: "ID", usage: "TileSet source ID"},
				{name: "atlas-x", meta: "AX", usage: "atlas tile column (default 0)"},
				{name: "atlas-y", meta: "AY", usage: "atlas tile row (default 0)"},
			},
		},
		{
			sub:  "cell-clear",
			line: "  gdctl [--host host] [--port port] [--token token] tilemap cell-clear --node PATH --layer N --x X --y Y",
			desc: "erase a cell on a TileMap node",
			flags: []helpFlag{
				{name: "node", meta: "PATH", usage: "TileMap node path in the open scene"},
				{name: "layer", meta: "N", usage: "layer index (default 0)"},
				{name: "x", meta: "X", usage: "cell column"},
				{name: "y", meta: "Y", usage: "cell row"},
			},
		},
	}},
	{name: "audio", cmds: []helpCmd{
		{
			sub:  "bus-add",
			line: "  gdctl [--host host] [--port port] [--token token] audio bus-add --name NAME",
			desc: "add a named audio bus",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "bus name"},
			},
		},
		{
			sub:  "bus-volume-set",
			line: "  gdctl [--host host] [--port port] [--token token] audio bus-volume-set --name NAME --volume-db DB",
			desc: "set the volume of an audio bus in dB",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "bus name"},
				{name: "volume-db", meta: "DB", usage: "volume in decibels (e.g. 0.0 for unity, -6.0 for half)"},
			},
		},
		{
			sub:  "bus-effect-add",
			line: "  gdctl [--host host] [--port port] [--token token] audio bus-effect-add --name NAME --effect-type TYPE",
			desc: "add an AudioEffect to a bus",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "bus name"},
				{name: "effect-type", meta: "TYPE", usage: "AudioEffect subclass name (e.g. AudioEffectReverb, AudioEffectCompressor)"},
			},
		},
	}},
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	for _, g := range helpGroups {
		for _, cmd := range g.cmds {
			fmt.Fprintln(w, cmd.line)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Dotted aliases are supported for multi-word commands, e.g. gdctl file.mkdir and gdctl project.setting.get.")
}

func runHelp(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	for _, g := range helpGroups {
		if g.name == args[0] {
			if len(args) == 1 {
				fmt.Fprintln(stdout, "Usage:")
				for _, cmd := range g.cmds {
					fmt.Fprintln(stdout, cmd.line)
				}
				return nil
			}
			sub := strings.Join(args[1:], " ")
			for _, cmd := range g.cmds {
				if cmd.sub == sub {
					return printCommandHelp(stdout, cmd)
				}
			}
			return fmt.Errorf("unknown subcommand %q under %q", sub, g.name)
		}
	}
	if len(args) == 1 {
		for _, g := range helpGroups {
			for _, cmd := range g.cmds {
				if cmd.sub == args[0] {
					return printCommandHelp(stdout, cmd)
				}
			}
		}
	}
	fmt.Fprintf(stdout, "Unknown help topic %q. Available topics:\n", args[0])
	for _, g := range helpGroups {
		fmt.Fprintf(stdout, "  %s\n", g.name)
	}
	return fmt.Errorf("unknown help topic: %s", args[0])
}

func printCommandHelp(stdout io.Writer, cmd helpCmd) error {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, cmd.line)
	if cmd.desc != "" {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, cmd.desc)
	}
	if len(cmd.flags) > 0 {
		fmt.Fprintln(stdout)
		maxWidth := 0
		for _, f := range cmd.flags {
			w := 2 + len(f.name)
			if f.meta != "" {
				w += 1 + len(f.meta)
			}
			if w > maxWidth {
				maxWidth = w
			}
		}
		for _, f := range cmd.flags {
			var flagStr string
			if f.meta != "" {
				flagStr = fmt.Sprintf("--%s %s", f.name, f.meta)
			} else {
				flagStr = fmt.Sprintf("--%s", f.name)
			}
			fmt.Fprintf(stdout, "  %-*s  %s\n", maxWidth, flagStr, f.usage)
		}
	}
	if len(cmd.notes) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Notes:")
		for _, note := range cmd.notes {
			fmt.Fprintf(stdout, "  %s\n", note)
		}
	}
	return nil
}
