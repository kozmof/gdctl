# Game Plan: Resin Protocol

**Genre:** Asymmetric 2-player local co-op action  
**Purpose:** Complex CLI prototyping session to find next-generation improvement points

---

## Concept

Two players share one screen. Player 1 is a ground agent navigating a ruined biotech facility on foot. Player 2 is a remote operator who sees a top-down tactical map and can open doors, trigger traps, and deploy drones. Neither can complete the mission alone. Think *Portal 2 co-op* structure but asymmetric information.

---

## Why This Finds Different CLI Gaps

Neither Gravwell Station nor Hollow Market tests **split-screen, multiple input devices, co-op state synchronisation, or runtime scene instancing under load**.

| System | New CLI territory |
|---|---|
| Split viewport (agent view + operator map) | `viewport add` in real co-op use; SubViewport camera assignment |
| Two simultaneous `InputMap` profiles | `input map` per-player action sets |
| Drones: runtime `instantiate` of PackedScene | Scene instancing at runtime via CLI; no `run instantiate` command |
| Door/trap state shared between views | `run probe node` — read door state from ground scene in operator view |
| Minimap shader (fog of war mask) | Shader uniform set at runtime; no `material set-uniform` |
| `AnimationTree` blend: agent crouch/sprint/climb | Shared gap with Gravwell Station |
| Networked-style event log (for co-op feedback) | `run logs` multi-source filtering at scale |
| Per-player audio listener position | Audio listener commands not yet implemented |
| Operator UI: tile-overlay tactical map | Tilemap driven entirely from CLI at runtime |
| Mid-session scene hot-swap for operator screen | Scene reload without losing ground agent state |

---

## Build Order (Five Sessions)

### Session 1 — Split Viewport & Cameras
- `viewport add` for agent view (3D) and operator map (top-down orthographic)
- `viewport camera-assign` — not yet implemented
- Set Camera3D and Camera2D per SubViewport via `node set`
- `viewport set-size` to divide screen 70/30
- **Expected friction:** no `viewport camera-assign` command; assigning a camera to a specific SubViewport requires multiple manual `node set` calls with no clear CLI pattern

### Session 2 — Agent & Drone System
- `scene apply-blueprint player3d` for agent
- `run instantiate` — runtime PackedScene spawn for drones — not yet implemented
- `animation create` + AnimationTree for agent states
- Drone pooling via script
- **Expected friction:** no `run instantiate` command to spawn packed scenes at runtime; drone spawning requires manual GDScript; no CLI way to trigger scene instantiation in a running game

### Session 3 — Operator Tactical Map
- TileMap overlay for operator view
- `tilemap cell-set` at runtime to update fog-of-war state
- Fog-of-war shader: `material set-uniform` to push player position — not yet implemented
- `run probe node` to read agent world position into operator map
- **Expected friction:** `material set-uniform` missing; `run probe node` missing; operator needs real-time position data but CLI only has log-based probing

### Session 4 — Co-op Input & Audio
- `input map` per-player action sets (joypad device 0 vs device 1)
- `audio listener` commands — position AudioListener3D per viewport — not yet implemented
- Per-player SFX/music mix via audio buses
- **Expected friction:** no `audio listener` commands; no per-device input map binding; audio spatialisation per-viewport not CLI-accessible

### Session 5 — Mission Flow & Hot-Swap
- Operator subscene hot-swap mid-mission (`run scene-reload`)
- `run smoke` multi-viewport — screenshot a specific SubViewport — not yet implemented
- Mission state probes across both players
- **Expected friction:** `run scene-reload` not implemented; `run screenshot` only captures main viewport; no way to screenshot a named SubViewport

## Comparison With Other Plans

| Theme | Primary stress | Unique items |
|---|---|---|
| Gravwell Station | 3D environment, physics, particles, persistence | environment, light, particle, autoload, input map |
| Hollow Market | 2D tilemap, dialogue, data-driven economy | tilemap layers, custom resource class, bulk node set, i18n |
| Resin Protocol | Split-screen, co-op, runtime instancing | viewport camera-assign, run instantiate, material set-uniform, audio listener, run scene-reload |

All three share `animation tree` and `autoload` as common gaps. Running all three sessions would systematically cover almost every major CLI surface area that v0.2.0 does not yet reach.
