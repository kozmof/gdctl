@tool
extends RefCounted


func handle_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Scene creation requires bearer token", {})
	if body.get("op", "") != "scene.create":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected scene.create operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var scene_path: String = String(params.get("path", ""))
	var root_type: String = String(params.get("root_type", ""))
	var root_name: String = String(params.get("root_name", ""))
	var force: bool = bool(params.get("force", false))
	if scene_path == "" or not scene_path.begins_with("res://") or not scene_path.ends_with(".tscn"):
		return context["bridge_error"].call(400, body.get("request_id", ""), "SCENE_PATH_INVALID", "Scene path must be a res:// .tscn path", {"path": scene_path})
	if FileAccess.file_exists(scene_path) and not force:
		return context["bridge_error"].call(409, body.get("request_id", ""), "SCENE_ALREADY_EXISTS", "Scene already exists", {"path": scene_path})
	if root_type == "" or not ClassDB.can_instantiate(root_type):
		return context["bridge_error"].call(400, body.get("request_id", ""), "ROOT_TYPE_INVALID", "Root type cannot be instantiated", {"root_type": root_type})
	if not ClassDB.is_parent_class(root_type, "Node") and root_type != "Node":
		return context["bridge_error"].call(400, body.get("request_id", ""), "ROOT_TYPE_INVALID", "Root type must inherit Node", {"root_type": root_type})
	if root_name == "" or not root_name.is_valid_identifier():
		return context["bridge_error"].call(400, body.get("request_id", ""), "ROOT_NAME_INVALID", "Root name must be a valid identifier", {"root_name": root_name})

	var root: Node = ClassDB.instantiate(root_type) as Node
	if root == null:
		return context["bridge_error"].call(500, body.get("request_id", ""), "ROOT_INSTANTIATE_FAILED", "Could not instantiate root node", {"root_type": root_type})
	root.name = root_name

	var packed := PackedScene.new()
	var pack_err: Error = packed.pack(root)
	if pack_err != OK:
		root.free()
		return context["bridge_error"].call(500, body.get("request_id", ""), "SCENE_PACK_FAILED", "Could not pack scene", {"error": error_string(pack_err)})

	var dir_err: Error = _ensure_resource_dir(scene_path)
	if dir_err != OK:
		root.free()
		return context["bridge_error"].call(500, body.get("request_id", ""), "SCENE_DIR_FAILED", "Could not create scene directory", {"path": scene_path, "error": error_string(dir_err)})

	var save_err: Error = ResourceSaver.save(packed, scene_path)
	root.free()
	if save_err != OK:
		return context["bridge_error"].call(500, body.get("request_id", ""), "SCENE_SAVE_FAILED", "Could not save scene", {"path": scene_path, "error": error_string(save_err)})
	return context["bridge_ok"].call(body.get("request_id", ""), {
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
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Scene open requires bearer token", {})
	if body.get("op", "") != "scene.open":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected scene.open operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var scene_path: String = String(params.get("path", ""))
	if scene_path == "" or not scene_path.begins_with("res://") or not scene_path.ends_with(".tscn"):
		return context["bridge_error"].call(400, body.get("request_id", ""), "SCENE_PATH_INVALID", "Scene path must be a res:// .tscn path", {"path": scene_path})
	if not FileAccess.file_exists(scene_path):
		return context["bridge_error"].call(404, body.get("request_id", ""), "SCENE_NOT_FOUND", "Scene does not exist", {"path": scene_path})
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(500, body.get("request_id", ""), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})

	var job_id: String = String(context["queue_job"].call("scene.open", {
		"path": scene_path,
		"request_id": body.get("request_id", ""),
	}))
	return context["bridge_ok"].call(body.get("request_id", ""), {
		"queued": true,
		"job_id": job_id,
		"path": scene_path,
	})


func handle_instance(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Scene instancing requires bearer token", {})
	if body.get("op", "") != "scene.instance":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected scene.instance operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var parent_path: String = String(params.get("parent", ""))
	var scene_path: String = String(params.get("scene", ""))
	var instance_name: String = String(params.get("name", ""))
	var parent: Node = context["node_by_path"].call(parent_path)
	if parent == null:
		return context["bridge_error"].call(404, body.get("request_id", ""), "NODE_PARENT_NOT_FOUND", "Parent node does not exist", {"parent": parent_path})
	if scene_path == "" or not scene_path.begins_with("res://") or not scene_path.ends_with(".tscn"):
		return context["bridge_error"].call(400, body.get("request_id", ""), "SCENE_PATH_INVALID", "Scene path must be a res:// .tscn path", {"path": scene_path})
	if not FileAccess.file_exists(scene_path):
		return context["bridge_error"].call(404, body.get("request_id", ""), "SCENE_NOT_FOUND", "Scene does not exist", {"path": scene_path})
	if instance_name == "" or not instance_name.is_valid_identifier():
		return context["bridge_error"].call(400, body.get("request_id", ""), "NODE_NAME_INVALID", "Instance name must be a valid identifier", {"name": instance_name})
	if parent.has_node(NodePath(instance_name)):
		return context["bridge_error"].call(409, body.get("request_id", ""), "NODE_ALREADY_EXISTS", "Parent already has a child with that name", {"parent": parent_path, "name": instance_name})

	var packed: PackedScene = ResourceLoader.load(scene_path, "PackedScene", ResourceLoader.CACHE_MODE_REPLACE) as PackedScene
	if packed == null:
		return context["bridge_error"].call(500, body.get("request_id", ""), "SCENE_LOAD_FAILED", "Could not load scene resource", {"path": scene_path})
	var instance: Node = packed.instantiate()
	if instance == null:
		return context["bridge_error"].call(500, body.get("request_id", ""), "SCENE_INSTANCE_FAILED", "Could not instantiate scene", {"path": scene_path})
	instance.name = instance_name
	parent.add_child(instance)
	instance.owner = context["edited_scene_root"].call()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(body.get("request_id", ""), {
		"path": context["logical_path"].call(instance),
		"scene": scene_path,
		"parent": parent_path,
		"name": instance_name,
		"instanced": true,
	})


func handle_save(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Scene save requires bearer token", {})
	if body.get("op", "") != "scene.save":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected scene.save operation", {})
	var params: Dictionary = context["params_or_empty"].call(body)
	if String(params.get("path", "")) != "":
		return context["bridge_error"].call(400, body.get("request_id", ""), "SCENE_SAVE_AS_UNSUPPORTED", "Scene save --path is temporarily unsupported", {})

	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, body.get("request_id", ""), "NO_SCENE_OPEN", "No edited scene is open", {})
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(500, body.get("request_id", ""), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	if root.scene_file_path == "":
		return context["bridge_error"].call(409, body.get("request_id", ""), "SCENE_PATH_MISSING", "Save the scene once in Godot before using gdctl scene save", {"root": context["logical_path"].call(root)})

	var job_id: String = String(context["queue_job"].call("scene.save", {
		"path": root.scene_file_path,
		"root": context["logical_path"].call(root),
		"request_id": body.get("request_id", ""),
	}))
	return context["bridge_ok"].call(body.get("request_id", ""), {
		"queued": true,
		"job_id": job_id,
		"path": root.scene_file_path,
		"root": context["logical_path"].call(root),
	})


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
