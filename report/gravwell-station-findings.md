# Gravwell Station Creation Report

Date: 2026-05-13

## Created Prototype

Built **Gravwell Station** in the connected Godot project under:

- `res://gravwell_station/scenes/station_hub.tscn`
- `res://gravwell_station/scenes/room_a.tscn`
- `res://gravwell_station/scenes/room_b.tscn`
- `res://gravwell_station/scenes/room_c.tscn`
- `res://gravwell_station/scripts/game_state.gd`
- `res://gravwell_station/scripts/gravwell_room.gd`
- `res://gravwell_station/scripts/room_data.gd`
- `res://gravwell_station/shaders/gravity_warp.gdshader`
- `res://gravwell_station/data/*.tres`

The game is a compact 3D puzzle-platformer prototype with:

- Four real scenes connected by scripted `change_scene_to_file` transitions.
- Runtime `GameState` autoload for inventory, gravity direction, room flags, and persistent puzzle state.
- Player movement, gravity direction swapping with WASD, arrow-key traversal, and interaction on `E`/space.
- Pushable mass cube, socket trigger, sealed/open door state, collectible crew-log shard, and per-room persistence.
- HUD with room name, inventory count, door state, and gravity indicator.
- World environment, directional/fill lighting, gravity field shader, and `GPUParticles3D` gravity dust.
- Three audio buses: `Music`, `Ambience`, and `SFX`, with ambience reverb.

## Verification

Smoke test passed:

```bash
GDCTL_BRIDGE_HOST=host.docker.internal GDCTL_BRIDGE_TOKEN=... \
go run ./cmd/gdctl run smoke \
  --scene res://gravwell_station/scenes/station_hub.tscn \
  --assert 'runtime.gravwell:transition_ready>=1' \
  --screenshot tmp/gravwell_station/smoke.png \
  --timeout 20s
```

Result:

```text
Smoke assert: runtime.gravwell:transition_ready>=1 ok
Smoke screenshot: tmp/gravwell_station/smoke.png (1152x648)
Smoke: PASS
```

The screenshot artifact was copied to `report/artifacts/gravwell-station-smoke.png`.

Latest runtime probe:

```json
{
  "source": "runtime.gravwell",
  "message": "smoke_ready",
  "detail": {
    "room": "hub",
    "inventory": 0,
    "cube_docked": 0,
    "door_open": 0,
    "gravity_changes": 0,
    "transition_ready": 1
  }
}
```

## Findings And Fix Status

| Finding | Status | Implemented / validated |
|---|---|---|
| 1. Autoload authoring | Fixed | Added `gdctl autoload add/remove/list`; live `autoload list --json` passed after reload. |
| 2. Custom resources | Fixed | Added `resource create --script SCRIPT`; live disposable `extends Resource` creation passed after the guard fix. |
| 3. Environment, lighting, particles | Partially fixed | Added `scene apply-blueprint` shortcuts for `world_environment`, `directional_light`, and `gpu_particles`; material/shader-param authoring remains open. |
| 4. Chatty scene mutation | Fixed for common cases | Added `scene batch --path SCENE --file OPS.json`; live batch add/set test passed with one open/save cycle. |
| 5. Smoke assertion UX | Fixed | Added split assertion flags; live `run smoke` with `--assert-source/--assert-key/--assert-op/--assert-value` passed. |
| InputMap gap | Fixed | Added `gdctl input action ...` and `gdctl input event add-key`; live add/bind/list/remove test passed. |

1. **Autoload authoring exists, but validation order is awkward.**

   `project setting set --key autoload/GameState --string '*res://...'` worked, but `script.write` validates scripts before that setting is active. A direct `GameState.foo()` reference failed during `gravwell_room.gd` upload. The workaround was to resolve the singleton dynamically with `get_node("/root/GameState")`.

   Suggested improvement: add `gdctl autoload add --name GameState --path res://...` and make `script.write` optionally validate against pending project settings or support an ordered transaction.

2. **Custom resource classes are not immediately usable.**

   `room_data.gd` with `class_name GravwellRoomData` passed `script.check`, but `resource create --type GravwellRoomData` returned `RESOURCE_TYPE_UNKNOWN`. The prototype fell back to plain `Resource` `.tres` files.

   Suggested improvement: add a class-name refresh endpoint, better diagnostics for script-class availability, or `resource create --script res://...` so custom resources can be authored without an editor reload.

3. **Environment, lighting, particles, and shader material setup are still script-heavy.**

   The CLI can create nodes and resources, but building a `WorldEnvironment`, `Environment`, `ParticleProcessMaterial`, and shader material graph through repeated `node set` / `node set-resource` calls would be very verbose. The prototype generated those runtime nodes from GDScript instead.

   Suggested improvement: add focused commands such as `environment create`, `light add`, `particles create`, and `material shader-create --shader`.

4. **Scene-scoped node mutation is reliable but chatty.**

   Creating four scenes and attaching one script worked well, including open/save job handling. Each `node add --scene` opens and saves the scene, which is safe but slow for many small edits.

   Suggested improvement: add a batched scene transaction command or extend `scene apply` to support script attachment, resources, and common 3D node presets in one save.

5. **Smoke testing is a strong feedback loop.**

   `run smoke`, runtime probes, and screenshot capture gave fast confidence that the generated game actually starts and renders. The only CLI footgun was shell quoting for assertions containing `>`.

   Suggested improvement: document assertion quoting prominently and consider accepting split flags like `--assert-source`, `--assert-key`, `--assert-op`, `--assert-value`.

## Notable Workarounds

- Used a token-free local helper script and passed `GDCTL_BRIDGE_TOKEN` only through the command environment.
- Used dynamic autoload lookup in room scripts to avoid pre-autoload validation failures.
- Used script-created runtime geometry and VFX for complex 3D setup because first-class CLI commands do not yet cover those authoring flows.
- Used plain `Resource` room data files after custom class resource creation failed.

## Next CLI Opportunities

- Done: `autoload add/remove/list`.
- Done: `input action add/remove/list` and `input event add-key`.
- Done: `resource create --script` for custom resources.
- `material create`, `material set-shader`, and shader uniform setters.
- Partial: `scene apply-blueprint --blueprint gpu_particles`; still open: `particles create` with common process-material flags.
- Done: batched scene mutation with one open/save cycle.
- Partial: added `world_environment`, `directional_light`, and `gpu_particles` blueprints; still open: richer room blueprint with environment, camera, light, collision, and HUD starter nodes.

## Follow-up Implemented

Implemented the first improvement pass from these findings:

- Added `gdctl autoload add/remove/list`.
- Added `resource create --script SCRIPT` for custom `Resource` scripts that are not registered in `ClassDB` yet.
- Added `scene batch --path SCENE --file OPS.json` to share one open/save cycle across multiple `node.add`, `node.set`, `node.attach-script`, and `node.set-resource` operations.
- Added split smoke assertions: `--assert-source`, `--assert-key`, `--assert-op`, and `--assert-value`.
- Added blueprint shortcuts for `world_environment`, `directional_light`, and `gpu_particles`.

Validation:

```text
go test ./... PASS
script check PASS for updated bridge command files
run smoke with split assertion PASS
```

Note: `gdctl addon update` pushed the bridge changes into the Godot project, but the live plugin still needs the normal disable/enable or editor restart before the new bridge endpoints appear in `bridge info`.

## Reload Test

After plugin reload, `bridge info` advertised the new autoload capabilities and `autoload list --json` returned the expected `GameState` and `GdctlRuntimeBridge` entries.

Live checks that passed:

- `autoload list --json`
- `scene batch --path res://gravwell_station/scenes/cli_batch_test.tscn --file ...`
- `scene apply-blueprint` for `world_environment`, `directional_light`, and `gpu_particles`

Live checks that found follow-up work:

- `resource create --script` rejected a valid disposable `extends Resource` script because the bridge used an over-strict `Script.can_instantiate()` guard. That guard was removed locally and `go test ./...` still passes. The addon needs another update/reload before retesting this endpoint.
- `run smoke` with split assertions returned HTTP 500 and then the bridge stopped accepting connections on `host.docker.internal:7777`. This looks like a live plugin/editor restart or bridge-server stop after smoke start. Retest once the bridge is reachable again.

After the next reload/update pass:

- `run smoke` with split assertions passed and the bridge remained reachable afterward.
- `resource create --script` passed with a disposable `extends Resource` script and produced `res://gravwell_station/data/cli_resource_script_test.tres`.

## InputMap Follow-up

Implemented the next Gravwell-driven CLI gap: project input-map authoring.

Added commands:

- `gdctl input action add --name NAME [--deadzone N]`
- `gdctl input action remove --name NAME`
- `gdctl input action list [--json] [--all]`
- `gdctl input event add-key --action ACTION --key KEY [--physical=false]`

Bridge additions:

- New `commands/input_commands.gd` handler.
- New capabilities: `input.action_add`, `input.action_remove`, `input.action_list`, `input.event_add_key`.
- Actions/events are persisted through `ProjectSettings` so they survive editor reloads.

Validation so far:

```text
go test ./... PASS
script check PASS for input_commands.gd
script check PASS for bridge_server.gd
CLI help renders for gdctl input
```

Note: `gdctl addon update` wrote the new handler into the Godot project. Because this adds a new preloaded command file, the live bridge needs another plugin reload before `bridge info` will advertise the new `input.*` capabilities.

Reload attempt:

- `bridge info` still did not advertise `input.*`.
- `input action list --json` returned `UNKNOWN_ENDPOINT`.
- Verified `res://addons/godot_tcp_bridge/commands/input_commands.gd` exists in the Godot project.
- `script check` passed for both `input_commands.gd` and `bridge_server.gd`.
- Re-ran `gdctl addon update`; it wrote 29 files and again requested plugin reload.

Conclusion: the project files are correct, but the currently running bridge instance has not loaded the updated `bridge_server.gd` preload set yet. Disable/enable the plugin again, or restart the editor, then retest `input action list --json`.

After reload, live InputMap validation passed:

- `bridge info` advertised `input.action_add`, `input.action_remove`, `input.action_list`, and `input.event_add_key`.
- `input action add --name gdctl_cli_input_test --deadzone 0.25` passed.
- `input event add-key --action gdctl_cli_input_test --key J` passed.
- `input action list` showed `gdctl_cli_input_test [J]`.
- `input action remove --name gdctl_cli_input_test` passed.
- Follow-up list check found no `gdctl_cli_input_test` entry.
- `go test ./...` still passed and the bridge remained reachable.

Remaining polish: `input action list` without `--all` is noisier than expected because Godot reports many default `ui_*` actions through project settings. The command works, but filtering should be refined if a concise project-only listing is important.
