@tool
extends RefCounted


func handle_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "scene.create", "Scene creation requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var scene_path: String = String(params.get("path", ""))
	var root_type: String = String(params.get("root_type", ""))
	var root_name: String = String(params.get("root_name", ""))
	var force: bool = bool(params.get("force", false))
	if scene_path == "" or not scene_path.begins_with("res://") or not scene_path.ends_with(".tscn"):
		return context["bridge_error"].call(400, request_id, "SCENE_PATH_INVALID", "Scene path must be a res:// .tscn path", {"path": scene_path})
	if FileAccess.file_exists(scene_path) and not force:
		return context["bridge_error"].call(409, request_id, "SCENE_ALREADY_EXISTS", "Scene already exists", {"path": scene_path})
	if root_type == "" or not ClassDB.can_instantiate(root_type):
		return context["bridge_error"].call(400, request_id, "ROOT_TYPE_INVALID", "Root type cannot be instantiated", {"root_type": root_type})
	if not ClassDB.is_parent_class(root_type, "Node") and root_type != "Node":
		return context["bridge_error"].call(400, request_id, "ROOT_TYPE_INVALID", "Root type must inherit Node", {"root_type": root_type})
	if root_name == "" or not root_name.is_valid_identifier():
		return context["bridge_error"].call(400, request_id, "ROOT_NAME_INVALID", "Root name must be a valid identifier", {"root_name": root_name})

	var root: Node = ClassDB.instantiate(root_type) as Node
	if root == null:
		return context["bridge_error"].call(500, request_id, "ROOT_INSTANTIATE_FAILED", "Could not instantiate root node", {"root_type": root_type})
	root.name = root_name

	var packed := PackedScene.new()
	var pack_err: Error = packed.pack(root)
	if pack_err != OK:
		root.free()
		return context["bridge_error"].call(500, request_id, "SCENE_PACK_FAILED", "Could not pack scene", {"error": error_string(pack_err)})

	var dir_err: Error = _ensure_resource_dir(scene_path)
	if dir_err != OK:
		root.free()
		return context["bridge_error"].call(500, request_id, "SCENE_DIR_FAILED", "Could not create scene directory", {"path": scene_path, "error": error_string(dir_err)})

	var save_err: Error = ResourceSaver.save(packed, scene_path)
	root.free()
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "SCENE_SAVE_FAILED", "Could not save scene", {"path": scene_path, "error": error_string(save_err)})
	return context["bridge_ok"].call(request_id, {
		"path": scene_path,
		"root_type": root_type,
		"root_name": root_name,
		"root_path": "/root/%s" % root_name,
		"created": true,
	})


func handle_tree(_request: Dictionary, context: Dictionary) -> Dictionary:
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, "", "NO_SCENE_OPEN", "No edited scene is open", {})
	return context["http_json"].call(200, {"ok": true, "root": context["node_info"].call(root)})


func handle_open(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "scene.open", "Scene open requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var scene_path: String = String(params.get("path", ""))
	if scene_path == "" or not scene_path.begins_with("res://") or not scene_path.ends_with(".tscn"):
		return context["bridge_error"].call(400, request_id, "SCENE_PATH_INVALID", "Scene path must be a res:// .tscn path", {"path": scene_path})
	if not FileAccess.file_exists(scene_path):
		return context["bridge_error"].call(404, request_id, "SCENE_NOT_FOUND", "Scene does not exist", {"path": scene_path})
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(500, request_id, "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})

	var job_id: String = String(context["queue_job"].call("scene.open", {
		"path": scene_path,
		"request_id": request_id,
	}))
	return context["bridge_ok"].call(request_id, {
		"queued": true,
		"job_id": job_id,
		"path": scene_path,
	})


func handle_instance(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "scene.instance", "Scene instancing requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var parent_path: String = String(params.get("parent", ""))
	var scene_path: String = String(params.get("scene", ""))
	var instance_name: String = String(params.get("name", ""))
	var parent: Node = context["node_by_path"].call(parent_path)
	if parent == null:
		return context["bridge_error"].call(404, request_id, "NODE_PARENT_NOT_FOUND", "Parent node does not exist", {"parent": parent_path})
	if scene_path == "" or not scene_path.begins_with("res://") or not scene_path.ends_with(".tscn"):
		return context["bridge_error"].call(400, request_id, "SCENE_PATH_INVALID", "Scene path must be a res:// .tscn path", {"path": scene_path})
	if not FileAccess.file_exists(scene_path):
		return context["bridge_error"].call(404, request_id, "SCENE_NOT_FOUND", "Scene does not exist", {"path": scene_path})
	if instance_name == "" or not instance_name.is_valid_identifier():
		return context["bridge_error"].call(400, request_id, "NODE_NAME_INVALID", "Instance name must be a valid identifier", {"name": instance_name})
	if parent.has_node(NodePath(instance_name)):
		return context["bridge_error"].call(409, request_id, "NODE_ALREADY_EXISTS", "Parent already has a child with that name", {"parent": parent_path, "name": instance_name})

	var packed: PackedScene = ResourceLoader.load(scene_path, "PackedScene", ResourceLoader.CACHE_MODE_REPLACE) as PackedScene
	if packed == null:
		return context["bridge_error"].call(500, request_id, "SCENE_LOAD_FAILED", "Could not load scene resource", {"path": scene_path})
	var instance: Node = packed.instantiate()
	if instance == null:
		return context["bridge_error"].call(500, request_id, "SCENE_INSTANCE_FAILED", "Could not instantiate scene", {"path": scene_path})
	instance.name = instance_name
	parent.add_child(instance)
	instance.owner = context["edited_scene_root"].call()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(instance),
		"scene": scene_path,
		"parent": parent_path,
		"name": instance_name,
		"instanced": true,
	})


func handle_save(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "scene.save", "Scene save requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	if String(params.get("path", "")) != "":
		return context["bridge_error"].call(400, request_id, "SCENE_SAVE_AS_UNSUPPORTED", "Scene save --path is temporarily unsupported", {})

	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(500, request_id, "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	if root.scene_file_path == "":
		return context["bridge_error"].call(409, request_id, "SCENE_PATH_MISSING", "Save the scene once in Godot before using gdctl scene save", {"root": context["logical_path"].call(root)})

	var job_id: String = String(context["queue_job"].call("scene.save", {
		"path": root.scene_file_path,
		"root": context["logical_path"].call(root),
		"request_id": request_id,
	}))
	return context["bridge_ok"].call(request_id, {
		"queued": true,
		"job_id": job_id,
		"path": root.scene_file_path,
		"root": context["logical_path"].call(root),
	})


func handle_apply(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "scene.apply", "Scene apply requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var dry_run: bool = bool(params.get("dry_run", false))
	if not params.has("tree") or typeof(params.get("tree")) != TYPE_DICTIONARY:
		return context["bridge_error"].call(400, request_id, "SCENE_APPLY_TREE_INVALID", "scene apply requires a tree object", {})
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})

	var tree: Dictionary = params["tree"]
	var root_spec: Dictionary = tree
	if tree.has("root"):
		if typeof(tree.get("root")) != TYPE_DICTIONARY:
			return context["bridge_error"].call(400, request_id, "SCENE_APPLY_ROOT_INVALID", "root must be an object", {})
		root_spec = tree["root"]
	var root_path := String(root_spec.get("path", ""))
	if root_path != "" and root_path != context["logical_path"].call(root):
		return context["bridge_error"].call(400, request_id, "SCENE_APPLY_ROOT_MISMATCH", "Tree root path does not match the edited scene root", {"path": root_path, "root": context["logical_path"].call(root)})

	var counts := {"created": 0, "updated": 0}
	var applied := _apply_existing_node(root, root_spec, context, dry_run, counts)
	if not bool(applied.get("ok", false)):
		return context["bridge_error"].call(400, request_id, String(applied.get("code", "SCENE_APPLY_FAILED")), String(applied.get("message", "Could not apply scene tree")), applied.get("detail", {}))
	if not dry_run:
		context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"root": context["logical_path"].call(root),
		"created": int(counts["created"]),
		"updated": int(counts["updated"]),
		"dry_run": dry_run,
	})


func handle_list(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "scene.list", "Scene list requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var dir: String = String(params.get("dir", "res://"))
	var recursive: bool = bool(params.get("recursive", true))
	if dir == "" or not dir.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "DIR_PATH_INVALID", "Directory must be a res:// path", {"dir": dir})
	if dir.find("..") != -1:
		return context["bridge_error"].call(400, request_id, "DIR_PATH_INVALID", "Directory must not contain ..", {"dir": dir})
	var abs_dir: String = ProjectSettings.globalize_path(dir)
	if not DirAccess.dir_exists_absolute(abs_dir):
		return context["bridge_error"].call(404, request_id, "DIR_NOT_FOUND", "Directory does not exist", {"dir": dir})

	var scenes: Array = []
	_collect_scenes(dir, scenes, recursive)
	return context["bridge_ok"].call(request_id, {
		"dir": dir,
		"scenes": scenes,
	})


func _collect_scenes(dir: String, out: Array, recursive: bool) -> void:
	for f: String in DirAccess.get_files_at(dir):
		if f.ends_with(".tscn"):
			out.append(dir.path_join(f))
	if recursive:
		for d: String in DirAccess.get_directories_at(dir):
			_collect_scenes(dir.path_join(d), out, true)


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))


func _apply_existing_node(node: Node, spec: Dictionary, context: Dictionary, dry_run: bool, counts: Dictionary) -> Dictionary:
	var props_result := _apply_properties(node, spec.get("properties", {}), context, dry_run)
	if not bool(props_result.get("ok", false)):
		return props_result
	if int(props_result.get("updated", 0)) > 0:
		counts["updated"] = int(counts["updated"]) + int(props_result["updated"])
	var children_value: Variant = spec.get("children", [])
	if typeof(children_value) != TYPE_ARRAY:
		return _apply_error("SCENE_APPLY_CHILDREN_INVALID", "children must be an array", {"node": context["logical_path"].call(node)})
	var children: Array = children_value
	for child_value in children:
		if typeof(child_value) != TYPE_DICTIONARY:
			return _apply_error("SCENE_APPLY_CHILD_INVALID", "Each child must be an object", {"node": context["logical_path"].call(node)})
		var child_spec: Dictionary = child_value
		var expanded_result := _expand_grid_child(child_spec)
		if not bool(expanded_result.get("ok", false)):
			return expanded_result
		var expanded_children: Array = expanded_result.get("children", [])
		for expanded_spec in expanded_children:
			var child_result := _apply_child(node, expanded_spec, context, dry_run, counts)
			if not bool(child_result.get("ok", false)):
				return child_result
	return {"ok": true}


func _expand_grid_child(spec: Dictionary) -> Dictionary:
	if not spec.has("grid"):
		return {"ok": true, "children": [spec]}
	var grid_value: Variant = spec.get("grid")
	if typeof(grid_value) != TYPE_DICTIONARY:
		return _apply_error("SCENE_APPLY_GRID_INVALID", "grid must be an object", {})
	var grid: Dictionary = grid_value
	var name_prefix := String(grid.get("name_prefix", "GridNode"))
	var type_name := String(grid.get("type", spec.get("type", "")))
	var count_x := int(grid.get("count_x", 0))
	var count_z := int(grid.get("count_z", 0))
	if name_prefix == "" or type_name == "" or count_x <= 0 or count_z <= 0:
		return _apply_error("SCENE_APPLY_GRID_INVALID", "grid requires name_prefix, type, count_x, and count_z", {})
	var origin_value: Variant = _array_to_vector3(grid.get("origin", [0, 0, 0]))
	var step_x_value: Variant = _array_to_vector3(grid.get("step_x", [1, 0, 0]))
	var step_z_value: Variant = _array_to_vector3(grid.get("step_z", [0, 0, 1]))
	if typeof(origin_value) != TYPE_VECTOR3 or typeof(step_x_value) != TYPE_VECTOR3 or typeof(step_z_value) != TYPE_VECTOR3:
		return _apply_error("SCENE_APPLY_GRID_INVALID", "grid origin, step_x, and step_z must be [x, y, z]", {})
	var origin: Vector3 = origin_value
	var step_x: Vector3 = step_x_value
	var step_z: Vector3 = step_z_value
	var name_format := String(grid.get("name_format", "%s_%03d"))
	var out: Array = []
	var index := 0
	for z in range(count_z):
		for x in range(count_x):
			var child: Dictionary = spec.duplicate(true)
			child.erase("grid")
			child["name"] = name_format % [name_prefix, index]
			child["type"] = type_name
			var props: Dictionary = {}
			if typeof(child.get("properties", {})) == TYPE_DICTIONARY:
				props = child.get("properties", {}).duplicate(true)
			var position := origin + step_x * float(x) + step_z * float(z)
			props["position"] = {"kind": "Vector3", "value": [position.x, position.y, position.z]}
			child["properties"] = props
			out.append(child)
			index += 1
	return {"ok": true, "children": out}


func _apply_child(parent: Node, spec: Dictionary, context: Dictionary, dry_run: bool, counts: Dictionary) -> Dictionary:
	var node_name := String(spec.get("name", ""))
	var type_name := String(spec.get("type", ""))
	if node_name == "" or not node_name.is_valid_identifier():
		return _apply_error("NODE_NAME_INVALID", "Node name must be a valid identifier", {"name": node_name})
	if type_name == "" or not ClassDB.can_instantiate(type_name):
		return _apply_error("NODE_TYPE_INVALID", "Node type cannot be instantiated", {"type": type_name, "name": node_name})
	if not ClassDB.is_parent_class(type_name, "Node") and type_name != "Node":
		return _apply_error("NODE_TYPE_INVALID", "Node type must inherit Node", {"type": type_name, "name": node_name})

	var node: Node = null
	var created := false
	if parent.has_node(NodePath(node_name)):
		node = parent.get_node(NodePath(node_name))
		if node.get_class() != type_name and not ClassDB.is_parent_class(node.get_class(), type_name):
			return _apply_error("NODE_TYPE_MISMATCH", "Existing node type does not match tree spec", {"name": node_name, "existing": node.get_class(), "type": type_name})
	else:
		node = ClassDB.instantiate(type_name) as Node
		if node == null:
			return _apply_error("NODE_INSTANTIATE_FAILED", "Could not instantiate node", {"type": type_name, "name": node_name})
		node.name = node_name
		created = true
		if not dry_run:
			parent.add_child(node)
			node.owner = context["edited_scene_root"].call()

	var result := _apply_existing_node(node, spec, context, dry_run, counts)
	if created and dry_run:
		node.free()
	if not bool(result.get("ok", false)):
		return result
	if created:
		counts["created"] = int(counts["created"]) + 1
	return {"ok": true}


func _apply_properties(target: Object, properties_value: Variant, context: Dictionary, dry_run: bool) -> Dictionary:
	if typeof(properties_value) != TYPE_DICTIONARY:
		return _apply_error("SCENE_APPLY_PROPERTIES_INVALID", "properties must be an object", {})
	var typed_values: RefCounted = context["typed_values"]
	var properties: Dictionary = properties_value
	var updated := 0
	for property in properties.keys():
		var decoded: Dictionary = typed_values.decode(properties[property])
		if not bool(decoded.get("ok", false)):
			return _apply_error("VALUE_INVALID", String(decoded.get("error", "Invalid typed value")), {"property": String(property)})
		if not dry_run:
			target.set(String(property), decoded.get("value"))
		updated += 1
	return {"ok": true, "updated": updated}


func _array_to_vector3(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_ARRAY:
		return null
	var items: Array = raw
	if items.size() != 3:
		return null
	return Vector3(float(items[0]), float(items[1]), float(items[2]))


func _apply_error(code: String, message: String, detail: Dictionary) -> Dictionary:
	return {"ok": false, "code": code, "message": message, "detail": detail}
