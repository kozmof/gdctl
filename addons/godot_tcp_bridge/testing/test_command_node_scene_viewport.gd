@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const NodeCommands = preload("res://addons/godot_tcp_bridge/commands/node_commands.gd")
const SceneCommands = preload("res://addons/godot_tcp_bridge/commands/scene_commands.gd")
const ViewportCommands = preload("res://addons/godot_tcp_bridge/commands/viewport_commands.gd")
const TypedValues = preload("res://addons/godot_tcp_bridge/typed_values.gd")

const TEMP_ROOT := "res://gdctl_tmp/gdctl_unit_scene_commands"
const TEMP_SCENE := TEMP_ROOT + "/unit_scene.tscn"
const TEMP_SCRIPT := TEMP_ROOT + "/unit_node_script.gd"

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var node_commands := NodeCommands.new()
var scene_commands := SceneCommands.new()
var viewport_commands := ViewportCommands.new()
var typed_values := TypedValues.new()
var scene_root: Node = null
var dirty_count := 0
var queued_jobs: Array = []
var plugin_available := true


func before_each() -> void:
	_cleanup_temp()
	dirty_count = 0
	queued_jobs = []
	plugin_available = true
	scene_root = Node.new()
	scene_root.name = "Root"


func after_each() -> void:
	if scene_root != null:
		scene_root.free()
		scene_root = null
	_cleanup_temp()


func test_node_add_set_get_groups_duplicate_and_remove() -> void:
	var add_response := node_commands.handle_add(_request("node.add", {
		"parent": "/root/Root",
		"type": "Label",
		"name": "Title",
		"props": {"text": _value("String", "Hello")},
	}), _context())
	assert_eq(add_response["status"], 200)
	assert_eq(add_response["body"]["result"]["path"], "/root/Root/Title")
	assert_eq(dirty_count, 1)
	var title := scene_root.get_node("Title") as Label
	assert_eq(title.text, "Hello")

	var set_response := node_commands.handle_set(_request("node.set", {"path": "/root/Root/Title", "property": "text", "value": _value("String", "Updated")}), _context())
	assert_eq(set_response["status"], 200)
	assert_eq(title.text, "Updated")

	var get_response := node_commands.handle_get(_request("node.get", {"path": "/root/Root/Title", "property": "text"}), _context())
	assert_eq(get_response["status"], 200)
	assert_eq(get_response["body"]["result"]["value"]["value"], "Updated")

	var set_many_response := node_commands.handle_set_many(_request("node.set_many", {"path": "/root/Root/Title", "properties": {"visible": _value("bool", false)}}), _context())
	assert_eq(set_many_response["status"], 200)
	assert_eq(set_many_response["body"]["result"]["updated"], 1)
	assert_eq(title.visible, false)

	var group_add := node_commands.handle_group_add(_request("node.group_add", {"path": "/root/Root/Title", "group": "ui"}), _context())
	assert_eq(group_add["status"], 200)
	assert_true(title.is_in_group("ui"))

	var group_list := node_commands.handle_group_list(_request("node.group_list", {"path": "/root/Root/Title"}), _context())
	assert_eq(group_list["status"], 200)
	assert_true(group_list["body"]["result"]["groups"].has("ui"))

	var duplicate := node_commands.handle_duplicate(_request("node.duplicate", {"path": "/root/Root/Title", "name": "TitleCopy"}), _context())
	assert_eq(duplicate["status"], 200)
	assert_true(scene_root.has_node("TitleCopy"))

	var remove_group := node_commands.handle_group_remove(_request("node.group_remove", {"path": "/root/Root/Title", "group": "ui"}), _context())
	assert_eq(remove_group["status"], 200)
	assert_false(title.is_in_group("ui"))

	var remove := node_commands.handle_remove(_request("node.remove", {"path": "/root/Root/TitleCopy"}), _context())
	assert_eq(remove["status"], 200)


func test_node_rename_move_resource_script_and_properties() -> void:
	var a := Node.new()
	a.name = "A"
	var b := Node.new()
	b.name = "B"
	scene_root.add_child(a)
	scene_root.add_child(b)

	var rename := node_commands.handle_rename(_request("node.rename", {"path": "/root/Root/A", "name": "Renamed"}), _context())
	assert_eq(rename["status"], 200)
	assert_eq(a.name, "Renamed")

	var move := node_commands.handle_move(_request("node.move", {"path": "/root/Root/Renamed", "parent": "/root/Root/B"}), _context())
	assert_eq(move["status"], 200)
	assert_true(b.has_node("Renamed"))

	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(TEMP_ROOT))
	var script_file := FileAccess.open(TEMP_SCRIPT, FileAccess.WRITE)
	script_file.store_string("extends Node\nvar unit_value := 3\n")
	script_file.close()
	var attach := node_commands.handle_attach_script(_request("node.attach_script", {"path": "/root/Root/B/Renamed", "script": TEMP_SCRIPT}), _context())
	assert_eq(attach["status"], 200)
	assert_true(a.get_script() != null)

	var resource := Theme.new()
	ResourceSaver.save(resource, TEMP_ROOT + "/node_theme.tres")
	var label := Label.new()
	label.name = "Label"
	scene_root.add_child(label)
	var set_resource := node_commands.handle_set_resource(_request("node.set_resource", {"path": "/root/Root/Label", "property": "theme", "resource": TEMP_ROOT + "/node_theme.tres"}), _context())
	assert_eq(set_resource["status"], 200)
	assert_true(label.theme is Theme)

	var props := node_commands.handle_list_properties(_request("node.list_properties", {"path": "/root/Root/Label"}), _context())
	assert_eq(props["status"], 200)
	assert_true(props["body"]["result"]["properties"].size() > 0)


func test_node_commands_validate_inputs() -> void:
	var bad_add := node_commands.handle_add(_request("node.add", {"parent": "/root/Root", "type": "RefCounted", "name": "Bad"}), _context())
	assert_eq(bad_add["status"], 400)
	assert_eq(bad_add["body"]["error"]["code"], "NODE_TYPE_INVALID")

	var bad_name := node_commands.handle_rename(_request("node.rename", {"path": "/root/Root", "name": "bad-name"}), _context())
	assert_eq(bad_name["status"], 400)
	assert_eq(bad_name["body"]["error"]["code"], "NODE_NAME_INVALID")

	var bad_set := node_commands.handle_set(_request("node.set", {"path": "/root/Root", "property": "name"}), _context())
	assert_eq(bad_set["status"], 400)
	assert_eq(bad_set["body"]["error"]["code"], "VALUE_MISSING")

	var bad_group_remove := node_commands.handle_group_remove(_request("node.group_remove", {"path": "/root/Root", "group": "missing"}), _context())
	assert_eq(bad_group_remove["status"], 404)
	assert_eq(bad_group_remove["body"]["error"]["code"], "GROUP_NOT_FOUND")


func test_scene_create_list_instance_apply_and_blueprint() -> void:
	var create := scene_commands.handle_create(_request("scene.create", {"path": TEMP_SCENE, "root_type": "Node3D", "root_name": "UnitScene"}), _context())
	assert_eq(create["status"], 200)
	assert_true(FileAccess.file_exists(TEMP_SCENE))

	var list := scene_commands.handle_list(_request("scene.list", {"dir": TEMP_ROOT, "recursive": true}), _context())
	assert_eq(list["status"], 200)
	assert_true(list["body"]["result"]["scenes"].has(TEMP_SCENE))

	var instance := scene_commands.handle_instance(_request("scene.instance", {"parent": "/root/Root", "scene": TEMP_SCENE, "name": "Instanced"}), _context())
	assert_eq(instance["status"], 200)
	assert_true(scene_root.has_node("Instanced"))

	var apply := scene_commands.handle_apply(_request("scene.apply", {"tree": {"children": [{"type": "Label", "name": "Applied", "properties": {"text": _value("String", "Hi")}}]}}), _context())
	assert_eq(apply["status"], 200)
	assert_true(scene_root.has_node("Applied"))
	assert_eq((scene_root.get_node("Applied") as Label).text, "Hi")

	var blueprint := scene_commands.handle_apply_blueprint(_request("scene.apply.blueprint", {"path": TEMP_SCENE, "blueprint": "hud_label", "dry_run": true}), _context())
	assert_eq(blueprint["status"], 200)
	assert_eq(blueprint["body"]["result"]["created"], 2)


func test_scene_tree_open_save_and_validations() -> void:
	var tree := scene_commands.handle_tree({}, _context())
	assert_eq(tree["status"], 200)
	assert_eq(tree["body"]["root"]["name"], "Root")

	var bad_create := scene_commands.handle_create(_request("scene.create", {"path": "res://bad.txt", "root_type": "Node", "root_name": "Root"}), _context())
	assert_eq(bad_create["status"], 400)
	assert_eq(bad_create["body"]["error"]["code"], "SCENE_PATH_INVALID")

	var create := scene_commands.handle_create(_request("scene.create", {"path": TEMP_SCENE, "root_type": "Node", "root_name": "SavedRoot", "force": true}), _context())
	assert_eq(create["status"], 200)
	var open := scene_commands.handle_open(_request("scene.open", {"path": TEMP_SCENE}), _context())
	assert_eq(open["status"], 200)
	assert_eq(open["body"]["result"]["queued"], true)
	assert_eq(queued_jobs[0]["op"], "scene.open")

	scene_root.scene_file_path = TEMP_SCENE
	var save := scene_commands.handle_save(_request("scene.save", {}), _context())
	assert_eq(save["status"], 200)
	assert_eq(save["body"]["result"]["queued"], true)

	var bad_apply := scene_commands.handle_apply(_request("scene.apply", {"tree": {"children": "bad"}}), _context())
	assert_eq(bad_apply["status"], 400)
	assert_eq(bad_apply["body"]["error"]["code"], "SCENE_APPLY_CHILDREN_INVALID")

	var bad_blueprint := scene_commands.handle_apply_blueprint(_request("scene.apply.blueprint", {"path": TEMP_SCENE, "blueprint": "missing"}), _context())
	assert_eq(bad_blueprint["status"], 400)
	assert_eq(bad_blueprint["body"]["error"]["code"], "BLUEPRINT_NOT_FOUND")


func test_viewport_add_set_camera_screenshot_and_validations() -> void:
	var add := viewport_commands.handle_add(_request("viewport.add", {"parent": "/root/Root", "width": 400, "height": 300, "add_camera": true}), _context())
	assert_eq(add["status"], 200)
	var viewport := scene_root.get_node("SubViewport") as SubViewport
	assert_eq(viewport.size, Vector2i(400, 300))
	assert_true(viewport.has_node("Camera3D"))

	var set_size := viewport_commands.handle_set_size(_request("viewport.set-size", {"path": "/root/Root/SubViewport", "width": 640, "height": 360}), _context())
	assert_eq(set_size["status"], 200)
	assert_eq(viewport.size, Vector2i(640, 360))

	var camera := viewport_commands.handle_camera_assign(_request("viewport.camera-assign", {"viewport": "/root/Root/SubViewport", "camera": "/root/Root/SubViewport/Camera3D"}), _context())
	assert_eq(camera["status"], 200)
	assert_eq((viewport.get_node("Camera3D") as Camera3D).current, true)

	var screenshot := viewport_commands.handle_screenshot(_request("viewport.screenshot", {"kind": "3d", "index": 1}), _context())
	assert_eq(screenshot["status"], 200)
	assert_eq(screenshot["body"]["result"]["queued"], true)

	var bad_size := viewport_commands.handle_set_size(_request("viewport.set-size", {"width": 0, "height": 1}), _context())
	assert_eq(bad_size["status"], 400)
	assert_eq(bad_size["body"]["error"]["code"], "VIEWPORT_SIZE_INVALID")

	var bad_window_assign := viewport_commands.handle_window_assign_viewport(_request("window.assign-viewport", {}), _context())
	assert_eq(bad_window_assign["status"], 400)
	assert_eq(bad_window_assign["body"]["error"]["code"], "WINDOW_ASSIGN_PARAMS_MISSING")

	var bad_screenshot := viewport_commands.handle_screenshot(_request("viewport.screenshot", {"kind": "bad"}), _context())
	assert_eq(bad_screenshot["status"], 400)
	assert_eq(bad_screenshot["body"]["error"]["code"], "VIEWPORT_KIND_INVALID")


func _context() -> Dictionary:
	return {
		"json_body_or_error": Callable(protocol, "json_body_or_error"),
		"params_or_empty": Callable(self, "_params_or_empty"),
		"authorized": Callable(self, "_authorized"),
		"bridge_error": Callable(protocol, "bridge_error"),
		"bridge_ok": Callable(protocol, "bridge_ok"),
		"http_json": Callable(protocol, "http_json"),
		"request": request_helper,
		"typed_values": typed_values,
		"node_by_path": Callable(self, "_node_by_path"),
		"logical_path": Callable(self, "_logical_path"),
		"mark_scene_dirty": Callable(self, "_mark_scene_dirty"),
		"edited_scene_root": Callable(self, "_edited_scene_root"),
		"node_info": Callable(self, "_node_info"),
		"editor_plugin_available": Callable(self, "_editor_plugin_available"),
		"queue_job": Callable(self, "_queue_job"),
	}


func _request(op: String, params: Dictionary) -> Dictionary:
	return {
		"headers": {"content-type": "application/json"},
		"body": JSON.stringify({"request_id": "req-1", "op": op, "params": params}),
	}


func _value(kind: String, value: Variant) -> Dictionary:
	return {"kind": kind, "value": value}


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


func _node_info(node: Node) -> Dictionary:
	var children: Array = []
	for child in node.get_children():
		children.append(_node_info(child))
	return {"name": String(node.name), "path": _logical_path(node), "type": node.get_class(), "children": children}


func _editor_plugin_available() -> bool:
	return plugin_available


func _queue_job(op: String, params: Dictionary) -> String:
	var id := "job-%d" % queued_jobs.size()
	queued_jobs.append({"id": id, "op": op, "params": params})
	return id


func _cleanup_temp() -> void:
	var files := [TEMP_SCRIPT, TEMP_ROOT + "/node_theme.tres", TEMP_SCENE]
	for path in files:
		if FileAccess.file_exists(path):
			DirAccess.remove_absolute(ProjectSettings.globalize_path(path))
	var dir_path := ProjectSettings.globalize_path(TEMP_ROOT)
	if DirAccess.dir_exists_absolute(dir_path):
		DirAccess.remove_absolute(dir_path)
