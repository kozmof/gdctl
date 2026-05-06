@tool
extends RefCounted

const PLUGIN_VERSION := "0.1.0"
const PROTOCOL_VERSION := "gdctl.v1"
const DEFAULT_HOST := "127.0.0.1"
const DEFAULT_PORT := 7777
const TOKEN_PATH := "res://.godot-bridge-token"
const ADDON_ROOT := "res://addons/godot_tcp_bridge/"
const ADDON_BACKUP_ROOT := "res://addons/.godot_tcp_bridge_backup/"

var editor_plugin: EditorPlugin
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
		return _handle_node_add(request)
	if method == "POST" and path == "/node/remove":
		return _handle_node_remove(request)
	if method == "POST" and path == "/addon/update":
		return _handle_addon_update(request)
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


func _handle_node_add(request: Dictionary) -> Dictionary:
	var body: Dictionary = _json_body_or_error(request)
	if body.has("error_response"):
		return body["error_response"]
	if not _authorized(request):
		return _bridge_error(401, body.get("request_id", ""), "UNAUTHORIZED", "Mutation endpoint requires bearer token", {})
	if body.get("op", "") != "node.add":
		return _bridge_error(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected node.add operation", {})

	var params := _params_or_empty(body)
	var parent_path := String(params.get("parent", ""))
	var type_name := String(params.get("type", ""))
	var node_name := String(params.get("name", ""))
	var dry_run := bool(params.get("dry_run", false))
	var parent := _node_by_path(parent_path)
	if parent == null:
		return _bridge_error(404, body.get("request_id", ""), "NODE_PARENT_NOT_FOUND", "Parent node does not exist", {"parent": parent_path})
	if type_name == "" or not ClassDB.can_instantiate(type_name):
		return _bridge_error(400, body.get("request_id", ""), "NODE_TYPE_INVALID", "Node type cannot be instantiated", {"type": type_name})
	if not ClassDB.is_parent_class(type_name, "Node") and type_name != "Node":
		return _bridge_error(400, body.get("request_id", ""), "NODE_TYPE_INVALID", "Node type must inherit Node", {"type": type_name})
	if node_name == "" or not node_name.is_valid_identifier():
		return _bridge_error(400, body.get("request_id", ""), "NODE_NAME_INVALID", "Node name must be a valid identifier", {"name": node_name})

	var path := "%s/%s" % [parent_path.rstrip("/"), node_name]
	if dry_run:
		return _bridge_ok(body.get("request_id", ""), {"path": path, "dry_run": true})

	var node := ClassDB.instantiate(type_name) as Node
	node.name = node_name
	parent.add_child(node)
	node.owner = _edited_scene_root()
	_mark_scene_dirty()
	return _bridge_ok(body.get("request_id", ""), {"path": _logical_path(node)})


func _handle_node_remove(request: Dictionary) -> Dictionary:
	var body := _json_body_or_error(request)
	if body.has("error_response"):
		return body["error_response"]
	if not _authorized(request):
		return _bridge_error(401, body.get("request_id", ""), "UNAUTHORIZED", "Mutation endpoint requires bearer token", {})
	if body.get("op", "") != "node.remove":
		return _bridge_error(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected node.remove operation", {})

	var params := _params_or_empty(body)
	var path := String(params.get("path", ""))
	var dry_run := bool(params.get("dry_run", false))
	var node := _node_by_path(path)
	if node == null:
		return _bridge_error(404, body.get("request_id", ""), "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if node == _edited_scene_root():
		return _bridge_error(400, body.get("request_id", ""), "CANNOT_REMOVE_SCENE_ROOT", "Scene root cannot be removed", {"path": path})
	if dry_run:
		return _bridge_ok(body.get("request_id", ""), {"removed": path, "dry_run": true})

	node.get_parent().remove_child(node)
	node.queue_free()
	_mark_scene_dirty()
	return _bridge_ok(body.get("request_id", ""), {"removed": path})


func _handle_addon_update(request: Dictionary) -> Dictionary:
	var body: Dictionary = _json_body_or_error(request)
	if body.has("error_response"):
		return body["error_response"]
	if not _authorized(request):
		return _bridge_error(401, body.get("request_id", ""), "UNAUTHORIZED", "Addon update requires bearer token", {})
	if body.get("op", "") != "addon.update":
		return _bridge_error(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected addon.update operation", {})

	var params: Dictionary = _params_or_empty(body)
	var manifest: Variant = params.get("manifest", {})
	if typeof(manifest) != TYPE_DICTIONARY:
		return _bridge_error(400, body.get("request_id", ""), "INVALID_MANIFEST", "Addon update requires a manifest object", {})
	var manifest_dict: Dictionary = manifest
	var manifest_files_value: Variant = manifest_dict.get("files", [])
	if typeof(manifest_files_value) != TYPE_ARRAY:
		return _bridge_error(400, body.get("request_id", ""), "INVALID_MANIFEST", "Manifest files must be an array", {})
	var manifest_files: Array = manifest_files_value
	var files_value: Variant = params.get("files", [])
	if typeof(files_value) != TYPE_ARRAY:
		return _bridge_error(400, body.get("request_id", ""), "INVALID_FILES", "Addon update files must be an array", {})
	var files: Array = files_value

	var allowed: Dictionary = {}
	for item in manifest_files:
		var rel: String = String(item)
		if not _is_safe_addon_path(rel):
			return _bridge_error(400, body.get("request_id", ""), "INVALID_ADDON_PATH", "Manifest contains an unsafe file path", {"path": rel})
		allowed[rel] = true

	var backup: String = _backup_addon_files(manifest_files)
	var written: int = 0
	for file_item in files:
		if typeof(file_item) != TYPE_DICTIONARY:
			return _bridge_error(400, body.get("request_id", ""), "INVALID_FILES", "Each addon update file must be an object", {})
		var file_dict: Dictionary = file_item
		var rel_path: String = String(file_dict.get("path", ""))
		if not _is_safe_addon_path(rel_path) or not allowed.has(rel_path):
			return _bridge_error(400, body.get("request_id", ""), "INVALID_ADDON_PATH", "Addon update file path is not allowed", {"path": rel_path})
		var content_base64: String = String(file_dict.get("content_base64", ""))
		var bytes: PackedByteArray = Marshalls.base64_to_raw(content_base64)
		if bytes.is_empty() and content_base64 != "":
			return _bridge_error(400, body.get("request_id", ""), "INVALID_FILE_CONTENT", "Addon update file content is not valid base64", {"path": rel_path})
		var write_err: Error = _write_addon_file(rel_path, bytes)
		if write_err != OK:
			return _bridge_error(500, body.get("request_id", ""), "ADDON_WRITE_FAILED", "Failed to write addon file", {"path": rel_path, "error": error_string(write_err)})
		written += 1

	return _bridge_ok(body.get("request_id", ""), {
		"updated": true,
		"files_written": written,
		"backup": backup,
		"reload_required": true,
	})


func _is_safe_addon_path(path: String) -> bool:
	if path == "":
		return false
	if path.begins_with("/") or path.begins_with("\\"):
		return false
	if path.find("..") != -1:
		return false
	if path.find(":") != -1:
		return false
	return true


func _backup_addon_files(files: Array) -> String:
	var timestamp: String = Time.get_datetime_string_from_system(true).replace(":", "-")
	var backup_root: String = ADDON_BACKUP_ROOT + timestamp + "/"
	var wrote_any: bool = false
	for item in files:
		var rel: String = String(item)
		if not _is_safe_addon_path(rel):
			continue
		var src: String = ADDON_ROOT + rel
		if not FileAccess.file_exists(src):
			continue
		var data: PackedByteArray = FileAccess.get_file_as_bytes(src)
		var dst: String = backup_root + rel
		var dir: String = dst.get_base_dir()
		DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir))
		var file: FileAccess = FileAccess.open(dst, FileAccess.WRITE)
		if file:
			file.store_buffer(data)
			file.close()
			wrote_any = true
	if not wrote_any:
		return ""
	return backup_root


func _write_addon_file(path: String, bytes: PackedByteArray) -> Error:
	var dst: String = ADDON_ROOT + path
	var dir: String = dst.get_base_dir()
	var dir_err: Error = DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir))
	if dir_err != OK:
		return dir_err
	var file: FileAccess = FileAccess.open(dst, FileAccess.WRITE)
	if file == null:
		return ERR_CANT_OPEN
	file.store_buffer(bytes)
	file.close()
	return OK


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
			"scene.save",
			"node.add",
			"node.remove",
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
