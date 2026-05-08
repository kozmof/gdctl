@tool
extends RefCounted


func handle_write_bytes(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "file.write_bytes", "File write requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var content_base64: String = String(params.get("content_base64", ""))
	if path == "" or not path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "File path must be a res:// path", {"path": path})
	if path.find("..") != -1:
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "File path must not contain ..", {"path": path})
	if content_base64 == "":
		return context["bridge_error"].call(400, request_id, "FILE_CONTENT_MISSING", "Base64 file content is required", {})

	var bytes: PackedByteArray = Marshalls.base64_to_raw(content_base64)
	if bytes.is_empty():
		return context["bridge_error"].call(400, request_id, "FILE_CONTENT_INVALID", "File content is not valid base64 or is empty", {"path": path})
	var dir_err: Error = _ensure_resource_dir(path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "FILE_DIR_FAILED", "Could not create file directory", {"path": path, "error": error_string(dir_err)})
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		return context["bridge_error"].call(500, request_id, "FILE_WRITE_FAILED", "Could not open file for writing", {"path": path})
	file.store_buffer(bytes)
	file.close()

	return context["bridge_ok"].call(request_id, {
		"path": path,
		"bytes": bytes.size(),
		"written": true,
	})


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
