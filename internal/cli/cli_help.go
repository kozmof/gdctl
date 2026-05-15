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
	{name: "autoload", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] autoload add --name NAME --path PATH",
			desc: "add an autoload singleton",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "autoload singleton name"},
				{name: "path", meta: "PATH", usage: "script or scene path (res://)"},
			},
		},
		{
			sub:  "remove",
			line: "  gdctl [--host host] [--port port] [--token token] autoload remove --name NAME",
			desc: "remove an autoload singleton",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "autoload singleton name"},
			},
		},
		{
			sub:  "list",
			line: "  gdctl [--host host] [--port port] [--token token] autoload list [--json]",
			desc: "list autoload singletons",
			flags: []helpFlag{
				{name: "json", usage: "print JSON"},
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
		},
		{
			sub:  "action remove",
			line: "  gdctl [--host host] [--port port] [--token token] input action remove --name NAME",
			desc: "remove an input action",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "input action name"},
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
			line: "  gdctl [--host host] [--port port] [--token token] run status [--json]",
			desc: "show whether the editor is running a scene and whether the runtime helper checked in",
			flags: []helpFlag{
				{name: "json", usage: "print status as JSON"},
			},
		},
		{
			sub:  "helper-status",
			line: "  gdctl [--host host] [--port port] [--token token] run helper-status [--json]",
			desc: "show gdctl runtime helper/autoload health",
			flags: []helpFlag{
				{name: "json", usage: "print helper status as JSON"},
			},
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
				"Use --source screen for the legacy whole-host-screen capture.",
				"Use --viewport to capture a specific SubViewport (e.g. a split-screen panel).",
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
		},
		{
			sub:  "profile",
			line: "  gdctl [--host host] [--port port] [--token token] run profile --metric METRICS --duration DURATION",
			desc: "sample runtime performance metrics while the game is running",
			flags: []helpFlag{
				{name: "metric", meta: "METRICS", usage: "comma-separated metrics: fps,draw_calls,physics_time,memory_usage"},
				{name: "duration", meta: "DURATION", usage: "sampling duration, e.g. 10s (default 5s)"},
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
				{name: "blueprint", meta: "NAME", usage: "blueprint name: player3d, spotlight, trigger_area, hud_label, world_environment, directional_light, gpu_particles"},
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
		{
			sub:  "camera-assign",
			line: "  gdctl [--host host] [--port port] [--token token] viewport camera-assign --viewport PATH --camera PATH",
			desc: "make a Camera3D or Camera2D current inside a SubViewport",
			flags: []helpFlag{
				{name: "viewport", meta: "PATH", usage: "SubViewport node path"},
				{name: "camera", meta: "PATH", usage: "Camera3D or Camera2D node path"},
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
		{
			sub:  "tree add-state",
			line: "  gdctl [--host host] [--port port] [--token token] animation tree add-state --tree PATH --name NAME --animation ANIM",
			desc: "add an animation state to an AnimationNodeStateMachine",
			flags: []helpFlag{
				{name: "tree", meta: "PATH", usage: "AnimationTree node path"},
				{name: "name", meta: "NAME", usage: "state name"},
				{name: "animation", meta: "ANIM", usage: "animation resource name"},
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
		},
		{
			sub:  "tree set-param",
			line: "  gdctl [--host host] [--port port] [--token token] animation tree set-param --tree PATH --param PARAM (--vector2 X,Y | --float N | --bool B | --int N)",
			desc: "set a runtime parameter on an AnimationTree",
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
			line: "  gdctl [--host host] [--port port] [--token token] audio bus-add --name NAME [--if-missing]",
			desc: "add a named audio bus",
			flags: []helpFlag{
				{name: "name", meta: "NAME", usage: "bus name"},
				{name: "if-missing", usage: "succeed without changes when the bus already exists"},
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
		{
			sub:  "listener-make-current",
			line: "  gdctl [--host host] [--port port] [--token token] audio listener-make-current --path PATH",
			desc: "make an AudioListener3D or AudioListener2D the active listener",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "AudioListener3D or AudioListener2D node path"},
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
		},
		{
			sub:  "playlist-autoplay",
			line: "  gdctl [--host host] [--port port] [--token token] audio playlist-autoplay --bus BUS --mode MODE",
			desc: "configure autoplay and shuffle on a bus playlist",
			flags: []helpFlag{
				{name: "bus", meta: "BUS", usage: "audio bus name"},
				{name: "mode", meta: "MODE", usage: "playback mode: sequential or random_no_repeat"},
			},
		},
	}},
	{name: "softbody", cmds: []helpCmd{
		{
			sub:  "pin-point",
			line: "  gdctl [--host host] [--port port] [--token token] softbody pin-point --path PATH --point N",
			desc: "pin a vertex of a SoftBody3D to prevent it from simulating",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "SoftBody3D node path"},
				{name: "point", meta: "N", usage: "vertex index to pin"},
			},
		},
		{
			sub:  "unpin-point",
			line: "  gdctl [--host host] [--port port] [--token token] softbody unpin-point --path PATH --point N",
			desc: "unpin a previously pinned SoftBody3D vertex",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "SoftBody3D node path"},
				{name: "point", meta: "N", usage: "vertex index to unpin"},
			},
		},
	}},
	{name: "lod", cmds: []helpCmd{
		{
			sub:  "set",
			line: "  gdctl [--host host] [--port port] [--token token] lod set --path PATH --begin N --end N",
			desc: "set LOD visibility range on a GeometryInstance3D node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "node path"},
				{name: "begin", meta: "N", usage: "distance at which the node begins to fade in"},
				{name: "end", meta: "N", usage: "distance at which the node is fully hidden"},
			},
		},
		{
			sub:  "set-many",
			line: "  gdctl [--host host] [--port port] [--token token] lod set-many --file FILE",
			desc: "set LOD ranges on multiple nodes from a JSON file",
			flags: []helpFlag{
				{name: "file", meta: "FILE", usage: `JSON array: [{"path":"/root/Node","begin":20.0,"end":40.0},...]`},
			},
		},
	}},
	{name: "terrain", cmds: []helpCmd{
		{
			sub:  "heightmap-import",
			line: "  gdctl [--host host] [--port port] [--token token] terrain heightmap-import --path PATH --texture TEX --min-height N --max-height N",
			desc: "import a heightmap image into a HeightMapShape3D node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "HeightMapShape3D node path"},
				{name: "texture", meta: "TEX", usage: "heightmap image resource path (res://)"},
				{name: "min-height", meta: "N", usage: "minimum height value in the heightmap"},
				{name: "max-height", meta: "N", usage: "maximum height value in the heightmap"},
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
		},
	}},
	{name: "voxelgi", cmds: []helpCmd{
		{
			sub:  "bake",
			line: "  gdctl [--host host] [--port port] [--token token] voxelgi bake --path PATH",
			desc: "trigger a VoxelGI bake",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "VoxelGI node path"},
			},
		},
	}},
	{name: "reflection-probe", cmds: []helpCmd{
		{
			sub:  "bake",
			line: "  gdctl [--host host] [--port port] [--token token] reflection-probe bake --path PATH",
			desc: "attempt to bake a ReflectionProbe (not supported via GDScript; returns status)",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "ReflectionProbe node path"},
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
		},
		{
			sub:  "assign-viewport",
			line: "  gdctl [--host host] [--port port] [--token token] window assign-viewport --window-id ID --viewport PATH",
			desc: "reparent a SubViewport node into a Window",
			flags: []helpFlag{
				{name: "window-id", meta: "ID", usage: "window ID returned by window create"},
				{name: "viewport", meta: "PATH", usage: "SubViewport node path to assign"},
			},
		},
	}},
	{name: "graph-edit", cmds: []helpCmd{
		{
			sub:  "node-add",
			line: "  gdctl [--host host] [--port port] [--token token] graph-edit node-add --path GRAPH --name NAME [--position X,Y]",
			desc: "add a GraphNode to a GraphEdit control",
			flags: []helpFlag{
				{name: "path", meta: "GRAPH", usage: "GraphEdit node path"},
				{name: "name", meta: "NAME", usage: "node name and title"},
				{name: "position", meta: "X,Y", usage: "node position offset in the graph (default 0,0)"},
			},
		},
		{
			sub:  "connection-add",
			line: "  gdctl [--host host] [--port port] [--token token] graph-edit connection-add --graph GRAPH --from NODE --from-port N --to NODE --to-port N",
			desc: "connect two GraphNodes in a GraphEdit",
			flags: []helpFlag{
				{name: "graph", meta: "GRAPH", usage: "GraphEdit node path"},
				{name: "from", meta: "NODE", usage: "source GraphNode name"},
				{name: "from-port", meta: "N", usage: "output port index on source"},
				{name: "to", meta: "NODE", usage: "destination GraphNode name"},
				{name: "to-port", meta: "N", usage: "input port index on destination"},
			},
		},
		{
			sub:  "node-remove",
			line: "  gdctl [--host host] [--port port] [--token token] graph-edit node-remove --path GRAPH --name NAME",
			desc: "remove a GraphNode from a GraphEdit",
			flags: []helpFlag{
				{name: "path", meta: "GRAPH", usage: "GraphEdit node path"},
				{name: "name", meta: "NAME", usage: "GraphNode name to remove"},
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
		},
		{
			sub:  "tts-stop",
			line: "  gdctl [--host host] [--port port] [--token token] accessibility tts-stop",
			desc: "stop any currently playing TTS speech",
		},
	}},
	{name: "i18n", cmds: []helpCmd{
		{
			sub:  "locale-set",
			line: "  gdctl [--host host] [--port port] [--token token] i18n locale-set --locale LOCALE",
			desc: "set the active locale in TranslationServer",
			flags: []helpFlag{
				{name: "locale", meta: "LOCALE", usage: "locale code, e.g. en, ja, fr"},
			},
		},
		{
			sub:  "string-add",
			line: "  gdctl [--host host] [--port port] [--token token] i18n string-add --key KEY --locale LOCALE --text TEXT",
			desc: "add or update a translation string at runtime",
			flags: []helpFlag{
				{name: "key", meta: "KEY", usage: "translation key"},
				{name: "locale", meta: "LOCALE", usage: "locale code"},
				{name: "text", meta: "TEXT", usage: "translated string"},
			},
		},
	}},
	{name: "csg", cmds: []helpCmd{
		{
			sub:  "node-add",
			line: "  gdctl [--host host] [--port port] [--token token] csg node-add --parent PATH --type TYPE --name NAME",
			desc: "add a CSG node (thin wrapper over node add)",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "type", meta: "TYPE", usage: "CSG node type: CSGBox3D, CSGSphere3D, CSGCylinder3D, CSGTorus3D, CSGMesh3D, CSGCombiner3D"},
				{name: "name", meta: "NAME", usage: "node name"},
			},
		},
		{
			sub:  "operation-set",
			line: "  gdctl [--host host] [--port port] [--token token] csg operation-set --path PATH --operation OP",
			desc: "set the CSG boolean operation on a CSG node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "CSG node path"},
				{name: "operation", meta: "OP", usage: "operation: union, intersection, or subtraction"},
			},
		},
		{
			sub:  "size-set",
			line: "  gdctl [--host host] [--port port] [--token token] csg size-set --path PATH --size X,Y,Z",
			desc: "set the size of a CSGBox3D node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "CSGBox3D node path"},
				{name: "size", meta: "X,Y,Z", usage: "size vector"},
			},
		},
	}},
	{name: "decal", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] decal add --parent PATH --texture TEX --size X,Y,Z",
			desc: "add a Decal node to a parent",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "texture", meta: "TEX", usage: "albedo texture resource path (res://)"},
				{name: "size", meta: "X,Y,Z", usage: "decal size in 3D space"},
			},
		},
		{
			sub:  "set-normal-fade",
			line: "  gdctl [--host host] [--port port] [--token token] decal set-normal-fade --path PATH --fade N",
			desc: "set the normal fade factor on a Decal node",
			flags: []helpFlag{
				{name: "path", meta: "PATH", usage: "Decal node path"},
				{name: "fade", meta: "N", usage: "normal fade value (0.0–1.0)"},
			},
		},
	}},
	{name: "fog-volume", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] fog-volume add --parent PATH --shape SHAPE --size X,Y,Z --density N",
			desc: "add a FogVolume node with a FogMaterial to a parent",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "shape", meta: "SHAPE", usage: "volume shape: box, ellipsoid, cone, cylinder, or world"},
				{name: "size", meta: "X,Y,Z", usage: "volume extents"},
				{name: "density", meta: "N", usage: "fog density"},
			},
		},
	}},
	{name: "occluder", cmds: []helpCmd{
		{
			sub:  "add",
			line: "  gdctl [--host host] [--port port] [--token token] occluder add --parent PATH --shape SHAPE --size X,Y,Z",
			desc: "add an OccluderInstance3D node to a parent",
			flags: []helpFlag{
				{name: "parent", meta: "PATH", usage: "parent node path"},
				{name: "shape", meta: "SHAPE", usage: "occluder shape: box, sphere, or quad"},
				{name: "size", meta: "X,Y,Z", usage: "occluder extents"},
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
