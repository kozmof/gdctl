# Game Plan: Gravwell Station

**Genre:** 3D puzzle-platformer / environmental storytelling  
**Purpose:** Complex CLI prototyping session to find next-generation improvement points

---

## Concept

An abandoned space station with gravity anomalies. The player navigates rooms by manipulating gravity direction, pushes objects into place to unlock doors, and pieces together what happened to the crew. 3–4 interconnected rooms, persistent room state, simple inventory.

---

## Why This Finds New CLI Gaps

Every previous game (Chrono Forge → Signal Harbor → Skyline Courier → Chromabreach → Echo Unit) was self-contained in one scene with a flat node tree. Gravwell Station requires:

| System | New CLI territory |
|---|---|
| Multi-scene with `change_scene_to_file` | Scene transition / loading flow |
| AnimationTree for player (idle/walk/fall/push) | No `animation tree` commands yet |
| `WorldEnvironment` + `DirectionalLight3D` per room | No environment/lighting commands |
| `GPUParticles3D` for gravity-field VFX | No particle commands |
| Autoload singleton `GameState` for room progress | No autoload management |
| Persistent per-room save (custom `.tres` resource) | Custom resource authoring gap |
| 3-bus audio: Music / Ambience / SFX | Validates existing audio commands at depth |
| `InputMap` — remap gravity directions to WASD | No input map commands |
| Shader: gravity-warp distortion on environment | Heavier shader workflow use |
| HUD: room name, gravity indicator, inventory count | Theme system in real use |

---

## Build Order (Five Sessions)

### Session 1 — Environment & Rooms
- Create `station_hub.tscn`, `room_a.tscn`, `room_b.tscn`
- `node add` WorldEnvironment + Sky + DirectionalLight3D in each
- Set environment properties with `node set`
- Scene-transition trigger via Area3D blueprint
- **Expected friction:** no `environment` shorthand flags; setting sky/light properties is verbose

### Session 2 — Player Character
- `scene apply-blueprint player3d` as base
- Layer AnimationTree on top: idle / walk / fall / push states
- `node set` to wire AnimationTree parameters
- **Expected friction:** no `animation tree` command; manually wiring AnimationTree nodes through `node set` will be very painful; `animation player-play` only covers simple cases

### Session 3 — Gravity Mechanic & Physics
- `GPUParticles3D` nodes for visual gravity field
- Physics gravity scale overrides per area
- `Area3D` gravity vector via `node set --property gravity_direction --vector3`
- Particle material setup
- **Expected friction:** no particle commands; setting `ParticleProcessMaterial` properties requires many `node set-resource` + `node set` round-trips

### Session 4 — Persistence & Game State
- Autoload `GameState` singleton via `script create` + project settings
- Custom resource `RoomData.tres` to store door/switch state
- Save/load via `resource.create` + `node set`
- **Expected friction:** no `autoload add/remove` command; no `project.setting set` for autoloads; custom resource class registration gap

### Session 5 — Polish: HUD, Audio, Shaders
- Full HUD with theme system
- Music / Ambience / SFX bus mixing (validates existing audio commands)
- Gravity-warp shader via `shader write`
- `run smoke` end-to-end pass: enter room → trigger gravity → assert probe
- **Expected friction:** `run smoke --assert` may need multi-source probe; shader uniform set at runtime not yet CLI-accessible

---

## Predicted New Improvement Items

| # | Item | Session | Area |
|---|---|---|---|
| 1 | `animation tree` commands — create AnimationTree, add StateMachine, states/transitions | 2 | Animation |
| 2 | `environment` commands — WorldEnvironment sky/fog/tone mapping shorthands | 1 | Rendering |
| 3 | `light` commands — DirectionalLight3D/OmniLight3D property shorthands | 1 | Rendering |
| 4 | `particle` commands — GPUParticles3D, ParticleProcessMaterial, emit/stop | 3 | VFX |
| 5 | `autoload` commands — add/remove/list backed by project.godot edits | 4 | Project |
| 6 | `input map` commands — add/remove/list input actions and key bindings | 4 | Project |
| 7 | `run smoke` multi-assert — assert against multiple sources in one pass | 5 | Testing |
| 8 | Material uniform set — `material set-uniform` for shader parameters at runtime | 5 | Rendering |
| 9 | Scene transition probe — wait for `scene.changed` event in run logs | 5 | Testing |
