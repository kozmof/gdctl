Below is a draft plan for **Godot-over-TCP communication from a devcontainer**.

# Draft: Godot TCP Bridge Plan

## 1. Goal

Create a communication path between:

```text
Devcontainer
  Go CLI tool

Windows Host
  Godot Editor
```

The devcontainer does not directly execute `Godot.exe`. Instead, it communicates with a Godot-side bridge over TCP.

```text
+-----------------------------+
| Linux Devcontainer          |
|                             |
|  godotctl / turnout-godot   |
|        |                    |
|        | TCP / HTTP / JSON   |
|        v                    |
+-----------------------------+
          host.docker.internal:7777
+-----------------------------+
| Windows Host                |
|                             |
|  Godot Editor               |
|    + Godot Bridge Plugin    |
|                             |
+-----------------------------+
```

## 2. Main Assumption

Godot runs as a normal Windows GUI application.

The CLI runs inside a Linux devcontainer.

Communication happens through:

```text
host.docker.internal:<port>
```

Example:

```bash
curl http://host.docker.internal:7777/ping
```

## 3. Components

## 3.1 Go CLI

Responsibilities:

```text
- ping Godot
- run doctor checks
- inspect Godot bridge status
- request scene tree snapshots
- request node operations
- receive structured responses
```

Example commands:

```bash
godotctl ping

godotctl doctor

godotctl scene tree

godotctl node add \
  --parent /root/Main \
  --type Node2D \
  --name EnemySpawner

godotctl node remove \
  --path /root/Main/EnemySpawner
```

## 3.2 Godot Bridge Plugin

A Godot editor plugin that opens a local TCP/HTTP server.

Responsibilities:

```text
- listen on a configured TCP port
- expose bridge endpoints
- validate requests
- operate on the currently opened project
- return structured JSON responses
- never expose dangerous operations by default
```

The plugin runs inside the Godot editor process.

## 3.3 Optional Windows Launcher

A separate Windows-side launcher may exist later.

Responsibilities:

```text
- start Godot.exe
- open a specific project
- report whether Godot is already running
```

This is optional. The first version can assume Godot is already open.

---

# 4. Transport Choice

For the first implementation, use **HTTP over TCP with JSON**.

Reason:

```text
- easy to debug with curl
- easy to implement in Go
- easy to implement in Godot
- works through host.docker.internal
- human-readable
- enough for doctor/ping/scene inspection
```

Later versions can switch to:

```text
- WebSocket
- raw TCP frames
- JSON-RPC
- msgpack
- protobuf
```

But HTTP+JSON is best for the first bridge.

---

# 5. Default Network Settings

## 5.1 Godot Bridge

```hcl
bridge {
  host = "127.0.0.1"
  port = 7777
}
```

However, for devcontainer access, `127.0.0.1` may only mean Windows localhost from the Windows side. Depending on Docker Desktop networking behavior, the bridge may need to listen on:

```hcl
bridge {
  host = "0.0.0.0"
  port = 7777
}
```

Then the container connects to:

```hcl
client {
  host = "host.docker.internal"
  port = 7777
}
```

## 5.2 CLI Config

```hcl
godot {
  bridge {
    host = "host.docker.internal"
    port = 7777
    protocol = "http"
  }
}
```

---

# 6. Minimum API

## 6.1 Ping

```http
GET /ping
```

Response:

```json
{
  "ok": true,
  "service": "godot-bridge",
  "engine": "Godot",
  "engine_version": "4.4.1",
  "plugin_version": "0.1.0",
  "project_name": "my-game",
  "project_path": "C:/Users/me/projects/my-game"
}
```

CLI command:

```bash
godotctl ping
```

Expected output:

```text
Godot bridge: ok
Engine: Godot 4.4.1
Project: my-game
Plugin: 0.1.0
```

## 6.2 Doctor

The Go CLI performs checks from the container side.

```bash
godotctl doctor
```

Checks:

```text
- can resolve host.docker.internal
- can connect to bridge port
- /ping returns ok
- Godot project is open
- plugin version is compatible
- path mapping is configured
- optional Linux headless Godot exists
```

Example output:

```text
Godot TCP Bridge Doctor

[ok] host.docker.internal resolved
[ok] bridge reachable at host.docker.internal:7777
[ok] ping returned ok
[ok] Godot editor project is open
[ok] plugin version 0.1.0
[warn] no Linux headless Godot configured
[ok] path map found

Result: usable
```

## 6.3 Scene Tree Snapshot

```http
GET /scene/tree
```

Response:

```json
{
  "ok": true,
  "root": {
    "name": "Main",
    "type": "Node2D",
    "path": "/root/Main",
    "children": [
      {
        "name": "Player",
        "type": "CharacterBody2D",
        "path": "/root/Main/Player",
        "children": [
          {
            "name": "Camera2D",
            "type": "Camera2D",
            "path": "/root/Main/Player/Camera2D",
            "children": []
          }
        ]
      }
    ]
  }
}
```

CLI:

```bash
godotctl scene tree
```

Output:

```text
Main Node2D
└── Player CharacterBody2D
    └── Camera2D Camera2D
```

## 6.4 Add Node

```http
POST /node/add
```

Request:

```json
{
  "parent": "/root/Main",
  "type": "Node2D",
  "name": "EnemySpawner"
}
```

Response:

```json
{
  "ok": true,
  "path": "/root/Main/EnemySpawner"
}
```

CLI:

```bash
godotctl node add \
  --parent /root/Main \
  --type Node2D \
  --name EnemySpawner
```

## 6.5 Remove Node

```http
POST /node/remove
```

Request:

```json
{
  "path": "/root/Main/EnemySpawner"
}
```

Response:

```json
{
  "ok": true,
  "removed": "/root/Main/EnemySpawner"
}
```

## 6.6 Save Scene

```http
POST /scene/save
```

Request:

```json
{
  "path": "res://scenes/main.tscn"
}
```

Response:

```json
{
  "ok": true,
  "saved": "res://scenes/main.tscn"
}
```

---

# 7. Path Mapping

The container and Windows Godot see different filesystem paths.

```text
Container:
  /workspace/my-game

Windows:
  C:\Users\me\projects\my-game

Godot resource path:
  res://
```

The bridge should prefer `res://` paths whenever possible.

For external paths, define explicit mapping:

```hcl
path_map {
  container = "/workspace/my-game"
  windows   = "C:\\Users\\me\\projects\\my-game"
}
```

Example conversion:

```text
/container:
/workspace/my-game/scenes/main.tscn

/windows:
C:\Users\me\projects\my-game\scenes\main.tscn

/godot:
res://scenes/main.tscn
```

The protocol should mostly avoid absolute paths and use:

```text
- node paths
- res:// paths
- project-relative paths
- stable ids where possible
```

---

# 8. Security Model

The bridge should not be exposed publicly.

Default:

```text
- bind to localhost or trusted interface only
- use allowlist token
- accept only JSON
- reject unknown commands
- disable file write operations unless explicitly enabled
```

Example config:

```hcl
bridge {
  host = "0.0.0.0"
  port = 7777

  auth {
    mode = "token"
    token_file = "res://.godot-bridge-token"
  }

  permissions {
    inspect_scene = true
    mutate_scene  = true
    save_scene    = false
    run_game      = false
    file_write    = false
  }
}
```

Request header:

```http
Authorization: Bearer <token>
```

For local development, the first prototype may allow unauthenticated `GET /ping`, but mutation endpoints should require a token.

---

# 9. Request Envelope

Instead of designing every endpoint independently, mutation APIs can use a common envelope.

```json
{
  "request_id": "cli-000001",
  "op": "node.add",
  "params": {
    "parent": "/root/Main",
    "type": "Node2D",
    "name": "EnemySpawner"
  }
}
```

Response:

```json
{
  "request_id": "cli-000001",
  "ok": true,
  "result": {
    "path": "/root/Main/EnemySpawner"
  },
  "error": null
}
```

Error response:

```json
{
  "request_id": "cli-000001",
  "ok": false,
  "result": null,
  "error": {
    "code": "NODE_PARENT_NOT_FOUND",
    "message": "Parent node does not exist",
    "detail": {
      "parent": "/root/Main/Missing"
    }
  }
}
```

This is close to JSON-RPC but simpler.

---

# 10. Initial Go Types

```go
type PingResponse struct {
	OK            bool   `json:"ok"`
	Service       string `json:"service"`
	Engine        string `json:"engine"`
	EngineVersion string `json:"engine_version"`
	PluginVersion string `json:"plugin_version"`
	ProjectName   string `json:"project_name"`
	ProjectPath   string `json:"project_path"`
}

type BridgeError struct {
	Code    string         `json:"code"`
	Message string        `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

type BridgeResponse[T any] struct {
	RequestID string       `json:"request_id,omitempty"`
	OK        bool         `json:"ok"`
	Result    T            `json:"result,omitempty"`
	Error     *BridgeError `json:"error,omitempty"`
}

type NodeInfo struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Path     string     `json:"path"`
	Children []NodeInfo `json:"children"`
}
```

---

# 11. Initial Godot Plugin Shape

Possible plugin layout:

```text
addons/godot_tcp_bridge/
  plugin.cfg
  bridge_plugin.gd
  bridge_server.gd
  handlers/
    ping_handler.gd
    scene_tree_handler.gd
    node_add_handler.gd
    node_remove_handler.gd
```

`plugin.cfg`:

```ini
[plugin]
name="Godot TCP Bridge"
description="Local TCP/HTTP bridge for external CLI tools"
author="kozmof"
version="0.1.0"
script="bridge_plugin.gd"
```

Conceptual Godot-side structure:

```gdscript
@tool
extends EditorPlugin

var server

func _enter_tree():
    server = preload("res://addons/godot_tcp_bridge/bridge_server.gd").new()
    server.start(7777)

func _exit_tree():
    if server:
        server.stop()
```

---

# 12. Milestones

## Milestone 1: Ping

Goal:

```text
CLI can confirm communication with Godot.
```

Deliverables:

```text
- Godot plugin starts local HTTP/TCP server
- GET /ping returns JSON
- Go CLI implements `godotctl ping`
- doctor checks connection
```

## Milestone 2: Scene Inspection

Goal:

```text
CLI can inspect current scene tree.
```

Deliverables:

```text
- GET /scene/tree
- Go CLI tree renderer
- error if no scene is open
```

## Milestone 3: Safe Node Mutation

Goal:

```text
CLI can add/remove nodes.
```

Deliverables:

```text
- POST /node/add
- POST /node/remove
- token auth for mutation
- dry-run mode
```

Example:

```bash
godotctl node add --parent /root/Main --type Node2D --name Marker --dry-run
```

## Milestone 4: Save and Reload

Goal:

```text
CLI can persist changes.
```

Deliverables:

```text
- POST /scene/save
- POST /scene/reload
- dirty state reporting
```

## Milestone 5: Higher-level DSL

Goal:

```text
Declarative node tree management.
```

Example:

```hcl
scene "res://scenes/main.tscn" {
  node "EnemySpawner" {
    parent = "/root/Main"
    type   = "Node2D"
  }

  node "DebugLabel" {
    parent = "/root/Main/UI"
    type   = "Label"
  }
}
```

Apply:

```bash
godotctl apply scene.hcl
```

---

# 13. Suggested First Scope

For the first prototype, avoid:

```text
- property editing
- script editing
- arbitrary file writes
- running game from CLI
- live bidirectional streaming
- complex authentication
```

Focus only on:

```text
- ping
- doctor
- scene tree read
- add node
- remove node
```

That matches your earlier scope: **inspect and node tree management only, no property editing**.

---

# 14. Summary

The recommended architecture is:

```text
Godot Editor Plugin
  exposes local HTTP/TCP bridge

Go CLI in Devcontainer
  connects to host.docker.internal

Protocol
  JSON request/response

Initial features
  ping
  doctor
  scene tree inspect
  add/remove nodes
```

This keeps the boundary clean:

```text
Windows Godot owns the editor/runtime.
Container CLI owns automation and declarative operations.
TCP protocol connects them.
```
