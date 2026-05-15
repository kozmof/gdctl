@tool
extends RefCounted


func handle_set_size(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "viewport.set-size", "Viewport set-size requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var width: int = int(params.get("width", 0))
	var height: int = int(params.get("height", 0))
	var viewport_path: String = String(params.get("path", ""))
	if width <= 0 or height <= 0:
		return context["bridge_error"].call(400, request_id, "VIEWPORT_SIZE_INVALID", "Width and height must be positive integers", {"width": width, "height": height})
	if viewport_path == "":
		# Resize the main window
		DisplayServer.window_set_size(Vector2i(width, height))
		return context["bridge_ok"].call(request_id, {"width": width, "height": height, "path": ""})
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})
	var node: Node = context["node_by_path"].call(viewport_path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node not found", {"path": viewport_path})
	if not node is SubViewport:
		return context["bridge_error"].call(400, request_id, "VIEWPORT_NODE_INVALID", "Node is not a SubViewport", {"path": viewport_path})
	var vp: SubViewport = node as SubViewport
	vp.size = Vector2i(width, height)
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"width": width, "height": height, "path": viewport_path})


func handle_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "viewport.add", "Viewport add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var parent_path: String = String(params.get("parent", ""))
	var width: int = int(params.get("width", 320))
	var height: int = int(params.get("height", 240))
	var add_camera: bool = bool(params.get("add_camera", false))
	if width <= 0 or height <= 0:
		return context["bridge_error"].call(400, request_id, "VIEWPORT_SIZE_INVALID", "Width and height must be positive integers", {"width": width, "height": height})
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})
	var parent: Node = root
	if parent_path != "":
		parent = context["node_by_path"].call(parent_path)
		if parent == null:
			return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Parent node not found", {"path": parent_path})
	var vp := SubViewport.new()
	vp.size = Vector2i(width, height)
	vp.name = "SubViewport"
	parent.add_child(vp)
	vp.owner = root
	if add_camera:
		var cam := Camera3D.new()
		cam.name = "Camera3D"
		vp.add_child(cam)
		cam.owner = root
	context["mark_scene_dirty"].call()
	var vp_path: String = context["logical_path"].call(vp)
	return context["bridge_ok"].call(request_id, {"path": vp_path, "width": width, "height": height, "added": true})


func handle_camera_assign(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "viewport.camera-assign", "Viewport camera-assign requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var viewport_path: String = String(params.get("viewport", ""))
	var camera_path: String = String(params.get("camera", ""))
	if viewport_path == "" or camera_path == "":
		return context["bridge_error"].call(400, request_id, "VIEWPORT_CAMERA_PARAMS_MISSING", "viewport and camera paths are required", {})
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})
	var vp_node: Node = context["node_by_path"].call(viewport_path)
	if vp_node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Viewport node not found", {"path": viewport_path})
	if not vp_node is SubViewport:
		return context["bridge_error"].call(400, request_id, "VIEWPORT_NODE_INVALID", "Node is not a SubViewport", {"path": viewport_path})
	var cam_node: Node = context["node_by_path"].call(camera_path)
	if cam_node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Camera node not found", {"path": camera_path})
	if not cam_node is Camera3D and not cam_node is Camera2D:
		return context["bridge_error"].call(400, request_id, "CAMERA_NODE_INVALID", "Node is not a Camera3D or Camera2D", {"path": camera_path})
	if cam_node is Camera3D:
		var cam3d: Camera3D = cam_node as Camera3D
		cam3d.current = true
	elif cam_node is Camera2D:
		var cam2d: Camera2D = cam_node as Camera2D
		cam2d.make_current()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"viewport": viewport_path,
		"camera": camera_path,
		"applied": true,
	})


func handle_window_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "window.create", "Window create requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var title: String = String(params.get("title", "Window"))
	var width: int = int(params.get("width", 640))
	var height: int = int(params.get("height", 480))
	var pos_x: int = int(params.get("position_x", 0))
	var pos_y: int = int(params.get("position_y", 0))
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var win := Window.new()
	win.title = title
	win.size = Vector2i(width, height)
	win.position = Vector2i(pos_x, pos_y)
	win.name = title.replace(" ", "_")
	scene_root.add_child(win)
	win.owner = scene_root
	win.show()
	context["mark_scene_dirty"].call()
	var win_path: String = context["logical_path"].call(win)
	var win_id: int = win.get_window_id()
	return context["bridge_ok"].call(request_id, {"path": win_path, "window_id": win_id, "title": title, "width": width, "height": height, "created": true})


func handle_window_assign_viewport(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "window.assign-viewport", "Window assign-viewport requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var window_id: int = int(params.get("window_id", -1))
	var viewport_path: String = String(params.get("viewport_path", ""))
	if window_id < 0 or viewport_path == "":
		return context["bridge_error"].call(400, request_id, "WINDOW_ASSIGN_PARAMS_MISSING", "window_id and viewport_path are required", {})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	# Find the Window node by its window_id
	var win_node: Window = null
	for child in scene_root.get_children():
		if child is Window and (child as Window).get_window_id() == window_id:
			win_node = child as Window
			break
	if win_node == null:
		return context["bridge_error"].call(404, request_id, "WINDOW_NOT_FOUND", "Window with given ID not found in scene root children", {"window_id": window_id})
	var vp_node: Node = context["node_by_path"].call(viewport_path)
	if vp_node == null:
		return context["bridge_error"].call(404, request_id, "VIEWPORT_NOT_FOUND", "SubViewport node not found", {"path": viewport_path})
	if not vp_node is SubViewport:
		return context["bridge_error"].call(400, request_id, "VIEWPORT_NODE_INVALID", "Node is not a SubViewport", {"path": viewport_path})
	# Reparent the SubViewport into the Window
	var vp: SubViewport = vp_node as SubViewport
	var old_parent: Node = vp.get_parent()
	if old_parent != null:
		old_parent.remove_child(vp)
	win_node.add_child(vp)
	vp.owner = scene_root
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"window_id": window_id, "viewport_path": viewport_path, "assigned": true})


func handle_screenshot(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "viewport.screenshot", "Viewport screenshot requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var kind: String = String(params.get("kind", "2d"))
	var index: int = int(params.get("index", 0))
	if kind != "2d" and kind != "3d":
		return context["bridge_error"].call(400, request_id, "VIEWPORT_KIND_INVALID", "Viewport kind must be 2d or 3d", {"kind": kind})
	if index < 0 or index > 3:
		return context["bridge_error"].call(400, request_id, "VIEWPORT_INDEX_INVALID", "Viewport index must be between 0 and 3", {"index": index})
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(500, request_id, "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})

	var job_id: String = String(context["queue_job"].call("viewport.screenshot", {
		"kind": kind,
		"index": index,
		"frames_remaining": 2,
		"request_id": request_id,
	}))
	return context["bridge_ok"].call(request_id, {
		"queued": true,
		"job_id": job_id,
		"kind": kind,
		"index": index,
	})
