@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const Protocol = preload("res://addons/godot_tcp_bridge/protocol.gd")
const CommandRequest = preload("res://addons/godot_tcp_bridge/commands/request.gd")
const LightingCommands = preload("res://addons/godot_tcp_bridge/commands/lighting_commands.gd")
const FogCommands = preload("res://addons/godot_tcp_bridge/commands/fog_commands.gd")
const DecalCommands = preload("res://addons/godot_tcp_bridge/commands/decal_commands.gd")
const OccluderCommands = preload("res://addons/godot_tcp_bridge/commands/occluder_commands.gd")
const LODCommands = preload("res://addons/godot_tcp_bridge/commands/lod_commands.gd")
const SoftBodyCommands = preload("res://addons/godot_tcp_bridge/commands/softbody_commands.gd")
const TerrainCommands = preload("res://addons/godot_tcp_bridge/commands/terrain_commands.gd")

const TEMP_ROOT := "res://gdctl_tmp/gdctl_unit_3d_commands"
const TEMP_HEIGHTMAP := TEMP_ROOT + "/heightmap.tres"

var protocol := Protocol.new()
var request_helper := CommandRequest.new()
var lighting_commands := LightingCommands.new()
var fog_commands := FogCommands.new()
var decal_commands := DecalCommands.new()
var occluder_commands := OccluderCommands.new()
var lod_commands := LODCommands.new()
var softbody_commands := SoftBodyCommands.new()
var terrain_commands := TerrainCommands.new()
var scene_root: Node3D = null
var dirty_count := 0


func before_each() -> void:
	_cleanup_temp()
	dirty_count = 0
	scene_root = Node3D.new()
	scene_root.name = "Root"


func after_each() -> void:
	if scene_root != null:
		scene_root.free()
		scene_root = null
	_cleanup_temp()


func test_fog_decal_and_occluder_add_and_set() -> void:
	var fog := fog_commands.handle_add(_request("fog-volume.add", {"parent": "/root/Root", "shape": "ellipsoid", "size": [3, 4, 5], "density": 0.25}), _context())
	assert_eq(fog["status"], 200)
	assert_true(scene_root.has_node("FogVolume"))
	var fog_node := scene_root.get_node("FogVolume") as FogVolume
	assert_eq(fog_node.size, Vector3(3, 4, 5))
	assert_eq((fog_node.material as FogMaterial).density, 0.25)

	var decal := decal_commands.handle_add(_request("decal.add", {"parent": "/root/Root", "size": {"x": 1.5, "y": 2.5, "z": 3.5}}), _context())
	assert_eq(decal["status"], 200)
	assert_true(scene_root.has_node("Decal"))
	var decal_node := scene_root.get_node("Decal") as Decal
	assert_eq(decal_node.size, Vector3(1.5, 2.5, 3.5))

	var fade := decal_commands.handle_set_normal_fade(_request("decal.set-normal-fade", {"path": "/root/Root/Decal", "fade": 0.75}), _context())
	assert_eq(fade["status"], 200)
	assert_eq(decal_node.normal_fade, 0.75)

	var occluder := occluder_commands.handle_add(_request("occluder.add", {"parent": "/root/Root", "shape": "sphere", "size": [4, 4, 4]}), _context())
	assert_eq(occluder["status"], 200)
	assert_true(scene_root.has_node("OccluderInstance3D"))
	assert_true((scene_root.get_node("OccluderInstance3D") as OccluderInstance3D).occluder is SphereOccluder3D)
	assert_eq(dirty_count, 4)


func test_environment_commands_validate_parent_and_node_inputs() -> void:
	var missing_fog_parent := fog_commands.handle_add(_request("fog-volume.add", {}), _context())
	assert_eq(missing_fog_parent["status"], 400)
	assert_eq(missing_fog_parent["body"]["error"]["code"], "FOG_PARAMS_MISSING")

	var missing_decal := decal_commands.handle_set_normal_fade(_request("decal.set-normal-fade", {"path": "/root/Root/Missing"}), _context())
	assert_eq(missing_decal["status"], 404)
	assert_eq(missing_decal["body"]["error"]["code"], "DECAL_NOT_FOUND")

	var missing_occluder_parent := occluder_commands.handle_add(_request("occluder.add", {"parent": "/root/Root/Missing"}), _context())
	assert_eq(missing_occluder_parent["status"], 404)
	assert_eq(missing_occluder_parent["body"]["error"]["code"], "OCCLUDER_PARENT_NOT_FOUND")


func test_lod_set_and_set_many() -> void:
	var mesh_a := MeshInstance3D.new()
	mesh_a.name = "MeshA"
	scene_root.add_child(mesh_a)
	var mesh_b := MeshInstance3D.new()
	mesh_b.name = "MeshB"
	scene_root.add_child(mesh_b)

	var one := lod_commands.handle_set(_request("lod.set", {"path": "/root/Root/MeshA", "begin": 5, "end": 50}), _context())
	assert_eq(one["status"], 200)
	assert_eq(mesh_a.visibility_range_begin, 5.0)
	assert_eq(mesh_a.visibility_range_end, 50.0)

	var many := lod_commands.handle_set_many(_request("lod.set-many", {"entries": [
		{"path": "/root/Root/MeshA", "begin": 10},
		{"path": "/root/Root/MeshB", "end": 80},
		{"path": "/root/Root/Missing", "end": 90},
	]}), _context())
	assert_eq(many["status"], 200)
	assert_eq(many["body"]["result"]["updated"], 2)
	assert_eq(many["body"]["result"]["errors"].size(), 1)

	var missing_entries := lod_commands.handle_set_many(_request("lod.set-many", {}), _context())
	assert_eq(missing_entries["status"], 400)
	assert_eq(missing_entries["body"]["error"]["code"], "LOD_PARAMS_MISSING")


func test_lighting_softbody_and_reflection_probe_paths() -> void:
	var light_missing := lighting_commands.handle_lightmap_bake(_request("lightmap.bake", {}), _context())
	assert_eq(light_missing["status"], 400)
	assert_eq(light_missing["body"]["error"]["code"], "LIGHTMAP_PARAMS_MISSING")

	var voxel_wrong := lighting_commands.handle_voxelgi_bake(_request("voxelgi.bake", {"path": "/root/Root"}), _context())
	assert_eq(voxel_wrong["status"], 400)
	assert_eq(voxel_wrong["body"]["error"]["code"], "VOXELGI_NODE_INVALID")

	var reflection := lighting_commands.handle_reflection_probe_bake(_request("reflection-probe.bake", {}), _context())
	assert_eq(reflection["status"], 200)
	assert_eq(reflection["body"]["result"]["status"], "not_supported")

	var soft_missing := softbody_commands.handle_pin_point(_request("softbody.pin-point", {}), _context())
	assert_eq(soft_missing["status"], 400)
	assert_eq(soft_missing["body"]["error"]["code"], "SOFTBODY_PARAMS_MISSING")

	var soft_wrong := softbody_commands.handle_unpin_point(_request("softbody.unpin-point", {"path": "/root/Root", "point": 0}), _context())
	assert_eq(soft_wrong["status"], 404)
	assert_eq(soft_wrong["body"]["error"]["code"], "SOFTBODY_NOT_FOUND")


func test_terrain_heightmap_import_and_validation() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(TEMP_ROOT))
	var image := Image.create(2, 2, false, Image.FORMAT_RGBA8)
	image.fill(Color(0.5, 0.5, 0.5, 1.0))
	var texture := ImageTexture.create_from_image(image)
	ResourceSaver.save(texture, TEMP_HEIGHTMAP)

	var collision := CollisionShape3D.new()
	collision.name = "Terrain"
	collision.shape = HeightMapShape3D.new()
	scene_root.add_child(collision)

	var imported := terrain_commands.handle_heightmap_import(_request("terrain.heightmap-import", {"path": "/root/Root/Terrain", "texture": TEMP_HEIGHTMAP, "min_height": -1.0, "max_height": 2.0}), _context())
	assert_eq(imported["status"], 200)
	assert_eq(imported["body"]["result"]["width"], 2)
	assert_eq(imported["body"]["result"]["height"], 2)
	assert_eq(dirty_count, 1)

	var missing := terrain_commands.handle_heightmap_import(_request("terrain.heightmap-import", {}), _context())
	assert_eq(missing["status"], 400)
	assert_eq(missing["body"]["error"]["code"], "TERRAIN_PARAMS_MISSING")

	var wrong_node := terrain_commands.handle_heightmap_import(_request("terrain.heightmap-import", {"path": "/root/Root", "texture": TEMP_HEIGHTMAP}), _context())
	assert_eq(wrong_node["status"], 400)
	assert_eq(wrong_node["body"]["error"]["code"], "TERRAIN_NODE_INVALID")


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


func _cleanup_temp() -> void:
	if FileAccess.file_exists(TEMP_HEIGHTMAP):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(TEMP_HEIGHTMAP))
	var dir_path := ProjectSettings.globalize_path(TEMP_ROOT)
	if DirAccess.dir_exists_absolute(dir_path):
		DirAccess.remove_absolute(dir_path)
