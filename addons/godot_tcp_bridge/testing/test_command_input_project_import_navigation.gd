@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const InputCommands = preload("res://addons/godot_tcp_bridge/commands/input_commands.gd")
const ProjectCommands = preload("res://addons/godot_tcp_bridge/commands/project_commands.gd")
const ImportCommands = preload("res://addons/godot_tcp_bridge/commands/import_commands.gd")
const NavigationCommands = preload("res://addons/godot_tcp_bridge/commands/navigation_commands.gd")
const TypedValues = preload("res://addons/godot_tcp_bridge/typed_values.gd")

const TEMP_ROOT := "res://gdctl_tmp/gdctl_unit_import"
const TEMP_ASSET := TEMP_ROOT + "/asset.png"
const TEMP_IMPORT := TEMP_ASSET + ".import"

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var input_commands := InputCommands.new()
var project_commands := ProjectCommands.new()
var import_commands := ImportCommands.new()
var navigation_commands := NavigationCommands.new()
var typed_values := TypedValues.new()
var reimported_files: PackedStringArray = []
var available_node: Node = null


func before_each() -> void:
	reimported_files = PackedStringArray()
	available_node = null
	_cleanup_temp()


func after_each() -> void:
	_cleanup_temp()
	if available_node != null:
		available_node.free()
		available_node = null


func test_input_action_add_rejects_missing_and_pathlike_action() -> void:
	var missing := input_commands.handle_action_add(_request("input.action_add", {}), _context())
	assert_eq(missing["status"], 400)
	assert_eq(missing["body"]["error"]["code"], "INPUT_ACTION_INVALID")

	var pathlike := input_commands.handle_action_add(_request("input.action_add", {"action": "../jump"}), _context())
	assert_eq(pathlike["status"], 400)
	assert_eq(pathlike["body"]["error"]["code"], "INPUT_ACTION_INVALID")


func test_input_key_event_rejects_missing_or_unknown_key() -> void:
	var missing_key := input_commands.handle_event_add_key(_request("input.event_add_key", {"action": "unit_jump"}), _context())
	assert_eq(missing_key["status"], 400)
	assert_eq(missing_key["body"]["error"]["code"], "INPUT_KEY_INVALID")

	var unknown_key := input_commands.handle_event_add_key(_request("input.event_add_key", {"action": "unit_jump", "key": "DefinitelyNotAKey"}), _context())
	assert_eq(unknown_key["status"], 400)
	assert_eq(unknown_key["body"]["error"]["code"], "INPUT_KEY_INVALID")


func test_input_joypad_event_requires_button_or_axis() -> void:
	var response := input_commands.handle_event_add_joypad(_request("input.event_add_joypad", {"action": "unit_jump"}), _context())
	assert_eq(response["status"], 400)
	assert_eq(response["body"]["error"]["code"], "INPUT_JOYPAD_PARAMS_MISSING")


func test_project_setting_get_rejects_missing_and_unknown_key() -> void:
	var missing := project_commands.handle_setting_get(_request("project.setting_get", {}), _context())
	assert_eq(missing["status"], 400)
	assert_eq(missing["body"]["error"]["code"], "SETTING_KEY_INVALID")

	var unknown := project_commands.handle_setting_get(_request("project.setting_get", {"key": "gdctl/unit/does_not_exist"}), _context())
	assert_eq(unknown["status"], 404)
	assert_eq(unknown["body"]["error"]["code"], "SETTING_NOT_FOUND")


func test_project_setting_get_returns_typed_value() -> void:
	var response := project_commands.handle_setting_get(_request("project.setting_get", {"key": "application/config/name"}), _context())
	assert_eq(response["status"], 200)
	assert_eq(response["body"]["result"]["key"], "application/config/name")
	assert_true(response["body"]["result"]["value"].has("kind"))


func test_project_setting_set_validates_key_and_value() -> void:
	var missing_key := project_commands.handle_setting_set(_request("project.setting_set", {"value": {"kind": "String", "value": "x"}}), _context())
	assert_eq(missing_key["status"], 400)
	assert_eq(missing_key["body"]["error"]["code"], "SETTING_KEY_INVALID")

	var missing_value := project_commands.handle_setting_set(_request("project.setting_set", {"key": "gdctl/unit/value"}), _context())
	assert_eq(missing_value["status"], 400)
	assert_eq(missing_value["body"]["error"]["code"], "VALUE_MISSING")

	var invalid_value := project_commands.handle_setting_set(_request("project.setting_set", {"key": "gdctl/unit/value", "value": {"kind": "Nope"}}), _context())
	assert_eq(invalid_value["status"], 400)
	assert_eq(invalid_value["body"]["error"]["code"], "VALUE_INVALID")


func test_autoload_commands_validate_inputs() -> void:
	var bad_name := project_commands.handle_autoload_add(_request("autoload.add", {"name": "bad-name", "path": "res://foo.gd"}), _context())
	assert_eq(bad_name["status"], 400)
	assert_eq(bad_name["body"]["error"]["code"], "AUTOLOAD_NAME_INVALID")

	var bad_path := project_commands.handle_autoload_add(_request("autoload.add", {"name": "GoodName", "path": "user://foo.gd"}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "AUTOLOAD_PATH_INVALID")

	var missing_resource := project_commands.handle_autoload_add(_request("autoload.add", {"name": "GoodName", "path": "res://missing_autoload.gd"}), _context())
	assert_eq(missing_resource["status"], 404)
	assert_eq(missing_resource["body"]["error"]["code"], "AUTOLOAD_RESOURCE_NOT_FOUND")

	var missing_remove := project_commands.handle_autoload_remove(_request("autoload.remove", {"name": "DefinitelyMissingAutoload"}), _context())
	assert_eq(missing_remove["status"], 404)
	assert_eq(missing_remove["body"]["error"]["code"], "AUTOLOAD_NOT_FOUND")


func test_autoload_list_returns_array() -> void:
	var response := project_commands.handle_autoload_list(_request("autoload.list", {}), _context())
	assert_eq(response["status"], 200)
	assert_true(response["body"]["result"].has("autoloads"))


func test_import_set_validates_path_and_params() -> void:
	var bad_path := import_commands.handle_set(_request("import.set", {"path": "user://asset.png"}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "IMPORT_PATH_INVALID")

	var bad_parent := import_commands.handle_set(_request("import.set", {"path": "res://../asset.png"}), _context())
	assert_eq(bad_parent["status"], 400)
	assert_eq(bad_parent["body"]["error"]["code"], "IMPORT_PATH_INVALID")

	var missing_import := import_commands.handle_set(_request("import.set", {"path": TEMP_ASSET}), _context())
	assert_eq(missing_import["status"], 404)
	assert_eq(missing_import["body"]["error"]["code"], "IMPORT_FILE_NOT_FOUND")

	_write_import_file()
	var bad_params := import_commands.handle_set(_request("import.set", {"path": TEMP_ASSET, "params": "bad"}), _context())
	assert_eq(bad_params["status"], 400)
	assert_eq(bad_params["body"]["error"]["code"], "IMPORT_PARAMS_INVALID")


func test_import_set_updates_import_file_and_reimports() -> void:
	_write_import_file()
	var response := import_commands.handle_set(_request("import.set", {"path": TEMP_ASSET, "params": {"compress/mode": 1, "mipmaps/generate": true}}), _context())
	assert_eq(response["status"], 200)
	assert_eq(response["body"]["result"]["params"], 2)
	assert_eq(reimported_files.size(), 1)
	assert_eq(reimported_files[0], TEMP_ASSET)

	var cfg := ConfigFile.new()
	assert_eq(cfg.load(ProjectSettings.globalize_path(TEMP_IMPORT)), OK)
	assert_eq(cfg.get_value("params", "compress/mode"), 1)
	assert_eq(cfg.get_value("params", "mipmaps/generate"), true)


func test_navigation_bake_rejects_missing_or_wrong_type_node() -> void:
	var missing := navigation_commands.handle_bake(_request("navigation.bake", {"path": "/root/Missing"}), _context())
	assert_eq(missing["status"], 404)
	assert_eq(missing["body"]["error"]["code"], "NODE_NOT_FOUND")

	available_node = Node.new()
	available_node.name = "Plain"
	var wrong_type := navigation_commands.handle_bake(_request("navigation.bake", {"path": "/root/Plain"}), _context())
	assert_eq(wrong_type["status"], 400)
	assert_eq(wrong_type["body"]["error"]["code"], "NODE_TYPE_INVALID")


func _context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"request": request_helper,
		"typed_values": typed_values,
		"editor_plugin_available": Callable(self, "_editor_plugin_available"),
		"reimport_files": Callable(self, "_reimport_files"),
		"node_by_path": Callable(self, "_node_by_path"),
		"logical_path": Callable(self, "_logical_path"),
		"mark_scene_dirty": Callable(self, "_mark_scene_dirty"),
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


func _editor_plugin_available() -> bool:
	return true


func _reimport_files(files: PackedStringArray) -> void:
	reimported_files = files


func _node_by_path(_path: String) -> Node:
	return available_node


func _logical_path(node: Node) -> String:
	return "/root/" + String(node.name)


func _mark_scene_dirty() -> void:
	pass


func _write_import_file() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(TEMP_ROOT))
	var file := FileAccess.open(TEMP_IMPORT, FileAccess.WRITE)
	file.store_string("[params]\n")
	file.close()


func _cleanup_temp() -> void:
	if FileAccess.file_exists(TEMP_IMPORT):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(TEMP_IMPORT))
	if FileAccess.file_exists(TEMP_ASSET):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(TEMP_ASSET))
	var dir_path := ProjectSettings.globalize_path(TEMP_ROOT)
	if DirAccess.dir_exists_absolute(dir_path):
		DirAccess.remove_absolute(dir_path)
