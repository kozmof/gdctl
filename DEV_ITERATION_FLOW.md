# gdctl Dev Iteration Flow

This document records the current proven workflow for developing `gdctl` from a Linux devcontainer while Godot runs on a Windows host.

## 1. Network Model

Default secure setup:

```text
Godot addon bind host: 127.0.0.1
Godot addon port: 7777
gdctl default host: 127.0.0.1
```

Inside a devcontainer, `127.0.0.1` means the container itself. To reach Godot running on the Windows host, connect through Docker Desktop's host route:

```bash
export GDCTL_BRIDGE_HOST=host.docker.internal
```

If a project was previously configured with:

```text
godot_tcp_bridge/host = "0.0.0.0"
```

the addon will preserve that setting across updates. New projects default to `127.0.0.1`.

## 2. Token Flow

After enabling the Godot addon, open the **gdctl Bridge** dock in Godot and copy the export command:

```bash
export GDCTL_BRIDGE_TOKEN=<token>
```

For devcontainer use:

```bash
export GDCTL_BRIDGE_HOST=host.docker.internal
export GDCTL_BRIDGE_TOKEN=<token copied from Godot>
```

Then verify:

```bash
go run ./cmd/gdctl bridge info
go run ./cmd/gdctl addon status
go run ./cmd/gdctl addon doctor
```

## 3. Projectless Addon Updates

Once the running addon supports `addon.update`, update it from the devcontainer without a mounted Godot project:

```bash
go run ./cmd/gdctl addon update
```

Equivalent command:

```bash
go run ./cmd/gdctl bridge addon-update
```

Expected output:

```text
Addon updated over bridge: 4 files written
Backup: res://addons/.godot_tcp_bridge_backup/<timestamp>/
Reload required: disable/enable the Godot plugin or restart Godot
```

After an addon update, disable/enable the plugin in Godot or restart Godot.

If the command returns:

```text
UNKNOWN_ENDPOINT: Unknown bridge endpoint
```

the running addon is too old for projectless update. Manually update/install the addon once, reload it, then projectless updates should work afterward.

## 4. Scene Path Smoke Test

Open or create a scene in Godot, then inspect it:

```bash
go run ./cmd/gdctl scene tree
```

Raw logical paths should look like:

```text
/root/Node3D
/root/Node3D/GridMap
```

Test an authenticated mutation without changing the scene:

```bash
go run ./cmd/gdctl node add \
  --parent /root/Node3D \
  --type Node3D \
  --name GdctlSmoke \
  --dry-run
```

Expected output:

```text
Dry run ok: /root/Node3D/GdctlSmoke
```

Scene save is currently disabled because direct Godot editor save calls crashed the editor from the bridge request handler. Save from Godot manually while bridge-safe save is implemented.

## 5. Common Failure Modes

`127.0.0.1:7777: connect: connection refused`

```text
You are probably inside the devcontainer. Use GDCTL_BRIDGE_HOST=host.docker.internal.
```

`UNAUTHORIZED: Mutation endpoint requires bearer token`

```text
Set GDCTL_BRIDGE_TOKEN from the gdctl Bridge dock, or pass --token explicitly.
```

`NO_SCENE_OPEN: No edited scene is open`

```text
Open or create a scene in Godot before using scene tree or node mutation commands.
```

`UNKNOWN_ENDPOINT: Unknown bridge endpoint`

```text
The running addon is older than the CLI command. Manually update/reload the addon once.
```

## 6. Godot Crash Case: Scene Save

Status:

```text
gdctl scene save is temporarily disabled.
```

What happened:

```text
Calling Godot editor scene-save APIs from the bridge request handler crashed/restarted Godot.
Both direct save_scene() and attempted save-as approaches proved unsafe in this bridge context.
```

Current safe behavior:

```text
The CLI rejects gdctl scene save before making a bridge request.
The bridge endpoint returns SCENE_SAVE_UNSUPPORTED before calling any Godot save APIs.
```

Recovery steps after a crash:

```text
1. Restart Godot if needed.
2. Manually copy or install the latest addon version that disables scene save.
3. Enable the plugin.
4. Verify the bridge:
   go run ./cmd/gdctl bridge info
5. Save scenes manually in Godot for now.
```

Do not retry save implementation by directly calling these from the request handler:

```text
EditorInterface.save_scene()
EditorInterface.save_scene_as(...)
PackedScene.pack(root) + ResourceSaver.save(...)
```

Before re-enabling CLI scene save, design a safer implementation. Candidate directions:

```text
Use deferred editor-main-thread execution with explicit state/result polling.
Use an EditorScript or Godot command-line batch script outside the live editor plugin.
Create scene/resource files through controlled file/resource operations instead of saving the active editor scene.
```

Acceptance criteria for revisiting scene save:

```text
It must not crash/restart Godot.
It must return structured success/error responses.
It must work for already-saved scenes before supporting save-as.
It must be tested manually on a disposable project before updating the normal addon.
```

## 7. Current Property Editing Step

The bridge bootstrap and projectless update loop are working. Logical scene paths are working.
The addon server has started moving reusable logic out of `bridge_server.gd`; typed CLI/Godot value conversion now lives in `addons/godot_tcp_bridge/typed_values.gd`.

The next implemented bridge primitive is:

```text
node set/get
```

That will let generated scenes configure positions, resources, collision shapes, and exported script parameters.

Use typed JSON values:

```bash
gdctl node set --path /root/Main/Player --property position --value '{"kind":"Vector2","value":[200,400]}'
gdctl node get --path /root/Main/Player --property position
```

Scene saving should be revisited later with a deferred/editor-safe implementation.
