@tool
extends RefCounted


func handle_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Mutation endpoint requires bearer token", {})
	if body.get("op", "") != "node.add":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected node.add operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var parent_path: String = String(params.get("parent", ""))
	var type_name: String = String(params.get("type", ""))
	var node_name: String = String(params.get("name", ""))
	var dry_run: bool = bool(params.get("dry_run", false))
	var parent: Node = context["node_by_path"].call(parent_path)
	if parent == null:
		return context["bridge_error"].call(404, body.get("request_id", ""), "NODE_PARENT_NOT_FOUND", "Parent node does not exist", {"parent": parent_path})
	if type_name == "" or not ClassDB.can_instantiate(type_name):
		return context["bridge_error"].call(400, body.get("request_id", ""), "NODE_TYPE_INVALID", "Node type cannot be instantiated", {"type": type_name})
	if not ClassDB.is_parent_class(type_name, "Node") and type_name != "Node":
		return context["bridge_error"].call(400, body.get("request_id", ""), "NODE_TYPE_INVALID", "Node type must inherit Node", {"type": type_name})
	if node_name == "" or not node_name.is_valid_identifier():
		return context["bridge_error"].call(400, body.get("request_id", ""), "NODE_NAME_INVALID", "Node name must be a valid identifier", {"name": node_name})

	var path: String = "%s/%s" % [parent_path.rstrip("/"), node_name]
	if dry_run:
		return context["bridge_ok"].call(body.get("request_id", ""), {"path": path, "dry_run": true})

	var node: Node = ClassDB.instantiate(type_name) as Node
	node.name = node_name
	parent.add_child(node)
	node.owner = context["edited_scene_root"].call()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(body.get("request_id", ""), {"path": context["logical_path"].call(node)})


func handle_remove(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Mutation endpoint requires bearer token", {})
	if body.get("op", "") != "node.remove":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected node.remove operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var path: String = String(params.get("path", ""))
	var dry_run: bool = bool(params.get("dry_run", false))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, body.get("request_id", ""), "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if node == context["edited_scene_root"].call():
		return context["bridge_error"].call(400, body.get("request_id", ""), "CANNOT_REMOVE_SCENE_ROOT", "Scene root cannot be removed", {"path": path})
	if dry_run:
		return context["bridge_ok"].call(body.get("request_id", ""), {"removed": path, "dry_run": true})

	node.get_parent().remove_child(node)
	node.queue_free()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(body.get("request_id", ""), {"removed": path})


func handle_get(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Node property read requires bearer token", {})
	if body.get("op", "") != "node.get":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected node.get operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var path: String = String(params.get("path", ""))
	var property: String = String(params.get("property", ""))
	if property == "":
		return context["bridge_error"].call(400, body.get("request_id", ""), "PROPERTY_INVALID", "Property name is required", {})
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, body.get("request_id", ""), "NODE_NOT_FOUND", "Node does not exist", {"path": path})

	var typed_values: RefCounted = context["typed_values"]
	var encoded: Dictionary = typed_values.encode(node.get(property))
	return context["bridge_ok"].call(body.get("request_id", ""), {
		"path": context["logical_path"].call(node),
		"property": property,
		"value": encoded,
	})


func handle_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Mutation endpoint requires bearer token", {})
	if body.get("op", "") != "node.set":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected node.set operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var path: String = String(params.get("path", ""))
	var property: String = String(params.get("property", ""))
	if property == "":
		return context["bridge_error"].call(400, body.get("request_id", ""), "PROPERTY_INVALID", "Property name is required", {})
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, body.get("request_id", ""), "NODE_NOT_FOUND", "Node does not exist", {"path": path})
	if not params.has("value"):
		return context["bridge_error"].call(400, body.get("request_id", ""), "VALUE_MISSING", "Typed value is required", {})

	var typed_values: RefCounted = context["typed_values"]
	var decoded: Dictionary = typed_values.decode(params.get("value"))
	if not bool(decoded.get("ok", false)):
		return context["bridge_error"].call(400, body.get("request_id", ""), "VALUE_INVALID", String(decoded.get("error", "Invalid typed value")), {})
	node.set(property, decoded.get("value"))
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(body.get("request_id", ""), {
		"path": context["logical_path"].call(node),
		"property": property,
		"value": typed_values.encode(node.get(property)),
	})
