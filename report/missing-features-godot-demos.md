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
| Material | write (ShaderMaterial only) |
| File | write-bytes |
| LUT | write |
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

**4. `resource create` (general)**
Currently only ShaderMaterial can be created. Missing:
- `StandardMaterial3D` / `ORM` (3D demos)
- `Environment` + `Sky` (3D lighting, sky shaders, volumetric fog)
- `PhysicsMaterial` (physics demos)
- `AudioStreamPlayer` resources
- `FontFile` (GUI/font demos)
- `AnimationLibrary`

**5. `animation` commands**
AnimationPlayer is central to platformers, cutscenes, UI transitions. Need: create track, add keyframe, set animation length, play.

**6. `node duplicate`**
Pattern used constantly — create one node, duplicate N times for terrain chunks, enemy spawners, UI items, etc.

**7. File system operations**
- `file list` — enumerate project files/dirs
- `file mkdir` — create subdirectories
- `file delete` — clean up generated assets
- `file exists` — conditional logic

**8. `node list-properties`**
Discover what properties a node type exposes — essential for AI-assisted building since you can't know all property names from memory.

---

### Medium priority — needed by specific demo categories

**9. `tilemap` commands**
2D tile demos require setting up TileSet resources and painting tiles onto a TileMap. No analog exists in the current CLI.

**10. `import set`**
Configure import settings for textures (filter, compression, sRGB), 3D models (scale, generate LODs), and audio (loop, compression). Without this, assets import with wrong defaults.

**11. `navigation bake`**
3D navigation demos need `NavigationMesh.bake()` triggered from CLI.

**12. `scene list` / `resource list`**
Enumerate scenes and resources in the project (`res://` tree) so a build script can discover what assets exist.

**13. `project run` / `scene run`**
Launch a scene headlessly to test it (requires a headless Godot binary — the `doctor` output already acknowledges this is missing).

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
| `resource create` (general) | Most 3D, GUI | High | — |
| `animation` commands | 2D/3D/GUI | High | — |
| `node duplicate` | Most | Low | — |
| File system ops | All | Low | — |
| `node list-properties` | All (tooling) | Low | — |
| `tilemap` | 2D tile demos | High | — |
| `import set` | 2D/3D asset demos | Medium | — |
| `navigation bake` | 3D navigation | Low | — |
| `project run` | All (testing) | Medium | — |
| `audio bus` | Audio demos | Medium | — |
| `theme` | GUI demos | Medium | — |

The next most impactful additions would be **node duplicate**, **file system ops**, and **node list-properties** —
all low-effort and broadly useful across demo categories.
