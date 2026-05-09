@tool
extends RefCounted


func handle_bake(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "navigation.bake", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var node: Node = context["node_by_path"].call(path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node does not exist", {"path": path})

	if node is NavigationRegion3D:
		node.bake_navigation_mesh(false)
		context["mark_scene_dirty"].call()
		return context["bridge_ok"].call(request_id, {
			"path": context["logical_path"].call(node),
			"kind": "NavigationRegion3D",
			"baked": true,
		})
	elif node is NavigationRegion2D:
		node.bake_navigation_polygon()
		context["mark_scene_dirty"].call()
		return context["bridge_ok"].call(request_id, {
			"path": context["logical_path"].call(node),
			"kind": "NavigationRegion2D",
			"baked": true,
		})
	return context["bridge_error"].call(400, request_id, "NODE_TYPE_INVALID", "Node must be NavigationRegion3D or NavigationRegion2D", {"path": path, "type": node.get_class()})
