@tool
extends RefCounted


func handle_lightmap_bake(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "lightmap.bake", "LightmapGI bake requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path: String = String(params.get("path", ""))
	if node_path == "":
		return context["bridge_error"].call(400, request_id, "LIGHTMAP_PARAMS_MISSING", "path is required", {})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var node: Node = context["node_by_path"].call(node_path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "LIGHTMAP_NODE_NOT_FOUND", "Node not found", {"path": node_path})
	if not node is LightmapGI:
		return context["bridge_error"].call(400, request_id, "LIGHTMAP_NODE_INVALID", "Node is not a LightmapGI", {"path": node_path})
	var lm: LightmapGI = node as LightmapGI
	lm.bake(scene_root, "")
	return context["bridge_ok"].call(request_id, {"path": node_path, "status": "started", "note": "LightmapGI bake is async; check editor output for completion"})


func handle_voxelgi_bake(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "voxelgi.bake", "VoxelGI bake requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path: String = String(params.get("path", ""))
	if node_path == "":
		return context["bridge_error"].call(400, request_id, "VOXELGI_PARAMS_MISSING", "path is required", {})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var node: Node = context["node_by_path"].call(node_path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "VOXELGI_NODE_NOT_FOUND", "Node not found", {"path": node_path})
	if not node is VoxelGI:
		return context["bridge_error"].call(400, request_id, "VOXELGI_NODE_INVALID", "Node is not a VoxelGI", {"path": node_path})
	var vgi: VoxelGI = node as VoxelGI
	vgi.bake(scene_root, true)
	return context["bridge_ok"].call(request_id, {"path": node_path, "status": "started", "note": "VoxelGI bake is async; check editor output for completion"})


func handle_reflection_probe_bake(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "reflection-probe.bake", "ReflectionProbe bake requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var request_id: String = String(checked["request_id"])
	return context["bridge_ok"].call(request_id, {
		"status": "not_supported",
		"note": "ReflectionProbe bake has no GDScript API; use the editor Bake button or set update_mode=UPDATE_ALWAYS for runtime baking",
	})
