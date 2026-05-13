# gdctl Godot TCP Bridge

`gdctl` is a prototype CLI for talking to a Godot 4 editor plugin over local HTTP from a Linux devcontainer.

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
gdctl doctor [--project ./my-game] [--fix]
gdctl addon install --project ./my-game
gdctl addon enable --project ./my-game
gdctl addon status [--project ./my-game] [--json]
gdctl addon update [--project ./my-game]
gdctl addon rollback --project ./my-game [--backup PATH]
gdctl addon disable --project ./my-game
gdctl addon remove --project ./my-game
gdctl addon doctor [--project ./my-game] [--fix]
gdctl bridge info
gdctl bridge logs
gdctl bridge logs --json
gdctl bridge logs --clear
gdctl run start --scene res://scenes/Main.tscn
gdctl run start --main
gdctl run status
gdctl run logs
gdctl run logs --json
gdctl run logs --clear
gdctl run screenshot --out ./run.png
gdctl run screenshot --source screen --out ./host-screen.png
gdctl run input --file ./input.json
gdctl run stop
gdctl scene create --path res://scenes/Main.tscn --root Node2D --name Main
gdctl scene open --path res://scenes/Main.tscn
gdctl scene apply --path res://scenes/Main.tscn --file ./tree.json
gdctl scene instance --parent /root/Main --scene res://scenes/PlayerCar.tscn --name PlayerCar
gdctl scene tree
gdctl scene save
gdctl node add --parent /root/Main --type Node2D --name EnemySpawner
gdctl node rename --path /root/Main/EnemySpawner --name SpawnPoint
gdctl node move --path /root/Main/SpawnPoint --parent /root/Main/Track
gdctl node set --path /root/Main/Track/SpawnPoint --property position --value '{"kind":"Vector2","value":[200,400]}'
gdctl node get --path /root/Main/Track/SpawnPoint --property position
gdctl node remove --path /root/Main/Track/SpawnPoint
gdctl script create --path res://scripts/player_car.gd --extends CharacterBody2D
gdctl script write --path res://scripts/player_car.gd --body-file ./player_car.gd
gdctl script check --path res://scripts/player_car.gd
gdctl node attach-script --path /root/PlayerCar --script res://scripts/player_car.gd
gdctl node attach-script --scene res://scenes/PlayerCar.tscn --path /root/PlayerCar --script res://scripts/player_car.gd
gdctl shader write --path res://shaders/edge_mix_3d.gdshader --body-file ./edge_mix_3d.gdshader
gdctl shader check --path res://shaders/edge_mix_3d.gdshader
gdctl material write --path res://materials/edge_mix.tres --shader res://shaders/edge_mix_3d.gdshader
gdctl material write --path res://materials/edge_mix.tres --shader res://shaders/edge_mix_3d.gdshader --texture-param edge_lut=res://textures/edge_lut.png
gdctl node set-resource --path /root/Main/Body --property material --resource res://materials/edge_mix.tres
gdctl file write-bytes --path res://textures/raw.bin --in ./raw.bin
gdctl lut write --path res://textures/edge_lut.png --profiles ./edge_profiles.json
gdctl viewport screenshot --out ./status.png
```

Multi-word commands can also be written with a dotted alias for quick interactive use:

```bash
gdctl file.mkdir --path res://scenes
gdctl project.setting.get --key application/run/main_scene
```

## Current Run/Debug Workflow

`gdctl` can ask the already-open Godot editor to run a scene through the bridge. This is useful in devcontainers where no container-visible headless Godot binary is configured:

```bash
gdctl run start --scene res://signal_harbor/scenes/SignalHarborMain.tscn
gdctl run status
gdctl run logs
gdctl run input --file ./smoke-input.json
gdctl run screenshot --out ./signal-harbor.png
gdctl run stop
```

`run start` clears run logs by default. Use `--clear-logs=false` to preserve prior entries. `run logs` returns bridge-captured run/error messages plus game-side helper logs. Use `run logs --clear` to read and then clear the run log buffer.

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

`run screenshot` captures the running game viewport by default. It uses a small gdctl runtime helper autoload that `run start` registers before launching the scene. Use `--source screen` for the legacy whole-host-screen capture.

Scene node paths are logical paths rooted at the edited scene root:

```text
/root/Main
/root/Main/Player
/root/Main/Camera
```

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

Mutation endpoints require the bearer token generated by the Godot addon at `res://.godot-bridge-token`, unless `godot_tcp_bridge/auth_enabled` is disabled in project settings.
Bridge logs are token-protected too because they may include request paths and diagnostics.

If a bad addon update prevents the bridge from starting, use the filesystem rollback path from a project checkout:

```bash
gdctl addon rollback --project ./my-game
gdctl addon rollback --project ./my-game --backup ./my-game/addons/.godot_tcp_bridge_backup/20260510T214304Z
```

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
gdctl project setting set --key display/window/size/viewport_width --int 1280
gdctl project setting set --key application/run/main_scene --string res://scenes/Main.tscn
```

Array shorthand always uses `;` between elements. Vector arrays use commas only inside each vector element, for example `--array-vector3 "0,1,2;3,4,5"`.

For GDScript typed arrays, prefer typed locals before passing array literals, or use plain `Array` storage with explicit element casts at use sites. `script check` and `script write` include a hint when Godot reports a typed-array mismatch.

`node add` can set initial properties while creating the node:

```bash
gdctl node add --parent /root/Main --type Node3D --name SpawnPoint \
  --prop 'position={"kind":"Vector3","value":[0,1,0]}'
```

## Current Scene Workflow

`gdctl` can now create, open, mutate, inspect, and save `.tscn` scenes over the running bridge:

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

gdctl scene create \
  --path res://gdctl_tmp/Child.tscn \
  --root Node2D \
  --name Child \
  --force

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

Save-as is still intentionally unsupported:

```bash
gdctl scene save --path res://other.tscn
```

Use `scene create --force` when you explicitly want to replace an existing scene file.

## Current Script Workflow

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

`script create` writes a minimal script like:

```gdscript
extends Node2D
```

`script write` syntax-checks the provided body with Godot before writing it. If Godot rejects the script, the command fails with `SCRIPT_SYNTAX_INVALID` and includes Godot's diagnostic, line number, and nearby source context when the running engine exposes those details.
`node attach-script` syntax-checks the target script again before loading and attaching it to the node, and reports the same diagnostics on failure. By default it mutates the currently open editor scene; with `--scene`, it opens that scene, attaches the script, and saves it.

## Current Shader Workflow

`gdctl` can write and check whole `.gdshader` files over the running bridge:

```bash
gdctl shader write \
  --path res://shaders/edge_mix_3d.gdshader \
  --body-file ./edge_mix_3d.gdshader

gdctl shader check --path res://shaders/edge_mix_3d.gdshader
```

Shader updates are whole-file rewrites. This keeps shader authoring deterministic and avoids partial resource mutation while the shader/material pipeline is still young.

Shader materials can also be generated as whole resources:

```bash
gdctl material write \
  --path res://materials/edge_mix.tres \
  --shader res://shaders/edge_mix_3d.gdshader \
  --texture-param edge_lut=res://textures/edge_lut.png

gdctl node set-resource \
  --path /root/Main3D/Player/Body \
  --property material \
  --resource res://materials/edge_mix.tres
```

`node set-resource` loads a `res://` resource and assigns it to a node property, which is enough to connect a generated `ShaderMaterial` to visible 3D geometry.

Edge ID LUTs are generated from JSON profile data as 256x1 PNG files:

```bash
gdctl lut write \
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

`file write-bytes` is the lower-level binary upload primitive used by `lut write`.

## Current Visual Check Workflow

`gdctl` can capture the Godot editor viewport and write the PNG in the CLI environment:

```bash
gdctl viewport screenshot --out ./status.png
```

By default this captures the 2D editor viewport. For a 3D viewport:

```bash
gdctl viewport screenshot --kind 3d --index 0 --out ./status-3d.png
```

The bridge captures PNG bytes from the editor viewport and returns them as base64 in a deferred job result. The CLI decodes those bytes and writes the local `--out` file. This keeps screenshots projectless and avoids writing temporary files into `res://`.

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

Addon command handlers live under:

```text
addons/godot_tcp_bridge/commands/
```

`commands/request.gd` centralizes JSON body parsing, token authorization, operation-name checks, and params extraction for command handlers. Keep new endpoint bodies on that helper path so command scripts stay small and return consistent bridge errors.

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
signal.connect
signal.disconnect
project.setting_get
project.setting_set
node.duplicate
node.list_properties
file.list
file.mkdir
file.delete
file.exists
navigation.bake
script.check
script.create
script.write
shader.check
shader.write
resource.create
file.write_bytes
viewport.screenshot
addon.update
bridge.logs
import.set
scene.list
resource.list
run.start
run.status
run.stop
run.logs
run.logs.clear
run.screenshot
run.input
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
