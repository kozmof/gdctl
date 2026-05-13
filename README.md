# gdctl Godot TCP Bridge

`gdctl` is a CLI for talking to a Godot 4 editor plugin over local HTTP from a Linux devcontainer.

## Build

```bash
go build ./cmd/gdctl
```

## CLI Defaults

The CLI connects to:

```text
http://127.0.0.1:7777
```

Override with flags or environment variables:

```bash
GDCTL_BRIDGE_HOST=127.0.0.1
GDCTL_BRIDGE_PORT=7777
GDCTL_BRIDGE_TOKEN=<token>
```

For Linux devcontainer access to Godot running on a Windows host, keep the addon on its default `127.0.0.1:7777` first and connect from the container with Docker Desktop's host route:

```bash
GDCTL_BRIDGE_HOST=host.docker.internal gdctl ping
```

Inside a devcontainer, `127.0.0.1` means the container itself, not the Windows host. If Godot says the bridge is `listening on 127.0.0.1:7777` but `gdctl ping` fails with `127.0.0.1:7777: connect: connection refused`, use:

```bash
export GDCTL_BRIDGE_HOST=host.docker.internal
export GDCTL_BRIDGE_TOKEN=<token copied from the gdctl Bridge dock>

gdctl ping
gdctl addon status
gdctl addon update
```

## Commands

```bash
gdctl ping
gdctl doctor [--project PATH] [--fix]
gdctl help [topic]

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

gdctl run start [--scene SCENE | --main] [--clear-logs=false]
gdctl run status
gdctl run stop
gdctl run logs [--json] [--clear] [--source SOURCE] [--latest] [--since-start]
gdctl run screenshot [--out FILE] [--source game|screen] [--screen N]
gdctl run input --file input.json [--timeout DURATION] [--summary-probe SOURCE]
gdctl run wait-probe --source SOURCE --assert KEY OP VALUE [--timeout DURATION] [--json]
gdctl run probe raycast [--json] [--timeout DURATION]
gdctl run smoke [--scene SCENE | --main] [--input FILE] [--assert SOURCE:KEY>=VALUE] [--screenshot OUT] [--timeout DURATION] [--keep-running]

gdctl scene create --path PATH --root TYPE --name NAME [--force]
gdctl scene open --path PATH
gdctl scene instance --parent PATH --scene SCENE --name NAME
gdctl scene tree
gdctl scene save
gdctl scene apply --path SCENE --file TREE.json [--dry-run]
gdctl scene apply-blueprint --path SCENE --blueprint NAME [--dry-run] [--timeout DURATION]
gdctl scene list [--dir res://] [--recursive]
gdctl scene run --path SCENE [--timeout DURATION]

gdctl node add --parent PATH --type TYPE --name NAME [--prop NAME=TYPED_JSON] [--dry-run] [--scene SCENE] [--timeout DURATION]
gdctl node remove --path PATH [--dry-run] [--scene SCENE] [--timeout DURATION]
gdctl node rename --path PATH --name NAME [--dry-run] [--scene SCENE] [--timeout DURATION]
gdctl node move --path PATH --parent PARENT [--index N] [--dry-run] [--scene SCENE] [--timeout DURATION]
gdctl node get --path PATH --property PROPERTY
gdctl node set --path PATH (--property PROPERTY VALUE | --position X,Y,Z | --rotation-degrees X,Y,Z | --scale X,Y,Z) [--scene SCENE] [--timeout DURATION]
gdctl node set-resource --path PATH --property PROPERTY --resource RESOURCE
gdctl node attach-script --path PATH --script SCRIPT [--scene SCENE] [--timeout DURATION]
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

gdctl resource create --path PATH --type TYPE [--prop NAME=TYPED_JSON] [--shader-param NAME=RESOURCE]
gdctl resource list [--dir res://] [--recursive] [--ext EXT]

gdctl import set --path PATH [--param NAME=VALUE]

gdctl file write-bytes --path PATH --in FILE
gdctl file lut-write --path PATH --profiles FILE
gdctl file list --path PATH [--recursive]
gdctl file mkdir --path PATH
gdctl file delete --path PATH
gdctl file exists --path PATH

gdctl navigation bake --path PATH

gdctl signal connect --from PATH --signal NAME --to PATH --method METHOD
gdctl signal disconnect --from PATH --signal NAME --to PATH --method METHOD

gdctl project setting get --key KEY
gdctl project setting set --key KEY (--value TYPED_JSON | --string S | --int N | --float N | --bool BOOL | --vector2 X,Y | --vector3 X,Y,Z | --color R,G,B[,A] | --resource PATH | --array-vector3 A;B)
gdctl project run [--scene SCENE] [--timeout DURATION]

gdctl viewport screenshot --out FILE [--kind 2d|3d] [--index N]
gdctl viewport set-size --width W --height H [--path NODE_PATH]
gdctl viewport add --width W --height H [--parent PATH] [--add-camera]

gdctl theme create --path PATH [--force]
gdctl theme set-color --path PATH --node-type TYPE --name NAME --value COLOR
gdctl theme set-font-size --path PATH --node-type TYPE --name NAME --value N
gdctl theme set-constant --path PATH --node-type TYPE --name NAME --value N

gdctl animation create --path LIBRARY --name NAME [--length N] [--loop]
gdctl animation track-add --path LIBRARY --animation NAME --node-path NODE --property PROP
gdctl animation keyframe-add --path LIBRARY --animation NAME --track-idx N --time T --value TYPED_JSON
gdctl animation length-set --path LIBRARY --animation NAME --length N
gdctl animation player-play --node-path PATH [--animation NAME]

gdctl tilemap tileset-create --path PATH [--tile-width W] [--tile-height H] [--force]
gdctl tilemap source-add --path TILESET --texture TEX [--tile-width W] [--tile-height H]
gdctl tilemap cell-set --node PATH --layer N --x X --y Y --source-id ID [--atlas-x AX] [--atlas-y AY]
gdctl tilemap cell-clear --node PATH --layer N --x X --y Y

gdctl audio bus-add --name NAME
gdctl audio bus-volume-set --name NAME --volume-db DB
gdctl audio bus-effect-add --name NAME --effect-type TYPE
```

Multi-word commands can also be written with a dotted alias for quick interactive use:

```bash
gdctl file.mkdir --path res://scenes
gdctl project.setting.get --key application/run/main_scene
```

## Run/Debug Workflow

`gdctl` asks the already-open Godot editor to run a scene through the bridge. This is useful in devcontainers where no container-visible headless Godot binary is configured:

```bash
gdctl run start --scene res://signal_harbor/scenes/SignalHarborMain.tscn
gdctl run status
gdctl run logs
gdctl run input --file ./smoke-input.json
gdctl run screenshot --out ./signal-harbor.png
gdctl run stop
```

`run start` clears run logs by default. Use `--clear-logs=false` to preserve prior entries. `run logs` returns bridge-captured run/error messages plus game-side helper logs. Use `run logs --clear` to read and then clear the run log buffer. Filter logs with `--source SOURCE`, `--latest` (most-recent entry per source), or `--since-start` (exclude entries before the current run start).

Game scripts can write standardized CLI-readable probes through the helper autoload that `run start` installs. Use `probe`, `info`, `warn`, `error`, or the generic `log_event(level, source, message, detail)`:

```gdscript
var gdctl := get_node_or_null("/root/GdctlRuntimeBridge")
if gdctl:
	gdctl.probe("foo", "ready", {
		"player_position": player.global_position,
		"parcels": total_parcels,
		"moving_platforms": moving_platforms.size(),
	})
```

The helper prefixes custom sources with `runtime.`, so the example appears in `gdctl run logs` as `runtime.foo`.

`run input` plays short input scripts into the running game through the runtime helper:

```json
{
  "steps": [
    {"type": "key", "key": "W", "action": "press"},
    {"type": "wait", "ms": 500},
    {"type": "key", "key": "W", "action": "release"},
    {"type": "mouse_button", "button": "left", "action": "tap", "duration_ms": 200}
  ]
}
```

`run screenshot` captures the running game viewport by default. Use `--source screen` for whole-host-screen capture, and `--screen N` to select the display index.

`run wait-probe` polls run logs until a probe field satisfies a predicate or timeout fires:

```bash
gdctl run wait-probe --source runtime.game --assert targets_disabled>=1 --timeout 30s
```

`run probe raycast` fires a center-screen ray in the running 3D game and reports the hit collider, position, and normal. Requires `GdctlRuntimeBridge` autoload and an active `Camera3D`.

`run smoke` is a one-shot automated test: start, optionally inject input, wait for a probe assertion, capture a screenshot, then stop:

```bash
gdctl run smoke \
  --scene res://scenes/Main.tscn \
  --input ./smoke-input.json \
  --assert runtime.game:targets_disabled>=1 \
  --screenshot ./smoke.png \
  --timeout 30s
```

Exits with code 0 on pass, 1 on failure. Prints `Smoke: PASS` or `Smoke: FAIL — <reason>`.

## JSON Scene Trees

`scene apply` can build a nested node tree in one scene-scoped operation. It opens the scene, applies the JSON, and saves the scene:

```bash
gdctl scene apply --path res://scenes/Main.tscn --file ./tree.json
```

Each node uses `name`, `type`, optional `properties`, and optional `children`. Properties use the same typed JSON shape as `node set`, and `Resource` values can be inline:

```json
{
  "root": {
    "path": "/root/Main",
    "children": [
      {
        "name": "StartRoof",
        "type": "StaticBody3D",
        "properties": {
          "position": { "kind": "Vector3", "value": [0, 0, 0] }
        },
        "children": [
          {
            "name": "Mesh",
            "type": "MeshInstance3D",
            "properties": {
              "mesh": {
                "kind": "Resource",
                "value": {
                  "type": "BoxMesh",
                  "properties": {
                    "size": { "kind": "Vector3", "value": [9, 0.6, 7] }
                  }
                }
              }
            }
          }
        ]
      }
    ]
  }
}
```

Repeated 3D grids can be expressed with a narrow `grid` child spec. The bridge expands it into normal child nodes before applying:

```json
{
  "grid": {
    "name_prefix": "PaintTile",
    "type": "StaticBody3D",
    "count_x": 11,
    "count_z": 11,
    "origin": [-10, -0.05, -10],
    "step_x": [2, 0, 0],
    "step_z": [0, 0, 2],
    "name_format": "%s_%03d"
  },
  "children": []
}
```

`scene apply-blueprint` applies a named pre-built node tree to a scene:

```bash
gdctl scene apply-blueprint --path res://scenes/Main.tscn --blueprint player3d
```

Available blueprints: `player3d`, `spotlight`, `trigger_area`, `hud_label`.

`scene list` lists `.tscn` files in the project:

```bash
gdctl scene list [--dir res://scenes] [--recursive]
```

`scene run` runs a scene with headless Godot (requires `GDCTL_GODOT_PATH` or `--godot`):

```bash
gdctl scene run --path res://scenes/Main.tscn --timeout 30s
```

## Scene Workflow

`gdctl` can create, open, mutate, inspect, and save `.tscn` scenes over the running bridge:

```bash
gdctl scene create \
  --path res://gdctl_tmp/SmokeScene.tscn \
  --root Node2D \
  --name SmokeScene \
  --force

gdctl scene open --path res://gdctl_tmp/SmokeScene.tscn
gdctl scene tree

gdctl node add \
  --parent /root/SmokeScene \
  --type Node2D \
  --name OpenSmokeChild

gdctl scene instance \
  --parent /root/SmokeScene \
  --scene res://gdctl_tmp/Child.tscn \
  --name Child

gdctl scene tree
gdctl scene save
```

Expected result:

```text
SmokeScene Node2D
├── OpenSmokeChild Node2D
└── Child Node2D
```

`scene create` writes a scene file directly with `PackedScene` and `ResourceSaver`.
`scene instance` loads a saved `.tscn`, instantiates it under the active scene, and marks the active scene dirty.
`scene open` and `scene save` run as deferred bridge jobs and the CLI polls `/jobs/<id>` until the editor-side operation finishes.

Save-as is intentionally unsupported:

```bash
gdctl scene save --path res://other.tscn  # not supported
```

Use `scene create --force` when you explicitly want to replace an existing scene file.

Scene node paths are logical paths rooted at the edited scene root:

```text
/root/Main
/root/Main/Player
/root/Main/Camera
```

## Node Operations

`node add` can set initial properties while creating the node:

```bash
gdctl node add --parent /root/Main --type Node3D --name SpawnPoint \
  --prop 'position={"kind":"Vector3","value":[0,1,0]}'
```

`node move` accepts `--index N` to place the node at a specific child position under the new parent (-1 = append).

`node group` manages Godot groups on a node:

```bash
gdctl node group add --path /root/Main/Player --group enemies
gdctl node group list --path /root/Main/Player
gdctl node group remove --path /root/Main/Player --group enemies
```

`node duplicate` copies a node with a new name, optionally under a different parent:

```bash
gdctl node duplicate --path /root/Main/Player --name Player2 --parent /root/Main
```

`node list-properties` lists all exported properties on a node:

```bash
gdctl node list-properties --path /root/Main/Player
```

## Property Values

Property values use typed JSON at the CLI boundary for complex values:

```json
{"kind":"String","value":"Player"}
{"kind":"bool","value":true}
{"kind":"int","value":3}
{"kind":"float","value":1.5}
{"kind":"Vector2","value":[200,400]}
{"kind":"Vector3","value":[0,1,0]}
{"kind":"Color","value":[1,0,0,1]}
{"kind":"Array[Vector3]","value":[[0,1,2],[3,4,5]]}
{"kind":"PackedVector2Array","value":[[0,0],[1200,0],[1200,800],[0,800]]}
```

For common scalar/vector/resource values, `node set` and `project setting set` also accept shorthand flags:

```bash
gdctl node set --path /root/Main/Player --property position --vector3 0,1,0
gdctl node set --path /root/Main/Label --property text --string "Ready"
gdctl node set --path /root/Main/Drone --property route --array-vector3 "0,1,2;3,4,5"
gdctl node set --path /root/Main/Switches --property active_flags --array-bool "true;false"
gdctl node set --path /root/Main/Player --position 0,1,0
gdctl node set --path /root/Main/Player --rotation-degrees 0,45,0
gdctl node set --path /root/Main/Player --scale 1,1,1
gdctl project setting set --key display/window/size/viewport_width --int 1280
gdctl project setting set --key application/run/main_scene --string res://scenes/Main.tscn
```

Array shorthand always uses `;` between elements. Vector arrays use commas only inside each vector element, for example `--array-vector3 "0,1,2;3,4,5"`.

`--position`, `--rotation-degrees`, and `--scale` are transform shorthands that cannot be combined with `--property`.

## Script Workflow

`gdctl` can create, write, and syntax-check GDScript files over the running bridge:

```bash
gdctl script create \
  --path res://gdctl_tmp/smoke_script.gd \
  --extends Node2D \
  --force

gdctl script check --path res://gdctl_tmp/smoke_script.gd

gdctl script write \
  --path res://gdctl_tmp/smoke_script.gd \
  --body-file ./smoke_script.gd

gdctl node attach-script \
  --path /root/SmokeScene \
  --script res://gdctl_tmp/smoke_script.gd

gdctl node attach-script \
  --scene res://gdctl_tmp/SmokeScene.tscn \
  --path /root/SmokeScene \
  --script res://gdctl_tmp/smoke_script.gd
```

`script write` syntax-checks the provided body with Godot before writing it. If Godot rejects the script, the command fails with `SCRIPT_SYNTAX_INVALID` and includes Godot's diagnostic, line number, and nearby source context when the running engine exposes those details.

Use `--allow-missing-preloads` during iterative authoring when preloaded scenes will be created shortly after. `node attach-script` syntax-checks the target script again before attaching it.

## Shader Workflow

```bash
gdctl shader write \
  --path res://shaders/edge_mix_3d.gdshader \
  --body-file ./edge_mix_3d.gdshader

gdctl shader check --path res://shaders/edge_mix_3d.gdshader
```

Shader updates are whole-file rewrites. Shader materials can also be generated as whole resources:

```bash
gdctl resource create \
  --path res://materials/edge_mix.tres \
  --type ShaderMaterial \
  --prop 'shader={"kind":"Resource","value":"res://shaders/edge_mix_3d.gdshader"}' \
  --shader-param edge_lut=res://textures/edge_lut.png

gdctl node set-resource \
  --path /root/Main3D/Player/Body \
  --property material \
  --resource res://materials/edge_mix.tres
```

Edge ID LUTs are generated from JSON profile data as 256x1 PNG files:

```bash
gdctl file lut-write \
  --path res://textures/edge_lut.png \
  --profiles ./edge_profiles.json
```

The profile fields pack into PNG channels:

```text
R = mix
G = blur
B = width
A = mode
```

`file write-bytes` is the lower-level binary upload primitive used by `file lut-write`.

## Resource and File Operations

`resource create` creates a Godot resource file:

```bash
gdctl resource create --path res://materials/floor.tres --type StandardMaterial3D \
  --prop 'albedo_color={"kind":"Color","value":[0.8,0.8,0.8,1]}'
gdctl resource list [--dir res://materials] [--recursive] [--ext .tres]
```

`file` commands manage the res:// filesystem:

```bash
gdctl file list --path res://scenes [--recursive]
gdctl file mkdir --path res://levels/world1
gdctl file delete --path res://temp/scratch.gd
gdctl file exists --path res://scenes/Main.tscn
gdctl file write-bytes --path res://textures/raw.bin --in ./raw.bin
```

`import set` configures import parameters for an asset:

```bash
gdctl import set --path res://textures/player.png --param compress/mode=0
```

## Signals and Project Settings

Connect and disconnect signals between nodes:

```bash
gdctl signal connect --from /root/Main/Player --signal hit --to /root/Main/HUD --method on_player_hit
gdctl signal disconnect --from /root/Main/Player --signal hit --to /root/Main/HUD --method on_player_hit
```

Read and write project settings:

```bash
gdctl project setting get --key display/window/size/viewport_width
gdctl project setting set --key display/window/size/viewport_width --int 1920
gdctl project setting set --key application/run/main_scene --string res://scenes/Main.tscn
```

## Navigation

Bake a navigation mesh on a `NavigationRegion` node:

```bash
gdctl navigation bake --path /root/Main/NavigationRegion3D
```

## Visual Check Workflow

Capture the Godot editor viewport:

```bash
gdctl viewport screenshot --out ./status.png
gdctl viewport screenshot --kind 3d --index 0 --out ./status-3d.png
```

Resize the main window or a SubViewport node:

```bash
gdctl viewport set-size --width 1920 --height 1080
gdctl viewport set-size --width 320 --height 240 --path /root/Main/SubViewport
```

Add a SubViewport node to the current scene:

```bash
gdctl viewport add --width 320 --height 240 --parent /root/Main --add-camera
```

## Theme

Create and configure UI theme resources:

```bash
gdctl theme create --path res://themes/main.tres
gdctl theme set-color --path res://themes/main.tres --node-type Label --name font_color --value 1,1,1,1
gdctl theme set-font-size --path res://themes/main.tres --node-type Label --name font_size --value 24
gdctl theme set-constant --path res://themes/main.tres --node-type Button --name margin_top --value 8
```

## Animation

Create and populate `AnimationLibrary` resources:

```bash
gdctl animation create --path res://animations/player.tres --name walk --length 1.0 --loop
gdctl animation track-add --path res://animations/player.tres --animation walk \
  --node-path Player --property position
gdctl animation keyframe-add --path res://animations/player.tres --animation walk \
  --track-idx 0 --time 0.0 --value '{"kind":"Vector3","value":[0,0,0]}'
gdctl animation keyframe-add --path res://animations/player.tres --animation walk \
  --track-idx 0 --time 1.0 --value '{"kind":"Vector3","value":[1,0,0]}'
gdctl animation length-set --path res://animations/player.tres --animation walk --length 2.0
gdctl animation player-play --node-path /root/Main/AnimationPlayer --animation walk
```

## TileMap

Create TileSets and paint cells:

```bash
gdctl tilemap tileset-create --path res://tilesets/world.tres --tile-width 16 --tile-height 16
gdctl tilemap source-add --path res://tilesets/world.tres --texture res://textures/tiles.png
gdctl tilemap cell-set --node /root/Main/TileMap --layer 0 --x 3 --y 5 --source-id 0 --atlas-x 1 --atlas-y 0
gdctl tilemap cell-clear --node /root/Main/TileMap --layer 0 --x 3 --y 5
```

## Audio

Manage audio buses:

```bash
gdctl audio bus-add --name Music
gdctl audio bus-volume-set --name Music --volume-db -6.0
gdctl audio bus-effect-add --name Music --effect-type AudioEffectReverb
```

## Godot Addon

The CLI embeds `addons/godot_tcp_bridge` and can install or update it inside a Godot 4 project:

```bash
gdctl addon install --project ./my-game
gdctl addon enable --project ./my-game
gdctl addon status --project ./my-game
```

The addon listens on `127.0.0.1:7777` by default. This is the safest local-only setup.

For devcontainer use, first keep the default addon host and connect from the container through `host.docker.internal`. If Docker Desktop cannot reach the bridge that way, change the Godot project setting `godot_tcp_bridge/host` to `0.0.0.0` as a fallback.

After enabling the plugin, Godot shows a **gdctl Bridge** dock. Use it to copy the mutation token or the `GDCTL_BRIDGE_TOKEN` export command when the Godot project folder is not mounted inside the devcontainer.

With a token configured, the devcontainer can inspect and update the running project's addon over the bridge without a mounted project path:

```bash
gdctl bridge info
gdctl addon status
gdctl bridge addon-update
# Equivalent when --project is omitted:
gdctl addon update
```

Reload the plugin in Godot after a bridge addon update.

If a bad addon update prevents the bridge from starting, use the filesystem rollback path from a project checkout:

```bash
gdctl addon rollback --project ./my-game
gdctl addon rollback --project ./my-game --backup ./my-game/addons/.godot_tcp_bridge_backup/20260510T214304Z
```

Addon command handlers live under:

```text
addons/godot_tcp_bridge/commands/
```

`commands/request.gd` centralizes JSON body parsing, token authorization, operation-name checks, and params extraction for command handlers.

## Current Bridge Capabilities

The current addon advertises these runtime capabilities through `gdctl bridge info`:

```text
ping
scene.create
scene.open
scene.instance
scene.tree
scene.save
scene.apply
scene.list
scene.apply.blueprint
jobs.get
node.add
node.remove
node.rename
node.move
node.get
node.set
node.set_resource
node.attach_script
node.group_add
node.group_remove
node.group_list
node.duplicate
node.list_properties
signal.connect
signal.disconnect
project.setting_get
project.setting_set
file.list
file.mkdir
file.delete
file.exists
file.write_bytes
navigation.bake
script.check
script.create
script.write
shader.check
shader.write
resource.create
resource.list
viewport.screenshot
viewport.set-size
viewport.add
addon.update
bridge.logs
import.set
run.start
run.status
run.stop
run.logs
run.logs.clear
run.screenshot
run.input
run.probe.raycast
theme.create
theme.set-color
theme.set-font-size
theme.set-constant
animation.create
animation.track-add
animation.keyframe-add
animation.length-set
animation.player-play
tilemap.tileset-create
tilemap.source-add
tilemap.cell-set
tilemap.cell-clear
audio.bus-add
audio.bus-volume-set
audio.bus-effect-add
```

Most commands are projectless once the addon is running: the CLI talks over TCP and does not need the Godot project mounted in the devcontainer. Filesystem addon install/enable/remove still need `--project` because they edit a local project directory directly.

## Security Notes

The default `127.0.0.1` bind is local-only. Binding the addon to `0.0.0.0` is a devcontainer fallback, but Windows may make the bridge reachable from other machines on networks where Godot is allowed through the firewall.

Recommended Windows setup:

```text
Allow Godot only on Private networks.
Do not allow Godot on Public networks.
Do not port-forward or otherwise expose port 7777 to the Internet.
Keep godot_tcp_bridge/auth_enabled enabled.
Rotate the token from the gdctl Bridge dock if it is exposed.
Disable the plugin when you are not using gdctl.
```

For native local use, keep the default `127.0.0.1` bind and connect with:

```bash
GDCTL_BRIDGE_HOST=127.0.0.1 gdctl ping
```
