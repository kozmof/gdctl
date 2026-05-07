@tool
extends RefCounted


func handle_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Script creation requires bearer token", {})
	if body.get("op", "") != "script.create":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected script.create operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var script_path: String = String(params.get("path", ""))
	var extends_type: String = String(params.get("extends", ""))
	var force: bool = bool(params.get("force", false))
	var path_error: Dictionary = _validate_script_path(script_path, false, context, body.get("request_id", ""))
	if not path_error.is_empty():
		return path_error
	if FileAccess.file_exists(script_path) and not force:
		return context["bridge_error"].call(409, body.get("request_id", ""), "SCRIPT_ALREADY_EXISTS", "Script already exists", {"path": script_path})
	if extends_type == "" or not extends_type.is_valid_identifier():
		return context["bridge_error"].call(400, body.get("request_id", ""), "SCRIPT_EXTENDS_INVALID", "Extends must be a valid class name", {"extends": extends_type})

	var source: String = "extends %s\n" % extends_type
	return _write_and_check(script_path, source, body.get("request_id", ""), context, {"created": true})


func handle_write(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Script write requires bearer token", {})
	if body.get("op", "") != "script.write":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected script.write operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var script_path: String = String(params.get("path", ""))
	var path_error: Dictionary = _validate_script_path(script_path, false, context, body.get("request_id", ""))
	if not path_error.is_empty():
		return path_error
	if not params.has("body"):
		return context["bridge_error"].call(400, body.get("request_id", ""), "SCRIPT_BODY_MISSING", "Script body is required", {})
	var source: String = String(params.get("body", ""))
	return _write_and_check(script_path, source, body.get("request_id", ""), context, {"written": true})


func handle_check(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Script check requires bearer token", {})
	if body.get("op", "") != "script.check":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected script.check operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var script_path: String = String(params.get("path", ""))
	var path_error: Dictionary = _validate_script_path(script_path, true, context, body.get("request_id", ""))
	if not path_error.is_empty():
		return path_error

	var source: String = FileAccess.get_file_as_string(script_path)
	var check_error: Dictionary = _syntax_error(script_path, source, body.get("request_id", ""), context)
	if not check_error.is_empty():
		return check_error
	return context["bridge_ok"].call(body.get("request_id", ""), {
		"path": script_path,
		"valid": true,
	})


func _write_and_check(script_path: String, source: String, request_id: String, context: Dictionary, extra: Dictionary) -> Dictionary:
	var dir_err: Error = _ensure_resource_dir(script_path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "SCRIPT_DIR_FAILED", "Could not create script directory", {"path": script_path, "error": error_string(dir_err)})
	var check_error: Dictionary = _syntax_error(script_path, source, request_id, context)
	if not check_error.is_empty():
		return check_error
	var file := FileAccess.open(script_path, FileAccess.WRITE)
	if file == null:
		return context["bridge_error"].call(500, request_id, "SCRIPT_WRITE_FAILED", "Could not open script for writing", {"path": script_path})
	file.store_string(source)
	file.close()
	var result: Dictionary = {
		"path": script_path,
		"valid": true,
	}
	for key in extra.keys():
		result[key] = extra[key]
	return context["bridge_ok"].call(request_id, result)


func _syntax_error(script_path: String, source: String, request_id: String, context: Dictionary) -> Dictionary:
	var script := GDScript.new()
	script.resource_path = script_path
	script.source_code = source
	var err: Error = script.reload()
	if err == OK:
		return {}
	return context["bridge_error"].call(400, request_id, "SCRIPT_SYNTAX_INVALID", "Script did not pass Godot syntax check", {
		"path": script_path,
		"error": error_string(err),
	})


func _validate_script_path(script_path: String, must_exist: bool, context: Dictionary, request_id: String) -> Dictionary:
	if script_path == "" or not script_path.begins_with("res://") or not script_path.ends_with(".gd"):
		return context["bridge_error"].call(400, request_id, "SCRIPT_PATH_INVALID", "Script path must be a res:// .gd path", {"path": script_path})
	if must_exist and not FileAccess.file_exists(script_path):
		return context["bridge_error"].call(404, request_id, "SCRIPT_NOT_FOUND", "Script does not exist", {"path": script_path})
	return {}


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
