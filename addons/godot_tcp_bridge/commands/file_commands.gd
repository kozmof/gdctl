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


func handle_list(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "file.list", "File listing requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var recursive: bool = bool(params.get("recursive", false))
	if path == "" or not path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Path must be a res:// path", {"path": path})
	if path.find("..") != -1:
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Path must not contain ..", {"path": path})
	var abs_path: String = ProjectSettings.globalize_path(path)
	if not DirAccess.dir_exists_absolute(abs_path):
		return context["bridge_error"].call(404, request_id, "DIR_NOT_FOUND", "Directory does not exist", {"path": path})

	var files: Array = []
	var dirs: Array = []
	_list_recursive(path, files, dirs, recursive)
	return context["bridge_ok"].call(request_id, {
		"path": path,
		"files": files,
		"dirs": dirs,
	})


func handle_mkdir(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "file.mkdir", "File mkdir requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	if path == "" or not path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Path must be a res:// path", {"path": path})
	if path.find("..") != -1:
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Path must not contain ..", {"path": path})
	var abs_path: String = ProjectSettings.globalize_path(path)
	var err: Error = DirAccess.make_dir_recursive_absolute(abs_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "DIR_CREATE_FAILED", "Could not create directory", {"path": path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {
		"path": path,
		"created": true,
	})


func handle_delete(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "file.delete", "File delete requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	if path == "" or not path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Path must be a res:// path", {"path": path})
	if path == "res://" or path == "res:///":
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Cannot delete project root", {"path": path})
	if path.find("..") != -1:
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Path must not contain ..", {"path": path})
	var abs_path: String = ProjectSettings.globalize_path(path)
	var err: Error = DirAccess.remove_absolute(abs_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "FILE_DELETE_FAILED", "Could not delete path", {"path": path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {
		"path": path,
		"deleted": true,
	})


func handle_exists(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "file.exists", "File exists requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	if path == "" or not path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "FILE_PATH_INVALID", "Path must be a res:// path", {"path": path})
	var is_file: bool = FileAccess.file_exists(path)
	var abs_path: String = ProjectSettings.globalize_path(path)
	var is_dir: bool = DirAccess.dir_exists_absolute(abs_path)
	return context["bridge_ok"].call(request_id, {
		"path": path,
		"exists": is_file or is_dir,
		"is_file": is_file,
		"is_dir": is_dir,
	})


func _list_recursive(path: String, files_out: Array, dirs_out: Array, recursive: bool) -> void:
	for f: String in DirAccess.get_files_at(path):
		files_out.append(path.path_join(f))
	for d: String in DirAccess.get_directories_at(path):
		dirs_out.append(path.path_join(d))
		if recursive:
			_list_recursive(path.path_join(d), files_out, dirs_out, true)


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
