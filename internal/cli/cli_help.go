package cli

import (
	"fmt"
	"io"
	"strings"
)

type helpFlag struct {
	name  string
	meta  string
	usage string
}

type helpCmd struct {
	sub     string
	line    string
	desc    string
	flags   []helpFlag
	notes   []string
	usecase []string
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
			usecase: []string{
				"Verify the bridge is alive before running any automation script.",
				"Confirm the correct Godot project is connected when switching projects.",
				"Quick sanity check after restarting Godot or re-enabling the plugin.",
			},
		},
		{
			sub:  "doctor",
			line: "  gdctl [--host host] [--port port] [--token token] [--project path] doctor [--project PATH] [--fix]",
			desc: "diagnose addon and bridge setup",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
				{name: "fix", usage: "install and enable the addon when needed"},
			},
			usecase: []string{
				"Diagnose why gdctl commands fail (missing addon, wrong host, auth issues).",
				"Run in CI to assert the environment is ready before executing scene mutations.",
				"Use --fix to automatically install and enable the addon in one step.",
			},
		},
		{
			sub:  "help",
			line: "  gdctl help [--usecase] [topic]",
			desc: "show usage information",
			flags: []helpFlag{
				{name: "usecase", usage: "show scenario-based descriptions for each command"},
				{name: "topic", meta: "TOPIC", usage: "command group or specific command (e.g. scene, scene create, scene.create)"},
			},
			usecase: []string{
				"Explore available commands when you are new to gdctl.",
				"Look up exact flags for a specific command without leaving the terminal.",
				"Use --usecase to find the right command for a given workflow scenario.",
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
			usecase: []string{
				"Set up the bridge addon in a freshly cloned or newly created Godot project.",
				"Re-install after accidentally deleting the addon directory.",
				"Use --force when files already exist and you want a clean overwrite.",
			},
		},
		{
			sub:  "enable",
			line: "  gdctl addon enable --project PATH",
			desc: "enable the addon in project.godot",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
			},
			usecase: []string{
				"Activate the addon in project.godot after a manual file copy or install.",
				"Re-enable after Godot disabled the plugin due to a crash or error.",
			},
		},
		{
			sub:  "disable",
			line: "  gdctl addon disable --project PATH",
			desc: "disable the addon in project.godot",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
			},
			usecase: []string{
				"Temporarily deactivate the bridge without removing its files.",
				"Prevent the bridge from starting in a production export build.",
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
			usecase: []string{
				"Check whether the addon is installed, enabled, and reachable at a glance.",
				"Use --json to feed addon health into a CI status dashboard.",
				"Audit the protocol version before running commands that require a specific capability.",
			},
		},
		{
			sub:  "update",
			line: "  gdctl addon update [--project PATH]",
			desc: "update the addon files",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path (omit to update over the bridge)"},
			},
			usecase: []string{
				"Deploy a new CLI version's bundled addon files to the running Godot project.",
				"Keep the addon in sync with the CLI without manually copying files.",
				"Run after upgrading gdctl to pick up new command handlers.",
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
			usecase: []string{
				"Recover after a bad addon update breaks the bridge.",
				"Restore a known-good addon version when the bridge fails to start.",
				"Use --backup to pick a specific backup rather than the latest.",
			},
		},
		{
			sub:  "remove",
			line: "  gdctl addon remove --project PATH",
			desc: "remove the addon from a Godot project",
			flags: []helpFlag{
				{name: "project", meta: "PATH", usage: "Godot project path"},
			},
			usecase: []string{
				"Clean up the addon files from a project that no longer needs gdctl.",
				"Prepare for a manual reinstall of a specific addon version.",
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
			usecase: []string{
				"Identify whether the addon is installed, enabled, and the correct version.",
				"Use --fix to automatically install and enable when issues are detected.",
				"Run in CI before any bridge-dependent step to catch setup drift.",
			},
		},
	}},
	{name: "bridge", cmds: []helpCmd{
		{
			sub:  "info",
			line: "  gdctl [--host host] [--port port] bridge info",
			desc: "show bridge connection details",
			usecase: []string{
				"Inspect all bridge connection details in one command (host, port, version, capabilities).",
				"Verify which capabilities the running addon exposes before using an advanced command.",
				"Debug connection problems by seeing the exact host and port the bridge is listening on.",
			},
		},
		{
			sub:  "logs",
			line: "  gdctl [--host host] [--port port] [--token token] bridge logs [--json] [--clear]",
			desc: "read bridge log entries",
			flags: []helpFlag{
				{name: "json", usage: "write logs as JSON"},
				{name: "clear", usage: "clear logs after reading"},
			},
			usecase: []string{
				"Investigate why a command returned an unexpected error.",
				"Use after a GDScript runtime fault to see which endpoint was last active.",
				"Use --clear to flush old log entries before a focused test run.",
			},
		},
		{
			sub:  "addon-update",
			line: "  gdctl [--host host] [--port port] [--token token] bridge addon-update",
			desc: "update the addon over the bridge",
			usecase: []string{
				"Alternative to 'addon update' when targeting the bridge endpoint directly.",
				"Use in scripts that already hold a bridge client handle.",
			},
		},
	}},
	{name: "autoload", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] autoload add --name NAME --path PATH",
			desc: "add an autoload singleton",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "autoload singleton name"},
				{name: "path", meta: "PATH", usage: "script or scene path (res://)"},
			},
			usecase: []string{
				"Register a persistent singleton (e.g., GameManager, AudioController) at project startup.",
				"Wire up the gdctl runtime helper autoload for run commands.",
			},
		},
		{
			sub:  "remove",
			line: "  gdctl [--host host] [--port port] [--token token] autoload remove --name NAME",
			desc: "remove an autoload singleton",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "autoload singleton name"},
			},
			usecase: []string{
				"Unregister a singleton that is no longer needed or was added by mistake.",
				"Clean up gdctl runtime helper autoloads after finishing a test run.",
			},
		},
		{
			sub:  "list",
			line: "  gdctl [--host host] [--port port] [--token token] autoload list [--json]",
			desc: "list autoload singletons",
			flags: []helpFlag{
				{name: "json", usage: "print JSON"},
			},
			usecase: []string{
				"Audit which singletons are currently registered in the project.",
				"Use --json to diff autoload state between two project configurations.",
			},
		},
	}},
	{name: "input", cmds: []helpCmd{
		{
			sub:  "action add",
			line: "  gdctl [--host host] [--port port] [--token token] input action add --name NAME [--deadzone N]",
			desc: "add or update an input action",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "input action name"},
				{name: "deadzone", meta: "N", usage: "input action deadzone (default 0.5)"},
			},
			usecase: []string{
				"Create a new input action (e.g., 'jump', 'fire') before binding keys to it.",
				"Adjust the deadzone of an existing action for fine-tuned analog input.",
			},
		},
		{
			sub:  "action remove",
			line: "  gdctl [--host host] [--port port] [--token token] input action remove --name NAME",
			desc: "remove an input action",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "input action name"},
			},
			usecase: []string{
				"Delete an input action that is no longer used.",
				"Clean up placeholder actions created during prototyping.",
			},
		},
		{
			sub:  "action list",
			line: "  gdctl [--host host] [--port port] [--token token] input action list [--json] [--all]",
			desc: "list project input actions",
			flags: []helpFlag{
				{name: "json", usage: "print JSON"},
				{name: "all", usage: "include built-in engine actions"},
			},
			usecase: []string{
				"Audit the full set of project input actions.",
				"Use --all to include built-in Godot actions for reference.",
				"Use --json to compare action definitions between project versions.",
			},
		},
		{
			sub:  "event add-key",
			line: "  gdctl [--host host] [--port port] [--token token] input event add-key --action ACTION --key KEY [--physical=false]",
			desc: "add a keyboard event to an input action",
			flags: []helpFlag{
				{name: "action", meta: "ACTION", usage: "input action name"},
				{name: "key", meta: "KEY", usage: "key name, e.g. W, Space, Up"},
				{name: "physical", meta: "BOOL", usage: "use physical keycode instead of layout keycode (default true)"},
			},
			usecase: []string{
				"Bind a keyboard key to an existing input action.",
				"Switch between physical and layout keycodes for locale-agnostic controls.",
			},
		},
		{
			sub:  "event add-joypad",
			line: "  gdctl [--host host] [--port port] [--token token] input event add-joypad --action ACTION (--button N | --axis N [--axis-value V]) [--device N]",
			desc: "add a joypad button or axis event to an input action",
			flags: []helpFlag{
				{name: "action", meta: "ACTION", usage: "input action name"},
				{name: "button", meta: "N", usage: "JoyButton index (e.g. 0=A/Cross, 1=B/Circle); mutually exclusive with --axis"},
				{name: "axis", meta: "N", usage: "JoyAxis index (e.g. 0=left_x, 1=left_y); mutually exclusive with --button"},
				{name: "axis-value", meta: "V", usage: "axis threshold value (default 1.0)"},
				{name: "device", meta: "N", usage: "joypad device index (-1 = any device, default -1)"},
			},
			usecase: []string{
				"Bind a joypad button or axis to an input action for gamepad support.",
				"Use --device -1 to match any connected controller.",
			},
		},
		{
			sub:  "event add-mouse-button",
			line: "  gdctl [--host host] [--port port] [--token token] input event add-mouse-button --action ACTION --button left|right|middle",
			desc: "add a mouse button event to an input action",
			flags: []helpFlag{
				{name: "action", meta: "ACTION", usage: "input action name"},
				{name: "button", meta: "BUTTON", usage: "mouse button: left, right, or middle"},
			},
			usecase: []string{
				"Bind left-click to a 'fire' action for a first-person shooter.",
				"Assign right-click to an 'aim' action without opening the Godot editor.",
			},
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
			usecase: []string{
				"Launch a specific scene in the editor for a rapid iteration loop.",
				"Trigger a playtest from a CI script without manually clicking Play in Godot.",
				"Use --main to run the project's configured main scene.",
			},
		},
		{
			sub:  "status",
			line: "  gdctl [--host host] [--port port] [--token token] run status [--json]",
			desc: "show whether the editor is running a scene and whether the runtime helper checked in",
			flags: []helpFlag{
				{name: "json", usage: "print status as JSON"},
			},
			notes: []string{
				"Plain output says running without runtime helper when a scene is active but GdctlRuntimeBridge has not checked in.",
			},
			usecase: []string{
				"Check whether the editor is currently running a scene.",
				"Poll in a script to wait until a scene finishes starting before probing.",
			},
		},
		{
			sub:  "helper-status",
			line: "  gdctl [--host host] [--port port] [--token token] run helper-status [--json]",
			desc: "show gdctl runtime helper/autoload health",
			flags: []helpFlag{
				{name: "json", usage: "print helper status as JSON"},
			},
			usecase: []string{
				"Confirm the GdctlRuntimeBridge autoload is active and checked in.",
				"Debug why run probe or run input commands are failing.",
			},
		},
		{
			sub:  "stop",
			line: "  gdctl [--host host] [--port port] [--token token] run stop",
			desc: "stop the scene currently running from the editor",
			usecase: []string{
				"Halt the running scene from a script after a test completes.",
				"Recover from a stuck game state without clicking Stop in the editor.",
			},
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
			usecase: []string{
				"Read GDScript print/push_error output captured during a test run.",
				"Use --source to filter logs from a specific system (e.g., runtime.game).",
				"Use --since-start to ignore stale logs from a previous run.",
			},
		},
		{
			sub:  "screenshot",
			line: "  gdctl [--host host] [--port port] [--token token] run screenshot [--out FILE] [--source game|screen] [--screen N] [--viewport PATH]",
			desc: "capture the running game viewport or host screen",
			flags: []helpFlag{
				{name: "out", meta: "FILE", usage: "local PNG output path (default screenshots/YYYYMMDD-HHMMSS.png)"},
				{name: "source", meta: "SOURCE", usage: "screenshot source: game or screen (default game)"},
				{name: "screen", meta: "N", usage: "host display screen index when --source screen is used (default 0)"},
				{name: "viewport", meta: "PATH", usage: "SubViewport node path in the running scene; captures that viewport instead of the root"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for screenshot job (default 5s)"},
			},
			notes: []string{
				"Game screenshots require the gdctl runtime helper autoload installed by run start.",
				"With --source game, gdctl checks helper health before queueing the screenshot job.",
				"Use --source screen for the legacy whole-host-screen capture; it does not require the helper.",
				"Use --viewport to capture a specific SubViewport (e.g. a split-screen panel).",
			},
			usecase: []string{
				"Capture the game viewport as a PNG for visual regression testing.",
				"Use --viewport to capture a specific SubViewport (e.g., a split-screen panel).",
				"Use --source screen for a full host-screen screenshot.",
			},
		},
		{
			sub:  "input",
			line: "  gdctl [--host host] [--port port] [--token token] run input --file input.json [--timeout DURATION] [--summary-probe SOURCE]",
			desc: "play a short input sequence into the running game",
			flags: []helpFlag{
				{name: "file", meta: "FILE", usage: "input JSON file containing validated steps"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for input job (default 5s)"},
				{name: "summary-probe", meta: "SOURCE", usage: "after input completes, print the latest log entry for this source"},
			},
			notes: []string{
				"Requires GdctlRuntimeBridge to be present in the running scene; gdctl checks this before queueing input.",
			},
			usecase: []string{
				"Replay a recorded input sequence to simulate player actions in a test.",
				"Drive a menu navigation or combat flow without human interaction.",
				"Use mouse_motion with relative: [x, y]; dx/dy is rejected.",
			},
		},
		{
			sub:  "wait-probe",
			line: "  gdctl [--host host] [--port port] [--token token] run wait-probe --source SOURCE (--assert KEY>=VALUE | --assert-key KEY --assert-op OP --assert-value VALUE) [--timeout DURATION] [--json]",
			desc: "poll run logs until a probe field satisfies a predicate or timeout fires",
			flags: []helpFlag{
				{name: "source", meta: "SOURCE", usage: "log source to watch (e.g. runtime.game)"},
				{name: "assert", meta: "EXPR", usage: "predicate expression, e.g. targets_disabled>=1 (ops: >= <= > < == !=)"},
				{name: "assert-key", meta: "KEY", usage: "predicate key for split assertion form"},
				{name: "assert-op", meta: "OP", usage: "predicate operator for split assertion form"},
				{name: "assert-value", meta: "VALUE", usage: "predicate value for split assertion form"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait (default 30s)"},
				{name: "json", usage: "print matching entry as JSON"},
			},
			notes: []string{
				"Requires GdctlRuntimeBridge to be present in the running scene; gdctl checks this before polling logs.",
			},
			usecase: []string{
				"Block until a game state condition is met (e.g., score >= 100) before asserting.",
				"Use in CI to wait for a loading screen to finish before screenshotting.",
				"Quote --assert expressions containing > or < in shells, or use the split assertion flags.",
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
			usecase: []string{
				"Verify the player camera can see an expected target in a 3D scene.",
				"Debug line-of-sight or occlusion issues from a script.",
			},
		},
		{
			sub:  "probe node",
			line: "  gdctl [--host host] [--port port] [--token token] run probe node --path PATH --property NAME [--property NAME] [--json] [--timeout DURATION]",
			desc: "read properties from a node in the running game tree",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "runtime node path, e.g. /root/Main/Player"},
				{name: "property", meta: "NAME", usage: "property to read; repeat for multiple properties"},
				{name: "json", usage: "print result as JSON"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for node probe result (default 5s)"},
			},
			notes: []string{
				"Requires GdctlRuntimeBridge autoload in the running scene.",
			},
			usecase: []string{
				"Read live node properties (position, health, state) from the running game.",
				"Assert that a node reached the expected state after an input sequence.",
			},
		},
		{
			sub:  "instantiate",
			line: "  gdctl [--host host] [--port port] [--token token] run instantiate --scene SCENE --parent PATH [--name NAME] [--timeout DURATION]",
			desc: "instantiate a packed scene at a parent node in the running game",
			flags: []helpFlag{
				{name: "scene", meta: "SCENE", usage: "packed scene path to instantiate (res://)"},
				{name: "parent", meta: "PATH", usage: "parent node path in the running scene"},
				{name: "name", meta: "NAME", usage: "name for the new node (optional, uses scene default)"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for instantiate job (default 5s)"},
			},
			notes: []string{
				"Requires GdctlRuntimeBridge autoload in the running scene.",
			},
			usecase: []string{
				"Spawn a packed scene at runtime under a specific parent node.",
				"Add an enemy, pickup, or UI overlay to the running game from a script.",
			},
		},
		{
			sub:  "scene-reload",
			line: "  gdctl [--host host] [--port port] [--token token] run scene-reload [--timeout DURATION]",
			desc: "reload the current scene in the running game",
			flags: []helpFlag{
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene reload (default 5s)"},
			},
			notes: []string{
				"Requires GdctlRuntimeBridge autoload in the running scene.",
				"Autoloads (including the runtime helper) persist across the reload.",
			},
			usecase: []string{
				"Restart the current scene without stopping and re-starting Godot.",
				"Reset game state between test iterations in a loop.",
			},
		},
		{
			sub:  "smoke",
			line: "  gdctl [--host host] [--port port] [--token token] run smoke [--scene SCENE | --main] [--input FILE] [--assert SOURCE:KEY>=VALUE | --assert-source SOURCE --assert-key KEY --assert-op OP --assert-value VALUE] [--screenshot OUT] [--timeout DURATION] [--keep-running]",
			desc: "one-shot automated test: start, optionally inject input, probe, screenshot, then stop",
			flags: []helpFlag{
				{name: "scene", meta: "SCENE", usage: "scene to run (res://)"},
				{name: "main", usage: "run the project main scene"},
				{name: "input", meta: "FILE", usage: "input JSON file to inject after start"},
				{name: "assert", meta: "SOURCE:KEY>=VALUE", usage: "wait for a probe predicate before proceeding"},
				{name: "assert-source", meta: "SOURCE", usage: "probe source for split assertion form"},
				{name: "assert-key", meta: "KEY", usage: "probe detail key for split assertion form"},
				{name: "assert-op", meta: "OP", usage: "predicate operator for split assertion form"},
				{name: "assert-value", meta: "VALUE", usage: "predicate value for split assertion form"},
				{name: "screenshot", meta: "FILE", usage: "capture game viewport to this PNG path"},
				{name: "screenshot-viewport", meta: "PATH", usage: "SubViewport node path to screenshot instead of main viewport"},
				{name: "timeout", meta: "DURATION", usage: "total time limit for the smoke run (default 30s)"},
				{name: "keep-running", usage: "do not stop the scene after the test"},
			},
			notes: []string{
				"Exits with code 0 on pass, 1 on failure.",
				"Quote --assert expressions containing > or < in shells, or use the split assertion flags.",
				"Prints 'Smoke: PASS' or 'Smoke: FAIL — <reason>'.",
			},
			usecase: []string{
				"One-shot automated test: start a scene, inject input, probe state, screenshot, stop.",
				"Integrate into CI to gate merges on a basic scene-runs-and-hits-a-state check.",
				"Use --assert to define the success condition as a probe predicate.",
			},
		},
		{
			sub:  "profile",
			line: "  gdctl [--host host] [--port port] [--token token] run profile --metric METRICS --duration DURATION",
			desc: "sample runtime performance metrics while the game is running",
			flags: []helpFlag{
				{name: "metric", meta: "METRICS", usage: "comma-separated metrics: fps,draw_calls,physics_time,memory_usage"},
				{name: "duration", meta: "DURATION", usage: "sampling duration, e.g. 10s (default 5s)"},
			},
			usecase: []string{
				"Measure FPS, draw calls, physics time, or memory while the game is running.",
				"Baseline performance before and after a graphics or physics change.",
			},
		},
	}},
	{name: "test", cmds: []helpCmd{
		{
			sub:  "gdscript",
			line: "  gdctl [--host host] [--port port] [--token token] test [gdscript] (--path PATH | --dir DIR) [--timeout DURATION] [--json]",
			desc: "run selected GDScript unit tests in the editor bridge",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "single GDScript test script path (res:// .gd)"},
				{name: "dir", meta: "DIR", usage: "directory to recursively discover test_*.gd files"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for the test job (default 30s)"},
				{name: "json", usage: "print the full structured result as JSON"},
			},
			notes: []string{
				"With --path or --dir and no subcommand, runs the selected GDScript tests.",
				"test gdscript is an explicit alias for the same GDScript test runner.",
				"Test scripts should extend res://addons/godot_tcp_bridge/testing/test_case.gd.",
				"Zero-argument methods named test_* are executed with optional before_all/after_all/before_each/after_each hooks.",
			},
			usecase: []string{
				"Run project-selected GDScript unit tests without launching the game scene.",
				"Run fast editor-side GDScript unit tests without launching the game scene.",
				"Use --json to feed test results into CI or another tool.",
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
			usecase: []string{
				"Bootstrap a new level, menu, or prefab scene file without opening Godot manually.",
				"Generate scene stubs in a batch script during project scaffolding.",
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
			usecase: []string{
				"Switch the editor to a specific scene from a script or CI step.",
				"Prepare a scene for node mutation commands that operate on the open scene.",
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
			usecase: []string{
				"Embed a packed scene under a parent node in the currently open scene.",
				"Compose complex scenes from reusable building blocks in one script.",
			},
		},
		{
			sub:  "tree",
			line: "  gdctl [--host host] [--port port] [--token token] scene tree",
			desc: "print the current scene node tree",
			usecase: []string{
				"Inspect the live node hierarchy of the currently open scene.",
				"Find the correct node path before running node commands.",
				"Confirm the scene structure matches expectations after a mutation.",
			},
		},
		{
			sub:  "save",
			line: "  gdctl [--host host] [--port port] [--token token] scene save",
			desc: "save the current scene",
			flags: []helpFlag{
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for save job (default 5s)"},
			},
			usecase: []string{
				"Persist scene changes made by node commands back to disk.",
				"Run at the end of a batch mutation script to commit all edits.",
			},
		},
		{
			sub:  "apply",
			line: "  gdctl [--host host] [--port port] [--token token] scene apply --path SCENE --file TREE.json [--dry-run] [--timeout DURATION]",
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
			usecase: []string{
				"Apply a full JSON node-tree definition to a scene file in one operation.",
				"Use for declarative scene authoring driven by a code generator.",
				"Use --dry-run to validate the tree shape without mutating the file.",
			},
		},
		{
			sub:  "batch",
			line: "  gdctl [--host host] [--port port] [--token token] scene batch --path SCENE --file OPS.json [--timeout DURATION]",
			desc: "open a scene once, run several mutations, and save once",
			flags: []helpFlag{
				{name: "path", meta: "SCENE", usage: "scene to open and mutate (res://main.tscn)"},
				{name: "file", meta: "OPS.json", usage: "JSON operations file"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for open/save jobs (default 5s)"},
			},
			notes: []string{
				"Supported ops: node.add, node.set, node.set-many, node.attach-script, node.set-resource.",
				"Use this when several small edits should share one open/save cycle.",
			},
			usecase: []string{
				"Apply several node mutations to a scene with a single open/save cycle.",
				"More efficient than calling individual node commands when many edits are needed.",
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
			usecase: []string{
				"Enumerate all .tscn files in the project for scripting or auditing.",
				"Use --dir to restrict the search to a specific directory.",
			},
		},
		{
			sub:  "apply-blueprint",
			line: "  gdctl [--host host] [--port port] [--token token] scene apply-blueprint --path SCENE --blueprint NAME [--dry-run] [--timeout DURATION]",
			desc: "apply a named node-tree blueprint to the current scene",
			flags: []helpFlag{
				{name: "path", meta: "SCENE", usage: "scene to open and mutate (res://main.tscn)"},
				{name: "blueprint", meta: "NAME", usage: "blueprint name: player3d, spotlight, trigger_area, hud_label, world_environment, directional_light, gpu_particles"},
				{name: "dry-run", usage: "validate without mutating or saving"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for open/save jobs (default 5s)"},
			},
			usecase: []string{
				"Quickly scaffold a common node subtree (player, HUD, lights) from a named template.",
				"Use in project setup scripts to add standard game structures in one command.",
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
			usecase: []string{
				"Run a scene headlessly (no editor required) for unit-level tests.",
				"Use in CI environments where a full Godot editor is not available.",
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
			usecase: []string{
				"Add a new node to the scene tree from a script or CI step.",
				"Use --dry-run to validate the parent path and type before committing.",
				"Set initial properties with --prop to avoid a separate node set call.",
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
			usecase: []string{
				"Delete a node and its children from the scene.",
				"Use --dry-run to confirm the target path before destructive removal.",
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
			usecase: []string{
				"Rename a node in place without changing its position in the hierarchy.",
				"Fix naming inconsistencies in a batch rename script.",
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
			usecase: []string{
				"Reparent a node to a different parent within the scene tree.",
				"Reorganize the scene hierarchy from a script.",
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
			usecase: []string{
				"Read a single node property for inspection or assertion.",
				"Verify a property value was set correctly after a node set call.",
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
				{name: "node-path", meta: "PATH", usage: "NodePath shorthand (e.g. /root/World/Player)"},
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
			usecase: []string{
				"Assign a material, texture, or other resource to a node property.",
				"Apply a shared resource across multiple nodes in a loop.",
			},
		},
		{
			sub:  "set-many",
			line: "  gdctl [--host host] [--port port] [--token token] node set-many --path PATH --file PROPS.json [--scene SCENE] [--timeout DURATION]",
			desc: "set several node properties from one JSON file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "file", meta: "PROPS.json", usage: `JSON file shaped as {"properties":{"text":{"kind":"String","value":"Hi"}}}`},
				{name: "scene", meta: "SCENE", usage: "open this scene before mutating and save it after"},
				{name: "timeout", meta: "DURATION", usage: "maximum time to wait for scene open/save jobs (default 5s)"},
			},
			usecase: []string{
				"Set several properties on one node from a JSON file in a single call.",
				"Efficient when initializing a node with many known values.",
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
			usecase: []string{
				"Attach a GDScript file to a node and validate its syntax at the same time.",
				"Wire up game logic to a newly added node in a scene setup script.",
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
			usecase: []string{
				"Add a node to a named group (e.g., 'enemies', 'collectibles') for group-based queries.",
				"Tag nodes for behavior systems that rely on group membership.",
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
			usecase: []string{
				"Remove a node from a group when it no longer belongs there.",
				"Clean up group memberships during scene refactoring.",
			},
		},
		{
			sub:  "group list",
			line: "  gdctl [--host host] [--port port] [--token token] node group list --path PATH",
			desc: "list groups on a node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
			},
			usecase: []string{
				"Audit which groups a specific node belongs to.",
				"Debug group membership issues when group-based game logic misbehaves.",
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
			usecase: []string{
				"Copy a node (and its subtree) to create a variant without recreating it manually.",
				"Batch-create similar nodes by duplicating a configured template node.",
			},
		},
		{
			sub:  "list-properties",
			line: "  gdctl [--host host] [--port port] [--token token] node list-properties --path PATH",
			desc: "list all exported properties on a node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
			},
			usecase: []string{
				"Discover all exported properties on a node type before using node set.",
				"Audit what properties are available on a custom script node.",
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
			usecase: []string{
				"Stub out a new GDScript file with the correct extends header.",
				"Generate script files during project scaffolding without opening the editor.",
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
			usecase: []string{
				"Validate and write a GDScript file body in one step.",
				"Use --body-file to write large scripts from local files.",
				"Safely deploy new command handler scripts before running addon update.",
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
			usecase: []string{
				"Lint a GDScript file for syntax errors without writing anything.",
				"Run in CI to catch script errors before deploying new addon code.",
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
			usecase: []string{
				"Write and validate a GDShader file in one step.",
				"Iterate on shader code from your editor and push it to Godot with a single command.",
			},
		},
		{
			sub:  "check",
			line: "  gdctl [--host host] [--port port] [--token token] shader check --path PATH",
			desc: "syntax-check a shader file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "shader path (res://)"},
			},
			usecase: []string{
				"Validate a shader file's syntax without writing it.",
				"Run in CI to catch shader errors early.",
			},
		},
	}},
	{name: "resource", cmds: []helpCmd{
		{
			sub:  "create",
			line: "  gdctl [--host host] [--port port] [--token token] resource create --path PATH (--type TYPE | --script SCRIPT) [--prop NAME=TYPED_JSON] [--shader-param NAME=RESOURCE]",
			desc: "create a Godot resource file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "resource file path (res://, .tres)"},
				{name: "type", meta: "TYPE", usage: "Godot resource class name (e.g. StandardMaterial3D)"},
				{name: "script", meta: "SCRIPT", usage: "GDScript Resource subclass to instantiate"},
				{name: "prop", meta: "NAME=TYPED_JSON", usage: "property value in name=TYPED_JSON form (repeatable)"},
				{name: "shader-param", meta: "NAME=PATH", usage: "ShaderMaterial param in name=res://path form (repeatable)"},
			},
			notes: []string{
				"Use --script for custom Resource scripts that are not yet registered with ClassDB.",
			},
			usecase: []string{
				"Create a Godot resource file (material, curve, custom Resource) from a script.",
				"Use --script for custom Resource subclasses not registered with ClassDB.",
				"Set initial properties with --prop to avoid a separate step.",
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
			usecase: []string{
				"Enumerate resource files in the project for auditing or batch processing.",
				"Use --ext to filter by file type (e.g., .tres).",
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
			usecase: []string{
				"Change import parameters (compression, mipmaps, etc.) for an asset without opening the editor.",
				"Batch-update texture import settings across many files from a script.",
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
			usecase: []string{
				"Upload any binary file (PNG, audio, etc.) to a res:// path in the project.",
				"Use as the final step after generating binary data locally.",
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
			usecase: []string{
				"Generate and upload a 256x1 edge LUT PNG from a JSON profile definition.",
				"Keep LUT authoring data-driven and reproducible from source files.",
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
			usecase: []string{
				"List files in a res:// directory for scripting or auditing.",
				"Verify a file was written to the expected path.",
			},
		},
		{
			sub:  "mkdir",
			line: "  gdctl [--host host] [--port port] [--token token] file mkdir --path PATH",
			desc: "create a res:// directory",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "res:// directory path to create"},
			},
			usecase: []string{
				"Create a res:// directory before writing files into it.",
				"Ensure directory structure exists in a project setup script.",
			},
		},
		{
			sub:  "delete",
			line: "  gdctl [--host host] [--port port] [--token token] file delete --path PATH",
			desc: "delete a res:// file",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "res:// path to delete"},
			},
			usecase: []string{
				"Remove a generated or temporary res:// file.",
				"Clean up stale assets during a build pipeline step.",
			},
		},
		{
			sub:  "exists",
			line: "  gdctl [--host host] [--port port] [--token token] file exists --path PATH",
			desc: "check whether a res:// path exists",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "res:// path to check"},
			},
			usecase: []string{
				"Check whether a res:// path exists before trying to read or write it.",
				"Gate conditional logic in automation scripts on file presence.",
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
			usecase: []string{
				"Bake a navigation mesh after adding or moving geometry that affects pathfinding.",
				"Automate navmesh re-baking as part of a level-build pipeline.",
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
			usecase: []string{
				"Wire a signal from one node to a method on another from a script.",
				"Set up event-driven communication between nodes during scene setup.",
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
			usecase: []string{
				"Remove a signal connection that is no longer needed.",
				"Clean up connections during scene refactoring.",
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
			usecase: []string{
				"Read a project setting value for inspection or assertion.",
				"Verify a setting was applied correctly after a project setting set call.",
			},
		},
		{
			sub:  "setting set",
			line: "  gdctl [--host host] [--port port] [--token token] project setting set --key KEY (--value TYPED_JSON | --string S | --int N | --float N | --bool BOOL | --node-path PATH | --vector2 X,Y | --vector3 X,Y,Z | --color R,G,B[,A] | --resource PATH | --array-vector3 A;B)",
			desc: "set a project setting value",
			flags: []helpFlag{
				{name: "key", meta: "KEY", usage: "project setting key"},
				{name: "value", meta: "TYPED_JSON", usage: `typed JSON value (e.g. {"kind":"int","value":1920})`},
				{name: "string", meta: "S", usage: "string shorthand"},
				{name: "int", meta: "N", usage: "integer shorthand"},
				{name: "float", meta: "N", usage: "float shorthand"},
				{name: "bool", meta: "BOOL", usage: "boolean shorthand"},
				{name: "node-path", meta: "PATH", usage: "NodePath shorthand (e.g. /root/World/Player)"},
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
			usecase: []string{
				"Update a project setting (window size, physics ticks, etc.) from a script.",
				"Apply environment-specific settings (e.g., resolution) in a CI step.",
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
			usecase: []string{
				"Run the project headlessly for integration tests without the editor.",
				"Use in CI environments where a full Godot editor is not available.",
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
			usecase: []string{
				"Capture the editor viewport (not the running game) as a PNG.",
				"Use for visual regression tests of scene layout in the editor.",
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
			usecase: []string{
				"Resize the main window or a SubViewport to a specific resolution.",
				"Set a known resolution before taking a screenshot for consistent comparisons.",
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
			usecase: []string{
				"Add a SubViewport for split-screen, picture-in-picture, or off-screen rendering.",
				"Create a secondary render target for a minimap or rear-view mirror effect.",
			},
		},
		{
			sub:  "camera-assign",
			line: "  gdctl [--host host] [--port port] [--token token] viewport camera-assign --viewport PATH --camera PATH",
			desc: "make a Camera3D or Camera2D current inside a SubViewport",
			flags: []helpFlag{
				{name: "viewport", meta: "PATH", usage: "SubViewport node path"},
				{name: "camera", meta: "PATH", usage: "Camera3D or Camera2D node path"},
			},
			usecase: []string{
				"Make a specific Camera3D or Camera2D the active camera inside a SubViewport.",
				"Set up the correct camera for each panel in a split-screen layout.",
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
			usecase: []string{
				"Create a reusable Theme resource file for consistent UI styling.",
				"Bootstrap a theme file during project setup before applying overrides.",
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
			usecase: []string{
				"Override a named color (e.g., font_color) on a UI control type in a theme.",
				"Apply branding colors to all Labels or Buttons through one command.",
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
			usecase: []string{
				"Set a named font size override on a UI control type in a theme.",
				"Adjust text size for accessibility or responsive layout changes.",
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
			usecase: []string{
				"Set a named integer constant (e.g., margin_top) on a UI control in a theme.",
				"Fine-tune spacing and layout constants without opening the editor.",
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
			usecase: []string{
				"Create a new named animation in an AnimationLibrary resource.",
				"Use --loop to enable looping for idle or walk-cycle animations.",
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
			usecase: []string{
				"Add a value track for a specific node property to an animation.",
				"Set up a track before inserting keyframes.",
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
			usecase: []string{
				"Insert a keyframe value at a specific time on an existing track.",
				"Build animation curves frame by frame from a script.",
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
			usecase: []string{
				"Adjust the total duration of an animation.",
				"Extend or trim an animation after adding or removing keyframes.",
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
			usecase: []string{
				"Trigger playback of an animation on an AnimationPlayer node in the open scene.",
				"Preview an animation programmatically without clicking Play in the editor.",
			},
		},
		{
			sub:  "tree add-state",
			line: "  gdctl [--host host] [--port port] [--token token] animation tree add-state --tree PATH --name NAME --animation ANIM",
			desc: "add an animation state to an AnimationNodeStateMachine",
			flags: []helpFlag{
				{name: "tree", meta: "PATH", usage: "AnimationTree node path"},
				{name: "name", meta: "NAME", usage: "state name"},
				{name: "animation", meta: "ANIM", usage: "animation resource name"},
			},
			usecase: []string{
				"Add an animation state to a state machine for character behavior (idle, walk, run).",
				"Scaffold the full animation state graph from a script.",
			},
		},
		{
			sub:  "tree add-transition",
			line: "  gdctl [--host host] [--port port] [--token token] animation tree add-transition --tree PATH --from STATE --to STATE --condition COND",
			desc: "add a transition between two states in an AnimationNodeStateMachine",
			flags: []helpFlag{
				{name: "tree", meta: "PATH", usage: "AnimationTree node path"},
				{name: "from", meta: "STATE", usage: "source state name"},
				{name: "to", meta: "STATE", usage: "destination state name"},
				{name: "condition", meta: "COND", usage: "advance condition expression"},
			},
			usecase: []string{
				"Connect two states in an AnimationNodeStateMachine with a condition.",
				"Define state transitions for character movement or combat logic.",
			},
		},
		{
			sub:  "tree blend-space-2d-add",
			line: "  gdctl [--host host] [--port port] [--token token] animation tree blend-space-2d-add --tree PATH --state STATE --blend-x PARAM --blend-y PARAM",
			desc: "replace a state with an AnimationNodeBlendSpace2D node",
			flags: []helpFlag{
				{name: "tree", meta: "PATH", usage: "AnimationTree node path"},
				{name: "state", meta: "STATE", usage: "state name to replace"},
				{name: "blend-x", meta: "PARAM", usage: "blend parameter name for X axis"},
				{name: "blend-y", meta: "PARAM", usage: "blend parameter name for Y axis"},
			},
			usecase: []string{
				"Replace a state with a 2D blend space for directional movement blending.",
				"Set up a blend tree that blends animations based on velocity X/Y parameters.",
			},
		},
		{
			sub:  "tree set-param",
			line: "  gdctl [--host host] [--port port] [--token token] animation tree set-param --tree PATH --param PARAM (--vector2 X,Y | --float N | --bool B | --int N)",
			desc: "set a runtime parameter on an AnimationTree",
			usecase: []string{
				"Drive an AnimationTree parameter (playback, blend position) from a script.",
				"Programmatically switch animation states during a run probe test.",
			},
			flags: []helpFlag{
				{name: "tree", meta: "PATH", usage: "AnimationTree node path"},
				{name: "param", meta: "PARAM", usage: "parameter path, e.g. parameters/playback"},
				{name: "vector2", meta: "X,Y", usage: "set parameter to a Vector2 value"},
				{name: "float", meta: "N", usage: "set parameter to a float value"},
				{name: "bool", meta: "B", usage: "set parameter to a bool value (true/false)"},
				{name: "int", meta: "N", usage: "set parameter to an integer value"},
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
			usecase: []string{
				"Create a new TileSet resource for a TileMap node.",
				"Bootstrap tilemap assets during project setup.",
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
			usecase: []string{
				"Add an atlas texture source to an existing TileSet.",
				"Register a spritesheet so its tiles can be painted onto a TileMap.",
			},
		},
		{
			sub:  "cell-set",
			line: "  gdctl [--host host] [--port port] [--token token] tilemap cell-set --node PATH --layer N --x X --y Y --source-id ID [--atlas-x AX] [--atlas-y AY]",
			desc: "paint a cell on a TileMap node",
			usecase: []string{
				"Paint a single tile at a specific grid cell on a TileMap layer.",
				"Place individual tiles in a scripted level generator.",
			},
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
			sub:  "cell-set-rect",
			line: "  gdctl [--host host] [--port port] [--token token] tilemap cell-set-rect --node PATH --layer N --x X --y Y --width W --height H --source-id ID [--atlas-x AX] [--atlas-y AY]",
			desc: "paint a rectangular area on a TileMap node",
			flags: []helpFlag{
				{name: "node", meta: "PATH", usage: "TileMap node path in the open scene"},
				{name: "layer", meta: "N", usage: "layer index (default 0)"},
				{name: "x", meta: "X", usage: "rectangle start column"},
				{name: "y", meta: "Y", usage: "rectangle start row"},
				{name: "width", meta: "W", usage: "rectangle width in cells"},
				{name: "height", meta: "H", usage: "rectangle height in cells"},
				{name: "source-id", meta: "ID", usage: "TileSet source ID"},
				{name: "atlas-x", meta: "AX", usage: "atlas tile column (default 0)"},
				{name: "atlas-y", meta: "AY", usage: "atlas tile row (default 0)"},
			},
			usecase: []string{
				"Fill a rectangular area with the same tile in one command.",
				"Paint floor, wall, or background tiles efficiently in a batch script.",
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
			usecase: []string{
				"Erase a single tile from a TileMap cell.",
				"Remove a tile placed by a level generator that is no longer needed.",
			},
		},
	}},
	{name: "audio", cmds: []helpCmd{
		{
			sub:  "bus-add",
			line: "  gdctl [--host host] [--port port] [--token token] audio bus-add --name NAME [--if-missing]",
			desc: "add a named audio bus",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "bus name"},
				{name: "if-missing", usage: "succeed without changes when the bus already exists"},
			},
			usecase: []string{
				"Create a named audio bus (e.g., Music, SFX, Ambient) for routing audio.",
				"Set up the audio bus layout programmatically during project configuration.",
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
			usecase: []string{
				"Adjust the volume of an audio bus in decibels.",
				"Apply volume presets for different game states (e.g., quieter during cutscenes).",
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
			usecase: []string{
				"Add a reverb, compressor, or other AudioEffect to an audio bus.",
				"Configure audio processing from a script without opening the Godot Audio panel.",
			},
		},
		{
			sub:  "listener-make-current",
			line: "  gdctl [--host host] [--port port] [--token token] audio listener-make-current --path PATH",
			desc: "make an AudioListener3D or AudioListener2D the active listener",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "AudioListener3D or AudioListener2D node path"},
			},
			usecase: []string{
				"Set the active audio listener for spatial audio in a 3D or 2D scene.",
				"Switch the listener when the camera or player changes during gameplay.",
			},
		},
		{
			sub:  "playlist-add",
			line: "  gdctl [--host host] [--port port] [--token token] audio playlist-add --bus BUS --stream PATH",
			desc: "add a stream to an AudioStreamPlaylist on the named bus",
			flags: []helpFlag{
				{name: "bus", meta: "BUS", usage: "audio bus name (e.g. Music)"},
				{name: "stream", meta: "PATH", usage: "stream resource path (res://*.ogg)"},
			},
			usecase: []string{
				"Add a music stream to an AudioStreamPlaylist on a named bus.",
				"Build a dynamic music playlist from a script.",
			},
		},
		{
			sub:  "playlist-autoplay",
			line: "  gdctl [--host host] [--port port] [--token token] audio playlist-autoplay --bus BUS --mode MODE",
			desc: "configure autoplay and shuffle on a bus playlist",
			flags: []helpFlag{
				{name: "bus", meta: "BUS", usage: "audio bus name"},
				{name: "mode", meta: "MODE", usage: "playback mode: sequential or random_no_repeat"},
			},
			usecase: []string{
				"Configure autoplay and shuffle behavior on a bus playlist.",
				"Set up sequential or random-no-repeat music playback.",
			},
		},
	}},
	{name: "softbody", cmds: []helpCmd{
		{
			sub:  "pin-point",
			line: "  gdctl [--host host] [--port port] [--token token] recipe softbody pin-point --path PATH --point N",
			desc: "pin a vertex of a SoftBody3D to prevent it from simulating",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "SoftBody3D node path"},
				{name: "point", meta: "N", usage: "vertex index to pin"},
			},
			usecase: []string{
				"Pin a SoftBody3D vertex to anchor it (e.g., a cloth hanging point).",
				"Prevent simulation on specific vertices for a partial soft-body effect.",
			},
		},
		{
			sub:  "unpin-point",
			line: "  gdctl [--host host] [--port port] [--token token] recipe softbody unpin-point --path PATH --point N",
			desc: "unpin a previously pinned SoftBody3D vertex",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "SoftBody3D node path"},
				{name: "point", meta: "N", usage: "vertex index to unpin"},
			},
			usecase: []string{
				"Release a previously pinned vertex to let it simulate freely.",
				"Dynamically toggle pinning during gameplay or test setup.",
			},
		},
	}},
	{name: "lod", cmds: []helpCmd{
		{
			sub:  "set",
			line: "  gdctl [--host host] [--port port] [--token token] recipe lod set --path PATH --begin N --end N",
			desc: "set LOD visibility range on a GeometryInstance3D node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "begin", meta: "N", usage: "distance at which the node begins to fade in"},
				{name: "end", meta: "N", usage: "distance at which the node is fully hidden"},
			},
			usecase: []string{
				"Set the visibility distance range on a mesh node for LOD optimization.",
				"Configure at what distance a mesh fades in or out during rendering.",
			},
		},
		{
			sub:  "set-many",
			line: "  gdctl [--host host] [--port port] [--token token] recipe lod set-many --file FILE",
			desc: "set LOD ranges on multiple nodes from a JSON file",
			flags: []helpFlag{
				{name: "file", meta: "FILE", usage: `JSON array: [{"path":"/root/Node","begin":20.0,"end":40.0},...]`},
			},
			usecase: []string{
				"Apply LOD ranges to many nodes from a single JSON file.",
				"Batch-configure LOD for all meshes in a scene during a performance pass.",
			},
		},
	}},
	{name: "terrain", cmds: []helpCmd{
		{
			sub:  "heightmap-import",
			line: "  gdctl [--host host] [--port port] [--token token] recipe terrain heightmap-import --path PATH --texture TEX --min-height N --max-height N",
			desc: "import a heightmap image into a HeightMapShape3D node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "HeightMapShape3D node path"},
				{name: "texture", meta: "TEX", usage: "heightmap image resource path (res://)"},
				{name: "min-height", meta: "N", usage: "minimum height value in the heightmap"},
				{name: "max-height", meta: "N", usage: "maximum height value in the heightmap"},
			},
			usecase: []string{
				"Import a heightmap image into a HeightMapShape3D for physics-accurate terrain collision.",
				"Apply a procedurally generated heightmap to terrain in a build pipeline.",
			},
		},
	}},
	{name: "lightmap", cmds: []helpCmd{
		{
			sub:  "bake",
			line: "  gdctl [--host host] [--port port] [--token token] lightmap bake --path PATH",
			desc: "trigger a LightmapGI bake (returns immediately; bake runs asynchronously)",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "LightmapGI node path"},
			},
			usecase: []string{
				"Trigger a LightmapGI bake from a script to precompute static lighting.",
				"Automate lightmap generation as part of a level-finalization pipeline.",
			},
		},
	}},
	{name: "voxelgi", cmds: []helpCmd{
		{
			sub:  "bake",
			line: "  gdctl [--host host] [--port port] [--token token] recipe voxelgi bake --path PATH",
			desc: "trigger a VoxelGI bake",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "VoxelGI node path"},
			},
			usecase: []string{
				"Bake a VoxelGI node for real-time indirect lighting in a scene.",
				"Refresh VoxelGI after moving static geometry that affects the light volume.",
			},
		},
	}},
	{name: "reflection-probe", cmds: []helpCmd{
		{
			sub:  "bake",
			line: "  gdctl [--host host] [--port port] [--token token] recipe reflection-probe bake --path PATH",
			desc: "attempt to bake a ReflectionProbe (not supported via GDScript; returns status)",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "ReflectionProbe node path"},
			},
			usecase: []string{
				"Attempt to bake a ReflectionProbe for static reflections.",
				"Use after placing a ReflectionProbe to precompute its reflection data.",
			},
		},
	}},
	{name: "window", cmds: []helpCmd{
		{
			sub:  "create",
			line: "  gdctl [--host host] [--port port] [--token token] window create [--title TITLE] [--width W] [--height H] [--position X,Y]",
			desc: "create a new floating Window node in the scene",
			flags: []helpFlag{
				{name: "title", meta: "TITLE", usage: "window title (default Window)"},
				{name: "width", meta: "W", usage: "window width in pixels (default 640)"},
				{name: "height", meta: "H", usage: "window height in pixels (default 480)"},
				{name: "position", meta: "X,Y", usage: "screen position (default 0,0)"},
			},
			usecase: []string{
				"Create a floating Window node for secondary UI (inventory, map, debug overlay).",
				"Open a new window programmatically from a scene setup script.",
			},
		},
		{
			sub:  "assign-viewport",
			line: "  gdctl [--host host] [--port port] [--token token] window assign-viewport --window-id ID --viewport PATH",
			desc: "reparent a SubViewport node into a Window",
			flags: []helpFlag{
				{name: "window-id", meta: "ID", usage: "window ID returned by window create"},
				{name: "viewport", meta: "PATH", usage: "SubViewport node path to assign"},
			},
			usecase: []string{
				"Embed a SubViewport inside a Window for a secondary render view in its own window.",
				"Combine with viewport add to create a detachable render panel.",
			},
		},
	}},
	{name: "graph-edit", cmds: []helpCmd{
		{
			sub:  "node-add",
			line: "  gdctl [--host host] [--port port] [--token token] recipe graph-edit node-add --path GRAPH --name NAME [--position X,Y]",
			desc: "add a GraphNode to a GraphEdit control",
			flags: []helpFlag{
				{name: "path", meta: "GRAPH", usage: "GraphEdit node path"},
				{name: "name", meta: "NAME", usage: "node name and title"},
				{name: "position", meta: "X,Y", usage: "node position offset in the graph (default 0,0)"},
			},
			usecase: []string{
				"Add a GraphNode to a GraphEdit for visual scripting or node-based UI layouts.",
				"Scaffold a node graph from a script without manual editor interaction.",
			},
		},
		{
			sub:  "connection-add",
			line: "  gdctl [--host host] [--port port] [--token token] recipe graph-edit connection-add --graph GRAPH --from NODE --from-port N --to NODE --to-port N",
			desc: "connect two GraphNodes in a GraphEdit",
			flags: []helpFlag{
				{name: "graph", meta: "GRAPH", usage: "GraphEdit node path"},
				{name: "from", meta: "NODE", usage: "source GraphNode name"},
				{name: "from-port", meta: "N", usage: "output port index on source"},
				{name: "to", meta: "NODE", usage: "destination GraphNode name"},
				{name: "to-port", meta: "N", usage: "input port index on destination"},
			},
			usecase: []string{
				"Wire two GraphNodes together in a GraphEdit to define graph logic.",
				"Connect nodes in a generated node graph.",
			},
		},
		{
			sub:  "node-remove",
			line: "  gdctl [--host host] [--port port] [--token token] recipe graph-edit node-remove --path GRAPH --name NAME",
			desc: "remove a GraphNode from a GraphEdit",
			flags: []helpFlag{
				{name: "path", meta: "GRAPH", usage: "GraphEdit node path"},
				{name: "name", meta: "NAME", usage: "GraphNode name to remove"},
			},
			usecase: []string{
				"Remove a GraphNode from a GraphEdit.",
				"Clean up dynamically generated graph nodes.",
			},
		},
	}},
	{name: "accessibility", cmds: []helpCmd{
		{
			sub:  "tts-speak",
			line: "  gdctl [--host host] [--port port] [--token token] accessibility tts-speak --text TEXT [--interrupt]",
			desc: "speak text via the OS text-to-speech engine",
			flags: []helpFlag{
				{name: "text", meta: "TEXT", usage: "text to speak"},
				{name: "interrupt", usage: "interrupt any currently speaking text"},
			},
			usecase: []string{
				"Read UI text aloud for screen-reader accessibility testing.",
				"Trigger voice feedback in game events during an accessibility audit.",
			},
		},
		{
			sub:  "tts-configure",
			line: "  gdctl [--host host] [--port port] [--token token] accessibility tts-configure [--pitch N] [--rate N] [--voice VOICE]",
			desc: "configure TTS voice parameters for subsequent tts-speak calls",
			flags: []helpFlag{
				{name: "pitch", meta: "N", usage: "voice pitch multiplier (default 1.0)"},
				{name: "rate", meta: "N", usage: "speech rate multiplier (default 1.0)"},
				{name: "voice", meta: "VOICE", usage: "voice identifier string"},
			},
			usecase: []string{
				"Set the TTS voice pitch, rate, or voice before speaking.",
				"Customize the speech voice for locale or character personality.",
			},
		},
		{
			sub:  "tts-stop",
			line: "  gdctl [--host host] [--port port] [--token token] accessibility tts-stop",
			desc: "stop any currently playing TTS speech",
			usecase: []string{
				"Interrupt ongoing TTS speech immediately.",
				"Stop voice feedback when a new UI state takes over.",
			},
		},
	}},
	{name: "localization", cmds: []helpCmd{
		{
			sub:  "locale-set",
			line: "  gdctl [--host host] [--port port] [--token token] localization locale-set --locale LOCALE",
			desc: "set the active locale in TranslationServer",
			flags: []helpFlag{
				{name: "locale", meta: "LOCALE", usage: "locale code, e.g. en, ja, fr"},
			},
			usecase: []string{
				"Switch the active locale to test UI layout and string rendering for a specific language.",
				"Run automated locale-specific screenshot tests.",
			},
		},
		{
			sub:  "string-add",
			line: "  gdctl [--host host] [--port port] [--token token] localization string-add --key KEY --locale LOCALE --text TEXT",
			desc: "add or update a translation string at runtime",
			flags: []helpFlag{
				{name: "key", meta: "KEY", usage: "translation key"},
				{name: "locale", meta: "LOCALE", usage: "locale code"},
				{name: "text", meta: "TEXT", usage: "translated string"},
			},
			usecase: []string{
				"Add or update a translation string at runtime for rapid localization iteration.",
				"Inject test strings to verify UI rendering without rebuilding the translation CSV.",
			},
		},
	}},
	{name: "csg", cmds: []helpCmd{
		{
			sub:  "node-add",
			line: "  gdctl [--host host] [--port port] [--token token] recipe csg node-add --parent PATH --type TYPE --name NAME [--no-collision]",
			desc: "add a CSG node with use_collision=true by default",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "type", meta: "TYPE", usage: "CSG node type: CSGBox3D, CSGSphere3D, CSGCylinder3D, CSGTorus3D, CSGMesh3D, CSGCombiner3D"},
				{name: "name", meta: "NAME", usage: "node name"},
				{name: "no-collision", usage: "skip setting use_collision=true (omit for non-physics CSG)"},
			},
			usecase: []string{
				"Add a CSG primitive (box, sphere, cylinder) for rapid level prototyping.",
				"Block out level geometry with CSG shapes before replacing with real meshes.",
				"Use --no-collision for decorative CSG that doesn't need a physics body.",
			},
		},
		{
			sub:  "operation-set",
			line: "  gdctl [--host host] [--port port] [--token token] recipe csg operation-set --path PATH --operation OP",
			desc: "set the CSG boolean operation on a CSG node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "CSG node path"},
				{name: "operation", meta: "OP", usage: "operation: union, intersection, or subtraction"},
			},
			usecase: []string{
				"Set the boolean operation (union, intersection, subtraction) on a CSG node.",
				"Carve holes or merge shapes during level construction.",
			},
		},
		{
			sub:  "size-set",
			line: "  gdctl [--host host] [--port port] [--token token] recipe csg size-set --path PATH --size X,Y,Z",
			desc: "set the size of a CSGBox3D node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "CSGBox3D node path"},
				{name: "size", meta: "X,Y,Z", usage: "size vector"},
			},
			usecase: []string{
				"Resize a CSGBox3D without opening the inspector.",
				"Adjust blocking geometry dimensions from a script.",
			},
		},
	}},
	{name: "environment", cmds: []helpCmd{
		{
			sub:  "set-background",
			line: "  gdctl [--host host] [--port port] [--token token] recipe environment set-background --path PATH --mode color|sky|clear [--color R,G,B[,A]]",
			desc: "set the background mode and color on a WorldEnvironment node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "WorldEnvironment node path in the open scene"},
				{name: "mode", meta: "MODE", usage: "background mode: color, sky, clear, canvas, keep, or camera_feed (default color)"},
				{name: "color", meta: "R,G,B[,A]", usage: "background color in 0.0–1.0 per channel; required when --mode color"},
			},
			usecase: []string{
				"Set a solid sky color without manually creating an Environment resource.",
				"Switch between sky, solid color, and transparent background in one command.",
				"Configure WorldEnvironment background during automated scene setup scripts.",
			},
		},
	}},
	{name: "decal", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] recipe decal add --parent PATH --texture TEX --size X,Y,Z",
			desc: "add a Decal node to a parent",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "texture", meta: "TEX", usage: "albedo texture resource path (res://)"},
				{name: "size", meta: "X,Y,Z", usage: "decal size in 3D space"},
			},
			usecase: []string{
				"Add a decal for blood splatter, bullet holes, or surface markings.",
				"Place a texture-projected marking on geometry from a script.",
			},
		},
		{
			sub:  "set-normal-fade",
			line: "  gdctl [--host host] [--port port] [--token token] recipe decal set-normal-fade --path PATH --fade N",
			desc: "set the normal fade factor on a Decal node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "Decal node path"},
				{name: "fade", meta: "N", usage: "normal fade value (0.0–1.0)"},
			},
			usecase: []string{
				"Control how aggressively the decal fades on surfaces facing away from its projection axis.",
				"Tune decal blending for surfaces at steep angles.",
			},
		},
	}},
	{name: "fog-volume", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] recipe fog-volume add --parent PATH --shape SHAPE --size X,Y,Z --density N",
			desc: "add a FogVolume node with a FogMaterial to a parent",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "shape", meta: "SHAPE", usage: "volume shape: box, ellipsoid, cone, cylinder, or world"},
				{name: "size", meta: "X,Y,Z", usage: "volume extents"},
				{name: "density", meta: "N", usage: "fog density"},
			},
			usecase: []string{
				"Add a FogVolume for atmospheric effects (smoke, mist, area fog).",
				"Define a fog zone with a specific shape and density.",
			},
		},
	}},
	{name: "occluder", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] recipe occluder add --parent PATH --shape SHAPE --size X,Y,Z",
			desc: "add an OccluderInstance3D node to a parent",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "shape", meta: "SHAPE", usage: "occluder shape: box, sphere, or quad"},
				{name: "size", meta: "X,Y,Z", usage: "occluder extents"},
			},
			usecase: []string{
				"Add an OccluderInstance3D to improve GPU occlusion culling performance.",
				"Place invisible geometry that tells the renderer what to cull behind it.",
			},
		},
	}},
	{name: "apply", cmds: []helpCmd{
		{
			sub:  "apply",
			line: "  gdctl [--host host] [--port port] [--token token] apply FILE --scene SCENE [--dry-run] [--json]",
			desc: "apply a JSON scene tree as desired state (top-level workflow command)",
			flags: []helpFlag{
				{name: "scene", meta: "SCENE", usage: "scene path to open and mutate"},
				{name: "dry-run", usage: "preview changes without saving"},
				{name: "json", usage: "print result as JSON"},
			},
			usecase: []string{
				"Apply a JSON scene descriptor to a scene file in a single command.",
				"Use in CI pipelines to converge scene state from a declarative file.",
			},
		},
	}},
	{name: "plan", cmds: []helpCmd{
		{
			sub:  "plan",
			line: "  gdctl [--host host] [--port port] [--token token] plan FILE --scene SCENE [--json]",
			desc: "preview what apply would change without writing (dry run)",
			flags: []helpFlag{
				{name: "scene", meta: "SCENE", usage: "scene path to preview"},
				{name: "json", usage: "print result as JSON"},
			},
			usecase: []string{
				"Preview nodes and properties that would be created or updated without writing anything.",
				"Use before apply to review changes in CI or review workflows.",
			},
		},
	}},
	{name: "gate", cmds: []helpCmd{
		{
			sub:  "run",
			line: "  gdctl [--host host] [--port port] [--token token] gate run [PROFILE] [--json]",
			desc: "run a validation gate profile (production, ci, quick)",
			flags: []helpFlag{
				{name: "profile", meta: "PROFILE", usage: "gate profile: production, ci, or quick (default: quick)"},
				{name: "json", usage: "print results as JSON"},
			},
			usecase: []string{
				"Run all project quality checks in one command before a release.",
				"Use gdctl gate run ci in continuous integration to validate bridge, scripts, and policy.",
			},
		},
	}},
	{name: "policy", cmds: []helpCmd{
		{
			sub:  "validate",
			line: "  gdctl policy validate FILE [--json]",
			desc: "validate the project against a policy file",
			flags: []helpFlag{
				{name: "file", meta: "FILE", usage: "JSON policy file path"},
				{name: "json", usage: "print results as JSON"},
			},
			usecase: []string{
				"Validate that project assets meet organizational requirements (size, format, node count).",
				"Use in outsourcing pipelines to check vendor submissions against acceptance criteria.",
				"Add to CI to block releases when policy rules are violated.",
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
	showUsecase := false
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--usecase" {
			showUsecase = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) == 0 {
		if showUsecase {
			printUsecaseAll(stdout)
			return nil
		}
		printUsage(stdout)
		return nil
	}
	for _, g := range helpGroups {
		if g.name == args[0] {
			if len(args) == 1 {
				if showUsecase {
					printUsecaseGroup(stdout, g)
					return nil
				}
				fmt.Fprintln(stdout, "Usage:")
				for _, cmd := range g.cmds {
					fmt.Fprintln(stdout, cmd.line)
				}
				return nil
			}
			sub := strings.Join(args[1:], " ")
			for _, cmd := range g.cmds {
				if cmd.sub == sub {
					return printCommandHelp(stdout, cmd, showUsecase)
				}
			}
			return fmt.Errorf("unknown subcommand %q under %q", sub, g.name)
		}
	}
	if len(args) == 1 {
		for _, g := range helpGroups {
			for _, cmd := range g.cmds {
				if cmd.sub == args[0] {
					return printCommandHelp(stdout, cmd, showUsecase)
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

func printUsecaseAll(w io.Writer) {
	for _, g := range helpGroups {
		name := strings.ToUpper(g.name[:1]) + g.name[1:]
		fmt.Fprintf(w, "%s\n", name)
		printUsecaseGroup(w, g)
	}
}

func printUsecaseGroup(w io.Writer, g helpGroup) {
	for _, cmd := range g.cmds {
		fmt.Fprintf(w, "  %s\n", cmd.sub)
		for _, u := range cmd.usecase {
			fmt.Fprintf(w, "    * %s\n", u)
		}
		fmt.Fprintln(w)
	}
}

func printCommandHelp(stdout io.Writer, cmd helpCmd, showUsecase bool) error {
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
	if showUsecase && len(cmd.usecase) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Use cases:")
		for _, u := range cmd.usecase {
			fmt.Fprintf(stdout, "  * %s\n", u)
		}
	}
	return nil
}
