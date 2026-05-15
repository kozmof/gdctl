@tool
extends RefCounted


func handle_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "fog-volume.add", "FogVolume add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var parent_path: String = String(params.get("parent", ""))
	if parent_path == "":
		return context["bridge_error"].call(400, request_id, "FOG_PARAMS_MISSING", "parent is required", {})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var parent: Node = context["node_by_path"].call(parent_path)
	if parent == null:
		return context["bridge_error"].call(404, request_id, "FOG_PARENT_NOT_FOUND", "Parent node not found", {"path": parent_path})
	var fv := FogVolume.new()
	fv.name = "FogVolume"
	var shape_name: String = String(params.get("shape", "box")).to_lower()
	match shape_name:
		"ellipsoid":
			fv.shape = RenderingServer.FOG_VOLUME_SHAPE_ELLIPSOID
		"cone":
			fv.shape = RenderingServer.FOG_VOLUME_SHAPE_CONE
		"cylinder":
			fv.shape = RenderingServer.FOG_VOLUME_SHAPE_CYLINDER
		"world":
			fv.shape = RenderingServer.FOG_VOLUME_SHAPE_WORLD
		_:
			fv.shape = RenderingServer.FOG_VOLUME_SHAPE_BOX
	var size_raw = params.get("size", null)
	if size_raw is Array and size_raw.size() >= 3:
		fv.size = Vector3(float(size_raw[0]), float(size_raw[1]), float(size_raw[2]))
	elif size_raw is Dictionary:
		fv.size = Vector3(float(size_raw.get("x", 2.0)), float(size_raw.get("y", 2.0)), float(size_raw.get("z", 2.0)))
	var mat := FogMaterial.new()
	var density: float = float(params.get("density", 1.0))
	mat.density = density
	fv.material = mat
	parent.add_child(fv)
	fv.owner = scene_root
	context["mark_scene_dirty"].call()
	var fv_path: String = context["logical_path"].call(fv)
	return context["bridge_ok"].call(request_id, {"path": fv_path, "parent": parent_path, "shape": shape_name, "density": density, "added": true})
