# Game Plan: Hollow Market

**Genre:** 2D top-down social sim / economy game  
**Purpose:** Complex CLI prototyping session to find next-generation improvement points

---

## Concept

A small underground bazaar run by eccentric merchants. The player manages stalls, talks to NPCs with branching dialogue, crafts items from gathered ingredients, and reacts to a fluctuating economy driven by a hidden event system. Think Stardew Valley economy layer crossed with Recettear.

---

## Why This Finds Different CLI Gaps

Gravwell Station stresses 3D physics and environment. Hollow Market stresses **2D systems, dialogue, data-driven content, and runtime event handling** — a completely different surface area:

| System | New CLI territory |
|---|---|
| TileMap multi-layer market floor | Tilemap source-add + cell-set at scale; layer management |
| Dialogue trees (branching NPC text) | No dialogue/text resource commands; custom resource authoring at depth |
| `AnimationPlayer` per NPC (idle, talk, sell) | `animation player-play` in real multi-actor use |
| Economy event bus (price changes, rumours) | Signal connect/disconnect in runtime; `run wait-probe` on custom signals |
| Crafting: ingredient → item resource chain | Resource dependency graph; `resource create` chaining |
| Day/night cycle via `AnimationPlayer` on light | Long-running animation probe |
| Dynamic shop UI (prices, icons, quantities) | Theme + dynamic Label updates via `node set` at runtime |
| 2D camera zoom and pan | Viewport commands under 2D conditions |
| Save system: per-day market state | `file write-bytes` + custom resource pattern |
| Localisation stub (StringName resources) | No i18n/translation commands |

---

## Build Order (Five Sessions)

### Session 1 — Market Floor & Tilemap
- Create `market.tscn` with TileMap node
- `tilemap tileset-create` + `tilemap source-add` for floor/wall/stall tiles
- `tilemap cell-set` to paint the market layout across multiple layers
- Add static props (shelves, lanterns) via `node add`
- **Expected friction:** no TileMap `layer add/rename` command; painting large areas with individual `cell-set` calls is slow; no `tilemap fill` or `cell-set-rect` batch command

### Session 2 — NPCs & Dialogue
- `scene apply-blueprint` for NPC base (CharacterBody2D + Sprite2D + CollisionShape2D)
- `animation create` + `animation track-add` for idle/talk/sell cycles per NPC
- Custom resource `DialogueTree.tres` for branching text
- Wire NPC interaction signal to dialogue display
- **Expected friction:** no custom resource class definition command; `signal connect` requires a running scene; no dialogue resource type; registering a custom resource class requires manual script editing

### Session 3 — Economy & Event Bus
- `script create` for `EventBus` autoload singleton
- `autoload add` — not yet implemented
- Economy tick: price fluctuation driven by timer signal
- `run wait-probe` to assert price change after event fires
- **Expected friction:** no `autoload` commands; `run wait-probe` works on log entries but not on arbitrary node properties; need `run probe node` to read a node property directly at runtime

### Session 4 — Crafting & Inventory
- Custom resource `ItemData.tres` per craftable item
- `resource create` chain for ingredient → product relationships
- UI inventory panel with dynamic slot count
- `node set` bulk property updates (icon texture, quantity label)
- **Expected friction:** `node set` is one-property-per-call; updating 20 inventory slots requires 20+ round-trips; need bulk/batch node set; no `node set-texture` shorthand

### Session 5 — Day/Night, Audio & Save
- Day/night via `AnimationPlayer` driving `DirectionalLight2D` energy/color
- 3 audio buses: Music / Ambience / Market SFX
- Save/load per-day market state with `file write-bytes`
- `run smoke` full pass: open market → trigger economy event → assert probe → screenshot
- **Expected friction:** `run smoke` can only assert one probe source; i18n/translation stub not supported; no `project setting set` for locale

---

## Predicted New Improvement Items

| # | Item | Session | Area |
|---|---|---|---|
| 1 | `tilemap layer` commands — add/remove/rename TileMap layers | 1 | Tilemap |
| 2 | `tilemap cell-set-rect` — paint a rectangular region in one call | 1 | Tilemap |
| 3 | Custom resource class definition — register a new Resource subclass via CLI | 2 | Resources |
| 4 | `signal connect` in editor scene (not only running scene) | 2 | Signals |
| 5 | `autoload` commands — add/remove/list backed by project.godot | 3 | Project |
| 6 | `run probe node` — read a node property value directly at runtime | 3 | Testing |
| 7 | `node set` bulk — set multiple properties in one call | 4 | Nodes |
| 8 | `run smoke` multi-assert — assert against multiple sources in one pass | 5 | Testing |
| 9 | `i18n` commands — add translations, set locale | 5 | Project |
