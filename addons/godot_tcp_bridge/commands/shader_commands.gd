@tool
extends RefCounted


func handle_write(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "shader.write", "Shader write requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var shader_path: String = String(params.get("path", ""))
	var path_error: Dictionary = _validate_shader_path(shader_path, false, context, request_id)
	if not path_error.is_empty():
		return path_error
	if not params.has("body"):
		return context["bridge_error"].call(400, request_id, "SHADER_BODY_MISSING", "Shader body is required", {})
	var source: String = String(params.get("body", ""))
	return _write_and_check(shader_path, source, request_id, context, {"written": true})


func handle_check(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "shader.check", "Shader check requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var shader_path: String = String(params.get("path", ""))
	var path_error: Dictionary = _validate_shader_path(shader_path, true, context, request_id)
	if not path_error.is_empty():
		return path_error

	var source: String = FileAccess.get_file_as_string(shader_path)
	var check_error: Dictionary = _syntax_error(shader_path, source, request_id, context)
	if not check_error.is_empty():
		return check_error
	return context["bridge_ok"].call(request_id, {
		"path": shader_path,
		"valid": true,
	})


func _write_and_check(shader_path: String, source: String, request_id: String, context: Dictionary, extra: Dictionary) -> Dictionary:
	var dir_err: Error = _ensure_resource_dir(shader_path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "SHADER_DIR_FAILED", "Could not create shader directory", {"path": shader_path, "error": error_string(dir_err)})
	var check_error: Dictionary = _syntax_error(shader_path, source, request_id, context)
	if not check_error.is_empty():
		return check_error
	var file := FileAccess.open(shader_path, FileAccess.WRITE)
	if file == null:
		return context["bridge_error"].call(500, request_id, "SHADER_WRITE_FAILED", "Could not open shader for writing", {"path": shader_path})
	file.store_string(source)
	file.close()
	ResourceLoader.load(shader_path, "Shader", ResourceLoader.CACHE_MODE_REPLACE)
	var result: Dictionary = {
		"path": shader_path,
		"valid": true,
	}
	for key in extra.keys():
		result[key] = extra[key]
	return context["bridge_ok"].call(request_id, result)


func _syntax_error(shader_path: String, source: String, request_id: String, context: Dictionary) -> Dictionary:
	if source.strip_edges() == "":
		return context["bridge_error"].call(400, request_id, "SHADER_EMPTY", "Shader body is empty", {"path": shader_path})
	if source.find("shader_type ") == -1:
		return context["bridge_error"].call(400, request_id, "SHADER_TYPE_MISSING", "Shader body must declare shader_type", {"path": shader_path})
	var shader := Shader.new()
	shader.resource_path = shader_path
	shader.code = source
	if shader.code.strip_edges() == "":
		return context["bridge_error"].call(400, request_id, "SHADER_SYNTAX_INVALID", "Shader did not keep source code after parsing", {"path": shader_path})
	return {}


func _validate_shader_path(shader_path: String, must_exist: bool, context: Dictionary, request_id: String) -> Dictionary:
	if shader_path == "" or not shader_path.begins_with("res://") or not shader_path.ends_with(".gdshader"):
		return context["bridge_error"].call(400, request_id, "SHADER_PATH_INVALID", "Shader path must be a res:// .gdshader path", {"path": shader_path})
	if must_exist and not FileAccess.file_exists(shader_path):
		return context["bridge_error"].call(404, request_id, "SHADER_NOT_FOUND", "Shader does not exist", {"path": shader_path})
	return {}


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
