@tool
extends RefCounted


class SyntaxLogCapture extends Logger:
	var entries: Array[Dictionary] = []

	func _log_error(function: String, file: String, line: int, code: String, rationale: String, editor_notify: bool, error_type: int, script_backtraces: Array[ScriptBacktrace]) -> void:
		entries.append({
			"function": function,
			"file": file,
			"line": line,
			"code": code,
			"rationale": rationale,
			"editor_notify": editor_notify,
			"error_type": error_type,
		})

	func _log_message(message: String, error: bool) -> void:
		if error:
			entries.append({
				"message": message,
				"error": error,
			})


func handle_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "script.create", "Script creation requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var script_path: String = String(params.get("path", ""))
	var extends_type: String = String(params.get("extends", ""))
	var force: bool = bool(params.get("force", false))
	var path_error: Dictionary = _validate_script_path(script_path, false, context, request_id)
	if not path_error.is_empty():
		return path_error
	if FileAccess.file_exists(script_path) and not force:
		return context["bridge_error"].call(409, request_id, "SCRIPT_ALREADY_EXISTS", "Script already exists", {"path": script_path})
	if extends_type == "" or not extends_type.is_valid_identifier():
		return context["bridge_error"].call(400, request_id, "SCRIPT_EXTENDS_INVALID", "Extends must be a valid class name", {"extends": extends_type})

	var source: String = "extends %s\n" % extends_type
	return _write_and_check(script_path, source, request_id, context, {"created": true})


func handle_write(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "script.write", "Script write requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var script_path: String = String(params.get("path", ""))
	var path_error: Dictionary = _validate_script_path(script_path, false, context, request_id)
	if not path_error.is_empty():
		return path_error
	if not params.has("body"):
		return context["bridge_error"].call(400, request_id, "SCRIPT_BODY_MISSING", "Script body is required", {})
	var source: String = String(params.get("body", ""))
	return _write_and_check(script_path, source, request_id, context, {"written": true})


func handle_check(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "script.check", "Script check requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var script_path: String = String(params.get("path", ""))
	var path_error: Dictionary = _validate_script_path(script_path, true, context, request_id)
	if not path_error.is_empty():
		return path_error

	var source: String = FileAccess.get_file_as_string(script_path)
	var check_error: Dictionary = _syntax_error(script_path, source, request_id, context)
	if not check_error.is_empty():
		return check_error
	return context["bridge_ok"].call(request_id, {
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
	var capture := SyntaxLogCapture.new()
	var capture_enabled := OS.has_method("add_logger") and OS.has_method("remove_logger")
	if capture_enabled:
		OS.add_logger(capture)
	var err: Error = script.reload()
	if capture_enabled:
		OS.remove_logger(capture)
	if err == OK:
		return {}
	var detail := _syntax_error_detail(script_path, source, err, capture.entries)
	return context["bridge_error"].call(400, request_id, "SCRIPT_SYNTAX_INVALID", "Script did not pass Godot syntax check", {
		"path": detail["path"],
		"error": detail["error"],
		"diagnostic": detail["diagnostic"],
		"line": detail["line"],
		"source": detail["source"],
	})


func _syntax_error_detail(script_path: String, source: String, err: Error, entries: Array[Dictionary]) -> Dictionary:
	var diagnostic := ""
	var line := -1
	for entry in entries:
		var entry_file := String(entry.get("file", ""))
		var message := _entry_message(entry)
		if entry_file == script_path or message.contains(script_path):
			diagnostic = message
			line = int(entry.get("line", -1))
			break
	if diagnostic == "" and not entries.is_empty():
		var entry := entries[entries.size() - 1]
		diagnostic = _entry_message(entry)
		line = int(entry.get("line", -1))
	if diagnostic == "":
		diagnostic = error_string(err)
	return {
		"path": script_path,
		"error": error_string(err),
		"diagnostic": diagnostic,
		"line": line,
		"source": _source_context(source, line),
	}


func _entry_message(entry: Dictionary) -> String:
	var message := String(entry.get("rationale", ""))
	if message == "":
		message = String(entry.get("code", ""))
	if message == "":
		message = String(entry.get("message", ""))
	return message


func _source_context(source: String, line: int) -> Array[Dictionary]:
	if line <= 0:
		var empty: Array[Dictionary] = []
		return empty
	var lines := source.split("\n", true)
	if line > lines.size():
		var empty: Array[Dictionary] = []
		return empty
	var start := max(1, line - 2)
	var end := min(lines.size(), line + 2)
	var context: Array[Dictionary] = []
	for number in range(start, end + 1):
		context.append({
			"line": number,
			"text": lines[number - 1],
			"error": number == line,
		})
	return context


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
