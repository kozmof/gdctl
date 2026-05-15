@tool
extends RefCounted


func handle_heightmap_import(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "terrain.heightmap-import", "Terrain heightmap-import requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path: String = String(params.get("path", ""))
	var texture_path: String = String(params.get("texture", ""))
	var min_height: float = float(params.get("min_height", -10.0))
	var max_height: float = float(params.get("max_height", 50.0))
	if node_path == "" or texture_path == "":
		return context["bridge_error"].call(400, request_id, "TERRAIN_PARAMS_MISSING", "path and texture are required", {})
	if not ResourceLoader.exists(texture_path):
		return context["bridge_error"].call(404, request_id, "TERRAIN_TEXTURE_NOT_FOUND", "Texture resource does not exist", {"texture": texture_path})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var node: Node = context["node_by_path"].call(node_path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "TERRAIN_NODE_NOT_FOUND", "Node not found", {"path": node_path})
	var height_map = null
	if node is CollisionShape3D:
		var cs: CollisionShape3D = node as CollisionShape3D
		if cs.shape is HeightMapShape3D:
			height_map = cs.shape as HeightMapShape3D
	if height_map == null:
		return context["bridge_error"].call(400, request_id, "TERRAIN_NODE_INVALID", "Node must be a CollisionShape3D whose shape is a HeightMapShape3D", {"path": node_path})
	var image: Image = null
	var tex_res = ResourceLoader.load(texture_path)
	if tex_res is Texture2D:
		image = (tex_res as Texture2D).get_image()
	elif tex_res is Image:
		image = tex_res as Image
	if image == null:
		return context["bridge_error"].call(500, request_id, "TERRAIN_IMAGE_LOAD_FAILED", "Could not load heightmap — resource must be a Texture2D or Image imported in the project", {"texture": texture_path})
	height_map.update_map_data_from_image(image, min_height, max_height)
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"path": node_path,
		"texture": texture_path,
		"width": image.get_width(),
		"height": image.get_height(),
		"min_height": min_height,
		"max_height": max_height,
		"applied": true,
	})
