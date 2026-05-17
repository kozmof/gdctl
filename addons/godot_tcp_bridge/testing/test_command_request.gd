@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var authorized := true


func before_each() -> void:
	authorized = true


func test_require_body_accepts_valid_request() -> void:
	var result := request_helper.require_body(_request("script.check", {"path": "res://test.gd"}), _context(), "script.check", "Nope")
	assert_true(result["ok"])
	assert_eq(result["request_id"], "req-1")
	assert_eq(result["params"]["path"], "res://test.gd")


func test_require_body_rejects_unauthorized_request() -> void:
	authorized = false
	var result := request_helper.require_body(_request("script.check", {}), _context(), "script.check", "Needs auth")
	assert_false(result["ok"])
	assert_eq(result["error_response"]["status"], 401)
	assert_eq(result["error_response"]["body"]["error"]["code"], "UNAUTHORIZED")
	assert_eq(result["error_response"]["body"]["error"]["message"], "Needs auth")


func test_require_body_rejects_wrong_operation() -> void:
	var result := request_helper.require_body(_request("node.add", {}), _context(), "script.check", "Nope")
	assert_false(result["ok"])
	assert_eq(result["error_response"]["status"], 400)
	assert_eq(result["error_response"]["body"]["error"]["code"], "INVALID_OPERATION")


func test_require_body_rejects_non_json_content_type() -> void:
	var request := _request("script.check", {})
	request["headers"] = {"content-type": "text/plain"}
	var result := request_helper.require_body(request, _context(), "script.check", "Nope")
	assert_false(result["ok"])
	assert_eq(result["error_response"]["status"], 415)
	assert_eq(result["error_response"]["body"]["error"]["code"], "UNSUPPORTED_MEDIA_TYPE")


func test_require_body_rejects_invalid_json_body() -> void:
	var request := _request("script.check", {})
	request["body"] = "{"
	var result := request_helper.require_body(request, _context(), "script.check", "Nope")
	assert_false(result["ok"])
	assert_eq(result["error_response"]["status"], 400)
	assert_eq(result["error_response"]["body"]["error"]["code"], "INVALID_JSON")


func test_require_body_uses_empty_params_when_missing_or_invalid() -> void:
	var request := _request("script.check", {})
	request["body"] = JSON.stringify({"request_id": "req-1", "op": "script.check", "params": "bad"})
	var result := request_helper.require_body(request, _context(), "script.check", "Nope")
	assert_true(result["ok"])
	assert_eq(result["params"].size(), 0)


func _context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_error": Callable(protocol, "bridge_error"),
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
