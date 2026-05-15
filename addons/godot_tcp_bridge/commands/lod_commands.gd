@tool
extends RefCounted


func handle_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "lod.set", "LOD set requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path: String = String(params.get("path", ""))
	if node_path == "":
		return context["bridge_error"].call(400, request_id, "LOD_PARAMS_MISSING", "path is required", {})
	var node: Node = context["node_by_path"].call(node_path)
	if node == null or not node is GeometryInstance3D:
		return context["bridge_error"].call(404, request_id, "LOD_NODE_NOT_FOUND", "GeometryInstance3D node not found at path", {"path": node_path})
	var geom: GeometryInstance3D = node as GeometryInstance3D
	if params.has("begin"):
		geom.visibility_range_begin = float(params["begin"])
	if params.has("end"):
		geom.visibility_range_end = float(params["end"])
	if params.has("use_for_shadows"):
		geom.visibility_range_fade_mode = GeometryInstance3D.VISIBILITY_RANGE_FADE_DISABLED
	return context["bridge_ok"].call(request_id, {"path": node_path, "begin": geom.visibility_range_begin, "end": geom.visibility_range_end})


func handle_set_many(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "lod.set-many", "LOD set-many requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var entries: Array = Array(params.get("entries", []))
	if entries.is_empty():
		return context["bridge_error"].call(400, request_id, "LOD_PARAMS_MISSING", "entries array is required and must not be empty", {})
	var updated: int = 0
	var errors: Array = []
	for entry in entries:
		if not entry is Dictionary:
			continue
		var node_path: String = String(entry.get("path", ""))
		if node_path == "":
			continue
		var node: Node = context["node_by_path"].call(node_path)
		if node == null or not node is GeometryInstance3D:
			errors.append({"path": node_path, "error": "not found or not GeometryInstance3D"})
			continue
		var geom: GeometryInstance3D = node as GeometryInstance3D
		if entry.has("begin"):
			geom.visibility_range_begin = float(entry["begin"])
		if entry.has("end"):
			geom.visibility_range_end = float(entry["end"])
		updated += 1
	return context["bridge_ok"].call(request_id, {"updated": updated, "errors": errors})


