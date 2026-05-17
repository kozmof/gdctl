@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const TestCommands = preload("res://addons/godot_tcp_bridge/commands/test_commands.gd")

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var command := TestCommands.new()
var queued_kind := ""
var queued_detail: Dictionary = {}


func before_each() -> void:
	queued_kind = ""
	queued_detail = {}


func test_handle_gdscript_queues_path_request() -> void:
	var response := command.handle_gdscript(_request({"path": "res://addons/godot_tcp_bridge/testing/test_test_case.gd"}), _context())
	assert_eq(response["status"], 200)
	assert_eq(response["body"]["result"]["queued"], true)
	assert_eq(response["body"]["result"]["job_id"], "job-1")
	assert_eq(queued_kind, "test.gdscript")
	assert_eq(queued_detail["path"], "res://addons/godot_tcp_bridge/testing/test_test_case.gd")
	assert_eq(queued_detail["dir"], "")


func test_handle_gdscript_queues_dir_request() -> void:
	var response := command.handle_gdscript(_request({"dir": "res://addons/godot_tcp_bridge/testing"}), _context())
	assert_eq(response["status"], 200)
	assert_eq(response["body"]["result"]["queued"], true)
	assert_eq(queued_kind, "test.gdscript")
	assert_eq(queued_detail["path"], "")
	assert_eq(queued_detail["dir"], "res://addons/godot_tcp_bridge/testing")


func test_handle_gdscript_rejects_missing_selector() -> void:
	var response := command.handle_gdscript(_request({}), _context())
	assert_eq(response["status"], 400)
	assert_eq(response["body"]["error"]["code"], "TEST_SELECTOR_INVALID")


func test_handle_gdscript_rejects_both_selectors() -> void:
	var response := command.handle_gdscript(_request({"path": "res://addons/godot_tcp_bridge/testing/test_test_case.gd", "dir": "res://addons/godot_tcp_bridge/testing"}), _context())
	assert_eq(response["status"], 400)
	assert_eq(response["body"]["error"]["code"], "TEST_SELECTOR_INVALID")


func test_handle_gdscript_rejects_invalid_path() -> void:
	var response := command.handle_gdscript(_request({"path": "user://test.gd"}), _context())
	assert_eq(response["status"], 400)
	assert_eq(response["body"]["error"]["code"], "TEST_PATH_INVALID")


func test_handle_gdscript_rejects_missing_path() -> void:
	var response := command.handle_gdscript(_request({"path": "res://addons/godot_tcp_bridge/testing/missing_test.gd"}), _context())
	assert_eq(response["status"], 404)
	assert_eq(response["body"]["error"]["code"], "TEST_NOT_FOUND")


func test_handle_gdscript_rejects_invalid_dir() -> void:
	var response := command.handle_gdscript(_request({"dir": "user://tests"}), _context())
	assert_eq(response["status"], 400)
	assert_eq(response["body"]["error"]["code"], "TEST_DIR_INVALID")


func test_handle_gdscript_rejects_missing_dir() -> void:
	var response := command.handle_gdscript(_request({"dir": "res://addons/godot_tcp_bridge/nope"}), _context())
	assert_eq(response["status"], 404)
	assert_eq(response["body"]["error"]["code"], "TEST_DIR_NOT_FOUND")


func _context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"request": request_helper,
		"queue_job": Callable(self, "_queue_job"),
	}


func _request(params: Dictionary) -> Dictionary:
	return {
		"headers": {"content-type": "application/json"},
		"body": JSON.stringify({"request_id": "req-1", "op": "test.gdscript", "params": params}),
	}


func _authorized(_request: Dictionary) -> bool:
	return true


func _params_or_empty(body: Dictionary) -> Dictionary:
	var params: Variant = body.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		return {}
	return params


func _queue_job(kind: String, detail: Dictionary) -> String:
	queued_kind = kind
	queued_detail = detail.duplicate(true)
	return "job-1"
