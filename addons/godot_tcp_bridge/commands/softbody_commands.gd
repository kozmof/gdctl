@tool
extends RefCounted


func handle_pin_point(request: Dictionary, context: Dictionary) -> Dictionary:
	return _set_pin(request, context, true)


func handle_unpin_point(request: Dictionary, context: Dictionary) -> Dictionary:
	return _set_pin(request, context, false)


func _set_pin(request: Dictionary, context: Dictionary, pinned: bool) -> Dictionary:
	var op: String = "pin-point" if pinned else "unpin-point"
	var checked: Dictionary = context["request"].require_body(request, context, "softbody." + op, "SoftBody " + op + " requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path: String = String(params.get("path", ""))
	var point_idx: int = int(params.get("point", -1))
	if node_path == "" or point_idx < 0:
		return context["bridge_error"].call(400, request_id, "SOFTBODY_PARAMS_MISSING", "path and point (>= 0) are required", {})
	var node: Node = context["node_by_path"].call(node_path)
	if node == null or not node is SoftBody3D:
		return context["bridge_error"].call(404, request_id, "SOFTBODY_NOT_FOUND", "SoftBody3D node not found at path", {"path": node_path})
	var sb: SoftBody3D = node as SoftBody3D
	sb.pin_point(point_idx, pinned)
	return context["bridge_ok"].call(request_id, {"path": node_path, "point": point_idx, "pinned": pinned})


