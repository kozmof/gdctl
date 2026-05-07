@tool
extends RefCounted

const PLUGIN_VERSION := "0.1.0"
const PROTOCOL_VERSION := "gdctl.v1"
const DEFAULT_HOST := "127.0.0.1"
const DEFAULT_PORT := 7777
const TOKEN_PATH := "res://.godot-bridge-token"
const ADDON_ROOT := "res://addons/godot_tcp_bridge/"
const ADDON_BACKUP_ROOT := "res://addons/.godot_tcp_bridge_backup/"
const TypedValues = preload("res://addons/godot_tcp_bridge/typed_values.gd")
const NodeCommands = preload("res://addons/godot_tcp_bridge/node_commands.gd")
const AddonUpdate = preload("res://addons/godot_tcp_bridge/addon_update.gd")
const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const LogBuffer = preload("res://addons/godot_tcp_bridge/log_buffer.gd")

var editor_plugin: EditorPlugin
var typed_values = TypedValues.new()
var node_commands = NodeCommands.new()
var addon_update = AddonUpdate.new()
var protocol = Protocol.new()
var log_buffer = LogBuffer.new()
var tcp_server := TCPServer.new()
var clients: Array[Dictionary] = []
var host := DEFAULT_HOST
var port := DEFAULT_PORT
var auth_enabled := true
var token := ""
var running := false


func start() -> void:
	host = str(_setting("godot_tcp_bridge/host", DEFAULT_HOST))
	port = int(_setting("godot_tcp_bridge/port", DEFAULT_PORT))
	auth_enabled = bool(_setting("godot_tcp_bridge/auth_enabled", true))
	token = _load_or_create_token()

	var bind := "*"
	if host != "0.0.0.0":
		bind = host
	var err := tcp_server.listen(port, bind)
	if err != OK:
		push_error("Godot TCP Bridge failed to listen on %s:%d: %s" % [host, port, error_string(err)])
		log_buffer.add("error", "bridge.start", "Failed to listen", {"host": host, "port": port, "error": error_string(err)})
		return
	running = true
	print("Godot TCP Bridge listening on %s:%d" % [host, port])
	log_buffer.add("info", "bridge.start", "Listening", {"host": host, "port": port, "auth_enabled": auth_enabled})


func stop() -> void:
	for client in clients:
		var peer: StreamPeerTCP = client.get("peer")
		if peer:
			peer.disconnect_from_host()
	clients.clear()
	tcp_server.stop()
	running = false
	log_buffer.add("info", "bridge.stop", "Stopped", {})


func restart() -> void:
	stop()
	start()


func get_token() -> String:
	return token


func reset_token() -> String:
	token = _generate_token()
	_save_token(token)
	log_buffer.add("info", "bridge.token", "Token reset", {})
	return token


func poll() -> void:
	if not running:
		return
	if tcp_server.is_connection_available():
		var peer := tcp_server.take_connection()
		clients.append({"peer": peer, "buffer": PackedByteArray()})

	var remaining: Array[Dictionary] = []
	for client in clients:
		var peer: StreamPeerTCP = client.get("peer")
		if peer.get_status() != StreamPeerTCP.STATUS_CONNECTED:
			continue
		var available := peer.get_available_bytes()
		if available > 0:
			var chunk := peer.get_data(available)
			if chunk[0] == OK:
				var buffer: PackedByteArray = client["buffer"]
				buffer.append_array(chunk[1])
				client["buffer"] = buffer
		var request: Dictionary = protocol.parse_request(client["buffer"])
		if request.is_empty():
			remaining.append(client)
			continue
		_log_request(request)
		var response: Dictionary = _handle_request(request)
		_log_response(request, response)
		protocol.write_response(peer, response)
		peer.disconnect_from_host()
	clients = remaining


func _setting(key: String, default_value: Variant) -> Variant:
	if ProjectSettings.has_setting(key):
		return ProjectSettings.get_setting(key)
	ProjectSettings.set_setting(key, default_value)
	return default_value


func _load_or_create_token() -> String:
	if FileAccess.file_exists(TOKEN_PATH):
		return FileAccess.get_file_as_string(TOKEN_PATH).strip_edges()
	var generated := _generate_token()
	_save_token(generated)
	return generated


func _generate_token() -> String:
	randomize()
	return "%d-%d" % [Time.get_unix_time_from_system(), randi()]


func _save_token(value: String) -> void:
	var file := FileAccess.open(TOKEN_PATH, FileAccess.WRITE)
	if file:
		file.store_string(value + "\n")
		file.close()


func _handle_request(request: Dictionary) -> Dictionary:
	var method := String(request.get("method", ""))
	var path := String(request.get("path", ""))
	if path.find("?") != -1:
		path = path.substr(0, path.find("?"))

	if method == "GET" and path == "/ping":
		return protocol.http_json(200, _ping())
	if method == "GET" and path == "/logs":
		return _handle_logs(request)
	if method == "POST" and path == "/logs/clear":
		return _handle_logs_clear(request)
	if method == "GET" and path == "/scene/tree":
		var root := _edited_scene_root()
		if root == null:
			return protocol.bridge_error(409, "", "NO_SCENE_OPEN", "No edited scene is open", {})
		return protocol.http_json(200, {"ok": true, "root": _node_info(root)})
	if method == "POST" and path == "/scene/save":
		return _handle_scene_save(request)
	if method == "POST" and path == "/node/add":
		return node_commands.handle_add(request, _command_context())
	if method == "POST" and path == "/node/remove":
		return node_commands.handle_remove(request, _command_context())
	if method == "POST" and path == "/node/rename":
		return node_commands.handle_rename(request, _command_context())
	if method == "POST" and path == "/node/move":
		return node_commands.handle_move(request, _command_context())
	if method == "POST" and path == "/node/get":
		return node_commands.handle_get(request, _command_context())
	if method == "POST" and path == "/node/set":
		return node_commands.handle_set(request, _command_context())
	if method == "POST" and path == "/addon/update":
		return addon_update.handle_update(request, _command_context())
	return protocol.bridge_error(404, "", "UNKNOWN_ENDPOINT", "Unknown bridge endpoint", {"method": method, "path": path})


func _handle_scene_save(request: Dictionary) -> Dictionary:
	var body: Dictionary = protocol.json_body_or_error(request)
	if body.has("error_response"):
		return body["error_response"]
	if not _authorized(request):
		return protocol.bridge_error(401, body.get("request_id", ""), "UNAUTHORIZED", "Scene save requires bearer token", {})
	if body.get("op", "") != "scene.save":
		return protocol.bridge_error(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected scene.save operation", {})

	var root := _edited_scene_root()
	if root == null:
		return protocol.bridge_error(409, body.get("request_id", ""), "NO_SCENE_OPEN", "No edited scene is open", {})
	if editor_plugin == null:
		return protocol.bridge_error(500, body.get("request_id", ""), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})

	return protocol.bridge_error(501, body.get("request_id", ""), "SCENE_SAVE_UNSUPPORTED", "Scene save is temporarily disabled because direct editor save calls are unstable in the bridge request handler", {"root": _logical_path(root), "path": root.scene_file_path})


func _handle_logs(request: Dictionary) -> Dictionary:
	if not _authorized(request):
		return protocol.bridge_error(401, "", "UNAUTHORIZED", "Bridge logs require bearer token", {})
	return protocol.http_json(200, {"ok": true, "entries": log_buffer.list()})


func _handle_logs_clear(request: Dictionary) -> Dictionary:
	if not _authorized(request):
		return protocol.bridge_error(401, "", "UNAUTHORIZED", "Bridge log clearing requires bearer token", {})
	log_buffer.clear()
	log_buffer.add("info", "bridge.logs", "Logs cleared", {})
	return protocol.bridge_ok("", {"cleared": true})


func _params_or_empty(body: Dictionary) -> Dictionary:
	var params := body.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		return {}
	return params


func _command_context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"edited_scene_root": Callable(self, "_edited_scene_root"),
		"node_by_path": Callable(self, "_node_by_path"),
		"logical_path": Callable(self, "_logical_path"),
		"mark_scene_dirty": Callable(self, "_mark_scene_dirty"),
		"typed_values": typed_values,
		"addon_root": ADDON_ROOT,
		"addon_backup_root": ADDON_BACKUP_ROOT,
		"log": Callable(log_buffer, "add"),
	}


func _authorized(request: Dictionary) -> bool:
	if not auth_enabled:
		return true
	var headers: Dictionary = request.get("headers", {})
	var expected := "Bearer " + token
	return String(headers.get("authorization", "")) == expected


func _ping() -> Dictionary:
	var root := _edited_scene_root()
	return {
		"ok": true,
		"service": "godot-bridge",
		"engine": "Godot",
		"engine_version": Engine.get_version_info().get("string", ""),
		"plugin_version": PLUGIN_VERSION,
		"project_name": ProjectSettings.get_setting("application/config/name", ""),
		"project_path": ProjectSettings.globalize_path("res://"),
		"scene_open": root != null,
		"auth_enabled": auth_enabled,
		"host": host,
		"port": port,
		"protocol_version": PROTOCOL_VERSION,
		"capabilities": [
			"ping",
			"scene.tree",
			"node.add",
			"node.remove",
			"node.rename",
			"node.move",
			"node.get",
			"node.set",
			"addon.update",
			"bridge.logs",
		],
	}


func _edited_scene_root() -> Node:
	if editor_plugin == null:
		return null
	return editor_plugin.get_editor_interface().get_edited_scene_root()


func _node_by_path(path: String) -> Node:
	var root := _edited_scene_root()
	if root == null:
		return null
	if path == _logical_path(root) or path == String(root.get_path()):
		return root
	var stack: Array[Node] = [root]
	while not stack.is_empty():
		var node := stack.pop_back()
		if _logical_path(node) == path or String(node.get_path()) == path:
			return node
		for child in node.get_children():
			stack.append(child)
	return null


func _node_info(node: Node) -> Dictionary:
	var children := []
	for child in node.get_children():
		children.append(_node_info(child))
	return {
		"name": node.name,
		"type": node.get_class(),
		"path": _logical_path(node),
		"children": children,
	}


func _logical_path(node: Node) -> String:
	var root := _edited_scene_root()
	if root == null:
		return ""
	if node == root:
		return "/root/%s" % root.name
	if not root.is_ancestor_of(node):
		return String(node.get_path())
	var names: Array[String] = []
	var current := node
	while current != null and current != root:
		names.push_front(String(current.name))
		current = current.get_parent()
	names.push_front(String(root.name))
	return "/root/" + "/".join(names)


func _mark_scene_dirty() -> void:
	if editor_plugin:
		editor_plugin.get_editor_interface().mark_scene_as_unsaved()


func _log_request(request: Dictionary) -> void:
	var path: String = String(request.get("path", ""))
	if path == "/ping" or path == "/logs":
		return
	log_buffer.add("debug", "bridge.request", "Request received", {
		"method": String(request.get("method", "")),
		"path": path,
	})


func _log_response(request: Dictionary, response: Dictionary) -> void:
	var path: String = String(request.get("path", ""))
	if path == "/ping" or path == "/logs":
		return
	var status: int = int(response.get("status", 500))
	var body: Dictionary = response.get("body", {})
	if status >= 400:
		var error_value: Variant = body.get("error", {})
		var error_detail: Dictionary = {}
		if typeof(error_value) == TYPE_DICTIONARY:
			error_detail = error_value
		log_buffer.add("error", "bridge.response", "Request failed", {
			"method": String(request.get("method", "")),
			"path": path,
			"status": status,
			"error": error_detail,
		})
	else:
		log_buffer.add("debug", "bridge.response", "Request completed", {
			"method": String(request.get("method", "")),
			"path": path,
			"status": status,
		})
