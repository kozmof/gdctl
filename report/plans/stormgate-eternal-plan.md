# Game Plan: Stormgate Eternal

**Genre:** 3D action-RPG with procedural dungeons, online co-op lobby, and in-engine cinematics  
**Purpose:** Most complex CLI prototyping session — surfaces all remaining major CLI gaps

---

## Concept

A two-player online co-op dungeon crawler. Each run generates a new dungeon from modular room templates. Players choose one of three character classes, descend through five floors, and face a final boss whose entrance is gated by a scripted cutscene. Between runs, a persistent overworld hub holds the shop, skill tree, and quest board. Think *Hades* meeting *Deep Rock Galactic* — procedural structure, heavy narrative framing, co-op at the core.

---

## Why This Is the Most Complex Yet

It combines all the stress areas of the three previous plans and adds four entirely new surfaces: **procedural runtime assembly, online networking config, cinematic camera systems, and behaviour tree AI**.

| System | Stress area |
|---|---|
| Procedural dungeon: runtime room instancing + stitching | `run instantiate` + scene graph mutation while running |
| 3 character classes × AnimationTree (14+ states each) | `animation tree` at real scale — state machines, blend spaces, conditions |
| Online lobby (ENetMultiplayerPeer, RPC stubs) | Zero networking commands in current CLI |
| Boss cutscene (AnimationPlayer choreography, camera cuts) | Cinematic camera commands; complex multi-track animation |
| Overworld hub: persistent cross-run state | `autoload` + custom resource save/load at depth |
| Skill tree UI (90+ nodes, icons, connection lines) | `node set` bulk; theme at scale; no graph node commands |
| Particle VFX: spells, impacts, ambient dungeon | `particle` commands; runtime emit/stop |
| Dynamic dungeon lighting (per-floor colour palette) | `environment` + `light` commands in runtime mutation |
| Behaviour tree AI for 5 enemy archetypes | No behaviour tree / `NavigationAgent3D` config commands |
| Weather in hub world (rain, fog, wind particles) | `particle` + `environment` compound mutations |
| Minimap: runtime TileMap updated as rooms explored | `tilemap cell-set` at runtime; `viewport camera-assign` |
| 4-bus audio: Music / Dungeon Ambience / Combat SFX / UI | Audio commands at full production depth |
| Loot: procedural item stat resource generation | `resource create` + `node set` chains; no stat-roll command |
| Death/respawn: scene reload preserving partner state | `run scene-reload` without losing co-op peer |
| In-game debug overlay (enemy HP, navmesh, hitboxes) | `run probe node` for real-time stat display |

---

## Build Order (Seven Sessions)

### Session 1 — Hub World: Environment, Lighting, Audio
- Create `hub_world.tscn` with WorldEnvironment, DirectionalLight3D, sky
- `node set` for fog, ambient light, tone mapping
- 4 audio buses: Music / Dungeon Ambience / Combat SFX / UI
- `audio bus-effect-add` reverb/eq per bus
- **Expected friction:** no `environment` shorthand; setting sky properties is verbose multi-step; no `light` energy/color shorthand flags

### Session 2 — Dungeon Room Templates & Procedural Stitching
- Create 5 modular room `.tscn` templates (corridor, chamber, trap room, treasure, boss antechamber)
- `run instantiate` with init params to spawn and stitch rooms at runtime — not yet implemented
- `tilemap cell-set` at runtime to paint floor/wall layout per generated room
- Navigation mesh bake after each room is placed
- **Expected friction:** no `run instantiate` command; no way to pass constructor data to spawned scene; runtime navmesh bake triggered only via existing `navigation bake` which may not handle incremental updates

### Session 3 — Character Classes & AnimationTree
- 3 classes: Warden (melee), Conduit (ranged magic), Specter (stealth)
- `scene apply-blueprint player3d` as base for each
- `animation create` for each state (idle, walk, run, sprint, attack1, attack2, dodge, hurt, stagger, death, interact, cast, aim, climb) — 14 states per class
- AnimationTree state machine wiring — not yet CLI-accessible
- Blend space 2D for locomotion
- **Expected friction:** no `animation tree` commands at all; 14-state machine requires GUI; blend spaces have no CLI representation; transition conditions cannot be set via CLI

### Session 4 — Enemy AI (5 Archetypes)
- Grunt, Brute, Archer, Specter, Elite — each with NavigationAgent3D
- Behaviour tree per archetype (patrol → detect → pursue → attack → retreat)
- `node set` for NavigationAgent3D max_speed, avoidance_enabled, radius
- `node set` bulk — 5+ properties per agent per archetype — not yet implemented
- **Expected friction:** no behaviour tree commands; no `NavigationAgent3D` config shorthand; bulk node set missing makes per-agent tuning extremely tedious; no way to inspect live agent state without `run probe node`

### Session 5 — Online Lobby Stub (ENetMultiplayerPeer)
- `script create` for `NetworkManager` autoload
- Configure ENetMultiplayerPeer host/join via project settings
- `autoload add` — not yet implemented
- Set multiplayer authority on player nodes
- **Expected friction:** no `network` commands whatsoever; no `autoload` commands; multiplayer peer configuration requires manual script edits; no CLI way to set RPC mode on nodes

### Session 6 — Boss Cutscene & Camera Choreography
- Boss antechamber: 4-camera setup (wide, close, over-shoulder, cinematic crane)
- AnimationPlayer multi-track: camera cuts, character positions, lighting shifts, audio stings
- `animation keyframe-add` for camera Transform3D tracks across 4 cameras
- `animation player-play` to trigger cutscene
- **Expected friction:** no camera-cut / cinematic command; `animation keyframe-add` one-at-a-time makes multi-track cutscenes impractical; no `animation track-add --camera-cut` shorthand; no way to preview cutscene progress via CLI

### Session 7 — Loot, Skill Tree, Persistence & Final Smoke Pass
- Procedural loot: item stat resource generated from seed
- Skill tree: 90-node UI graph
- `node set` bulk for all skill tree nodes — not yet implemented
- `resource create` with random stat fields
- `run smoke` multi-assert: spawn room → kill enemy → assert loot drop → screenshot minimap SubViewport
- **Expected friction:** `node set` one-at-a-time unusable for 90-node graphs; no `resource procedural` command; `run smoke` single-assert only; no SubViewport screenshot; no `run probe node` for loot verification

---

## Predicted New Improvement Items

| # | Item | Session | Area |
|---|---|---|---|
| 1 | `environment` commands — WorldEnvironment sky/fog/tone mapping shorthands | 1 | Rendering |
| 2 | `light` commands — energy/color/range shorthands for all light types | 1 | Rendering |
| 3 | `run instantiate` with params — spawn PackedScene into running scene with init data | 2 | Runtime |
| 4 | `navigation bake` incremental — rebake partial navmesh after runtime room add | 2 | Navigation |
| 5 | `animation tree` commands — create AnimationTree, StateMachine, blend space, transitions | 3 | Animation |
| 6 | `node set` bulk — set multiple properties on one node in a single call | 4 | Nodes |
| 7 | `run probe node` — read a live node property value at runtime | 4 | Testing |
| 8 | `network` commands — configure ENetMultiplayerPeer, set multiplayer authority | 5 | Networking |
| 9 | `autoload` commands — add/remove/list backed by project.godot | 5 | Project |
| 10 | `cinematic` / camera-cut shorthand — sequence camera transitions in AnimationPlayer | 6 | Animation |
| 11 | `animation keyframe-add` bulk — add multiple keyframes in one call | 6 | Animation |
| 12 | `node set` bulk | 7 | Nodes |
| 13 | `resource procedural` — generate resource with randomised fields from a seed | 7 | Resources |
| 14 | `run smoke` multi-assert — assert multiple source:key>=value in one pass | 7 | Testing |
| 15 | `run screenshot --viewport NAME` — capture a named SubViewport | 7 | Testing |

---

## Comparison Across All Four Plans

| Theme | Sessions | Primary stress | Unique new items |
|---|---|---|---|
| Gravwell Station | 5 | 3D environment, physics, particles, persistence | environment, light, particle, autoload, input map |
| Hollow Market | 5 | 2D tilemap, dialogue, data-driven economy | tilemap layers, custom resource class, bulk node set, i18n |
| Resin Protocol | 5 | Split-screen, co-op, runtime instancing | viewport camera-assign, run instantiate, material set-uniform, audio listener, run scene-reload |
| Stormgate Eternal | 7 | Procedural, networking, cinematics, AI at scale | network, cinematic/camera-cut, behaviour tree config, run instantiate+params, resource procedural, animation tree at scale |

Items shared by all four: `animation tree`, `autoload`, `run probe node`, `node set` bulk.
