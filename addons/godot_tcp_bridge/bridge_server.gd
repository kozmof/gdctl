@tool
extends RefCounted

const PLUGIN_VERSION := "0.1.8"
const PROTOCOL_VERSION := "gdctl.v1"
const DEFAULT_HOST := "127.0.0.1"
const DEFAULT_PORT := 7777
const TOKEN_PATH := "res://.godot-bridge-token"
const ADDON_ROOT := "res://addons/godot_tcp_bridge/"
const ADDON_BACKUP_ROOT := "res://addons/.godot_tcp_bridge_backup/"
const RUNTIME_AUTOLOAD_NAME := "GdctlRuntimeBridge"
const RUNTIME_AUTOLOAD_PATH := "res://addons/godot_tcp_bridge/runtime/runtime_bridge.gd"
const RUNTIME_LOG_PATH := "res://.gdctl_runtime/logs/runtime.jsonl"
const TypedValues = preload("res://addons/godot_tcp_bridge/typed_values.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const BridgeCommands = preload("res://addons/godot_tcp_bridge/commands/bridge_commands.gd")
const SceneCommands = preload("res://addons/godot_tcp_bridge/commands/scene_commands.gd")
const NodeCommands = preload("res://addons/godot_tcp_bridge/commands/node_commands.gd")
const ScriptCommands = preload("res://addons/godot_tcp_bridge/commands/script_commands.gd")
const ShaderCommands = preload("res://addons/godot_tcp_bridge/commands/shader_commands.gd")
const ResourceCommands = preload("res://addons/godot_tcp_bridge/commands/resource_commands.gd")
const FileCommands = preload("res://addons/godot_tcp_bridge/commands/file_commands.gd")
const ViewportCommands = preload("res://addons/godot_tcp_bridge/commands/viewport_commands.gd")
const SignalCommands = preload("res://addons/godot_tcp_bridge/commands/signal_commands.gd")
const ProjectCommands = preload("res://addons/godot_tcp_bridge/commands/project_commands.gd")
const NavigationCommands = preload("res://addons/godot_tcp_bridge/commands/navigation_commands.gd")
const ImportCommands = preload("res://addons/godot_tcp_bridge/commands/import_commands.gd")
const AddonUpdate = preload("res://addons/godot_tcp_bridge/addon_update.gd")
const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const LogBuffer = preload("res://addons/godot_tcp_bridge/log_buffer.gd")
const LogCommands = preload("res://addons/godot_tcp_bridge/commands/log_commands.gd")
const Jobs = preload("res://addons/godot_tcp_bridge/jobs.gd")

class RuntimeLogCapture extends Logger:
	var log_buffer

	func _log_error(function: String, file: String, line: int, code: String, rationale: String, editor_notify: bool, error_type: int, script_backtraces: Array[ScriptBacktrace]) -> void:
		var message := rationale
		if message == "":
			message = code
		log_buffer.add("error", "runtime.error", message, {
			"function": function,
			"file": file,
			"line": line,
			"code": code,
			"rationale": rationale,
			"editor_notify": editor_notify,
			"error_type": error_type,
		})

	func _log_message(message: String, error: bool) -> void:
		if error:
			log_buffer.add("error", "runtime.stderr", message, {})

var editor_plugin: EditorPlugin
var typed_values = TypedValues.new()
var command_request = CommandRequest.new()
var bridge_commands = BridgeCommands.new()
var scene_commands = SceneCommands.new()
var node_commands = NodeCommands.new()
var script_commands = ScriptCommands.new()
var shader_commands = ShaderCommands.new()
var resource_commands = ResourceCommands.new()
var file_commands = FileCommands.new()
var viewport_commands = ViewportCommands.new()
var signal_commands = SignalCommands.new()
var project_commands = ProjectCommands.new()
var navigation_commands = NavigationCommands.new()
var import_commands = ImportCommands.new()
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
var runtime_logger


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
	if runtime_logger == null:
		runtime_logger = RuntimeLogCapture.new()
		runtime_logger.log_buffer = log_buffer
		OS.add_logger(runtime_logger)
	print("Godot TCP Bridge listening on %s:%d" % [host, port])
	log_buffer.add("info", "bridge.start", "Listening", {"host": host, "port": port, "auth_enabled": auth_enabled})


func stop() -> void:
	for client in clients:
		var peer: StreamPeerTCP = client.get("peer")
		if peer:
			peer.disconnect_from_host()
	clients.clear()
	if tcp_server != null:
		tcp_server.stop()
	if runtime_logger != null:
		OS.remove_logger(runtime_logger)
		runtime_logger = null
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
	if method == "POST" and path == "/scene/apply":
		return scene_commands.handle_apply(request, _command_context())
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
	if method == "POST" and path == "/node/duplicate":
		return node_commands.handle_duplicate(request, _command_context())
	if method == "POST" and path == "/node/list-properties":
		return node_commands.handle_list_properties(request, _command_context())
	if method == "POST" and path == "/file/list":
		return file_commands.handle_list(request, _command_context())
	if method == "POST" and path == "/file/mkdir":
		return file_commands.handle_mkdir(request, _command_context())
	if method == "POST" and path == "/file/delete":
		return file_commands.handle_delete(request, _command_context())
	if method == "POST" and path == "/file/exists":
		return file_commands.handle_exists(request, _command_context())
	if method == "POST" and path == "/navigation/bake":
		return navigation_commands.handle_bake(request, _command_context())
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
	if method == "POST" and path == "/resource/create":
		return resource_commands.handle_create(request, _command_context())
	if method == "POST" and path == "/file/write-bytes":
		return file_commands.handle_write_bytes(request, _command_context())
	if method == "POST" and path == "/viewport/screenshot":
		return viewport_commands.handle_screenshot(request, _command_context())
	if method == "POST" and path == "/addon/update":
		return addon_update.handle_update(request, _command_context())
	if method == "POST" and path == "/import/set":
		return import_commands.handle_set(request, _command_context())
	if method == "POST" and path == "/scene/list":
		return scene_commands.handle_list(request, _command_context())
	if method == "POST" and path == "/resource/list":
		return resource_commands.handle_list(request, _command_context())
	if method == "POST" and path == "/run/start":
		return _handle_run_start(request)
	if method == "POST" and path == "/run/status":
		return _handle_run_status(request)
	if method == "POST" and path == "/run/stop":
		return _handle_run_stop(request)
	if method == "GET" and path == "/run/logs":
		return _handle_run_logs(request)
	if method == "POST" and path == "/run/logs/clear":
		return _handle_run_logs_clear(request)
	if method == "POST" and path == "/run/screenshot":
		return _handle_run_screenshot(request)
	return protocol.bridge_error(404, "", "UNKNOWN_ENDPOINT", "Unknown bridge endpoint", {"method": method, "path": path})


func _handle_run_start(request: Dictionary) -> Dictionary:
	var checked: Dictionary = command_request.require_body(request, _command_context(), "run.start", "Run start requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	if not _editor_plugin_available():
		return protocol.bridge_error(503, String(checked["request_id"]), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var scene: String = String(params.get("scene", ""))
	var main: bool = bool(params.get("main", false))
	var clear_logs: bool = bool(params.get("clear_logs", true))
	var editor_interface := editor_plugin.get_editor_interface()
	if clear_logs:
		log_buffer.clear()
		_clear_runtime_logs()
		log_buffer.add("info", "run.logs", "Runtime logs cleared", {})
	if editor_interface.is_playing_scene():
		return protocol.bridge_error(409, request_id, "RUN_ALREADY_PLAYING", "A scene is already running", {"playing_scene": editor_interface.get_playing_scene()})
	var autoload_err := _ensure_runtime_autoload()
	if autoload_err != OK:
		return protocol.bridge_error(500, request_id, "RUNTIME_HELPER_SETUP_FAILED", "Could not register gdctl runtime helper autoload", {"error": error_string(autoload_err)})
	if scene != "":
		if not ResourceLoader.exists(scene):
			return protocol.bridge_error(404, request_id, "RUN_SCENE_NOT_FOUND", "Scene does not exist", {"scene": scene})
		editor_interface.play_custom_scene(scene)
	elif main:
		editor_interface.play_main_scene()
	else:
		editor_interface.play_current_scene()
	log_buffer.add("info", "run.start", "Started editor run", {"scene": scene, "main": main})
	return protocol.bridge_ok(request_id, {
		"running": true,
		"scene": scene,
		"playing_scene": editor_interface.get_playing_scene(),
	})


func _handle_run_status(request: Dictionary) -> Dictionary:
	var checked: Dictionary = command_request.require_body(request, _command_context(), "run.status", "Run status requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	if not _editor_plugin_available():
		return protocol.bridge_error(503, String(checked["request_id"]), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	var editor_interface := editor_plugin.get_editor_interface()
	return protocol.bridge_ok(String(checked["request_id"]), {
		"running": editor_interface.is_playing_scene(),
		"playing_scene": editor_interface.get_playing_scene(),
	})


func _handle_run_stop(request: Dictionary) -> Dictionary:
	var checked: Dictionary = command_request.require_body(request, _command_context(), "run.stop", "Run stop requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	if not _editor_plugin_available():
		return protocol.bridge_error(503, String(checked["request_id"]), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	var editor_interface := editor_plugin.get_editor_interface()
	var was_running := editor_interface.is_playing_scene()
	var playing_scene := editor_interface.get_playing_scene()
	if was_running:
		editor_interface.stop_playing_scene()
		log_buffer.add("info", "run.stop", "Stopped editor run", {"playing_scene": playing_scene})
	return protocol.bridge_ok(String(checked["request_id"]), {
		"stopped": was_running,
		"running": false,
		"playing_scene": playing_scene,
	})


func _handle_run_logs(request: Dictionary) -> Dictionary:
	if not _authorized(request):
		return protocol.bridge_error(401, "", "UNAUTHORIZED", "Run logs require bearer token", {})
	var entries: Array[Dictionary] = []
	for entry in log_buffer.list():
		var source := String(entry.get("source", ""))
		if source.begins_with("run.") or source.begins_with("runtime."):
			entries.append(entry)
	for entry in _read_runtime_logs():
		entries.append(entry)
	return protocol.http_json(200, {"ok": true, "entries": entries})


func _handle_run_logs_clear(request: Dictionary) -> Dictionary:
	var checked: Dictionary = command_request.require_body(request, _command_context(), "run.logs.clear", "Run logs clear requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	log_buffer.clear()
	_clear_runtime_logs()
	log_buffer.add("info", "run.logs", "Runtime logs cleared", {})
	return protocol.bridge_ok(String(checked["request_id"]), {"cleared": true})


func _handle_run_screenshot(request: Dictionary) -> Dictionary:
	var checked: Dictionary = command_request.require_body(request, _command_context(), "run.screenshot", "Run screenshot requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	if not _editor_plugin_available():
		return protocol.bridge_error(503, String(checked["request_id"]), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	var editor_interface := editor_plugin.get_editor_interface()
	if not editor_interface.is_playing_scene():
		return protocol.bridge_error(409, String(checked["request_id"]), "RUN_NOT_PLAYING", "No scene is currently running", {})
	var params: Dictionary = checked["params"]
	var source: String = String(params.get("source", "game"))
	if source != "game" and source != "screen":
		return protocol.bridge_error(400, String(checked["request_id"]), "RUN_SCREENSHOT_SOURCE_INVALID", "Run screenshot source must be game or screen", {"source": source})
	var screen: int = int(params.get("screen", 0))
	if source == "screen" and (screen < 0 or screen >= DisplayServer.get_screen_count()):
		return protocol.bridge_error(400, String(checked["request_id"]), "RUN_SCREEN_INVALID", "Screen index is out of range", {"screen": screen})
	var job_id: String = String(_queue_job("run.screenshot", {
		"source": source,
		"screen": screen,
		"frames_remaining": 2,
		"request_id": String(checked["request_id"]),
	}))
	return protocol.bridge_ok(String(checked["request_id"]), {
		"queued": true,
		"job_id": job_id,
		"source": source,
		"screen": screen,
	})


func _read_runtime_logs() -> Array[Dictionary]:
	var entries: Array[Dictionary] = []
	if not FileAccess.file_exists(RUNTIME_LOG_PATH):
		return entries
	var file := FileAccess.open(RUNTIME_LOG_PATH, FileAccess.READ)
	if file == null:
		return entries
	while not file.eof_reached():
		var line := file.get_line().strip_edges()
		if line == "":
			continue
		var parsed: Variant = JSON.parse_string(line)
		if typeof(parsed) != TYPE_DICTIONARY:
			continue
		var entry: Dictionary = parsed
		entries.append({
			"time": String(entry.get("time", "")),
			"level": String(entry.get("level", "info")),
			"source": String(entry.get("source", "runtime.game")),
			"message": String(entry.get("message", "")),
			"detail": _dictionary_or_empty(entry.get("detail", {})),
		})
	file.close()
	while entries.size() > 200:
		entries.pop_front()
	return entries


func _clear_runtime_logs() -> void:
	if FileAccess.file_exists(RUNTIME_LOG_PATH):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(RUNTIME_LOG_PATH))


func _dictionary_or_empty(value: Variant) -> Dictionary:
	if typeof(value) == TYPE_DICTIONARY:
		return value
	return {}


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
		"reimport_files": Callable(self, "_reimport_files"),
		"queue_job": Callable(self, "_queue_job"),
		"editor_plugin": editor_plugin,
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


func _ensure_runtime_autoload() -> Error:
	if not FileAccess.file_exists(RUNTIME_AUTOLOAD_PATH):
		return ERR_FILE_NOT_FOUND
	var key := "autoload/%s" % RUNTIME_AUTOLOAD_NAME
	var value := "*" + RUNTIME_AUTOLOAD_PATH
	if ProjectSettings.has_setting(key) and String(ProjectSettings.get_setting(key)) == value:
		return OK
	ProjectSettings.set_setting(key, value)
	return ProjectSettings.save()


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
		"scene.apply",
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
		"node.duplicate",
		"node.list_properties",
		"file.list",
		"file.mkdir",
		"file.delete",
		"file.exists",
		"navigation.bake",
		"script.check",
		"script.create",
		"script.write",
		"shader.check",
		"shader.write",
		"resource.create",
		"file.write_bytes",
		"viewport.screenshot",
		"addon.update",
		"bridge.logs",
		"import.set",
		"scene.list",
		"resource.list",
		"run.start",
		"run.status",
		"run.stop",
		"run.logs",
		"run.logs.clear",
		"run.screenshot",
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


func _reimport_files(paths: PackedStringArray) -> void:
	if editor_plugin:
		editor_plugin.get_editor_interface().get_resource_filesystem().reimport_files(paths)


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
