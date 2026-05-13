# gdctl Dev Iteration Flow

This document records the current proven workflow for developing `gdctl` from a Linux devcontainer while Godot runs on a Windows host.

## 1. Network Model

Default secure setup:

```text
Godot addon bind host: 127.0.0.1
Godot addon port: 7777
gdctl default host: 127.0.0.1
```

Inside a devcontainer, `127.0.0.1` means the container itself. To reach Godot running on the Windows host, connect through Docker Desktop's host route:

```bash
export GDCTL_BRIDGE_HOST=host.docker.internal
```

## 2. Token Flow

After enabling the Godot addon, open the **gdctl Bridge** dock in Godot and copy the export command:

```bash
export GDCTL_BRIDGE_TOKEN=<token>
```

For devcontainer use:

```bash
export GDCTL_BRIDGE_HOST=host.docker.internal
export GDCTL_BRIDGE_TOKEN=<token copied from Godot>
```

Then verify:

```bash
go run ./cmd/gdctl bridge info
go run ./cmd/gdctl addon status
go run ./cmd/gdctl addon doctor
```

## 3. Projectless Addon Updates

Once the running addon supports `addon.update`, update it from the devcontainer without a mounted Godot project:

```bash
go run ./cmd/gdctl addon update
```

Equivalent command:

```bash
go run ./cmd/gdctl bridge addon-update
```

Expected output:

```text
Addon updated over bridge: 4 files written
Backup: res://addons/.godot_tcp_bridge_backup/<timestamp>/
Reload required: disable/enable the Godot plugin or restart Godot
```

After an addon update, disable/enable the plugin in Godot or restart Godot.

If the command returns:

```text
UNKNOWN_ENDPOINT: Unknown bridge endpoint
```

the running addon is too old for projectless update. Manually update/install the addon once, reload it, then projectless updates should work afterward.

Note: check gdctl_manifest.json is also modified before running a command.

## 4. Scene Path Smoke Test

Open or create a scene in Godot, then inspect it:

```bash
go run ./cmd/gdctl scene tree
```

Raw logical paths should look like:

```text
/root/Node3D
/root/Node3D/GridMap
```

Test an authenticated mutation without changing the scene:

```bash
go run ./cmd/gdctl node add \
  --parent /root/Node3D \
  --type Node3D \
  --name GdctlSmoke \
  --dry-run
```

Expected output:

```text
Dry run ok: /root/Node3D/GdctlSmoke
```

Scene save now runs through a deferred bridge job:

```bash
go run ./cmd/gdctl scene save
```

If Godot crashes or restarts during save work, treat it as a regression in the async save path and follow the crash recovery notes below.

## 5. Common Failure Modes

`127.0.0.1:7777: connect: connection refused`

```text
You are probably inside the devcontainer. Use GDCTL_BRIDGE_HOST=host.docker.internal.
```

`UNAUTHORIZED: Mutation endpoint requires bearer token`

```text
Set GDCTL_BRIDGE_TOKEN from the gdctl Bridge dock, or pass --token explicitly.
```

`NO_SCENE_OPEN: No edited scene is open`

```text
Open or create a scene in Godot before using scene tree or node mutation commands.
```

`UNKNOWN_ENDPOINT: Unknown bridge endpoint`

```text
The running addon is older than the CLI command. Manually update/reload the addon once.
```

## 6. Godot Crash Case: Scene Save

Status:

```text
gdctl scene save is enabled through a deferred bridge job.
```

What happened:

```text
Calling Godot editor scene-save APIs from the bridge request handler crashed/restarted Godot.
Both direct save_scene() and attempted save-as approaches proved unsafe in this bridge context.
```

Current safer behavior:

```text
The CLI queues gdctl scene save and polls /jobs/<id>.
The bridge performs the save from its editor process loop, not directly in the TCP request handler.
Only already-saved open scenes are supported for now; save-as remains unsupported.
```

Recovery steps after a crash:

```text
1. Restart Godot if needed.
2. Check gdctl bridge logs for bridge.job entries.
3. Enable the plugin.
4. Verify the bridge:
   go run ./cmd/gdctl bridge info
5. If async save still fails, save scenes manually in Godot for that session.
```

Do not retry save implementation by directly calling these from the request handler:

```text
EditorInterface.save_scene()
EditorInterface.save_scene_as(...)
PackedScene.pack(root) + ResourceSaver.save(...)
```

Future scene save directions:

```text
Keep deferred editor-main-thread execution with explicit state/result polling.
Use an EditorScript or Godot command-line batch script outside the live editor plugin.
Create scene/resource files through controlled file/resource operations instead of saving the active editor scene.
Add save-as only after already-saved scenes remain stable.
```

Acceptance criteria for revisiting scene save:

```text
It must not crash/restart Godot.
It must return structured success/error responses.
It must work for already-saved scenes before supporting save-as.
It must be tested manually on a disposable project before updating the normal addon.
```

## 7. Current Bridge Structure

The bridge bootstrap and projectless update loop are working. Logical scene paths are working.
The addon server has started moving reusable logic out of `bridge_server.gd`. Typed CLI/Godot value conversion now lives in `addons/godot_tcp_bridge/typed_values.gd`, command handlers now live under `addons/godot_tcp_bridge/commands/`, bridge self-update logic now lives in `addons/godot_tcp_bridge/addon_update.gd`, request/response protocol helpers now live in `addons/godot_tcp_bridge/protocol.gd`, and async job processing now lives in `addons/godot_tcp_bridge/jobs.gd`.

Command request validation is centralized in:

```text
addons/godot_tcp_bridge/commands/request.gd
```

New command handlers should use that helper for JSON body parsing, authorization, operation checks, and params extraction.

Implemented bridge primitives include:

```text
scene create/open/instance/tree/save
node add/remove/set/get/rename/move/attach-script
script create/write/check
shader write/check
material write
node set-resource
file write-bytes
lut write
viewport screenshot
bridge logs
```

Use typed JSON values:

```bash
gdctl node set --path /root/Main/Player --property position --value '{"kind":"Vector2","value":[200,400]}'
gdctl node get --path /root/Main/Player --property position
gdctl node rename --path /root/Main/Player --name PlayerCar
gdctl node move --path /root/Main/PlayerCar --parent /root/Main/Track
```

Scene saving now uses a deferred/editor-process job. Save-as is still intentionally unsupported.

## 8. Shader Whole-File Period

Shader authoring starts with whole-file operations:

```bash
gdctl shader write --path res://shaders/edge_mix_3d.gdshader --body-file examples/edge_mix_3d.gdshader
gdctl shader check --path res://shaders/edge_mix_3d.gdshader
gdctl material write --path res://materials/edge_mix.tres --shader res://shaders/edge_mix_3d.gdshader --texture-param edge_lut=res://textures/edge_lut.png
gdctl node set-resource --path /root/Main3D/Player/Body --property material --resource res://materials/edge_mix.tres
gdctl lut write --path res://textures/edge_lut.png --profiles examples/edge_profiles.json
```

This intentionally avoids partial shader or material parameter mutation at first. Rewriting the whole `.gdshader` file is easier to reason about, easier to diff, and matches the current bridge pattern for scripts.

`lut write` generates a 256x1 PNG locally from JSON edge profiles and uploads it through `file write-bytes`. This keeps Edge ID + LUT authoring data-driven while reusing one general binary file primitive.

## 9. Current Self-Update Caveat

After extracting `addon_update.gd`, the currently running Godot addon may need a manual file refresh if `/addon/update` returns:

```text
HTTP 500 {}
```

That means the live in-memory self-update path is failing before it can install the next addon package. The bridge can still respond to `bridge info`, `scene tree`, and node commands, but the addon files must be copied/installed manually once so the fixed local addon code becomes active again. After reloading that version, verify:

```bash
gdctl bridge info
gdctl addon update
gdctl node rename --path /root/Node3D/Temp --name TempRenamed --dry-run
gdctl node move --path /root/Node3D/Temp --parent /root/Node3D --dry-run
```

## 10. Bridge Diagnostics

The addon keeps a small in-memory diagnostic buffer for bridge activity. Use it when a command returns an unclear error or when Godot reports a GDScript issue:

```bash
gdctl bridge logs
gdctl bridge logs --json
gdctl bridge logs --clear
```

Logs are token-protected and reset when the plugin/editor restarts. They record bridge startup/shutdown, request paths, response status, and structured bridge errors. Full Godot parser/runtime errors are not always catchable from inside GDScript, but these logs narrow down which endpoint and command path ran before a failure.

## 11. Adding New Commands

This section documents the proven workflow for adding a new CLI command end-to-end.

### Files to touch

| Layer | File | What to add |
|---|---|---|
| GDScript handler | `addons/godot_tcp_bridge/commands/<category>_commands.gd` | `handle_<op>` function |
| Bridge wiring | `addons/godot_tcp_bridge/bridge_server.gd` | `const`, `var`, route `if`, capability string |
| Addon manifest | `addons/godot_tcp_bridge/gdctl_manifest.json` | filename in `files` array (new files only) |
| Go types | `internal/bridge/types.go` | result struct |
| Go client | `internal/bridge/client.go` | method calling `postEnvelope` |
| Go CLI | `internal/cli/cli.go` | router case, handler function, `printUsage` line |
| Go tests | `internal/cli/cli_test.go` | happy-path test + validation test |

For a new category (e.g. `signal`, `project`) also create a new `*_commands.gd` file.

### Deployment workflow

The key constraint: `addon update` rewrites `bridge_server.gd`, which preloads all command files. If a new command file has a syntax error and you reload the plugin, the bridge dies and you can no longer use `addon update` or `script check` to recover — you must manually copy files into the Godot project.

The two-stage workflow avoids this:

**Stage A — syntax validation (bridge stays on old code)**

Write each new `.gd` file directly to its `res://` path using `script write`. This command runs Godot's parser on the body before writing, so a syntax error returns immediately without touching `bridge_server.gd`.

```bash
go run ./cmd/gdctl script write \
  --path res://addons/godot_tcp_bridge/commands/signal_commands.gd \
  --body-file addons/godot_tcp_bridge/commands/signal_commands.gd

go run ./cmd/gdctl script write \
  --path res://addons/godot_tcp_bridge/commands/project_commands.gd \
  --body-file addons/godot_tcp_bridge/commands/project_commands.gd
```

If `SCRIPT_SYNTAX_INVALID` is returned, fix the file locally and retry. The bridge is still running with the old plugin; nothing is broken.

**Stage B — full deploy (syntax already verified)**

```bash
go run ./cmd/gdctl addon update
# Reload the plugin in Godot (disable/enable in plugin manager)
go run ./cmd/gdctl bridge info
# Verify new capability strings appear in the capabilities list
```

**Stage C — smoke test**

Run each new command against the open scene to confirm end-to-end behaviour. Use `scene tree` first to find valid node paths:

```bash
go run ./cmd/gdctl scene tree
go run ./cmd/gdctl node group add --path /root/Main/Player --group enemies
go run ./cmd/gdctl node group list --path /root/Main/Player
go run ./cmd/gdctl node group remove --path /root/Main/Player --group enemies
```

If a command returns `UNKNOWN_ENDPOINT`, the running addon is older than the CLI. Go back to Stage B.

### Go unit tests

Before Stage A, verify all Go tests pass against mock servers:

```bash
go test ./...
```

Each new command needs at minimum:
- A happy-path test: mock the HTTP endpoint, verify the correct `op` and params are sent, check stdout
- A validation test: omit a required flag and confirm the error fires before any network call (use a bare `[]string{...}` with no server, not `serverArgs`)

### Recovery if the bridge breaks

If the plugin fails to reload (syntax error reached `bridge_server.gd`):

1. Fix the offending `.gd` file locally.
2. Manually copy the fixed file and `bridge_server.gd` into the Godot project at `res://addons/godot_tcp_bridge/`.
3. Reload the plugin.
4. Verify: `go run ./cmd/gdctl bridge info`
5. Once the bridge is back, continue from Stage B.
