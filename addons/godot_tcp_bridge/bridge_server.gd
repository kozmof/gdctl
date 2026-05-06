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

var editor_plugin: EditorPlugin
var typed_values = TypedValues.new()
var node_commands = NodeCommands.new()
var addon_update = AddonUpdate.new()
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
		return
	running = true
	print("Godot TCP Bridge listening on %s:%d" % [host, port])


func stop() -> void:
	for client in clients:
		var peer: StreamPeerTCP = client.get("peer")
		if peer:
			peer.disconnect_from_host()
	clients.clear()
	tcp_server.stop()
	running = false


func restart() -> void:
	stop()
	start()


func get_token() -> String:
	return token


func reset_token() -> String:
	token = _generate_token()
	_save_token(token)
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
		var request := _try_parse_request(client["buffer"])
		if request.is_empty():
			remaining.append(client)
			continue
		var response := _handle_request(request)
		_write_response(peer, response)
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


func _try_parse_request(buffer: PackedByteArray) -> Dictionary:
	var text := buffer.get_string_from_utf8()
	var header_end := text.find("\r\n\r\n")
	if header_end == -1:
		return {}

	var header_text := text.substr(0, header_end)
	var lines := header_text.split("\r\n")
	if lines.is_empty():
		return {}
	var request_line := lines[0].split(" ")
	if request_line.size() < 2:
		return {"method": "", "path": "", "headers": {}, "body": ""}

	var headers := {}
	var content_length := 0
	for i in range(1, lines.size()):
		var line := String(lines[i])
		var idx := line.find(":")
		if idx == -1:
			continue
		var name := line.substr(0, idx).strip_edges().to_lower()
		var value := line.substr(idx + 1).strip_edges()
		headers[name] = value
		if name == "content-length":
			content_length = int(value)

	var body_start := header_end + 4
	var body := text.substr(body_start)
	if body.length() < content_length:
		return {}
	body = body.substr(0, content_length)
	return {
		"method": String(request_line[0]),
		"path": String(request_line[1]),
		"headers": headers,
		"body": body,
	}


func _handle_request(request: Dictionary) -> Dictionary:
	var method := String(request.get("method", ""))
	var path := String(request.get("path", ""))
	if path.find("?") != -1:
		path = path.substr(0, path.find("?"))

	if method == "GET" and path == "/ping":
		return _http_json(200, _ping())
	if method == "GET" and path == "/scene/tree":
		var root := _edited_scene_root()
		if root == null:
			return _bridge_error(409, "", "NO_SCENE_OPEN", "No edited scene is open", {})
		return _http_json(200, {"ok": true, "root": _node_info(root)})
	if method == "POST" and path == "/scene/save":
		return _handle_scene_save(request)
	if method == "POST" and path == "/node/add":
		return node_commands.handle_add(request, _command_context())
	if method == "POST" and path == "/node/remove":
		return node_commands.handle_remove(request, _command_context())
	if method == "POST" and path == "/node/get":
		return node_commands.handle_get(request, _command_context())
	if method == "POST" and path == "/node/set":
		return node_commands.handle_set(request, _command_context())
	if method == "POST" and path == "/addon/update":
		return addon_update.handle_update(request, _command_context())
	return _bridge_error(404, "", "UNKNOWN_ENDPOINT", "Unknown bridge endpoint", {"method": method, "path": path})


func _handle_scene_save(request: Dictionary) -> Dictionary:
	var body: Dictionary = _json_body_or_error(request)
	if body.has("error_response"):
		return body["error_response"]
	if not _authorized(request):
		return _bridge_error(401, body.get("request_id", ""), "UNAUTHORIZED", "Scene save requires bearer token", {})
	if body.get("op", "") != "scene.save":
		return _bridge_error(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected scene.save operation", {})

	var root := _edited_scene_root()
	if root == null:
		return _bridge_error(409, body.get("request_id", ""), "NO_SCENE_OPEN", "No edited scene is open", {})
	if editor_plugin == null:
		return _bridge_error(500, body.get("request_id", ""), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})

	return _bridge_error(501, body.get("request_id", ""), "SCENE_SAVE_UNSUPPORTED", "Scene save is temporarily disabled because direct editor save calls are unstable in the bridge request handler", {"root": _logical_path(root), "path": root.scene_file_path})


func _json_body_or_error(request: Dictionary) -> Dictionary:
	var headers: Dictionary = request.get("headers", {})
	var content_type := String(headers.get("content-type", ""))
	if not content_type.to_lower().begins_with("application/json"):
		return {"error_response": _bridge_error(415, "", "UNSUPPORTED_MEDIA_TYPE", "Mutation endpoints accept only application/json", {})}
	var parsed := JSON.parse_string(String(request.get("body", "")))
	if typeof(parsed) != TYPE_DICTIONARY:
		return {"error_response": _bridge_error(400, "", "INVALID_JSON", "Request body must be a JSON object", {})}
	var parsed_dict: Dictionary = parsed
	return parsed_dict


func _params_or_empty(body: Dictionary) -> Dictionary:
	var params := body.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		return {}
	return params


func _command_context() -> Dictionary:
	return {
		"json_body_or_error": Callable(self, "_json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_ok": Callable(self, "_bridge_ok"),
		"bridge_error": Callable(self, "_bridge_error"),
		"edited_scene_root": Callable(self, "_edited_scene_root"),
		"node_by_path": Callable(self, "_node_by_path"),
		"logical_path": Callable(self, "_logical_path"),
		"mark_scene_dirty": Callable(self, "_mark_scene_dirty"),
		"typed_values": typed_values,
		"addon_root": ADDON_ROOT,
		"addon_backup_root": ADDON_BACKUP_ROOT,
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
			"node.get",
			"node.set",
			"addon.update",
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


func _bridge_ok(request_id: String, result: Dictionary) -> Dictionary:
	return _http_json(200, {
		"request_id": request_id,
		"ok": true,
		"result": result,
		"error": null,
	})


func _bridge_error(status: int, request_id: String, code: String, message: String, detail: Dictionary) -> Dictionary:
	return _http_json(status, {
		"request_id": request_id,
		"ok": false,
		"result": null,
		"error": {
			"code": code,
			"message": message,
			"detail": detail,
		},
	})


func _http_json(status: int, body: Dictionary) -> Dictionary:
	return {"status": status, "body": body}


func _write_response(peer: StreamPeerTCP, response: Dictionary) -> void:
	var status := int(response.get("status", 500))
	var body := JSON.stringify(response.get("body", {}))
	var reason := "OK"
	if status == 400:
		reason = "Bad Request"
	elif status == 401:
		reason = "Unauthorized"
	elif status == 404:
		reason = "Not Found"
	elif status == 409:
		reason = "Conflict"
	elif status == 415:
		reason = "Unsupported Media Type"
	elif status >= 500:
		reason = "Internal Server Error"

	var head := "HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n" % [status, reason, body.to_utf8_buffer().size()]
	peer.put_data((head + body).to_utf8_buffer())
