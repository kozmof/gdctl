# Gravwell Station — CLI Prototyping Findings

**Date:** 2026-05-14  
**Game:** Gravwell Station (3D puzzle-platformer, gravity anomalies)  
**Scenes built:** `station_hub.tscn`, `room_a.tscn`, `room_b.tscn`, `player.tscn`, `hud.tscn`  
**Smoke result:** All three room scenes pass (`Smoke: PASS`)  
**Visual verification:** Passed — movement, scene transition, and inverted gravity all confirmed with screenshots

---

## Summary

Five sessions were executed end-to-end: environment setup, player character, gravity mechanic + particles, persistence/autoloads, and polish (HUD, audio, shaders). The build surfaced confirmed friction in five areas and one positive surprise where friction was predicted but did not materialize.

---

## Session 1 — Environment & Rooms

### What worked
- `scene apply` with deep nested JSON built full room geometry (walls, floors, ceilings, trigger areas) in a single round-trip. 25–36 nodes created in one call per room.
- `scene apply-blueprint --blueprint world_environment` and `--blueprint directional_light` applied cleanly.

### Friction: `node set --scene` round-trips per property
Every `node set --property X ... --scene SCENE` call independently opens and saves the scene. Setting three properties on DirectionalLight3D produced three open/save cycles. For batching property writes to an offline scene, `node set-many` exists but requires the scene to be open; there is no `node set-many --scene` flag. This compounds badly with multi-property configuration of particles and other nodes.

**Gap:** Add `--scene` flag to `node set-many`, or a new `node configure --scene SCENE --file PROPS.json` command that opens, applies many properties across multiple nodes, and saves in one job.

### Friction: `scene apply-blueprint` to a non-open scene does not persist
When `scene apply-blueprint` is called on a scene that is not currently open in the editor, the CLI reports success but the node does not appear in the scene file afterward (`NODE_NOT_FOUND` on the next command). The blueprint was applied to an ephemeral in-memory open without saving.

**Gap:** `scene apply-blueprint` should either require the scene to be open (and error if not), or open → apply → save automatically, mirroring how `scene apply` behaves.

### Observation: Duplicate blueprint nodes
Calling `scene apply-blueprint --blueprint world_environment` on a scene that already has a `WorldEnvironment` node adds a second one with a generated name (`@WorldEnvironment@25307`). No deduplication or replace flag exists.

**Gap:** Add `--replace` or `--if-missing` flag to `scene apply-blueprint`.

---

## Session 2 — Player Character & AnimationTree

### What worked
- `node add --type AnimationTree` added the node cleanly.
- `node set --property anim_player --node-path ../AnimationPlayer` wired the AnimationTree to the AnimationPlayer via NodePath — the `--node-path` shorthand flag worked as expected.
- `node set --property tree_root --value '{"kind":"Resource","value":{"type":"AnimationNodeStateMachine","properties":{}}}'` set an inline AnimationNodeStateMachine as tree_root. The `--value` typed JSON path handles inline resources where `--resource` only accepts file paths.
- `animation create/track-add/keyframe-add` built an AnimationLibrary with idle/walk/fall/push states.

### Friction: AnimationTree state machine population has no CLI path
After setting `tree_root` to an `AnimationNodeStateMachine`, there is no way to add `AnimationNodeAnimation` states, connect transitions, or set transition conditions through the CLI. The entire state machine topology must be driven from GDScript at runtime. The `animation` commands operate on `AnimationLibrary` resources only — they do not touch `AnimationTree` graph topology.

**Gap:** New commands needed:
```
gdctl animation tree add-state --tree PATH --name NAME --animation ANIM_NAME
gdctl animation tree add-transition --tree PATH --from STATE --to STATE [--condition PARAM]
gdctl animation tree set-param --tree PATH --param NAME --value TYPED_JSON
```

### Friction: `--resource` flag does not accept inline resources
`node set --property X --resource '{"type":"AnimationNodeStateMachine",...}'` fails with `VALUE_INVALID: Resource value must be a res:// path`. Inline resource creation requires `--value` with typed JSON, but only a subset of kinds are supported there (no `Dictionary`, no `AABB`).

**Gap:** Either extend `--value` to cover all Godot variant kinds, or add an `--inline-resource TYPE` flag that accepts a properties JSON.

### Friction: `node set-resource` does not accept `--scene`
While `node set` supports `--scene SCENE` for offline scenes, `node set-resource` does not. It always operates on the currently open scene. This means mixing `node set` and `node set-resource` calls on an offline scene requires opening the scene between each pair of commands.

**Gap:** Add `--scene` flag to `node set-resource` for consistency.

---

## Session 3 — Gravity Mechanic & GPUParticles3D

### What worked
- `scene apply-blueprint --blueprint gpu_particles` applied when the scene was already open.
- `resource create --type ParticleProcessMaterial` created the particle material with initial properties in one call.
- `node set-resource --property process_material --resource ...` linked the material cleanly.

### Friction: `ParticleProcessMaterial` property iteration requires many round-trips
There is no `gdctl particle` or `gdctl particles` command group. Configuring a `GPUParticles3D` node fully (amount, lifetime, direction, velocity, emission shape, color gradient, etc.) requires a separate `node set --scene` call per property, each causing an open/save cycle. For 8–10 particle properties this is 8–10 round-trips instead of one.

**Gap:** New `gdctl particles configure --path NODE --scene SCENE --file PROPS.json` command, or extend `node set-many` with `--scene` support (see Session 1 gap).

### Friction: `AABB` is not a supported `--value` kind
Attempting `node set --property visibility_aabb --value '{"kind":"AABB","value":...}'` fails with `VALUE_INVALID: Unsupported value kind: aabb`. The GPUParticles3D `visibility_aabb` property cannot be set from the CLI.

**Gap:** Add `AABB` to the supported typed JSON kinds, with shape `{"kind":"AABB","value":{"position":[x,y,z],"size":[w,h,d]}}`.

### Observation: `gravity_direction` is a built-in `Area3D` property
Attempting `@export var gravity_direction := Vector3.DOWN` in a script extending `Area3D` raises `Member redefined (original in native class 'Area3D')`. The Godot syntax checker treats this as a hard error. The CLI's `script write` surfaces Godot's diagnostic precisely, which made the root cause immediately obvious.

**Positive:** Error reporting from `script write` is excellent — includes line number, surrounding context, and Godot's full diagnostic message.

---

## Session 4 — Persistence & GameState Autoload

### What worked
- `gdctl autoload add --name GameState --path PATH` and `gdctl autoload list` worked without any issues. The plan predicted this would be a friction point ("no autoload add/remove command"), but the commands exist and work correctly.
- `gdctl autoload list` confirmed registration alongside `GdctlRuntimeBridge`.

### Friction: Newly added autoloads are not recognized by `script check` / `node attach-script`
After `autoload add --name GameState`, scripts referencing `GameState` as a global fail syntax check (`Compile Error: Identifier not found: GameState`). The Godot script parser does not pick up autoloads added mid-session without a project reload. `node attach-script` runs a syntax check and therefore also blocks attachment.

**Workarounds tried:**
1. `--allow-missing-preloads` — does not suppress autoload errors, only preload errors.
2. Use `get_node_or_null("/root/GameState")` at runtime — eliminates compile-time dependency and works cleanly.

**Gap:** Either add `--allow-missing-autoloads` flag to `script write` and `node attach-script`, or document that autoloads require a project reload to take effect in the syntax checker.

### Friction: GDScript Variant type inference treated as error
`var result := JSON.parse_string(...)` causes `Parse Error: The variable type is being inferred from a Variant value, so it will be typed as Variant. (Warning treated as error.)` The project has warnings-as-errors enabled. The fix is explicit `var result: Variant = ...`, but this is a non-obvious GDScript 4 strictness trap. The CLI error output made the issue immediately diagnosable.

**Observation:** The strict type-checking environment means scripts require more explicit typing than minimal GDScript examples show. This is a project configuration concern, not a CLI gap.

---

## Session 5 — HUD, Audio, Shaders

### What worked
- `theme create`, `theme set-color`, `theme set-font-size`, `theme set-constant` built a complete HUD theme cleanly.
- `audio bus-add --if-missing`, `bus-volume-set`, `bus-effect-add` set up the 3-bus audio system (Music / Ambience / SFX) correctly.
- `shader write` + `shader check` wrote and validated the gravity-warp GLSL shader in one round-trip.
- **Positive surprise:** Shader uniform parameters are settable via dotted sub-property syntax: `node set --property "material_override:shader_parameter/warp_strength" --float 0.5`. The plan predicted "shader uniform set at runtime not yet CLI-accessible" — this is already working in the current CLI.

### Friction: Intermittent `audio bus-volume-set` timeouts
Two of three `audio bus-volume-set` calls timed out on the first attempt (Music and Ambience) and succeeded on retry. The SFX call succeeded first try. No pattern was identifiable; likely a transient editor-thread contention issue when multiple bus operations follow quickly.

**Gap:** The audio bus commands may need a small internal retry or should warn on timeout to retry rather than silently failing.

### Friction: `scene instance` requires scene to be open
`scene instance --parent PATH --scene SCENE --name NAME` fails with `NODE_PARENT_NOT_FOUND` if the target scene is not currently open in the editor. The command does not auto-open the target scene. This is inconsistent with `node add --scene SCENE` which opens and saves automatically.

**Gap:** `scene instance` should accept `--scene SCENE` to specify the target scene, or should auto-open the scene if not already open, mirroring `node add`.

### Observation: `run smoke --assert` format
`--assert runtime.player:ready>=1` fails with "must be in KEY>=VALUE form". The correct form requires a numeric value: e.g., `--assert "runtime.player:helper_present>=1"`. Probe field names that contain boolean-like values need numeric coercion. Non-numeric probe values (strings, booleans) cannot be asserted via `--assert`.

**Gap:** Add `--assert-exists KEY` flag to check that a probe field exists regardless of value, or support `--assert KEY==STRING` for string comparison.

---

---

## Visual Verification Results (input + screenshot)

A live input/screenshot session confirmed or corrected the following after the build:

### Bug found: nested CharacterBody3D from blueprint
`scene apply-blueprint --blueprint player3d` added a `Player CharacterBody3D` child under an existing `Player CharacterBody3D` root — the `CollisionShape3D` and `Camera3D` landed on the inner node. The root had no collision, causing the player to fall through the floor at Y=-4028 on first run. Fixed by moving `CollisionShape3D` and `Camera3D` to the root with `node move`.

**New gap:** `scene apply-blueprint` does not merge into an existing root of matching type — it always adds a new subtree. This makes it unsafe to use as an "add components to root" pattern.

### Bug found: `node attach-script` clears exported property values
When `node attach-script` re-attaches a script to a node (e.g., after fixing a syntax error), any `@export` properties previously set with `node set` are silently cleared. `target_scene` on `DoorToRoomA` was wiped when the fixed `scene_transition.gd` was re-attached, which caused the door trigger to do nothing at runtime.

**New gap (P1):** `node attach-script` should preserve existing exported property values when re-attaching. Alternatively, add a `--preserve-props` flag. The current behavior causes silent data loss that is very difficult to diagnose without a runtime probe.

### What worked correctly (visually confirmed)

| Test | Result |
|---|---|
| Player spawns on floor, collision working | ✓ — Y=0.25 on floor |
| WASD movement + camera follow | ✓ — player moves in all directions |
| Scene transition (hub → room A via door trigger) | ✓ — after re-setting `target_scene` |
| Gravity zone fires `body_entered`, calls `set_gravity` | ✓ — probe confirms `direction:[0,1,0]` |
| Player rises to ceiling under inverted gravity | ✓ — Y went from 0.25 to 2.94 (ceiling at Y=5) |
| Gravity warp shader sphere renders visually | ✓ — teal translucent sphere visible in Room A |
| Push block (RigidBody3D) rendered in Room A | ✓ — visible in screenshots |
| HUD bar renders with correct labels | ✓ — "STATION HUB", "GRAVITY: DOWN", "ITEMS: 0" |

### What did not work correctly (requires further work)

| Issue | Root cause |
|---|---|
| HUD gravity label stays "GRAVITY: DOWN" after zone entry | `set_gravity()` in player.gd doesn't emit a signal; HUD has no poll — needs a signal or `GameState` relay |
| HUD room label says "STATION HUB" in Room A when launched directly | `GameState.visit_room()` only called via `scene_transition.gd`; direct scene launch skips it |

---

## Gap Priority Matrix

| Gap | Impact | Effort est. | Priority | Status |
|---|---|---|---|---|
| `node set-many --scene` (batch offline property writes) | High — every multi-property configure is N round-trips | Low | **P1** | Open |
| `animation tree` commands (state machine topology) | High — AnimationTree is completely opaque to CLI | High | **P1** | Open |
| `scene apply-blueprint` auto-save for non-open scenes | High — silent data loss | Low | **P1** | **Fixed** |
| `node attach-script` clears exported property values on re-attach | **High** — silent data loss, hard to diagnose | Low | **P1** | **Fixed** |
| `scene instance --scene` flag (offline instancing) | Medium — workaround is `scene open` first | Low | **P2** | Open |
| `node set-resource --scene` flag (consistency) | Medium — forces open/close cycles | Low | **P2** | **Fixed** |
| `AABB` typed JSON kind support | Medium — blocks visibility_aabb and similar | Low | **P2** | **Fixed** |
| `--allow-missing-autoloads` for script check | Medium — autoloads added mid-session break attach | Medium | **P2** | **Fixed** |
| `scene apply-blueprint` adds subtree instead of merging into root | Medium — wrong pattern when root type matches | Low | **P2** | Open |
| `scene apply-blueprint --replace / --if-missing` | Low — easy to clean up manually | Low | **P3** | Open |
| `--assert-exists KEY` or string assert in smoke | Low — workaround is numeric probe fields | Low | **P3** | Open |
| Audio bus-volume-set retry / timeout warning | Low — transient, retries succeed | Low | **P3** | Open |

---

## Positive Findings (Better Than Expected)

| Feature | Status |
|---|---|
| `autoload add/remove/list` | **Exists** — plan said it didn't |
| Shader uniform set via `node set --property "mat:shader_parameter/X"` | **Works** — plan said it didn't |
| `node set --node-path` for NodePath properties | **Works cleanly** |
| `script write` error diagnostics | **Excellent** — line + context + Godot message |
| `scene apply` deep nested JSON | **Fast** — 36 nodes in one call |
| `resource create` with initial properties | **Works** for most property types |
| `run smoke` / `run probe` / `run logs` | **Solid** throughout all sessions |

---

---

## CLI Improvements Implemented

**Date:** 2026-05-14  
**All changes built, deployed via `gdctl bridge addon-update`, and verified against a live Godot 4.6.2 engine.**

### Fix: `node attach-script` preserves exported property values on re-attach (P1)

**Files:** `addons/godot_tcp_bridge/commands/node_commands.gd`

Before calling `node.set_script(script)`, the handler now snapshots all `PROPERTY_USAGE_SCRIPT_VARIABLE` values from the existing script. After `set_script()`, any property that exists on the new script under the same name is restored.

**Verified:** `DoorToRoomA.target_scene = "res://gravwell_station/scenes/room_a.tscn"` survived re-attachment of `scene_transition.gd`. Previously it was silently cleared to `""`.

### Fix: `scene apply-blueprint` auto-opens and saves non-open scenes (P1)

**Files:** `internal/cli/cli_media_commands.go`

`runSceneApplyBlueprint` now calls `openSceneAndWait` → `ApplyBlueprint` → `saveSceneAndWait`, wrapping the operation in `sceneMu` for concurrency safety. A `--timeout` flag was also added (default 5 s).

**Verified:** Called `scene apply-blueprint --path res://gravwell_station/scenes/room_b.tscn --blueprint spotlight` while `station_hub.tscn` was open. `room_b.tscn` was automatically opened, the `SpotLight3D` node was applied, and the file was saved — confirmed by reading the scene tree after the call.

### Fix: `node set-resource --scene` flag (P2)

**Files:** `internal/cli/cli_scene_node.go`

`runNodeSetResource` now accepts `--scene SCENE` and `--timeout DURATION`, mirroring the pattern in `node set`. When `--scene` is provided, it opens the scene before the resource assignment and saves after.

**Verified:** Set `GravParticles.process_material` on `room_b.tscn` from the command line while `room_b` was not the active scene. Scene opened, resource linked, scene saved — all in one command.

### Fix: `AABB` typed JSON kind (P2)

**Files:** `internal/cli/cli_values.go`, `addons/godot_tcp_bridge/typed_values.gd`

**CLI:** Added `--aabb px,py,pz,sx,sy,sz` shorthand flag to `typedValueFlags`. Parses six comma-separated floats and emits `{"kind":"AABB","value":{"position":[x,y,z],"size":[w,h,d]}}`.

**Bridge:** Added `"aabb"` to the `decode` match in `typed_values.gd` (via new `_dict_to_aabb` helper) and to `encode` (`TYPE_AABB` case). Both directions now round-trip correctly.

**Verified:** `node set --property visibility_aabb --aabb -5,-5,-5,10,10,10` on `GravityZoneUp/GravParticles` succeeded. `node get` returned `{"kind":"AABB","value":{"position":[-5,-5,-5],"size":[10,10,10]}}`.

### Fix: `--allow-missing-autoloads` for `script write` (P2)

**Files:** `internal/bridge/client_assets.go`, `internal/cli/cli_project_assets.go`, `addons/godot_tcp_bridge/commands/script_commands.gd`

Added `--allow-missing-autoloads` flag to `script write`. When set, errors whose message contains `"Identifier not found"` are treated as soft and suppressed — the file is written anyway. Real syntax errors (parse errors, type mismatches, other compile errors) are still reported.

The internal `_write_skipping_preload_errors` function was refactored into `_write_skipping_soft_errors(skip_preloads, skip_autoloads)` so both modes share the same write/reload/filter path. The old function delegates to the new one for backward compatibility.

**Verified:**
- Without flag: `GameState.save()` → `SCRIPT_SYNTAX_INVALID: Identifier not found: GameState` (expected failure).
- With `--allow-missing-autoloads`: same script written successfully.
- With flag + deliberate parse error (`var x = 1 +`): still rejected with `SCRIPT_SYNTAX_INVALID` — the flag does not suppress real errors.

---

## File Inventory

```
res://gravwell_station/
  scenes/
    station_hub.tscn    - Main hub: Floor/Ceiling/4 Walls, 2 door triggers, gravity switch
    room_a.tscn         - Inverted gravity room: push block, gravity zone with warp VFX
    room_b.tscn         - Multi-platform room: 2 gravity zones (up + sideways)
    player.tscn         - CharacterBody3D + Camera3D + AnimationPlayer + AnimationTree
    hud.tscn            - CanvasLayer: room name, gravity indicator, inventory count
  scripts/
    player.gd           - Movement, gravity response, push-block detection, anim state
    gravity_zone.gd     - Area3D: applies gravity to entering bodies, drives particles
    game_state.gd       - Autoload singleton: room state, inventory, save/load JSON
    scene_transition.gd - Area3D: change_scene_to_file with GameState persistence
    hud.gd              - HUD controller: listens to GameState signals
  shaders/
    gravity_warp.gdshader - Vertex warp + ripple alpha distortion for gravity fields
  materials/
    gravity_particles.tres - ParticleProcessMaterial for gravity zone VFX
    gravity_warp.tres      - ShaderMaterial referencing gravity_warp.gdshader
  animations/
    player.tres         - AnimationLibrary: idle, walk, fall, push (1 track each)
  themes/
    hud.tres            - Theme: Label font_color/font_size, Panel bg_color
```
