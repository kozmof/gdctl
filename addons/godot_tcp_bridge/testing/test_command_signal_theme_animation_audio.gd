@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const SignalCommands = preload("res://addons/godot_tcp_bridge/commands/signal_commands.gd")
const ThemeCommands = preload("res://addons/godot_tcp_bridge/commands/theme_commands.gd")
const AnimationCommands = preload("res://addons/godot_tcp_bridge/commands/animation_commands.gd")
const AudioCommands = preload("res://addons/godot_tcp_bridge/commands/audio_commands.gd")

const TEMP_ROOT := "res://gdctl_tmp/gdctl_unit_media"
const TEMP_THEME := TEMP_ROOT + "/unit_theme.tres"
const TEMP_ANIM := TEMP_ROOT + "/unit_animation.tres"
const UNIT_BUS := "GdctlUnitBus"

class SignalTarget extends Node:
	func _unit_signal_target() -> void:
		pass

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var signal_commands := SignalCommands.new()
var theme_commands := ThemeCommands.new()
var animation_commands := AnimationCommands.new()
var audio_commands := AudioCommands.new()
var nodes: Dictionary = {}
var scene_root: Node = null
var dirty_count := 0


func before_each() -> void:
	dirty_count = 0
	nodes = {}
	scene_root = Node.new()
	scene_root.name = "Root"
	_cleanup_temp()
	_cleanup_audio_bus()


func after_each() -> void:
	_cleanup_audio_bus()
	_cleanup_temp()
	for node in nodes.values():
		if node is Node and (node as Node).get_parent() == null:
			(node as Node).free()
	nodes = {}
	if scene_root != null:
		scene_root.free()
		scene_root = null


func test_signal_connect_and_disconnect() -> void:
	var source := Button.new()
	source.name = "Source"
	var target := SignalTarget.new()
	target.name = "Target"
	nodes["/root/Source"] = source
	nodes["/root/Target"] = target

	var connect_response := signal_commands.handle_connect(_request("signal.connect", {
		"from": "/root/Source",
		"signal": "pressed",
		"to": "/root/Target",
		"method": "_unit_signal_target",
	}), _context())
	assert_eq(connect_response["status"], 200)
	assert_eq(connect_response["body"]["result"]["connected"], true)
	assert_eq(dirty_count, 1)

	var duplicate := signal_commands.handle_connect(_request("signal.connect", {
		"from": "/root/Source",
		"signal": "pressed",
		"to": "/root/Target",
		"method": "_unit_signal_target",
	}), _context())
	assert_eq(duplicate["status"], 409)
	assert_eq(duplicate["body"]["error"]["code"], "SIGNAL_ALREADY_CONNECTED")

	var disconnect_response := signal_commands.handle_disconnect(_request("signal.disconnect", {
		"from": "/root/Source",
		"signal": "pressed",
		"to": "/root/Target",
		"method": "_unit_signal_target",
	}), _context())
	assert_eq(disconnect_response["status"], 200)
	assert_eq(disconnect_response["body"]["result"]["disconnected"], true)
	assert_eq(dirty_count, 2)


func test_signal_commands_validate_inputs() -> void:
	var missing_signal := signal_commands.handle_connect(_request("signal.connect", {"method": "_unit_signal_target"}), _context())
	assert_eq(missing_signal["status"], 400)
	assert_eq(missing_signal["body"]["error"]["code"], "SIGNAL_NAME_INVALID")

	var missing_method := signal_commands.handle_connect(_request("signal.connect", {"signal": "pressed"}), _context())
	assert_eq(missing_method["status"], 400)
	assert_eq(missing_method["body"]["error"]["code"], "METHOD_NAME_INVALID")

	var missing_node := signal_commands.handle_connect(_request("signal.connect", {"from": "/root/Missing", "signal": "pressed", "to": "/root/Target", "method": "_unit_signal_target"}), _context())
	assert_eq(missing_node["status"], 404)
	assert_eq(missing_node["body"]["error"]["code"], "NODE_NOT_FOUND")


func test_theme_create_and_setters() -> void:
	var create_response := theme_commands.handle_create(_request("theme.create", {"path": TEMP_THEME}), _context())
	assert_eq(create_response["status"], 200)
	assert_eq(create_response["body"]["result"]["created"], true)

	var duplicate := theme_commands.handle_create(_request("theme.create", {"path": TEMP_THEME}), _context())
	assert_eq(duplicate["status"], 409)
	assert_eq(duplicate["body"]["error"]["code"], "THEME_ALREADY_EXISTS")

	var color_response := theme_commands.handle_set_color(_request("theme.set-color", {"path": TEMP_THEME, "node_type": "Label", "name": "font_color", "value": "#ff00ff"}), _context())
	assert_eq(color_response["status"], 200)
	assert_eq(color_response["body"]["result"]["data_type"], "color")

	var size_response := theme_commands.handle_set_font_size(_request("theme.set-font-size", {"path": TEMP_THEME, "node_type": "Label", "name": "font_size", "value": 18}), _context())
	assert_eq(size_response["status"], 200)
	assert_eq(size_response["body"]["result"]["data_type"], "font_size")

	var constant_response := theme_commands.handle_set_constant(_request("theme.set-constant", {"path": TEMP_THEME, "node_type": "Panel", "name": "margin", "value": 4}), _context())
	assert_eq(constant_response["status"], 200)
	assert_eq(constant_response["body"]["result"]["data_type"], "constant")


func test_theme_commands_validate_inputs() -> void:
	var bad_path := theme_commands.handle_create(_request("theme.create", {"path": "res://bad.txt"}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "THEME_PATH_INVALID")

	var bad_color := theme_commands.handle_set_color(_request("theme.set-color", {"path": TEMP_THEME, "node_type": "Label", "name": "font_color", "value": "nope"}), _context())
	assert_eq(bad_color["status"], 400)
	assert_eq(bad_color["body"]["error"]["code"], "THEME_COLOR_INVALID")

	var bad_size := theme_commands.handle_set_font_size(_request("theme.set-font-size", {"path": TEMP_THEME, "node_type": "Label", "name": "font_size", "value": 0}), _context())
	assert_eq(bad_size["status"], 400)
	assert_eq(bad_size["body"]["error"]["code"], "THEME_FONT_SIZE_INVALID")


func test_animation_create_track_keyframe_and_length() -> void:
	var create_response := animation_commands.handle_create(_request("animation.create", {"path": TEMP_ANIM, "name": "walk", "length": 1.5, "loop": true}), _context())
	assert_eq(create_response["status"], 200)
	assert_eq(create_response["body"]["result"]["created"], true)

	var track_response := animation_commands.handle_track_add(_request("animation.track-add", {"path": TEMP_ANIM, "animation": "walk", "node_path": "Sprite", "property": "visible"}), _context())
	assert_eq(track_response["status"], 200)
	var track_idx: int = int(track_response["body"]["result"]["track_idx"])

	var key_response := animation_commands.handle_keyframe_add(_request("animation.keyframe-add", {"path": TEMP_ANIM, "animation": "walk", "track_idx": track_idx, "time": 0.25, "value": true}), _context())
	assert_eq(key_response["status"], 200)
	assert_eq(key_response["body"]["result"]["added"], true)

	var length_response := animation_commands.handle_length_set(_request("animation.length-set", {"path": TEMP_ANIM, "animation": "walk", "length": 2.0}), _context())
	assert_eq(length_response["status"], 200)


func test_animation_commands_validate_inputs() -> void:
	var bad_path := animation_commands.handle_create(_request("animation.create", {"path": "res://bad.txt", "name": "walk"}), _context())
	assert_eq(bad_path["status"], 400)
	assert_eq(bad_path["body"]["error"]["code"], "ANIMATION_PATH_INVALID")

	var bad_name := animation_commands.handle_create(_request("animation.create", {"path": TEMP_ANIM, "name": "bad-name"}), _context())
	assert_eq(bad_name["status"], 400)
	assert_eq(bad_name["body"]["error"]["code"], "ANIMATION_NAME_INVALID")

	var missing_track_params := animation_commands.handle_track_add(_request("animation.track-add", {"path": TEMP_ANIM}), _context())
	assert_eq(missing_track_params["status"], 400)
	assert_eq(missing_track_params["body"]["error"]["code"], "ANIMATION_PARAMS_MISSING")

	var bad_length := animation_commands.handle_length_set(_request("animation.length-set", {"path": TEMP_ANIM, "animation": "walk", "length": 0}), _context())
	assert_eq(bad_length["status"], 400)
	assert_eq(bad_length["body"]["error"]["code"], "ANIMATION_LENGTH_INVALID")


func test_animation_player_play_validates_scene_and_node() -> void:
	var missing_path := animation_commands.handle_player_play(_request("animation.player-play", {}), _context())
	assert_eq(missing_path["status"], 400)
	assert_eq(missing_path["body"]["error"]["code"], "ANIMATION_NODE_PATH_MISSING")

	var missing_node := animation_commands.handle_player_play(_request("animation.player-play", {"node_path": "/root/Missing"}), _context())
	assert_eq(missing_node["status"], 404)
	assert_eq(missing_node["body"]["error"]["code"], "NODE_NOT_FOUND")

	var plain := Node.new()
	plain.name = "Plain"
	nodes["/root/Plain"] = plain
	var wrong_type := animation_commands.handle_player_play(_request("animation.player-play", {"node_path": "/root/Plain"}), _context())
	assert_eq(wrong_type["status"], 400)
	assert_eq(wrong_type["body"]["error"]["code"], "ANIMATION_NODE_NOT_PLAYER")


func test_audio_bus_add_volume_and_effect() -> void:
	var add_response := audio_commands.handle_bus_add(_request("audio.bus-add", {"name": UNIT_BUS}), _context())
	assert_eq(add_response["status"], 200)
	assert_eq(add_response["body"]["result"]["created"], true)

	var duplicate := audio_commands.handle_bus_add(_request("audio.bus-add", {"name": UNIT_BUS}), _context())
	assert_eq(duplicate["status"], 409)
	assert_eq(duplicate["body"]["error"]["code"], "AUDIO_BUS_ALREADY_EXISTS")

	var volume_response := audio_commands.handle_bus_volume_set(_request("audio.bus-volume-set", {"name": UNIT_BUS, "volume_db": -6.0}), _context())
	assert_eq(volume_response["status"], 200)

	var effect_response := audio_commands.handle_bus_effect_add(_request("audio.bus-effect-add", {"name": UNIT_BUS, "effect_type": "AudioEffectLowPassFilter"}), _context())
	assert_eq(effect_response["status"], 200)


func test_audio_commands_validate_inputs() -> void:
	var missing_bus := audio_commands.handle_bus_add(_request("audio.bus-add", {}), _context())
	assert_eq(missing_bus["status"], 400)
	assert_eq(missing_bus["body"]["error"]["code"], "AUDIO_BUS_NAME_MISSING")

	var volume_missing := audio_commands.handle_bus_volume_set(_request("audio.bus-volume-set", {"name": "MissingBus"}), _context())
	assert_eq(volume_missing["status"], 404)
	assert_eq(volume_missing["body"]["error"]["code"], "AUDIO_BUS_NOT_FOUND")

	var bad_effect := audio_commands.handle_bus_effect_add(_request("audio.bus-effect-add", {"name": "Master", "effect_type": "Node"}), _context())
	assert_eq(bad_effect["status"], 400)
	assert_eq(bad_effect["body"]["error"]["code"], "AUDIO_EFFECT_TYPE_INVALID")

	var missing_playlist_params := audio_commands.handle_playlist_add(_request("audio.playlist-add", {}), _context())
	assert_eq(missing_playlist_params["status"], 400)
	assert_eq(missing_playlist_params["body"]["error"]["code"], "AUDIO_PLAYLIST_PARAMS_MISSING")

	var missing_listener_path := audio_commands.handle_listener_make_current(_request("audio.listener-make-current", {}), _context())
	assert_eq(missing_listener_path["status"], 400)
	assert_eq(missing_listener_path["body"]["error"]["code"], "AUDIO_LISTENER_PATH_MISSING")


func _context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"request": request_helper,
		"node_by_path": Callable(self, "_node_by_path"),
		"logical_path": Callable(self, "_logical_path"),
		"mark_scene_dirty": Callable(self, "_mark_scene_dirty"),
		"edited_scene_root": Callable(self, "_edited_scene_root"),
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


func _node_by_path(path: String) -> Node:
	return nodes.get(path, null)


func _logical_path(node: Node) -> String:
	return "/root/" + String(node.name)


func _mark_scene_dirty() -> void:
	dirty_count += 1


func _edited_scene_root() -> Node:
	return scene_root


func _cleanup_temp() -> void:
	for path in [TEMP_THEME, TEMP_ANIM]:
		if FileAccess.file_exists(path):
			DirAccess.remove_absolute(ProjectSettings.globalize_path(path))
	var dir_path := ProjectSettings.globalize_path(TEMP_ROOT)
	if DirAccess.dir_exists_absolute(dir_path):
		DirAccess.remove_absolute(dir_path)


func _cleanup_audio_bus() -> void:
	var idx := AudioServer.get_bus_index(UNIT_BUS)
	if idx >= 0:
		AudioServer.remove_bus(idx)
