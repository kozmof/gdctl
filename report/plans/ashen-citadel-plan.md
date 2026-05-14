# Game Plan: Ashen Citadel

**Genre:** Asymmetric local competitive base-assault  
**Purpose:** Most ambitious CLI prototyping session — covers the confirmed P1 AnimationTree gap, all unexecuted gaps from prior plans, and introduces new CLI surface areas that appear in no prior plan

---

## Concept

A two-player local asymmetric game. ARCHITECTS spend a build phase constructing a fortified base from modular room blueprints, assembling structure using CSG boolean operations and wiring sections together with physics joints. BREAKERS assault the completed base with siege weapons, cracking structural joints and triggering cascading collapses. A cinematic cutscene introduces each match. Between matches, a persistent hub holds a GraphEdit-based blueprint designer and a second OS window carrying a live tactical replay.

Think *Fortnite Save the World* (build/defend) crossed with an asymmetric local tactics game, with a Godot-native CSG construction pipeline as the core design challenge.

---

## Why This Is More Complex Than Stormgate Eternal

Stormgate Eternal proposed cinematics, AI, and procedural content but left most unexecuted. Ashen Citadel executes each of those and adds **new CLI surface areas** present in no prior plan:

| System | New CLI territory |
|---|---|
| CSG modular construction (build phase) | `csg` command group — zero prior coverage |
| LightmapGI bake (static base lighting) | `lightmap bake` — zero prior coverage |
| ReflectionProbe bake per room | `reflection-probe bake` — zero prior coverage |
| VoxelGI runtime update (GI shifts as rooms are CSG-destroyed) | `voxelgi bake` — zero prior coverage |
| Decal3D (impact marks, structural stress cracks) | `decal add/set` — zero prior coverage |
| FogVolume (smoke grenades, gas hazards) | `fog-volume add` — zero prior coverage |
| Physics joints (structural integrity of base sections) | `joint add` — zero prior coverage |
| SoftBody3D (banners, debris cloth) | `softbody pin-point/unpin-point` — zero prior coverage |
| LOD batch assignment (100+ building piece meshes) | `lod set-many` — zero prior coverage |
| OccluderInstance3D (large static geometry) | `occluder add` — zero prior coverage |
| Multi-window tactical replay | `window create`, `window assign-viewport` — zero prior coverage |
| GraphEdit blueprint designer | `graph-edit node-add`, `graph-edit connection-add` — zero prior coverage |
| TTS accessibility announcer | `accessibility tts-speak` — zero prior coverage |
| i18n: 4-language localization | `i18n locale-set`, `i18n string-add` — zero prior coverage |
| Heightmap terrain (outdoor arena) | `terrain heightmap-import` — zero prior coverage |
| AudioStreamPlaylist (adaptive battle music) | `audio playlist-add` — zero prior coverage |
| Runtime performance profiling | `run profile --metric` — zero prior coverage |
| AnimationTree state machine topology | `animation tree add-state`, `add-transition` — confirmed P1 gap (Gravwell findings) |
| AnimationTree blend space 2D | `animation blend-space-2d-add` — confirmed gap, never executed |
| `run instantiate` (runtime PackedScene spawn) | Shared gap — proposed in three plans, never executed |
| `audio listener` per viewport | Shared gap — proposed in Resin Protocol, never executed |
| `run scene-reload` mid-session | Shared gap — proposed in two plans, never executed |
| Multi-assert `run smoke` | Shared gap — single-assert limit confirmed in Gravwell findings |
| SubViewport screenshot in smoke | Shared gap — no SubViewport targeting in `run smoke` |

Items shared by all four prior plans (still unimplemented): `animation tree`, `run instantiate`, `audio listener`, `run scene-reload`.

---

## Build Order (Nine Sessions)

### Session 1 — Arena Terrain + Static Lighting Pipeline
- Heightmap terrain for the outdoor arena
- `terrain heightmap-import --path NODE --texture res://textures/arena_height.png` — not yet implemented
- LightmapGI node for interior base rooms
- `lightmap bake --path /root/Base/LightmapGI` — not yet implemented
- ReflectionProbe per major room section
- `reflection-probe bake --path /root/Base/Room_A/ReflectionProbe` — not yet implemented
- **Expected friction:** No `terrain`, `lightmap`, or `reflection-probe` command groups; LightmapGI and ReflectionProbe baking are async editor-side operations that require polling a job handle — the bridge exposes no bake trigger or progress query; heightmap terrain via `HeightMapShape3D` requires a texture-to-data step that the bridge cannot easily perform

### Session 2 — CSG Construction System (Build Phase)
- ARCHITECT build phase: modular rooms assembled from `CSGBox3D` + `CSGCombiner3D`
- `csg node-add --parent PATH --type CSGBox3D --operation union --name "HubRoom"` — not yet implemented
- Runtime CSG operation change when BREAKERS destroy a room
- `csg operation-set --path PATH --operation subtract` — not yet implemented
- `csg size-set --path PATH --size X,Y,Z` — not yet implemented (CSG geometry is not accessible via generic `node set` because `CSGBox3D.size` requires a `Vector3` typed property assignment — possible but verbose; a shorthand is needed at scale)
- **Expected friction:** No `csg` command group; CSG is a Godot node family with no bridge representation; switching `CSGCombiner3D.operation` at runtime via `node set --property operation --int 1` works technically but is opaque without semantic naming; building a room from 6–8 CSG pieces requires 6–8 `node add` + `csg size-set` round-trips per room; the bridge has no knowledge of CSG topology (union/subtract graph) as a unit

### Session 3 — Physics Destruction: Joints + SoftBody
- Structural `HingeJoint3D` on door hinges and support beam connections
- `joint add --type hinge --node-a /root/Base/Beam_A --node-b /root/Base/Beam_B --anchor-a 0,1,0` — not yet implemented
- `joint limit-set --path PATH --axis angular --min -45 --max 45` — not yet implemented
- `SoftBody3D` for ARCHITECT banners and fabric debris
- `softbody pin-point --path PATH --point 0` — not yet implemented (anchor top edge of banner)
- `softbody unpin-point --path PATH --point 12` — not yet implemented (releases bottom during destruction)
- **Expected friction:** No `joint` or `softbody` command groups; `HingeJoint3D` configuration requires `nodes/node_a` and `nodes/node_b` as `NodePath` values plus 6+ angular/linear limit properties — verbose without semantic joint commands; `SoftBody3D` pin indices cannot be set from CLI at all; configuring joint limits requires a `node set` call per property with non-obvious property path names

### Session 4 — AnimationTree at Scale: 8 Characters × 16 States
- 4 ARCHITECT classes: Constructor, Shield, Engineer, Medic
- 4 BREAKER classes: Assault, Demolisher, Scout, Vanguard
- Each class: 16 states (idle, walk, sprint, attack_1, attack_2, reload, hurt, stagger, death, interact, crouch, vault, emote_1, emote_2, revive, outro)
- AnimationNodeStateMachine per character
- `animation tree add-state --tree PATH --name "walk" --animation "walk"` — not yet implemented
- `animation tree add-transition --tree PATH --from "idle" --to "walk" --condition "speed>0.1"` — not yet implemented
- Blend space 2D for locomotion (walk/sprint directional blend)
- `animation blend-space-2d-add --tree PATH --state "locomotion" --blend-x "strafe" --blend-y "speed"` — not yet implemented
- `animation tree set-param --tree PATH --param "parameters/locomotion/blend_position" --vector2 0,1` — not yet implemented
- **Expected friction:** Confirmed P1 gap from Gravwell Station findings — `animation tree` commands do not exist; `AnimationNodeStateMachine` topology (states, transitions, conditions) is completely opaque to CLI; the `animation` commands only target `AnimationLibrary` resources; 8 × 16 state machines make GUI-only authoring impractical at this scale; blend space 2D coordinate axes have no CLI representation

### Session 5 — AI, LOD, and Occlusion
- 5 BREAKER AI archetypes (Grunt, Brute, Sapper, Sharpshooter, Commander) with `NavigationAgent3D`
- Behaviour tree per archetype (patrol → detect → pursue → build-breach → attack → retreat) — no `behaviour-tree` commands (shared gap from Stormgate)
- LOD on all 100+ building piece meshes per base
- `lod set --path PATH --begin 20.0 --end 40.0` — not yet implemented
- `lod set-many --file lod_config.json` — not yet implemented (batch assignment needed; 100+ meshes make per-mesh `node set visibility_range_begin` impractical)
- `OccluderInstance3D` on large structural walls
- `occluder add --parent PATH --shape box --size 8,4,6` — not yet implemented
- Incremental navigation bake after each CSG mutation during the BREAKER assault
- **Expected friction:** LOD configuration per mesh requires `node set --property visibility_range_begin` and `visibility_range_end` — 2 round-trips per mesh, 200+ round-trips total; no batch LOD command; occluder shape configuration has no CLI path; navigation incremental bake after CSG geometry change not known to trigger correctly via the existing `navigation bake` command

### Session 6 — Rendering Pipeline: Decals + FogVolume + VoxelGI
- `Decal3D` for bullet impact marks, structural stress indicators, scorch marks
- `decal add --parent PATH --texture res://textures/impact.png --size 0.3,0.3,0.3` — not yet implemented
- `decal set-normal-fade --path PATH --fade 0.5` — not yet implemented
- `FogVolume` for smoke grenades (gameplay-affecting opaque fog) and gas hazards (area-denial)
- `fog-volume add --parent PATH --shape ellipsoid --size 2,2,2 --density 1.2` — not yet implemented
- `FogMaterial` uniform set for color and density over time
- `material set-uniform --path FOG_MAT --param density --float 0.0` — not yet implemented (runtime fade-out; note: shader parameter syntax via `node set --property "material:shader_parameter/X"` works for `ShaderMaterial` but not for `FogMaterial` which uses a different resource type)
- VoxelGI bake invalidation as CSG geometry changes during the assault
- `voxelgi bake --path /root/Arena/VoxelGI` — not yet implemented; deferred editor-side bake operation
- **Expected friction:** `decal`, `fog-volume`, and `voxelgi` have zero CLI coverage; `FogMaterial` property updates require `node set-resource` + `node set` chains; VoxelGI bake during active gameplay (not in editor) is undefined behavior — may require a different CLI path than an editor bake

### Session 7 — GraphEdit Blueprint Designer + Multi-Window Replay
- ARCHITECT pre-match blueprint designer built with `GraphEdit` + `GraphNode` UI
- `graph-edit node-add --path /root/Hub/BlueprintGraph --name "Hub Room" --position 100,200` — not yet implemented
- `graph-edit connection-add --graph PATH --from "Hub Room" --from-port 0 --to "Barracks" --to-port 0` — not yet implemented
- `graph-edit node-remove --graph PATH --name "Barracks"` — not yet implemented
- Second OS window for live tactical replay
- `window create --title "Replay" --width 1280 --height 720 --position 1920,0` — not yet implemented
- `window assign-viewport --window-id 1 --viewport /root/ReplayViewport` — not yet implemented
- **Expected friction:** `GraphEdit` and `GraphNode` nodes have zero CLI representation; node position, title, slot count, and connection topology are entirely GUI-driven; multi-window creation is a `DisplayServer` call with no bridge command; assigning a `SubViewport` to a specific OS window index has no CLI path; the blueprint designer may need drag-and-drop behavior that cannot be simulated via `run input` alone

### Session 8 — Accessibility + i18n
- TTS match announcer ("Architects win!", "Base breach detected")
- `accessibility tts-speak --text "Match starts in 3" --interrupt true` — not yet implemented
- `accessibility tts-configure --pitch 1.0 --rate 1.1 --voice "en_US"` — not yet implemented
- `accessibility tts-stop` — not yet implemented
- 4-language localization (English, Japanese, Korean, French)
- `i18n locale-set --locale ja` — not yet implemented
- `i18n string-add --key BASE_BREACH --locale en --text "Base breach detected!"` — not yet implemented (batch string import from CSV/PO file would be needed; per-key CLI entry impractical for a real translation workflow)
- `AudioStreamPlaylist` for adaptive battle music that switches tracks based on tension level
- `audio playlist-add --bus Music --stream res://audio/battle_tense.ogg` — not yet implemented
- `audio playlist-autoplay --bus Music --mode random_no_repeat` — not yet implemented
- **Expected friction:** No `accessibility` or `i18n` command groups; `DisplayServer.tts_speak()` is a runtime call with no bridge command; `TranslationServer.load_translation()` requires a `.po` file resource, not a key-value CLI command; locale switching during a live session has undefined behavior in the bridge; no `audio playlist` commands

### Session 9 — Performance Profiling + Final Smoke Pass
- Runtime profiling under full load (both players + AI + active physics joints + FogVolumes)
- `run profile --metric fps,draw_calls,physics_time,memory_usage --duration 30s` — not yet implemented
- Multi-assert smoke test covering both factions
- `run smoke --assert runtime.base:integrity>=0.5 --assert runtime.breakers:unit_count>=2` — currently `run smoke` supports only one `--assert`; multi-assert requires multiple `run wait-probe` calls with no atomic pass/fail
- SubViewport screenshot of the replay window mid-smoke
- `run smoke --screenshot-viewport ReplayViewport ./replay.png` — not yet implemented; `run smoke --screenshot` only captures the main viewport
- Scene hot-reload mid-match to simulate a live patch cycle
- `run scene-reload --preserve-autoloads` — not yet implemented; current scene-reload equivalent drops all runtime state
- **Expected friction:** Multi-assert `run smoke` gap confirmed in Gravwell findings; SubViewport screenshot not CLI-accessible; `AudioStreamPlaylist` has no command group; performance metric sampling not exposed via bridge; `run scene-reload` entirely absent
