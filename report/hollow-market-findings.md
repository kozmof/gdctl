# Hollow Market Creation Report

Date: 2026-05-14

## Created Prototype

Built **Hollow Market** in the connected Godot project under:

- `res://hollow_market/scenes/market.tscn`
- `res://hollow_market/scripts/market.gd`
- `res://hollow_market/scripts/event_bus.gd`
- `res://hollow_market/scripts/item_data.gd`
- `res://hollow_market/scripts/dialogue_tree.gd`
- `res://hollow_market/data/mushroom.tres`
- `res://hollow_market/data/charm.tres`
- `res://hollow_market/data/lantern.tres`
- `res://hollow_market/data/mara_dialogue.tres`

The game is a compact 2D underground market prototype with:

- Top-down market navigation with keyboard movement.
- Three NPC merchants with branching-style sequential dialogue and trade hooks.
- Event bus autoload, market events, rumors, and price-change signals.
- Runtime economy ticks that fluctuate item prices.
- Inventory, coins, stall purchases, NPC sales, and two craftable items.
- Day/night tinting and a next-day loop with new ingredient arrivals.
- Save/load ledger support through `user://hollow_market_save.json`.
- Dynamic HUD and dialogue UI.
- Runtime probes for readiness, movement, crafting, trading, saving, loading, and economy ticks.

The project main scene was set to:

```text
res://hollow_market/scenes/market.tscn
```

## Verification

Smoke boot passed:

```bash
GDCTL_BRIDGE_HOST=host.docker.internal GDCTL_BRIDGE_TOKEN=... \
go run ./cmd/gdctl run smoke \
  --scene res://hollow_market/scenes/market.tscn \
  --assert-source runtime.hollow_market \
  --assert-key ready \
  --assert-op '>=' \
  --assert-value 1 \
  --screenshot /workspace/tmp/hollow_market/smoke.png \
  --timeout 20s
```

Result:

```text
Smoke assert: runtime.hollow_market:ready>=1 ok
Smoke screenshot: /workspace/tmp/hollow_market/smoke.png (1152x648)
Smoke: PASS
```

Crafting smoke passed with synthetic `C` key input:

```text
Smoke input: 3 steps
Smoke assert: runtime.hollow_market:crafted>=1 ok
Smoke screenshot: /workspace/tmp/hollow_market/craft.png (1152x648)
Smoke: PASS
```

Economy smoke passed:

```text
Smoke assert: runtime.hollow_market:economy_ticks>=1 ok
Smoke screenshot: /workspace/tmp/hollow_market/economy.png (1152x648)
Smoke: PASS
```

Screenshot artifacts:

- `report/artifacts/hollow-market-smoke.png`
- `report/artifacts/hollow-market-craft.png`

Latest runtime economy probe:

```json
{
  "source": "runtime.hollow_market",
  "message": "economy",
  "detail": {
    "coins": 34,
    "crafted": 0,
    "day": 1,
    "dialogue_open": 0,
    "economy_ticks": 1,
    "ready": 1,
    "selected_npc": ""
  }
}
```

## Findings

| Finding | Status | Notes |
|---|---|---|
| 1. `scene batch` is now essential | Validated | Created the scene UI and attached the main script in one open/save cycle. This avoided the old chatty mutation issue. |
| 2. Custom resource creation works | Validated | `resource create --script` successfully produced item and dialogue `.tres` files from custom `Resource` scripts. |
| 3. Autoload authoring works | Validated | `autoload add --name HollowMarketBus` worked and the runtime scene connected to its signals. |
| 4. Runtime probes are productive | Validated | `run smoke` covered boot, crafting input, and economy tick behavior with screenshots. |
| 5. TileMap authoring is still the hardest part | Open | The playable floor was drawn by script instead of using CLI `tilemap cell-set` because large painted layouts still require too many one-cell calls and source setup needs prepared texture assets. |
| 6. UI property setup still wants presets | Open | Labels, panels, margins, and wrapping were easier to configure in script than by repeated `node set` calls. |
| 7. Audio bus commands need idempotent behavior | Open | `audio bus-add --name Ambience` failed when the bus already existed; a `--if-missing` flag or clearer success mode would help repeatable creation scripts. |

## Notable Workarounds

- Used script drawing for the multi-layer market floor and stalls instead of CLI TileMap painting.
- Used script-side UI layout for HUD and dialogue control sizing.
- Kept item/dialogue resources as authored data artifacts while the runtime script also contains fallback data for robustness.
- Used keycode handling directly instead of creating a project InputMap, keeping the prototype portable across existing project settings.

## Next CLI Opportunities

- Add `tilemap fill` or `tilemap cell-set-rect` for rectangular floor, wall, and stall painting.
- Add a texture or atlas bootstrap command for simple generated TileSet sources.
- Add `node set-many` or extend `scene batch` with multi-property updates per node.
- Add UI blueprints for common HUD/dialogue panels.
- Add `audio bus-add --if-missing`.
- Add a runtime `probe node` command for direct property inspection, complementing log-based probes.

## Follow-up Implemented

The first Hollow Market follow-up pass implemented the repeatable authoring gaps directly:

- `tilemap cell-set-rect` for rectangular TileMap painting.
- `node set-many` plus `scene batch` support for multi-property UI setup.
- `audio bus-add --if-missing` for idempotent bus creation.

Validation:

```text
go test ./... PASS
script check PASS for tilemap_commands.gd
script check PASS for node_commands.gd
script check PASS for audio_commands.gd
script check PASS for bridge_server.gd
```

Note: `gdctl addon update` wrote the changed bridge files into the connected Godot project and reported that a plugin reload is required. The currently running bridge still advertises the pre-update capability list, so live endpoint checks for `tilemap.cell-set-rect` and `node.set_many` should be run after disabling/enabling the plugin or restarting Godot.

After plugin reload, live validation passed:

- `bridge info` advertised `node.set_many` and `tilemap.cell-set-rect`.
- `audio bus-add --name Ambience --if-missing` passed twice and reported the existing bus without error.
- `node set-many --scene res://hollow_market/scenes/cli_followup_validation.tscn --path /root/LiveValidation/HUD --file ...` set 3 properties and saved the scene.
- `tilemap cell-set-rect --node /root/LiveValidation/Floor --layer 0 --x 1 --y 2 --width 6 --height 4 --source-id 0` painted 24 cells and the scene saved afterward.
