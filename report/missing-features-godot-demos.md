# Missing CLI Features for Godot Demo Projects

## Current CLI capabilities

| Category | Commands |
|---|---|
| Connection | `ping`, `doctor` |
| Addon | install, update, enable, disable, remove, status, doctor |
| Bridge | info, logs, addon-update |
| Scene | create, open, instance, tree, save |
| Node | add, remove, rename, move, get, set, set-resource, attach-script, group add/remove/list |
| Signal | connect, disconnect |
| Project | setting get, setting set |
| Script | create, write, check |
| Shader | write, check |
| Resource | create (any Resource subclass; typed props, resource-typed props, shader params) |
| File | write-bytes, lut-write, list, mkdir, delete, exists |
| Viewport | screenshot |

---

## Missing features by priority

### Critical — needed by virtually every demo

~~**1. `signal connect / disconnect`**~~
~~**2. `project setting set / get`**~~
~~**3. `node group add / remove / list`**~~

All three implemented (2026-05-09). See current capabilities table above.

---

### High priority — needed by most demos

~~**4. `resource create` (general)**~~
~~Currently only ShaderMaterial can be created. Missing:~~
~~- `StandardMaterial3D` / `ORM` (3D demos)~~
~~- `Environment` + `Sky` (3D lighting, sky shaders, volumetric fog)~~
~~- `PhysicsMaterial` (physics demos)~~
~~- `AudioStreamPlayer` resources~~
~~- `FontFile` (GUI/font demos)~~
~~- `AnimationLibrary`~~

Implemented (2026-05-10). `resource create --path PATH --type TYPE` works for any Godot Resource subclass. Supports `--prop name=TYPED_JSON` (including `{"kind":"Resource","value":"res://path"}` for resource-typed properties) and `--shader-param name=res://path` for ShaderMaterial texture parameters. `material write` is removed — ShaderMaterial creation goes through `resource create`. `lut write` moved to `file lut-write`.

**5. `animation` commands**
AnimationPlayer is central to platformers, cutscenes, UI transitions. Need: create track, add keyframe, set animation length, play.

~~**6. `node duplicate`**~~
~~Pattern used constantly — create one node, duplicate N times for terrain chunks, enemy spawners, UI items, etc.~~

~~**7. File system operations**~~
~~- `file list` — enumerate project files/dirs~~
~~- `file mkdir` — create subdirectories~~
~~- `file delete` — clean up generated assets~~
~~- `file exists` — conditional logic~~

~~**8. `node list-properties`**~~
~~Discover what properties a node type exposes — essential for AI-assisted building since you can't know all property names from memory.~~

---

### Medium priority — needed by specific demo categories

**9. `tilemap` commands**
2D tile demos require setting up TileSet resources and painting tiles onto a TileMap. No analog exists in the current CLI.

~~**10. `import set`**~~
~~Configure import settings for textures (filter, compression, sRGB), 3D models (scale, generate LODs), and audio (loop, compression). Without this, assets import with wrong defaults.~~

Implemented (2026-05-10). `import set --path PATH --param name=VALUE` reads the asset's `.import` ConfigFile, patches the `[params]` section, saves it, and triggers `EditorFileSystem.reimport_files`. Param values are raw JSON (integers, booleans, floats, strings).

~~**11. `navigation bake`**~~
~~3D navigation demos need `NavigationMesh.bake()` triggered from CLI.~~

~~**12. `scene list` / `resource list`**~~
~~Enumerate scenes and resources in the project (`res://` tree) so a build script can discover what assets exist.~~

Implemented (2026-05-10). `scene list [--dir res://] [--recursive]` finds all `.tscn` files; `resource list [--dir res://] [--recursive] [--ext .tres]` finds `.tres`/`.res` files (or any extension via `--ext`). Both return JSON arrays.

~~**13. `project run` / `scene run`**~~
~~Launch a scene headlessly to test it (requires a headless Godot binary — the `doctor` output already acknowledges this is missing).~~

Implemented (2026-05-10). `project run [--scene res://main.tscn] [--timeout 30s]` and `scene run --path res://scene.tscn` exec a headless Godot binary (`GDCTL_GODOT_PATH` env or `--godot` flag) with `--headless --path <project>`. `doctor` now reports the configured binary or warns when missing.

---

### Lower priority — specialized demos

**14. `audio bus` commands**
Audio effect demos configure AudioServer buses (add effects, set volumes). Needed only for `audio/*` demos.

**15. `theme create / apply`**
GUI theme demos set custom fonts, colors, margins on Control nodes via Theme resources.

**16. `viewport set-size` / `viewport add`**
`viewport/*` demos create SubViewports with custom sizes and cameras.

**17. Networking setup**
Multiplayer demos need `MultiplayerPeer` configuration, but this is largely done in GDScript so CLI support has lower leverage.

---

## Summary table

| Feature | Demos affected | Effort estimate | Status |
|---|---|---|---|
| `signal connect` | All | Medium | ✅ done |
| `project setting set/get` | All | Medium | ✅ done |
| `node group` | Most 2D/3D | Low | ✅ done |
| `resource create` (general) | Most 3D, GUI | High | ✅ done |
| `animation` commands | 2D/3D/GUI | High | — |
| `node duplicate` | Most | Low | ✅ done |
| File system ops | All | Low | ✅ done |
| `node list-properties` | All (tooling) | Low | ✅ done |
| `tilemap` | 2D tile demos | High | — |
| `import set` | 2D/3D asset demos | Medium | ✅ done |
| `navigation bake` | 3D navigation | Low | ✅ done |
| `scene list` / `resource list` | All (tooling) | Medium | ✅ done |
| `project run` / `scene run` | All (testing) | Medium | ✅ done |
| `audio bus` | Audio demos | Medium | — |
| `theme` | GUI demos | Medium | — |

All low-effort items (**node duplicate**, **file system ops**, **node list-properties**, **navigation bake**) are now done (2026-05-09).
**resource create** is now done (2026-05-10) — covers all resource types with typed props, resource-typed props, and shader params.
**import set**, **scene list** / **resource list**, and **project run** / **scene run** are now done (2026-05-10).
The remaining impactful additions are **animation commands** (high effort) and **tilemap** (high effort).
