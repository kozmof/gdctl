@tool
extends RefCounted


func handle_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "occluder.add", "Occluder add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var parent_path: String = String(params.get("parent", ""))
	if parent_path == "":
		return context["bridge_error"].call(400, request_id, "OCCLUDER_PARAMS_MISSING", "parent is required", {})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var parent: Node = context["node_by_path"].call(parent_path)
	if parent == null:
		return context["bridge_error"].call(404, request_id, "OCCLUDER_PARENT_NOT_FOUND", "Parent node not found", {"path": parent_path})
	var oi := OccluderInstance3D.new()
	oi.name = "OccluderInstance3D"
	var shape_name: String = String(params.get("shape", "box")).to_lower()
	var size_raw = params.get("size", null)
	var size := Vector3(1.0, 1.0, 1.0)
	if size_raw is Array and size_raw.size() >= 3:
		size = Vector3(float(size_raw[0]), float(size_raw[1]), float(size_raw[2]))
	elif size_raw is Dictionary:
		size = Vector3(float(size_raw.get("x", 1.0)), float(size_raw.get("y", 1.0)), float(size_raw.get("z", 1.0)))
	match shape_name:
		"sphere":
			var sphere := SphereOccluder3D.new()
			sphere.radius = size.x * 0.5
			oi.occluder = sphere
		"quad":
			var quad := QuadOccluder3D.new()
			quad.size = Vector2(size.x, size.y)
			oi.occluder = quad
		_:
			var box := BoxOccluder3D.new()
			box.size = size
			oi.occluder = box
	parent.add_child(oi)
	oi.owner = scene_root
	context["mark_scene_dirty"].call()
	var oi_path: String = context["logical_path"].call(oi)
	return context["bridge_ok"].call(request_id, {"path": oi_path, "parent": parent_path, "shape": shape_name, "added": true})
