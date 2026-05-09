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
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const BridgeCommands = preload("res://addons/godot_tcp_bridge/commands/bridge_commands.gd")
const SceneCommands = preload("res://addons/godot_tcp_bridge/commands/scene_commands.gd")
const NodeCommands = preload("res://addons/godot_tcp_bridge/commands/node_commands.gd")
const ScriptCommands = preload("res://addons/godot_tcp_bridge/commands/script_commands.gd")
const ShaderCommands = preload("res://addons/godot_tcp_bridge/commands/shader_commands.gd")
const MaterialCommands = preload("res://addons/godot_tcp_bridge/commands/material_commands.gd")
const FileCommands = preload("res://addons/godot_tcp_bridge/commands/file_commands.gd")
const ViewportCommands = preload("res://addons/godot_tcp_bridge/commands/viewport_commands.gd")
const SignalCommands = preload("res://addons/godot_tcp_bridge/commands/signal_commands.gd")
const ProjectCommands = preload("res://addons/godot_tcp_bridge/commands/project_commands.gd")
const AddonUpdate = preload("res://addons/godot_tcp_bridge/addon_update.gd")
const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const LogBuffer = preload("res://addons/godot_tcp_bridge/log_buffer.gd")
const LogCommands = preload("res://addons/godot_tcp_bridge/commands/log_commands.gd")
const Jobs = preload("res://addons/godot_tcp_bridge/jobs.gd")

var editor_plugin: EditorPlugin
var typed_values = TypedValues.new()
var command_request = CommandRequest.new()
var bridge_commands = BridgeCommands.new()
var scene_commands = SceneCommands.new()
var node_commands = NodeCommands.new()
var script_commands = ScriptCommands.new()
var shader_commands = ShaderCommands.new()
var material_commands = MaterialCommands.new()
var file_commands = FileCommands.new()
var viewport_commands = ViewportCommands.new()
var signal_commands = SignalCommands.new()
var project_commands = ProjectCommands.new()
var addon_update = AddonUpdate.new()
var protocol = Protocol.new()
var log_buffer = LogBuffer.new()
var log_commands = LogCommands.new()
var jobs = Jobs.new()
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
	jobs.process(_job_context())
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
		return bridge_commands.handle_ping(request, _command_context())
	if method == "GET" and path == "/logs":
		return log_commands.handle_list(request, _command_context())
	if method == "POST" and path == "/logs/clear":
		return log_commands.handle_clear(request, _command_context())
	if method == "GET" and path.begins_with("/jobs/"):
		return _handle_job_status(request, path)
	if method == "POST" and path == "/scene/create":
		return scene_commands.handle_create(request, _command_context())
	if method == "POST" and path == "/scene/open":
		return scene_commands.handle_open(request, _command_context())
	if method == "POST" and path == "/scene/instance":
		return scene_commands.handle_instance(request, _command_context())
	if method == "GET" and path == "/scene/tree":
		return scene_commands.handle_tree(request, _command_context())
	if method == "POST" and path == "/scene/save":
		return scene_commands.handle_save(request, _command_context())
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
	if method == "POST" and path == "/node/set-resource":
		return node_commands.handle_set_resource(request, _command_context())
	if method == "POST" and path == "/node/attach-script":
		return node_commands.handle_attach_script(request, _command_context())
	if method == "POST" and path == "/node/group-add":
		return node_commands.handle_group_add(request, _command_context())
	if method == "POST" and path == "/node/group-remove":
		return node_commands.handle_group_remove(request, _command_context())
	if method == "POST" and path == "/node/group-list":
		return node_commands.handle_group_list(request, _command_context())
	if method == "POST" and path == "/signal/connect":
		return signal_commands.handle_connect(request, _command_context())
	if method == "POST" and path == "/signal/disconnect":
		return signal_commands.handle_disconnect(request, _command_context())
	if method == "POST" and path == "/project/setting-get":
		return project_commands.handle_setting_get(request, _command_context())
	if method == "POST" and path == "/project/setting-set":
		return project_commands.handle_setting_set(request, _command_context())
	if method == "POST" and path == "/script/check":
		return script_commands.handle_check(request, _command_context())
	if method == "POST" and path == "/script/create":
		return script_commands.handle_create(request, _command_context())
	if method == "POST" and path == "/script/write":
		return script_commands.handle_write(request, _command_context())
	if method == "POST" and path == "/shader/check":
		return shader_commands.handle_check(request, _command_context())
	if method == "POST" and path == "/shader/write":
		return shader_commands.handle_write(request, _command_context())
	if method == "POST" and path == "/material/write":
		return material_commands.handle_write(request, _command_context())
	if method == "POST" and path == "/file/write-bytes":
		return file_commands.handle_write_bytes(request, _command_context())
	if method == "POST" and path == "/viewport/screenshot":
		return viewport_commands.handle_screenshot(request, _command_context())
	if method == "POST" and path == "/addon/update":
		return addon_update.handle_update(request, _command_context())
	return protocol.bridge_error(404, "", "UNKNOWN_ENDPOINT", "Unknown bridge endpoint", {"method": method, "path": path})


func _handle_job_status(_request: Dictionary, path: String) -> Dictionary:
	var job_id: String = path.trim_prefix("/jobs/")
	return jobs.status_response(job_id, protocol)


func _params_or_empty(body: Dictionary) -> Dictionary:
	var params := body.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		return {}
	return params


func _command_context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"request": command_request,
		"authorized": Callable(self, "_authorized"),
		"http_json": Callable(protocol, "http_json"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"edited_scene_root": Callable(self, "_edited_scene_root"),
		"editor_plugin_available": Callable(self, "_editor_plugin_available"),
		"node_by_path": Callable(self, "_node_by_path"),
		"node_info": Callable(self, "_node_info"),
		"logical_path": Callable(self, "_logical_path"),
		"mark_scene_dirty": Callable(self, "_mark_scene_dirty"),
		"queue_job": Callable(self, "_queue_job"),
		"typed_values": typed_values,
		"log_buffer": log_buffer,
		"addon_root": ADDON_ROOT,
		"addon_backup_root": ADDON_BACKUP_ROOT,
		"plugin_version": PLUGIN_VERSION,
		"protocol_version": PROTOCOL_VERSION,
		"auth_enabled": auth_enabled,
		"host": host,
		"port": port,
		"capabilities": _capabilities(),
		"log": Callable(log_buffer, "add"),
	}


func _job_context() -> Dictionary:
	return {
		"editor_plugin": editor_plugin,
		"edited_scene_root": Callable(self, "_edited_scene_root"),
		"logical_path": Callable(self, "_logical_path"),
		"log": Callable(log_buffer, "add"),
	}


func _queue_job(kind: String, detail: Dictionary) -> String:
	return jobs.queue(kind, detail, _job_context())


func _editor_plugin_available() -> bool:
	return editor_plugin != null


func _authorized(request: Dictionary) -> bool:
	if not auth_enabled:
		return true
	var headers: Dictionary = request.get("headers", {})
	var expected := "Bearer " + token
	return String(headers.get("authorization", "")) == expected


func _capabilities() -> Array:
	return [
		"ping",
		"scene.create",
		"scene.open",
		"scene.instance",
		"scene.tree",
		"scene.save",
		"jobs.get",
		"node.add",
		"node.remove",
		"node.rename",
		"node.move",
		"node.get",
		"node.set",
		"node.set_resource",
		"node.attach_script",
		"node.group_add",
		"node.group_remove",
		"node.group_list",
		"signal.connect",
		"signal.disconnect",
		"project.setting_get",
		"project.setting_set",
		"script.check",
		"script.create",
		"script.write",
		"shader.check",
		"shader.write",
		"material.write",
		"file.write_bytes",
		"viewport.screenshot",
		"addon.update",
		"bridge.logs",
	]


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
