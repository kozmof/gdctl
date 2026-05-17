@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const RuntimeBridge = preload("res://addons/godot_tcp_bridge/runtime/runtime_bridge.gd")


func test_mouse_motion_requires_relative() -> void:
	var subject := RuntimeBridge.new()
	var missing: Dictionary = subject._execute_input_step({"type": "mouse_motion"})
	assert_false(bool(missing["ok"]))
	assert_true(String(missing["error"]).contains("relative"))

	var malformed: Dictionary = subject._execute_input_step({"type": "mouse_motion", "relative": [1]})
	assert_false(bool(malformed["ok"]))
	assert_true(String(malformed["error"]).contains("[x, y]"))
	subject.free()


func test_mouse_motion_accepts_relative_pair() -> void:
	var subject := RuntimeBridge.new()
	var result: Dictionary = subject._execute_input_step({"type": "mouse_motion", "relative": [180, -220]})
	assert_true(bool(result["ok"]))
	subject.free()
