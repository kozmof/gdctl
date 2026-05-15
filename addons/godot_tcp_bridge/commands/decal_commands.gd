@tool
extends RefCounted


func handle_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "decal.add", "Decal add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var parent_path: String = String(params.get("parent", ""))
	var texture_path: String = String(params.get("texture", ""))
	if parent_path == "":
		return context["bridge_error"].call(400, request_id, "DECAL_PARAMS_MISSING", "parent is required", {})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var parent: Node = context["node_by_path"].call(parent_path)
	if parent == null:
		return context["bridge_error"].call(404, request_id, "DECAL_PARENT_NOT_FOUND", "Parent node not found", {"path": parent_path})
	var decal := Decal.new()
	decal.name = "Decal"
	if texture_path != "" and FileAccess.file_exists(texture_path):
		var tex: Texture2D = load(texture_path)
		if tex != null:
			decal.texture_albedo = tex
	var size_raw = params.get("size", null)
	if size_raw is Array and size_raw.size() >= 3:
		decal.size = Vector3(float(size_raw[0]), float(size_raw[1]), float(size_raw[2]))
	elif size_raw is Dictionary:
		decal.size = Vector3(float(size_raw.get("x", 1.0)), float(size_raw.get("y", 1.0)), float(size_raw.get("z", 1.0)))
	parent.add_child(decal)
	decal.owner = scene_root
	context["mark_scene_dirty"].call()
	var decal_path: String = context["logical_path"].call(decal)
	return context["bridge_ok"].call(request_id, {"path": decal_path, "parent": parent_path, "texture": texture_path, "added": true})


func handle_set_normal_fade(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "decal.set-normal-fade", "Decal set-normal-fade requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path: String = String(params.get("path", ""))
	var fade: float = float(params.get("fade", 0.0))
	if node_path == "":
		return context["bridge_error"].call(400, request_id, "DECAL_PARAMS_MISSING", "path is required", {})
	var node: Node = context["node_by_path"].call(node_path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "DECAL_NOT_FOUND", "Decal node not found", {"path": node_path})
	if not node is Decal:
		return context["bridge_error"].call(400, request_id, "DECAL_NODE_INVALID", "Node is not a Decal", {"path": node_path})
	(node as Decal).normal_fade = fade
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"path": node_path, "normal_fade": fade, "applied": true})
