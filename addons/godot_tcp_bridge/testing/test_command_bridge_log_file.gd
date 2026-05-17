@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const BridgeCommands = preload("res://addons/godot_tcp_bridge/commands/bridge_commands.gd")
const LogCommands = preload("res://addons/godot_tcp_bridge/commands/log_commands.gd")
const FileCommands = preload("res://addons/godot_tcp_bridge/commands/file_commands.gd")
const LogBuffer = preload("res://addons/godot_tcp_bridge/log_buffer.gd")

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var bridge_commands := BridgeCommands.new()
var log_commands := LogCommands.new()
var file_commands := FileCommands.new()
var log_buffer := LogBuffer.new()
var authorized := true

const TEMP_ROOT := "res://gdctl_tmp/gdctl_unit_file_commands"


func before_each() -> void:
	authorized = true
	log_buffer.clear()
	_cleanup_temp()


func after_each() -> void:
	_cleanup_temp()


func test_bridge_ping_reports_context_values() -> void:
	var response := bridge_commands.handle_ping({}, _context())
	var body: Dictionary = response["body"]
	assert_eq(response["status"], 200)
	assert_eq(body["ok"], true)
	assert_eq(body["service"], "godot-bridge")
	assert_eq(body["plugin_version"], "unit-version")
	assert_eq(body["protocol_version"], "unit-protocol")
	assert_eq(body["auth_enabled"], true)
	assert_eq(body["host"], "unit-host")
	assert_eq(body["port"], 1234)
	assert_eq(body["scene_open"], false)
	assert_eq(body["capabilities"], ["ping", "test.gdscript"])


func test_log_list_requires_authorization() -> void:
	authorized = false
	var response := log_commands.handle_list({}, _context())
	assert_eq(response["status"], 401)
	assert_eq(response["body"]["error"]["code"], "UNAUTHORIZED")


func test_log_list_returns_entries() -> void:
	log_buffer.add("info", "unit", "hello", {"value": 1})
	var response := log_commands.handle_list({}, _context())
	assert_eq(response["status"], 200)
	assert_eq(response["body"]["ok"], true)
	assert_eq(response["body"]["entries"].size(), 1)
	assert_eq(response["body"]["entries"][0]["source"], "unit")


func test_log_clear_clears_and_records_marker() -> void:
	log_buffer.add("info", "unit", "old", {})
	var response := log_commands.handle_clear(_request("logs.clear", {}), _context())
	assert_eq(response["status"], 200)
	assert_eq(response["body"]["result"]["cleared"], true)
	var entries := log_buffer.list()
	assert_eq(entries.size(), 1)
	assert_eq(entries[0]["source"], "bridge.logs")


func test_file_write_bytes_rejects_invalid_path_and_content() -> void:
	var bad_path := file_commands.handle_write_bytes(_request("file.write_bytes", {"path": "user://bad.txt", "content_base64": "aGVsbG8="}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "FILE_PATH_INVALID")

	var bad_content := file_commands.handle_write_bytes(_request("file.write_bytes", {"path": TEMP_ROOT + "/bad.txt", "content_base64": ""}), _context())
	assert_eq(bad_content["status"], 400)
	assert_eq(bad_content["body"]["error"]["code"], "FILE_CONTENT_MISSING")


func test_file_commands_write_exists_list_and_delete() -> void:
	var mkdir_response := file_commands.handle_mkdir(_request("file.mkdir", {"path": TEMP_ROOT}), _context())
	assert_eq(mkdir_response["status"], 200)
	assert_eq(mkdir_response["body"]["result"]["created"], true)

	var write_response := file_commands.handle_write_bytes(_request("file.write_bytes", {"path": TEMP_ROOT + "/hello.txt", "content_base64": "aGVsbG8="}), _context())
	assert_eq(write_response["status"], 200)
	assert_eq(write_response["body"]["result"]["bytes"], 5)

	var exists_response := file_commands.handle_exists(_request("file.exists", {"path": TEMP_ROOT + "/hello.txt"}), _context())
	assert_eq(exists_response["status"], 200)
	assert_eq(exists_response["body"]["result"]["exists"], true)
	assert_eq(exists_response["body"]["result"]["is_file"], true)

	var list_response := file_commands.handle_list(_request("file.list", {"path": TEMP_ROOT, "recursive": false}), _context())
	assert_eq(list_response["status"], 200)
	assert_true(list_response["body"]["result"]["files"].has(TEMP_ROOT + "/hello.txt"))

	var delete_response := file_commands.handle_delete(_request("file.delete", {"path": TEMP_ROOT + "/hello.txt"}), _context())
	assert_eq(delete_response["status"], 200)
	assert_eq(delete_response["body"]["result"]["deleted"], true)


func test_file_delete_rejects_project_root() -> void:
	var response := file_commands.handle_delete(_request("file.delete", {"path": "res://"}), _context())
	assert_eq(response["status"], 400)
	assert_eq(response["body"]["error"]["code"], "FILE_PATH_INVALID")


func _context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"http_json": Callable(protocol, "http_json"),
		"request": request_helper,
		"edited_scene_root": Callable(self, "_edited_scene_root"),
		"log_buffer": log_buffer,
		"plugin_version": "unit-version",
		"protocol_version": "unit-protocol",
		"auth_enabled": true,
		"host": "unit-host",
		"port": 1234,
		"capabilities": ["ping", "test.gdscript"],
	}


func _request(op: String, params: Dictionary) -> Dictionary:
	return {
		"headers": {"content-type": "application/json"},
		"body": JSON.stringify({"request_id": "req-1", "op": op, "params": params}),
	}


func _authorized(_request: Dictionary) -> bool:
	return authorized


func _params_or_empty(body: Dictionary) -> Dictionary:
	var params: Variant = body.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		return {}
	return params


func _edited_scene_root() -> Node:
	return null


func _cleanup_temp() -> void:
	var file_path := TEMP_ROOT + "/hello.txt"
	if FileAccess.file_exists(file_path):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(file_path))
	var dir_path := ProjectSettings.globalize_path(TEMP_ROOT)
	if DirAccess.dir_exists_absolute(dir_path):
		DirAccess.remove_absolute(dir_path)
