# Open CLI Improvement Points

Consolidated from five game prototyping sessions (Chrono Forge → Signal Harbor → Skyline Courier → Chromabreach Arena → Echo Unit Breach). Each entry links to the source finding. Items already addressed in addon/CLI releases are excluded.

---

## Smoke / Automated Testing

**1. One-shot smoke command** *(Chrono Forge #3, Signal Harbor #3, Skyline Courier #4, Chromabreach #1/#4, Echo Unit #1)*

The single most-requested improvement across all sessions. Proposed form:

```bash
gdctl run smoke --scene SCENE --input FILE --probe SOURCE --assert KEY>=VALUE --screenshot OUT --timeout 10s
```

Chains `run start`, `run input`, probe predicate wait, screenshot capture, and `run stop` into a single command with a clear pass/fail exit code.

**2. `run wait-probe` / `run assert`** *(Echo Unit #1, Echo Unit #5)*

Stand-alone predicate command that polls `run logs` until a JSON probe field satisfies an expression (e.g. `targets_disabled >= 1`) or a timeout fires with the latest probe state printed. Can compose with the smoke command or be used independently.

**3. `run logs` filters** *(Echo Unit #3, Echo Unit #6)*

- `--source SOURCE` — filter by log source (e.g. `runtime.echo_unit`)
- `--latest` — print only the most-recent entry per source
- `--since-start` — exclude entries from before the current run start

Reduces manual JSON parsing for long input test sessions.

**4. `run input --summary-probe SOURCE`** *(Echo Unit #6)*

After input playback completes, automatically fetch and print selected latest probe fields instead of requiring a separate `run logs --json` pass.

---

## Scene Authoring

**5. Dependency-aware project/script sync** *(Chromabreach #3, Signal Harbor #2)*

- A `project sync` or `script write --deps-first` mode that writes dependencies in topological order (scenes before scripts that preload them).
- `script write --allow-missing-preloads` flag for iterative authoring when scenes will be created shortly after.
- Dry-run dependency scan that reports missing preloads before any mutation.

**6. Prefab/blueprint support for behavior-bearing repeated structures** *(Chromabreach #5, Skyline Courier #2, Echo Unit #4)*

`scene apply` grid expansion covers static geometry. What remains:

- Blueprint definitions that bundle a node tree with a script attachment and initial property set.
- Common 3D gameplay templates: `CharacterBody3D` player + camera rig, `SpotLight3D` + flashlight, raycast weapon, HUD reticle, collision mesh pairs.
- Friendly frontends: `mesh.box`, `collision.box`, `area.trigger`, `light.directional`.

**7. Higher-level transform flags** *(Skyline Courier #2)*

`--position X,Y,Z`, `--rotation-degrees X,Y,Z`, `--scale X,Y,Z` on `node set` and `scene apply` for small one-off edits without a full typed JSON value.

**8. `--scene` flag on remaining node mutation commands** *(Chrono Forge #4)*

`node attach-script --scene` proved valuable. Extend the same open/mutate/save shorthand to `node add`, `node set`, `node remove`, `node rename`, and `node move`.

**9. Batch / atomic open-mutate-save** *(Chrono Forge #4, Signal Harbor #1)*

- Client-side serialization to prevent the race condition where parallel `--scene` commands cross-wire different scenes (Signal Harbor: `CargoContainer` root overwriting `SignalPost.tscn`).
- Or a bridge-side batch endpoint that executes multiple node mutations atomically within a single open/save cycle.

---

## Runtime Observation

**10. Runtime center-ray probe for 3D** *(Echo Unit #2)*

`gdctl run probe raycast` — returns current camera direction, collider name, hit position, and distance. Useful for validating weapon aim and camera framing without relying on manual screenshot inspection.

**11. Screenshot sanity check** *(Skyline Courier #1)*

Warn when a `--source game` screenshot appears to be the editor background or desktop (e.g. solid-color heuristic, low-entropy detection) rather than a live game viewport.

**12. Deeper debugger integration** *(Chromabreach #1)*

`run status` already surfaces the paused-debugger state and error message. The remaining gap is stack frame detail: file, line, and local variable snapshot from an active debugger pause, exposed through the bridge.

---

## Addon Lifecycle

**13. `bridge addon-update` stale-file cleanup** *(Chrono Forge #6, Signal Harbor #7)*

Compare old and new addon manifests; delete files that were present in the previous install but are absent from the new package. Include removed-file count in update output and keep rollback backups before deletion.

---

## UI / Theme

**14. `theme` commands** *(Signal Harbor #5, missing-features-godot-demos #15)*

```bash
gdctl theme create --path res://ui/main_theme.tres
gdctl theme set-color --path ... --node-type Label --color-name font_color --value ff8800
gdctl theme set-font-size --path ... --node-type Label --size 18
gdctl theme set-constant --path ... --node-type MarginContainer --constant margin_top --value 8
```

Currently the fastest path is authoring HUD/UI layouts entirely in GDScript, which bypasses the theme system.

---

## Still-Open High-Effort Items from `missing-features-godot-demos.md`

**15. `animation` commands**

Create tracks, add keyframes, set animation length, and trigger playback. Affects every demo that uses AnimationPlayer (platformers, cutscenes, UI transitions). Estimated high effort.

**16. `tilemap` commands**

Set up `TileSet` resources and paint tiles onto a `TileMap`. No analog exists in the current CLI. Estimated high effort.

**17. `audio bus` commands**

Configure AudioServer buses: add effects, set volumes/sends. Needed for audio demos.

**18. `viewport set-size` / `viewport add`**

Create SubViewports with custom sizes and cameras for split-screen or render-target workflows.

---

## Priority Summary

| # | Item | Sessions | Effort |
|---|---|---|---|
| 1 | One-shot smoke command | All 5 | Medium |
| 2 | `run wait-probe` / assert | Echo Unit, Chromabreach | Low |
| 3 | `run logs` filters | Echo Unit | Low |
| 5 | Dependency-aware sync | Chromabreach, Signal Harbor | Medium |
| 9 | Batch atomic scene mutations | Signal Harbor, Chrono Forge | Medium |
| 6 | Blueprint/prefab templates | Chromabreach, Skyline, Echo Unit | High |
| 13 | `addon-update` stale cleanup | Chrono Forge, Signal Harbor | Low |
| 8 | `--scene` on more node commands | Chrono Forge | Low |
| 7 | Transform shorthand flags | Skyline Courier | Low |
| 10 | Center-ray probe | Echo Unit | Medium |
| 12 | Deeper debugger / stack frames | Chromabreach | Medium |
| 11 | Screenshot sanity check | Skyline Courier | Low |
| 4 | `run input --summary-probe` | Echo Unit | Low |
| 14 | `theme` commands | Signal Harbor | Medium |
| 15 | `animation` commands | All demos | High |
| 16 | `tilemap` commands | 2D tile demos | High |
| 17 | `audio bus` commands | Audio demos | Medium |
| 18 | `viewport set-size / add` | Viewport demos | Medium |
