@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const AnimationTreeCommands = preload("res://addons/godot_tcp_bridge/commands/animation_tree_commands.gd")
const GraphEditCommands = preload("res://addons/godot_tcp_bridge/commands/graph_edit_commands.gd")
const I18NCommands = preload("res://addons/godot_tcp_bridge/commands/i18n_commands.gd")
const AccessibilityCommands = preload("res://addons/godot_tcp_bridge/commands/accessibility_commands.gd")
const TileMapCommands = preload("res://addons/godot_tcp_bridge/commands/tilemap_commands.gd")

const TEMP_ROOT := "res://gdctl_tmp/gdctl_unit_ui_data_commands"
const TEMP_TILESET := TEMP_ROOT + "/unit_tileset.tres"
const TEMP_TEXTURE := TEMP_ROOT + "/unit_texture.tres"

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var animation_tree_commands := AnimationTreeCommands.new()
var graph_edit_commands := GraphEditCommands.new()
var i18n_commands := I18NCommands.new()
var accessibility_commands := AccessibilityCommands.new()
var tilemap_commands := TileMapCommands.new()
var scene_root: Node = null
var dirty_count := 0
var previous_locale := ""


func before_each() -> void:
	_cleanup_temp()
	dirty_count = 0
	previous_locale = TranslationServer.get_locale()
	scene_root = Node.new()
	scene_root.name = "Root"


func after_each() -> void:
	TranslationServer.set_locale(previous_locale)
	if scene_root != null:
		scene_root.free()
		scene_root = null
	_cleanup_temp()


func test_animation_tree_state_transition_blend_and_param() -> void:
	var anim_tree := AnimationTree.new()
	anim_tree.name = "AnimTree"
	anim_tree.tree_root = AnimationNodeStateMachine.new()
	scene_root.add_child(anim_tree)

	var state := animation_tree_commands.handle_add_state(_request("animation-tree.add-state", {"tree_path": "/root/Root/AnimTree", "name": "Idle", "animation": "idle"}), _context())
	assert_eq(state["status"], 200)
	assert_eq(state["body"]["result"]["created"], true)

	var duplicate := animation_tree_commands.handle_add_state(_request("animation-tree.add-state", {"tree_path": "/root/Root/AnimTree", "name": "Idle"}), _context())
	assert_eq(duplicate["status"], 200)
	assert_eq(duplicate["body"]["result"]["created"], false)

	var walk := animation_tree_commands.handle_add_state(_request("animation-tree.add-state", {"tree_path": "/root/Root/AnimTree", "name": "Walk"}), _context())
	assert_eq(walk["status"], 200)
	var transition := animation_tree_commands.handle_add_transition(_request("animation-tree.add-transition", {"tree_path": "/root/Root/AnimTree", "from": "Idle", "to": "Walk", "condition": "moving"}), _context())
	assert_eq(transition["status"], 200)

	var blend := animation_tree_commands.handle_blend_space_2d_add(_request("animation-tree.blend-space-2d-add", {"tree_path": "/root/Root/AnimTree", "state": "Locomotion", "blend_x": "x", "blend_y": "y"}), _context())
	assert_eq(blend["status"], 200)
	assert_true((anim_tree.tree_root as AnimationNodeStateMachine).has_node("Locomotion"))

	var set_param := animation_tree_commands.handle_set_param(_request("animation-tree.set-param", {"tree_path": "/root/Root/AnimTree", "param": "parameters/unit_value", "float": 0.5}), _context())
	assert_eq(set_param["status"], 200)

	var missing := animation_tree_commands.handle_set_param(_request("animation-tree.set-param", {"tree_path": "/root/Root/AnimTree", "param": "parameters/missing"}), _context())
	assert_eq(missing["status"], 400)
	assert_eq(missing["body"]["error"]["code"], "ANIM_TREE_VALUE_MISSING")


func test_animation_tree_validates_wrong_nodes() -> void:
	var plain := Node.new()
	plain.name = "Plain"
	scene_root.add_child(plain)
	var wrong := animation_tree_commands.handle_add_state(_request("animation-tree.add-state", {"tree_path": "/root/Root/Plain", "name": "Idle"}), _context())
	assert_eq(wrong["status"], 404)
	assert_eq(wrong["body"]["error"]["code"], "ANIM_TREE_NOT_FOUND")

	var missing_params := animation_tree_commands.handle_add_transition(_request("animation-tree.add-transition", {}), _context())
	assert_eq(missing_params["status"], 400)
	assert_eq(missing_params["body"]["error"]["code"], "ANIM_TREE_PARAMS_MISSING")


func test_graph_edit_node_add_remove_and_validation() -> void:
	var graph := GraphEdit.new()
	graph.name = "Graph"
	scene_root.add_child(graph)

	var add := graph_edit_commands.handle_node_add(_request("graph-edit.node-add", {"path": "/root/Root/Graph", "name": "A", "position": [12, 34]}), _context())
	assert_eq(add["status"], 200)
	assert_true(graph.has_node("A"))
	assert_eq((graph.get_node("A") as GraphNode).position_offset, Vector2(12, 34))

	var connection_missing := graph_edit_commands.handle_connection_add(_request("graph-edit.connection-add", {"graph": "/root/Root/Graph", "from": "A"}), _context())
	assert_eq(connection_missing["status"], 400)
	assert_eq(connection_missing["body"]["error"]["code"], "GRAPH_EDIT_PARAMS_MISSING")

	var remove := graph_edit_commands.handle_node_remove(_request("graph-edit.node-remove", {"path": "/root/Root/Graph", "name": "A"}), _context())
	assert_eq(remove["status"], 200)

	var wrong := graph_edit_commands.handle_node_add(_request("graph-edit.node-add", {"path": "/root/Root", "name": "Bad"}), _context())
	assert_eq(wrong["status"], 400)
	assert_eq(wrong["body"]["error"]["code"], "GRAPH_EDIT_NODE_INVALID")


func test_i18n_locale_and_string_add() -> void:
	var locale := i18n_commands.handle_locale_set(_request("i18n.locale-set", {"locale": "fr"}), _context())
	assert_eq(locale["status"], 200)
	assert_eq(locale["body"]["result"]["applied"], true)

	var add := i18n_commands.handle_string_add(_request("i18n.string-add", {"key": "HELLO", "locale": "fr", "text": "Bonjour"}), _context())
	assert_eq(add["status"], 200)
	assert_true(i18n_commands._translations.has("fr"))
	assert_eq((i18n_commands._translations["fr"] as Translation).get_message("HELLO"), "Bonjour")

	var missing_locale := i18n_commands.handle_locale_set(_request("i18n.locale-set", {}), _context())
	assert_eq(missing_locale["status"], 400)
	assert_eq(missing_locale["body"]["error"]["code"], "I18N_LOCALE_MISSING")

	var missing_string_params := i18n_commands.handle_string_add(_request("i18n.string-add", {"key": "HELLO"}), _context())
	assert_eq(missing_string_params["status"], 400)
	assert_eq(missing_string_params["body"]["error"]["code"], "I18N_PARAMS_MISSING")


func test_accessibility_configure_stop_and_speak_validation() -> void:
	var configure := accessibility_commands.handle_tts_configure(_request("accessibility.tts-configure", {"pitch": 1.2, "rate": 0.9, "volume": 60, "voice": "unit"}), _context())
	assert_eq(configure["status"], 200)
	assert_eq(configure["body"]["result"]["pitch"], 1.2)
	assert_eq(configure["body"]["result"]["rate"], 0.9)
	assert_eq(configure["body"]["result"]["volume"], 60)
	assert_eq(configure["body"]["result"]["voice"], "unit")

	var stop := accessibility_commands.handle_tts_stop(_request("accessibility.tts-stop", {}), _context())
	assert_eq(stop["status"], 200)

	var speak_missing := accessibility_commands.handle_tts_speak(_request("accessibility.tts-speak", {}), _context())
	assert_eq(speak_missing["status"], 400)
	assert_eq(speak_missing["body"]["error"]["code"], "TTS_TEXT_MISSING")


func test_tilemap_tileset_create_source_cell_rect_and_clear() -> void:
	_create_texture_resource()
	var create := tilemap_commands.handle_tileset_create(_request("tilemap.tileset-create", {"path": TEMP_TILESET, "tile_width": 8, "tile_height": 8}), _context())
	assert_eq(create["status"], 200)
	assert_true(ResourceLoader.exists(TEMP_TILESET))

	var duplicate := tilemap_commands.handle_tileset_create(_request("tilemap.tileset-create", {"path": TEMP_TILESET}), _context())
	assert_eq(duplicate["status"], 409)
	assert_eq(duplicate["body"]["error"]["code"], "TILESET_ALREADY_EXISTS")

	var source := tilemap_commands.handle_source_add(_request("tilemap.source-add", {"path": TEMP_TILESET, "texture": TEMP_TEXTURE, "tile_width": 8, "tile_height": 8}), _context())
	assert_eq(source["status"], 200)

	var tilemap := TileMap.new()
	tilemap.name = "TileMap"
	tilemap.tile_set = ResourceLoader.load(TEMP_TILESET, "TileSet", ResourceLoader.CACHE_MODE_REPLACE)
	scene_root.add_child(tilemap)
	if tilemap.get_layers_count() == 0:
		tilemap.add_layer(0)

	var cell := tilemap_commands.handle_cell_set(_request("tilemap.cell-set", {"node": "/root/Root/TileMap", "layer": 0, "x": 1, "y": 2, "source_id": 0}), _context())
	assert_eq(cell["status"], 200)

	var rect := tilemap_commands.handle_cell_set_rect(_request("tilemap.cell-set-rect", {"node": "/root/Root/TileMap", "layer": 0, "x": 0, "y": 0, "width": 2, "height": 2, "source_id": 0}), _context())
	assert_eq(rect["status"], 200)
	assert_eq(rect["body"]["result"]["cells"], 4)

	var clear := tilemap_commands.handle_cell_clear(_request("tilemap.cell-clear", {"node": "/root/Root/TileMap", "layer": 0, "x": 1, "y": 2}), _context())
	assert_eq(clear["status"], 200)


func test_tilemap_validates_inputs() -> void:
	var bad_tileset := tilemap_commands.handle_tileset_create(_request("tilemap.tileset-create", {"path": "", "tile_width": 8}), _context())
	assert_eq(bad_tileset["status"], 400)
	assert_eq(bad_tileset["body"]["error"]["code"], "TILESET_PATH_INVALID")

	var bad_source_texture := tilemap_commands.handle_source_add(_request("tilemap.source-add", {"path": TEMP_TILESET, "texture": "user://bad.png"}), _context())
	assert_eq(bad_source_texture["status"], 400)
	assert_eq(bad_source_texture["body"]["error"]["code"], "TILESET_TEXTURE_INVALID")

	var missing_node := tilemap_commands.handle_cell_set(_request("tilemap.cell-set", {}), _context())
	assert_eq(missing_node["status"], 400)
	assert_eq(missing_node["body"]["error"]["code"], "TILEMAP_NODE_MISSING")

	var bad_rect := tilemap_commands.handle_cell_set_rect(_request("tilemap.cell-set-rect", {"node": "/root/Root", "width": 0, "height": 1}), _context())
	assert_eq(bad_rect["status"], 400)
	assert_eq(bad_rect["body"]["error"]["code"], "TILEMAP_RECT_INVALID")


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


func _edited_scene_root() -> Node:
	return scene_root


func _node_by_path(path: String) -> Node:
	if scene_root == null:
		return null
	if path == "/root/Root":
		return scene_root
	var prefix := "/root/Root/"
	if path.begins_with(prefix):
		var local_path := path.substr(prefix.length())
		if scene_root.has_node(NodePath(local_path)):
			return scene_root.get_node(NodePath(local_path))
	return null


func _logical_path(node: Node) -> String:
	if node == scene_root:
		return "/root/Root"
	var names: Array[String] = []
	var current := node
	while current != null and current != scene_root:
		names.push_front(String(current.name))
		current = current.get_parent()
	if current == scene_root:
		return "/root/Root/" + "/".join(names)
	return "/root/" + String(node.name)


func _mark_scene_dirty() -> void:
	dirty_count += 1


func _create_texture_resource() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(TEMP_ROOT))
	var image := Image.create(16, 16, false, Image.FORMAT_RGBA8)
	image.fill(Color.WHITE)
	var texture := ImageTexture.create_from_image(image)
	ResourceSaver.save(texture, TEMP_TEXTURE)


func _cleanup_temp() -> void:
	for path in [TEMP_TILESET, TEMP_TEXTURE]:
		if FileAccess.file_exists(path):
			DirAccess.remove_absolute(ProjectSettings.globalize_path(path))
	var dir_path := ProjectSettings.globalize_path(TEMP_ROOT)
	if DirAccess.dir_exists_absolute(dir_path):
		DirAccess.remove_absolute(dir_path)
