@tool
extends RefCounted


func handle_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.add", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var parent_path: String = String(params.get("parent", ""))
	var type_name: String = String(params.get("type", ""))
	var node_name: String = String(params.get("name", ""))
	var dry_run: bool = bool(params.get("dry_run", false))
	var parent: Node = context["node_by_path"].call(parent_path)
	if parent == null:
		return context["bridge_error"].call(404, request_id, "NODE_PARENT_NOT_FOUND", "Parent node does not exist", {"parent": parent_path})
	if type_name == "" or not ClassDB.can_instantiate(type_name):
		return context["bridge_error"].call(400, request_id, "NODE_TYPE_INVALID", "Node type cannot be instantiated", {"type": type_name})
	if not ClassDB.is_parent_class(type_name, "Node") and type_name != "Node":
		return context["bridge_error"].call(400, request_id, "NODE_TYPE_INVALID", "Node type must inherit Node", {"type": type_name})
	if node_name == "" or not node_name.is_valid_identifier():
		return context["bridge_error"].call(400, request_id, "NODE_NAME_INVALID", "Node name must be a valid identifier", {"name": node_name})

	var path: String = "%s/%s" % [parent_path.rstrip("/"), node_name]
	if dry_run:
		return context["bridge_ok"].call(request_id, {"path": path, "dry_run": true})

	var node: Node = ClassDB.instantiate(type_name) as Node
	node.name = node_name
	parent.add_child(node)
	node.owner = context["edited_scene_root"].call()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"path": context["logical_path"].call(node)})


func handle_remove(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.remove", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var dry_run: bool = bool(params.get("dry_run", false))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if node == context["edited_scene_root"].call():
		return context["bridge_error"].call(400, request_id, "CANNOT_REMOVE_SCENE_ROOT", "Scene root cannot be removed", {"path": path})
	if dry_run:
		return context["bridge_ok"].call(request_id, {"removed": path, "dry_run": true})

	node.get_parent().remove_child(node)
	node.queue_free()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"removed": path})


func handle_rename(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.rename", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var new_name: String = String(params.get("name", ""))
	var dry_run: bool = bool(params.get("dry_run", false))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if new_name == "" or not new_name.is_valid_identifier():
		return context["bridge_error"].call(400, request_id, "NODE_NAME_INVALID", "Node name must be a valid identifier", {"name": new_name})

	var old_path: String = context["logical_path"].call(node)
	var new_path: String = _renamed_path(old_path, new_name)
	if dry_run:
		return context["bridge_ok"].call(request_id, {"old_path": old_path, "path": new_path, "dry_run": true})

	node.name = new_name
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"old_path": old_path,
		"path": context["logical_path"].call(node),
	})


func handle_move(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.move", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var parent_path: String = String(params.get("parent", ""))
	var index: int = int(params.get("index", -1))
	var dry_run: bool = bool(params.get("dry_run", false))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	var new_parent: Node = context["node_by_path"].call(parent_path)
	if new_parent == null:
		return context["bridge_error"].call(404, request_id, "NODE_PARENT_NOT_FOUND", "Parent node does not exist", {"parent": parent_path})
	if node == context["edited_scene_root"].call():
		return context["bridge_error"].call(400, request_id, "CANNOT_MOVE_SCENE_ROOT", "Scene root cannot be moved", {"path": path})
	if node == new_parent or node.is_ancestor_of(new_parent):
		return context["bridge_error"].call(400, request_id, "NODE_MOVE_INVALID", "Node cannot be moved under itself or its descendant", {"path": path, "parent": parent_path})

	var old_path: String = context["logical_path"].call(node)
	var new_path: String = "%s/%s" % [parent_path.rstrip("/"), node.name]
	if dry_run:
		return context["bridge_ok"].call(request_id, {"old_path": old_path, "path": new_path, "dry_run": true})

	node.reparent(new_parent)
	if index >= 0:
		new_parent.move_child(node, index)
	node.owner = context["edited_scene_root"].call()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"old_path": old_path,
		"path": context["logical_path"].call(node),
	})


func handle_get(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.get", "Node property read requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var property: String = String(params.get("property", ""))
	if property == "":
		return context["bridge_error"].call(400, request_id, "PROPERTY_INVALID", "Property name is required", {})
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})

	var typed_values: RefCounted = context["typed_values"]
	var encoded: Dictionary = typed_values.encode(node.get(property))
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"property": property,
		"value": encoded,
	})


func handle_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.set", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var property: String = String(params.get("property", ""))
	if property == "":
		return context["bridge_error"].call(400, request_id, "PROPERTY_INVALID", "Property name is required", {})
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if not params.has("value"):
		return context["bridge_error"].call(400, request_id, "VALUE_MISSING", "Typed value is required", {})

	var typed_values: RefCounted = context["typed_values"]
	var decoded: Dictionary = typed_values.decode(params.get("value"))
	if not bool(decoded.get("ok", false)):
		return context["bridge_error"].call(400, request_id, "VALUE_INVALID", String(decoded.get("error", "Invalid typed value")), {})
	node.set(property, decoded.get("value"))
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"property": property,
		"value": typed_values.encode(node.get(property)),
	})


func handle_set_resource(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.set_resource", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var property: String = String(params.get("property", ""))
	var resource_path: String = String(params.get("resource", ""))
	if property == "":
		return context["bridge_error"].call(400, request_id, "PROPERTY_INVALID", "Property name is required", {})
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if resource_path == "" or not resource_path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "RESOURCE_PATH_INVALID", "Resource path must be a res:// path", {"path": resource_path})
	if not FileAccess.file_exists(resource_path):
		return context["bridge_error"].call(404, request_id, "RESOURCE_NOT_FOUND", "Resource does not exist", {"path": resource_path})

	var resource: Resource = ResourceLoader.load(resource_path, "", ResourceLoader.CACHE_MODE_REPLACE)
	if resource == null:
		return context["bridge_error"].call(500, request_id, "RESOURCE_LOAD_FAILED", "Could not load resource", {"path": resource_path})
	node.set(property, resource)
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"property": property,
		"resource": resource_path,
		"set": true,
	})


func handle_attach_script(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.attach_script", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var script_path: String = String(params.get("script", ""))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if script_path == "" or not script_path.begins_with("res://") or not script_path.ends_with(".gd"):
		return context["bridge_error"].call(400, request_id, "SCRIPT_PATH_INVALID", "Script path must be a res:// .gd path", {"path": script_path})
	if not FileAccess.file_exists(script_path):
		return context["bridge_error"].call(404, request_id, "SCRIPT_NOT_FOUND", "Script does not exist", {"path": script_path})

	var source: String = FileAccess.get_file_as_string(script_path)
	var syntax_error: Dictionary = _script_syntax_error(script_path, source, request_id, context)
	if not syntax_error.is_empty():
		return syntax_error

	var script: Script = ResourceLoader.load(script_path, "", ResourceLoader.CACHE_MODE_REPLACE) as Script
	if script == null:
		return context["bridge_error"].call(500, request_id, "SCRIPT_LOAD_FAILED", "Could not load script resource", {"path": script_path})
	node.set_script(script)
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"script": script_path,
		"attached": true,
	})


func handle_group_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.group_add", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var group: String = String(params.get("group", ""))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if group == "":
		return context["bridge_error"].call(400, request_id, "GROUP_NAME_INVALID", "Group name is required", {})
	node.add_to_group(group)
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"group": group,
		"added": true,
	})


func handle_group_remove(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.group_remove", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var group: String = String(params.get("group", ""))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if group == "":
		return context["bridge_error"].call(400, request_id, "GROUP_NAME_INVALID", "Group name is required", {})
	if not node.is_in_group(group):
		return context["bridge_error"].call(404, request_id, "GROUP_NOT_FOUND", "Node is not in group", {"path": path, "group": group})
	node.remove_from_group(group)
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"group": group,
		"removed": true,
	})


func handle_group_list(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.group_list", "Node group read requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"groups": node.get_groups(),
	})


func handle_duplicate(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.duplicate", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var node_name: String = String(params.get("name", ""))
	var parent_path: String = String(params.get("parent", ""))
	var dry_run: bool = bool(params.get("dry_run", false))

	var source: Node = context["node_by_path"].call(path)
	if source == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Source node does not exist", {"path": path})
	if source == context["edited_scene_root"].call():
		return context["bridge_error"].call(400, request_id, "CANNOT_DUPLICATE_SCENE_ROOT", "Scene root cannot be duplicated", {"path": path})
	if node_name == "" or not node_name.is_valid_identifier():
		return context["bridge_error"].call(400, request_id, "NODE_NAME_INVALID", "Node name must be a valid identifier", {"name": node_name})

	var target_parent: Node
	if parent_path == "":
		target_parent = source.get_parent()
	else:
		target_parent = context["node_by_path"].call(parent_path)
		if target_parent == null:
			return context["bridge_error"].call(404, request_id, "NODE_PARENT_NOT_FOUND", "Parent node does not exist", {"parent": parent_path})

	var parent_logical: String = context["logical_path"].call(target_parent)
	var new_path: String = "%s/%s" % [parent_logical.rstrip("/"), node_name]
	if dry_run:
		return context["bridge_ok"].call(request_id, {"source_path": path, "path": new_path, "dry_run": true})

	var dup: Node = source.duplicate()
	dup.name = node_name
	target_parent.add_child(dup)
	_set_owner_recursive(dup, context["edited_scene_root"].call())
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"source_path": path,
		"path": context["logical_path"].call(dup),
		"duplicated": true,
	})


func handle_list_properties(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "node.list_properties", "Node property listing requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})

	var props: Array = []
	for p: Dictionary in node.get_property_list():
		if p["usage"] & PROPERTY_USAGE_EDITOR:
			props.append({
				"name": p["name"],
				"type": type_string(p["type"]),
				"usage": p["usage"],
			})
	return context["bridge_ok"].call(request_id, {
		"path": context["logical_path"].call(node),
		"properties": props,
	})


func _set_owner_recursive(node: Node, owner: Node) -> void:
	node.owner = owner
	for child: Node in node.get_children():
		_set_owner_recursive(child, owner)


func _renamed_path(path: String, new_name: String) -> String:
	var index: int = path.rfind("/")
	if index == -1:
		return new_name
	return path.substr(0, index + 1) + new_name


func _script_syntax_error(script_path: String, source: String, request_id: String, context: Dictionary) -> Dictionary:
	var script := GDScript.new()
	script.resource_path = script_path
	script.source_code = source
	var err: Error = script.reload()
	if err == OK:
		return {}
	return context["bridge_error"].call(400, request_id, "SCRIPT_SYNTAX_INVALID", "Script did not pass Godot syntax check", {
		"path": script_path,
		"error": error_string(err),
	})
