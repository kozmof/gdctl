# gdctl

`gdctl` is a CLI for Godot 4 projects. It connects to a running Godot editor over a local TCP bridge and exposes the engine through six command layers designed for automation.

> **Note:** gdctl is under active development. Do not use it in production.

## Architecture

Commands are organized into six layers, each with a distinct purpose:

```
Object Layer     — project structure: scene, node, script, shader, resource, file
System Layer     — engine subsystems: navigation, localization, audio, animation,
                   tilemap, theme, viewport, window, accessibility, lightmap, save
Policy Layer     — organizational rules: policy validate
Workflow Layer   — desired-state and batch operations: apply, plan, diff, tx, workflow, scaffold
Execution Layer  — validation and quality: asset, lint, test, gate, perf
Recipe Layer     — node-type shortcuts: recipe <name> <verb> [--print-core]
```

## Build

```bash
go build ./cmd/gdctl
```

## Connection

The CLI connects to the Godot bridge at `http://127.0.0.1:7777` by default. Override with flags or environment variables:

```bash
GDCTL_BRIDGE_HOST=127.0.0.1
GDCTL_BRIDGE_PORT=7777
GDCTL_BRIDGE_TOKEN=<token>
```

For a Linux devcontainer with Godot running on a Windows host, use Docker Desktop's host route:

```bash
export GDCTL_BRIDGE_HOST=host.docker.internal
export GDCTL_BRIDGE_TOKEN=<token copied from the gdctl Bridge dock>

gdctl ping
```

Inside a devcontainer, `127.0.0.1` means the container itself. If `gdctl ping` fails with `connection refused`, use `host.docker.internal` instead.

## Commands

### Infrastructure

```bash
gdctl ping
gdctl doctor [--project PATH] [--fix]
gdctl help [--usecase] [topic]

gdctl addon install --project PATH [--force]
gdctl addon enable --project PATH
gdctl addon disable --project PATH
gdctl addon status [--project PATH] [--json]
gdctl addon update [--project PATH]
gdctl addon rollback --project PATH [--backup PATH]
gdctl addon remove --project PATH
gdctl addon doctor [--project PATH] [--fix]

gdctl bridge info
gdctl bridge logs [--json] [--clear]
gdctl bridge addon-update

gdctl autoload add --name NAME --path PATH
gdctl autoload remove --name NAME
gdctl autoload list [--json]

gdctl input action add --name NAME [--deadzone N]
gdctl input action remove --name NAME
gdctl input action list [--json] [--all]
gdctl input event add-key --action ACTION --key KEY [--physical=false]
gdctl input event add-joypad --action ACTION (--button N | --axis N [--axis-value V]) [--device N]
gdctl input event add-mouse-button --action ACTION --button left|right|middle

gdctl run start [--scene SCENE | --main] [--clear-logs=false]
gdctl run status [--json]
gdctl run helper-status [--json]
gdctl run stop
gdctl run logs [--json] [--clear] [--source SOURCE] [--latest] [--since-start]
gdctl run screenshot [--out FILE] [--source game|screen] [--screen N] [--viewport PATH]
gdctl run input --file input.json [--timeout DURATION] [--summary-probe SOURCE]
gdctl run wait-probe --source SOURCE (--assert KEY>=VALUE | --assert-key KEY --assert-op OP --assert-value VALUE) [--timeout DURATION] [--json]
gdctl run probe raycast [--json] [--timeout DURATION]
gdctl run probe node --path PATH --property NAME [--property NAME] [--json] [--timeout DURATION]
gdctl run instantiate --scene SCENE --parent PATH [--name NAME] [--timeout DURATION]
gdctl run scene-reload [--timeout DURATION]
gdctl run smoke [--scene SCENE | --main] [--input FILE] [--assert SOURCE:KEY>=VALUE] [--screenshot OUT] [--timeout DURATION] [--keep-running]
gdctl run profile --metric METRICS --duration DURATION
```

### Object Layer

```bash
gdctl scene create --path PATH --root TYPE --name NAME [--force]
gdctl scene open --path PATH
gdctl scene instance --parent PATH --scene SCENE --name NAME
gdctl scene tree
gdctl scene save
gdctl scene apply --path SCENE --file TREE.json [--dry-run] [--json]
gdctl scene batch --path SCENE --file OPS.json [--timeout DURATION]
gdctl scene apply-blueprint --path SCENE --blueprint NAME [--dry-run] [--timeout DURATION]
gdctl scene list [--dir res://] [--recursive]
gdctl scene run --path SCENE [--timeout DURATION]

gdctl node add --parent PATH --type TYPE --name NAME [--prop NAME=TYPED_JSON] [--dry-run] [--scene SCENE]
gdctl node remove --path PATH [--dry-run] [--scene SCENE]
gdctl node rename --path PATH --name NAME [--dry-run] [--scene SCENE]
gdctl node move --path PATH --parent PARENT [--index N] [--dry-run] [--scene SCENE]
gdctl node get --path PATH --property PROPERTY
gdctl node set --path PATH (--property PROPERTY VALUE | --position X,Y,Z | --rotation-degrees X,Y,Z | --scale X,Y,Z) [--scene SCENE]
gdctl node set-many --path PATH --file PROPS.json [--scene SCENE]
gdctl node set-resource --path PATH --property PROPERTY --resource RESOURCE
gdctl node attach-script --path PATH --script SCRIPT [--scene SCENE]
gdctl node group add --path PATH --group GROUP
gdctl node group remove --path PATH --group GROUP
gdctl node group list --path PATH
gdctl node duplicate --path PATH --name NAME [--parent PARENT] [--dry-run]
gdctl node list-properties --path PATH

gdctl script create --path PATH --extends CLASS [--force]
gdctl script write --path PATH (--body TEXT | --body-file FILE) [--allow-missing-preloads]
gdctl script check --path PATH

gdctl shader write --path PATH (--body TEXT | --body-file FILE)
gdctl shader check --path PATH

gdctl resource create --path PATH (--type TYPE | --script SCRIPT) [--prop NAME=TYPED_JSON] [--shader-param NAME=RESOURCE]
gdctl resource list [--dir res://] [--recursive] [--ext EXT]

gdctl import set --path PATH [--param NAME=VALUE]

gdctl file write-bytes --path PATH --in FILE
gdctl file lut-write --path PATH --profiles FILE
gdctl file list --path PATH [--recursive]
gdctl file mkdir --path PATH
gdctl file delete --path PATH
gdctl file exists --path PATH
```

### System Layer

```bash
gdctl navigation bake --path PATH

gdctl localization locale-set --locale LOCALE
gdctl localization string-add --key KEY --locale LOCALE --text TEXT

gdctl audio bus-add --name NAME [--if-missing]
gdctl audio bus-volume-set --name NAME --volume-db DB
gdctl audio bus-effect-add --name NAME --effect-type TYPE
gdctl audio listener-make-current --path PATH
gdctl audio playlist-add --bus BUS --stream PATH
gdctl audio playlist-autoplay --bus BUS --mode MODE

gdctl animation create --path LIBRARY --name NAME [--length N] [--loop]
gdctl animation track-add --path LIBRARY --animation NAME --node-path NODE --property PROP
gdctl animation keyframe-add --path LIBRARY --animation NAME --track-idx N --time T --value TYPED_JSON
gdctl animation length-set --path LIBRARY --animation NAME --length N
gdctl animation player-play --node-path PATH [--animation NAME]
gdctl animation tree add-state --tree PATH --name NAME --animation ANIM
gdctl animation tree add-transition --tree PATH --from STATE --to STATE --condition COND
gdctl animation tree blend-space-2d-add --tree PATH --state STATE --blend-x PARAM --blend-y PARAM
gdctl animation tree set-param --tree PATH --param PARAM (--vector2 X,Y | --float N | --bool B | --int N)

gdctl tilemap tileset-create --path PATH [--tile-width W] [--tile-height H] [--force]
gdctl tilemap source-add --path TILESET --texture TEX [--tile-width W] [--tile-height H]
gdctl tilemap cell-set --node PATH --layer N --x X --y Y --source-id ID [--atlas-x AX] [--atlas-y AY]
gdctl tilemap cell-set-rect --node PATH --layer N --x X --y Y --width W --height H --source-id ID
gdctl tilemap cell-clear --node PATH --layer N --x X --y Y

gdctl theme create --path PATH [--force]
gdctl theme set-color --path PATH --node-type TYPE --name NAME --value COLOR
gdctl theme set-font-size --path PATH --node-type TYPE --name NAME --value N
gdctl theme set-constant --path PATH --node-type TYPE --name NAME --value N

gdctl viewport screenshot --out FILE [--kind 2d|3d] [--index N]
gdctl viewport set-size --width W --height H [--path NODE_PATH]
gdctl viewport add --width W --height H [--parent PATH] [--add-camera]
gdctl viewport camera-assign --viewport PATH --camera PATH

gdctl window create [--title TITLE] [--width W] [--height H] [--position X,Y]
gdctl window assign-viewport --window-id ID --viewport PATH

gdctl accessibility tts-speak --text TEXT [--interrupt]
gdctl accessibility tts-configure [--pitch N] [--rate N] [--voice VOICE]
gdctl accessibility tts-stop

gdctl lightmap bake --path PATH

gdctl save
```

### Policy Layer

```bash
# Flags before positional file argument (standard convention for all layered commands)
gdctl policy validate --dir res:// [--json] POLICY.json
```

Policy file format (JSON):

```json
{
  "textures": { "allowed_formats": ["png", "webp"] },
  "scripts":  {},
  "scenes":   { "max_node_count": 500 },
  "assets":   { "max_file_size_mb": 10 }
}
```

**Bridge requirement:** `policy validate` requires a live Godot bridge for all checks. It is intended for the inner development loop (devcontainer with Godot open), not for headless CI runners that have no Godot editor running.

| Rule | Implemented | How |
|---|---|---|
| `textures.allowed_formats` | Yes | `file list` via bridge |
| `scripts.check_all` | Yes | `script check` per file via bridge |
| `scenes.max_node_count` | Skipped | Requires opening each scene |
| `assets.max_file_size_mb` | Skipped | File size not exposed by bridge |

### Workflow Layer

```bash
# Apply a JSON scene tree as desired state
gdctl apply --scene SCENE [--dry-run] [--json] TREE.json

# Preview what apply would change without writing
gdctl plan --scene SCENE [--json] TREE.json

# Compare current scene tree against a desired-state file (node-level)
gdctl diff --scene SCENE [--json] TREE.json

# Batch operations (same format as scene batch; --scene maps to --path)
gdctl tx run --scene SCENE --file OPS.json

# Run a named workflow from a JSON file
gdctl workflow run --file WORKFLOWS.json [--continue-on-error] [--json] NAME

# Scaffold project structure from templates
gdctl scaffold player  --out res://scenes/player.tscn [--name NAME]
gdctl scaffold scene   --out res://scenes/world.tscn  [--root Node3D] [--name NAME]
gdctl scaffold autoload --out res://autoloads/game.gd [--name NAME]
gdctl scaffold test    --out res://tests/test_foo.gd  [--subject NAME]
```

Workflow file format:

```json
{
  "ci":      ["lint", "gate run quick"],
  "nightly": ["asset scan --dir res://", "lint", "test gdscript --dir res://tests"]
}
```

### Execution Layer

```bash
# Enumerate project files by category
gdctl asset scan [--dir res://] [--json]

# Check that autoload paths exist
gdctl asset missing [--json]

# Check all scripts and shaders for errors
gdctl lint [--dir res://] [--json]

# Run GDScript tests
gdctl test [gdscript] (--path PATH | --dir DIR) [--timeout DURATION] [--json]

# Unified validation gate
gdctl gate run [--json] PROFILE   # profiles: quick, ci, production

# Performance profiling (requires running game)
gdctl perf [--metric fps,draw_calls] [--duration 5s] [--timeout 120s]
```

### Recipe Layer

Recipes are convenience shortcuts for common node patterns. All recipes support `--print-core` to display the underlying object-layer commands without executing.

```bash
gdctl recipe fog-volume add --parent PATH [--shape box|ellipsoid|cone|cylinder|world] [--size X,Y,Z] [--density N] [--print-core]
gdctl recipe decal add --parent PATH [--texture TEX] [--size X,Y,Z] [--print-core]
gdctl recipe decal set-normal-fade --path PATH --fade N
gdctl recipe occluder add --parent PATH [--shape box|sphere|quad] [--size X,Y,Z] [--print-core]
gdctl recipe voxelgi bake --path PATH
gdctl recipe reflection-probe bake --path PATH
gdctl recipe csg node-add --parent PATH --type TYPE --name NAME [--no-collision] [--print-core]
gdctl recipe csg operation-set --path PATH --operation union|intersection|subtraction
gdctl recipe csg size-set --path PATH --size X,Y,Z
gdctl recipe lod set --path PATH --begin N --end N
gdctl recipe lod set-many --file FILE
gdctl recipe softbody pin-point --path PATH --point N
gdctl recipe softbody unpin-point --path PATH --point N
gdctl recipe terrain heightmap-import --path PATH --texture TEX [--min-height N] [--max-height N]
gdctl recipe graph-edit node-add --path GRAPH --name NAME [--position X,Y]
gdctl recipe graph-edit connection-add --graph GRAPH --from NODE --from-port N --to NODE --to-port N
gdctl recipe graph-edit node-remove --path GRAPH --name NAME
gdctl recipe environment set-background --path PATH --mode color|sky|clear [--color R,G,B[,A]]
gdctl recipe character-body add --parent PATH [--name NAME] [--print-core]
gdctl recipe area-trigger add --parent PATH [--name NAME] [--print-core]
gdctl recipe camera-3d add --parent PATH [--name NAME] [--current] [--print-core]
gdctl recipe camera-2d add --parent PATH [--name NAME] [--enabled] [--print-core]
gdctl recipe light-3d add --parent PATH [--name NAME] [--type directional|omni|spot] [--print-core]
gdctl recipe light-2d add --parent PATH [--name NAME] [--type directional|point] [--print-core]
gdctl recipe ui-button add --parent PATH [--name NAME] [--text TEXT] [--print-core]
```

`--print-core` shows what the recipe does without running it:

```bash
$ gdctl recipe character-body add --parent /root/World --name Player --print-core
gdctl node add --parent /root/World --type CharacterBody3D --name Player
gdctl node add --parent /root/World/Player --type CollisionShape3D --name CollisionShape3D
```

Multi-word commands also accept dotted aliases for quick interactive use:

```bash
gdctl file.mkdir --path res://scenes
gdctl project.setting.get --key application/run/main_scene
```

---

## Key Workflows

### Desired-State Scene Management

`scene apply` and the top-level `apply` command build or update a node tree from a JSON descriptor. They open the scene, apply the tree, and save:

```bash
gdctl apply --scene res://scenes/Main.tscn game.json
gdctl plan  --scene res://scenes/Main.tscn game.json   # preview only
gdctl diff  --scene res://scenes/Main.tscn game.json   # show structural changes
```

The JSON format uses `name`, `type`, optional `properties`, and optional `children`:

```json
{
  "root": {
    "path": "/root/Main",
    "children": [
      {
        "name": "Player",
        "type": "CharacterBody3D",
        "properties": {
          "position": { "kind": "Vector3", "value": [0, 1, 0] }
        }
      }
    ]
  }
}
```

Repeated grids can be expressed compactly; the bridge expands them before applying:

```json
{
  "grid": {
    "name_prefix": "Tile",
    "type": "StaticBody3D",
    "count_x": 10,
    "count_z": 10,
    "origin": [0, 0, 0],
    "step_x": [2, 0, 0],
    "step_z": [0, 0, 2]
  }
}
```

### Policy Validation

Define organizational rules in a policy file and enforce them in CI:

```bash
gdctl policy validate --dir res:// --json policy.json
```

Exits with code 1 if any violations are found. Use in CI to enforce texture formats, script health, and asset limits.

### Workflow Automation

Chain commands into named workflows:

```json
{
  "ci": ["lint", "gate run --json quick"],
  "release-check": ["asset scan --dir res://", "lint", "policy validate --dir res:// policy.json"]
}
```

```bash
gdctl workflow run --file gdctl-workflows.json --json ci
```

### Gate Profiles

`gate run` runs a validation sequence and reports a single pass/fail:

| Profile | Checks |
|---|---|
| `quick` | Bridge reachable |
| `ci` | Bridge + GDScript tests |
| `production` | Bridge + GDScript tests |

```bash
gdctl gate run --json production
```

### Run/Debug Workflow

`gdctl` can start, inspect, inject input into, and screenshot a running game through the bridge:

```bash
gdctl run start --scene res://scenes/Main.tscn
gdctl run status
gdctl run logs
gdctl run input --file ./smoke-input.json
gdctl run screenshot --out ./frame.png
gdctl run stop
```

Input scripts inject key/mouse events into the running game:

```json
{
  "steps": [
    {"type": "key", "key": "W", "action": "press"},
    {"type": "wait", "ms": 500},
    {"type": "mouse_motion", "relative": [180, -120]},
    {"type": "key", "key": "W", "action": "release"}
  ]
}
```

`run smoke` is a one-shot automated test — start, inject input, assert probe, screenshot, stop:

```bash
gdctl run smoke \
  --scene res://scenes/Main.tscn \
  --input ./smoke-input.json \
  --assert runtime.game:targets_disabled>=1 \
  --screenshot ./smoke.png \
  --timeout 30s
```

Game scripts can emit structured probes through the `GdctlRuntimeBridge` autoload:

```gdscript
var gdctl := get_node_or_null("/root/GdctlRuntimeBridge")
if gdctl:
    gdctl.probe("game", "level_loaded", {"level": current_level, "enemies": enemy_count})
```

Read probes with `run wait-probe` or `run probe node`.

### Property Values

Complex values use typed JSON at the CLI boundary:

```bash
gdctl node set --path /root/Main/Player --property position --vector3 0,1,0
gdctl node set --path /root/Main/Label  --property text --string "Ready"
gdctl node set --path /root/Main/Drone  --property route --array-vector3 "0,1,2;3,4,5"
gdctl node set --path /root/Main/Player --position 0,1,0
gdctl node set --path /root/Main/Player --rotation-degrees 0,45,0
```

Full typed JSON for complex or composite types:

```json
{"kind": "Vector3",  "value": [0, 1, 0]}
{"kind": "Color",    "value": [1, 0, 0, 1]}
{"kind": "Resource", "value": "res://materials/floor.tres"}
{"kind": "Array[Vector3]", "value": [[0,1,2],[3,4,5]]}
```

---

## GDScript Tests

`test gdscript` runs editor-side tests without launching the game. Test scripts extend the bridge's `test_case.gd` and use `test_*` methods:

```gdscript
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

func test_addition() -> void:
    assert_eq(1 + 1, 2)
```

```bash
gdctl test gdscript --dir res://tests --json
gdctl test gdscript --path res://tests/test_player.gd
```

Assertion helpers: `assert_true`, `assert_false`, `assert_eq`, `assert_ne`, `fail`.

---

## Godot Addon

The CLI embeds `addons/godot_tcp_bridge` and can install or update it without a mounted project:

```bash
gdctl addon install --project ./my-game
gdctl addon update                          # via bridge, no project mount needed
gdctl addon rollback --project ./my-game   # revert to last backup
```

After enabling the plugin, Godot shows a **gdctl Bridge** dock. Copy the token from there:

```bash
export GDCTL_BRIDGE_TOKEN=<token>
gdctl bridge info
gdctl addon status
```

Reload the plugin after every `addon update`.

---

## Current Bridge Capabilities

Run `gdctl bridge info` to see exactly what the running addon exposes. As of addon v0.2.0:

```text
scene.create  scene.open  scene.instance  scene.tree  scene.save  scene.apply
scene.apply.blueprint  scene.list  jobs.get
node.add  node.remove  node.rename  node.move  node.get  node.set  node.set_many
node.set_resource  node.attach_script  node.group_add  node.group_remove
node.group_list  node.duplicate  node.list_properties
script.check  script.create  script.write
shader.check  shader.write
resource.create  resource.list
file.list  file.mkdir  file.delete  file.exists  file.write_bytes
navigation.bake  signal.connect  signal.disconnect
project.setting_get  project.setting_set
autoload.add  autoload.remove  autoload.list
input.action_add  input.action_remove  input.action_list  input.event_add_key
run.start  run.status  run.stop  run.logs  run.screenshot  run.input
run.probe.raycast  run.probe.node  run.instantiate  run.scene-reload  test.gdscript
theme.create  theme.set-color  theme.set-font-size  theme.set-constant
animation.create  animation.track-add  animation.keyframe-add  animation.length-set  animation.player-play
tilemap.tileset-create  tilemap.source-add  tilemap.cell-set  tilemap.cell-clear
audio.bus-add  audio.bus-volume-set  audio.bus-effect-add  audio.listener-make-current
viewport.screenshot  viewport.set-size  viewport.add  viewport.camera-assign
addon.update  bridge.logs  import.set
```

Recipe commands for `window`, `graph-edit`, `softbody`, `lod`, `terrain`, `lightmap`, `voxelgi`, `reflection-probe`, `accessibility`, `csg`, `decal`, `fog-volume`, and `occluder` require a newer addon. Run `gdctl addon update` then reload the plugin.

---

## Security

The default `127.0.0.1` bind is local-only. Recommended setup:

```text
Keep godot_tcp_bridge/host at 127.0.0.1 (default).
Keep auth_enabled on.
Allow Godot on Private networks only (Windows Firewall).
Rotate the token from the gdctl Bridge dock if exposed.
Disable the plugin when not actively using gdctl.
```

For devcontainer access, use `host.docker.internal` on the CLI side and leave the addon at `127.0.0.1:7777`. Only set the addon host to `0.0.0.0` as a last resort when Docker Desktop's routing cannot reach the default address.
