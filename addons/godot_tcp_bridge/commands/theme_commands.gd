@tool
extends RefCounted


func handle_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "theme.create", "Theme create requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var theme_path: String = String(params.get("path", ""))
	var force: bool = bool(params.get("force", false))
	if not _valid_theme_path(theme_path):
		return context["bridge_error"].call(400, request_id, "THEME_PATH_INVALID", "Theme path must be a res:// .tres path", {"path": theme_path})
	if ResourceLoader.exists(theme_path) and not force:
		return context["bridge_error"].call(409, request_id, "THEME_ALREADY_EXISTS", "Theme already exists", {"path": theme_path})
	var dir_err: Error = _ensure_dir(theme_path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "THEME_DIR_FAILED", "Could not create theme directory", {"path": theme_path, "error": error_string(dir_err)})
	var theme := Theme.new()
	var err: Error = ResourceSaver.save(theme, theme_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "THEME_SAVE_FAILED", "Could not save theme", {"path": theme_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": theme_path, "created": true})


func handle_set_color(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "theme.set-color", "Theme set-color requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var theme_path: String = String(params.get("path", ""))
	var node_type: String = String(params.get("node_type", ""))
	var name: String = String(params.get("name", ""))
	var value_raw: Variant = params.get("value", null)
	if not _valid_theme_path(theme_path):
		return context["bridge_error"].call(400, request_id, "THEME_PATH_INVALID", "Theme path must be a res:// .tres path", {"path": theme_path})
	if node_type == "":
		return context["bridge_error"].call(400, request_id, "THEME_NODE_TYPE_MISSING", "node_type is required", {})
	if name == "":
		return context["bridge_error"].call(400, request_id, "THEME_NAME_MISSING", "name is required", {})
	var color_result := _parse_color(value_raw, request_id, context)
	if color_result.has("error_response"):
		return color_result["error_response"]
	var color: Color = color_result["color"]
	var theme: Theme = _load_theme(theme_path, request_id, context)
	if theme == null:
		return context["bridge_error"].call(404, request_id, "THEME_NOT_FOUND", "Theme does not exist", {"path": theme_path})
	theme.set_color(name, node_type, color)
	var err: Error = ResourceSaver.save(theme, theme_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "THEME_SAVE_FAILED", "Could not save theme", {"path": theme_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": theme_path, "node_type": node_type, "data_type": "color", "name": name, "set": true})


func handle_set_font_size(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "theme.set-font-size", "Theme set-font-size requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var theme_path: String = String(params.get("path", ""))
	var node_type: String = String(params.get("node_type", ""))
	var name: String = String(params.get("name", ""))
	var size: int = int(params.get("value", 0))
	if not _valid_theme_path(theme_path):
		return context["bridge_error"].call(400, request_id, "THEME_PATH_INVALID", "Theme path must be a res:// .tres path", {"path": theme_path})
	if node_type == "" or name == "":
		return context["bridge_error"].call(400, request_id, "THEME_PARAMS_MISSING", "node_type and name are required", {})
	if size <= 0:
		return context["bridge_error"].call(400, request_id, "THEME_FONT_SIZE_INVALID", "font size must be a positive integer", {"value": size})
	var theme: Theme = _load_theme(theme_path, request_id, context)
	if theme == null:
		return context["bridge_error"].call(404, request_id, "THEME_NOT_FOUND", "Theme does not exist", {"path": theme_path})
	theme.set_font_size(name, node_type, size)
	var err: Error = ResourceSaver.save(theme, theme_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "THEME_SAVE_FAILED", "Could not save theme", {"path": theme_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": theme_path, "node_type": node_type, "data_type": "font_size", "name": name, "set": true})


func handle_set_constant(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "theme.set-constant", "Theme set-constant requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var theme_path: String = String(params.get("path", ""))
	var node_type: String = String(params.get("node_type", ""))
	var name: String = String(params.get("name", ""))
	var value: int = int(params.get("value", 0))
	if not _valid_theme_path(theme_path):
		return context["bridge_error"].call(400, request_id, "THEME_PATH_INVALID", "Theme path must be a res:// .tres path", {"path": theme_path})
	if node_type == "" or name == "":
		return context["bridge_error"].call(400, request_id, "THEME_PARAMS_MISSING", "node_type and name are required", {})
	var theme: Theme = _load_theme(theme_path, request_id, context)
	if theme == null:
		return context["bridge_error"].call(404, request_id, "THEME_NOT_FOUND", "Theme does not exist", {"path": theme_path})
	theme.set_constant(name, node_type, value)
	var err: Error = ResourceSaver.save(theme, theme_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "THEME_SAVE_FAILED", "Could not save theme", {"path": theme_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": theme_path, "node_type": node_type, "data_type": "constant", "name": name, "set": true})


func _parse_color(value: Variant, request_id: String, context: Dictionary) -> Dictionary:
	if typeof(value) == TYPE_ARRAY:
		var arr: Array = value
		if arr.size() == 3:
			return {"color": Color(float(arr[0]), float(arr[1]), float(arr[2]), 1.0)}
		if arr.size() == 4:
			return {"color": Color(float(arr[0]), float(arr[1]), float(arr[2]), float(arr[3]))}
	if typeof(value) == TYPE_STRING:
		var s: String = value
		if s.is_valid_html_color():
			return {"color": Color(s)}
	return {"error_response": context["bridge_error"].call(400, request_id, "THEME_COLOR_INVALID", "Color must be [r,g,b,a] array or HTML color string", {"value": str(value)})}


func _load_theme(theme_path: String, _request_id: String, _context: Dictionary) -> Theme:
	if not ResourceLoader.exists(theme_path):
		return null
	var res: Resource = ResourceLoader.load(theme_path, "", ResourceLoader.CACHE_MODE_IGNORE)
	if res is Theme:
		return res as Theme
	return null


func _valid_theme_path(path: String) -> bool:
	return path != "" and path.begins_with("res://") and path.ends_with(".tres")


func _ensure_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
