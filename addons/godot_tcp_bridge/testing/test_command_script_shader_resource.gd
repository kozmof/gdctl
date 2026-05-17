@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const ScriptCommands = preload("res://addons/godot_tcp_bridge/commands/script_commands.gd")
const ShaderCommands = preload("res://addons/godot_tcp_bridge/commands/shader_commands.gd")
const ResourceCommands = preload("res://addons/godot_tcp_bridge/commands/resource_commands.gd")
const TypedValues = preload("res://addons/godot_tcp_bridge/typed_values.gd")

const TEMP_ROOT := "res://gdctl_tmp/gdctl_unit_assets"
const TEMP_SCRIPT := TEMP_ROOT + "/unit_script.gd"
const TEMP_SHADER := TEMP_ROOT + "/unit_shader.gdshader"
const TEMP_RESOURCE := TEMP_ROOT + "/unit_resource.tres"

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var script_commands := ScriptCommands.new()
var shader_commands := ShaderCommands.new()
var resource_commands := ResourceCommands.new()
var typed_values := TypedValues.new()


func before_each() -> void:
	_cleanup_temp()


func after_each() -> void:
	_cleanup_temp()


func test_script_create_write_and_check() -> void:
	var create_response := script_commands.handle_create(_request("script.create", {"path": TEMP_SCRIPT, "extends": "Node"}), _context())
	assert_eq(create_response["status"], 200)
	assert_eq(create_response["body"]["result"]["created"], true)

	var duplicate := script_commands.handle_create(_request("script.create", {"path": TEMP_SCRIPT, "extends": "Node"}), _context())
	assert_eq(duplicate["status"], 409)
	assert_eq(duplicate["body"]["error"]["code"], "SCRIPT_ALREADY_EXISTS")

	var write_response := script_commands.handle_write(_request("script.write", {"path": TEMP_SCRIPT, "body": "extends RefCounted\n\nfunc answer() -> int:\n\treturn 42\n"}), _context())
	assert_eq(write_response["status"], 200)
	assert_eq(write_response["body"]["result"]["written"], true)

	var check_response := script_commands.handle_check(_request("script.check", {"path": TEMP_SCRIPT}), _context())
	assert_eq(check_response["status"], 200)
	assert_eq(check_response["body"]["result"]["valid"], true)


func test_script_commands_validate_path_extends_and_body() -> void:
	var bad_path := script_commands.handle_create(_request("script.create", {"path": "user://bad.gd", "extends": "Node"}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "SCRIPT_PATH_INVALID")

	var bad_extends := script_commands.handle_create(_request("script.create", {"path": TEMP_SCRIPT, "extends": "not-valid"}), _context())
	assert_eq(bad_extends["status"], 400)
	assert_eq(bad_extends["body"]["error"]["code"], "SCRIPT_EXTENDS_INVALID")

	var missing_body := script_commands.handle_write(_request("script.write", {"path": TEMP_SCRIPT}), _context())
	assert_eq(missing_body["status"], 400)
	assert_eq(missing_body["body"]["error"]["code"], "SCRIPT_BODY_MISSING")

	var missing_check := script_commands.handle_check(_request("script.check", {"path": TEMP_SCRIPT}), _context())
	assert_eq(missing_check["status"], 404)
	assert_eq(missing_check["body"]["error"]["code"], "SCRIPT_NOT_FOUND")


func test_script_write_rejects_invalid_source() -> void:
	var response := script_commands.handle_write(_request("script.write", {"path": TEMP_SCRIPT, "body": "extends Node\nfunc nope(:\n"}), _context())
	assert_eq(response["status"], 400)
	assert_eq(response["body"]["error"]["code"], "SCRIPT_SYNTAX_INVALID")


func test_shader_write_and_check() -> void:
	var source := "shader_type canvas_item;\nvoid fragment() { COLOR = vec4(1.0); }\n"
	var write_response := shader_commands.handle_write(_request("shader.write", {"path": TEMP_SHADER, "body": source}), _context())
	assert_eq(write_response["status"], 200)
	assert_eq(write_response["body"]["result"]["written"], true)

	var check_response := shader_commands.handle_check(_request("shader.check", {"path": TEMP_SHADER}), _context())
	assert_eq(check_response["status"], 200)
	assert_eq(check_response["body"]["result"]["valid"], true)


func test_shader_commands_validate_path_body_and_type() -> void:
	var bad_path := shader_commands.handle_write(_request("shader.write", {"path": "res://bad.txt", "body": "shader_type canvas_item;"}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "SHADER_PATH_INVALID")

	var missing_body := shader_commands.handle_write(_request("shader.write", {"path": TEMP_SHADER}), _context())
	assert_eq(missing_body["status"], 400)
	assert_eq(missing_body["body"]["error"]["code"], "SHADER_BODY_MISSING")

	var missing_type := shader_commands.handle_write(_request("shader.write", {"path": TEMP_SHADER, "body": "void fragment() {}"}), _context())
	assert_eq(missing_type["status"], 400)
	assert_eq(missing_type["body"]["error"]["code"], "SHADER_TYPE_MISSING")

	var missing_check := shader_commands.handle_check(_request("shader.check", {"path": TEMP_SHADER}), _context())
	assert_eq(missing_check["status"], 404)
	assert_eq(missing_check["body"]["error"]["code"], "SHADER_NOT_FOUND")


func test_resource_create_and_list() -> void:
	var response := resource_commands.handle_create(_request("resource.create", {"path": TEMP_RESOURCE, "type": "Gradient"}), _context())
	assert_eq(response["status"], 200)
	assert_eq(response["body"]["result"]["created"], true)

	var list_response := resource_commands.handle_list(_request("resource.list", {"dir": TEMP_ROOT, "recursive": false}), _context())
	assert_eq(list_response["status"], 200)
	assert_true(list_response["body"]["result"]["resources"].has(TEMP_RESOURCE))


func test_resource_commands_validate_inputs() -> void:
	var bad_path := resource_commands.handle_create(_request("resource.create", {"path": "res://bad.txt", "type": "Gradient"}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "RESOURCE_PATH_INVALID")

	var missing_type := resource_commands.handle_create(_request("resource.create", {"path": TEMP_RESOURCE}), _context())
	assert_eq(missing_type["status"], 400)
	assert_eq(missing_type["body"]["error"]["code"], "RESOURCE_TYPE_REQUIRED")

	var unknown_type := resource_commands.handle_create(_request("resource.create", {"path": TEMP_RESOURCE, "type": "DefinitelyNotAType"}), _context())
	assert_eq(unknown_type["status"], 400)
	assert_eq(unknown_type["body"]["error"]["code"], "RESOURCE_TYPE_UNKNOWN")

	var non_resource := resource_commands.handle_create(_request("resource.create", {"path": TEMP_RESOURCE, "type": "Node"}), _context())
	assert_eq(non_resource["status"], 400)
	assert_eq(non_resource["body"]["error"]["code"], "RESOURCE_TYPE_INVALID")

	var bad_list_dir := resource_commands.handle_list(_request("resource.list", {"dir": "user://bad"}), _context())
	assert_eq(bad_list_dir["status"], 400)
	assert_eq(bad_list_dir["body"]["error"]["code"], "DIR_PATH_INVALID")

	var missing_dir := resource_commands.handle_list(_request("resource.list", {"dir": TEMP_ROOT}), _context())
	assert_eq(missing_dir["status"], 404)
	assert_eq(missing_dir["body"]["error"]["code"], "DIR_NOT_FOUND")


func _context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"request": request_helper,
		"typed_values": typed_values,
	}


func _request(op: String, params: Dictionary) -> Dictionary:
	return {
		"headers": {"content-type": "application/json"},
		"body": JSON.stringify({"request_id": "req-1", "op": op, "params": params}),
	}


func _authorized(_request: Dictionary) -> bool:
	return true


func _params_or_empty(body: Dictionary) -> Dictionary:
	var params: Variant = body.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		return {}
	return params


func _cleanup_temp() -> void:
	for path in [TEMP_SCRIPT, TEMP_SHADER, TEMP_RESOURCE]:
		if FileAccess.file_exists(path):
			DirAccess.remove_absolute(ProjectSettings.globalize_path(path))
	var dir_path := ProjectSettings.globalize_path(TEMP_ROOT)
	if DirAccess.dir_exists_absolute(dir_path):
		DirAccess.remove_absolute(dir_path)
