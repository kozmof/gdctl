@tool
extends RefCounted


func handle_update(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "addon.update", "Addon update requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var manifest: Variant = params.get("manifest", {})
	if typeof(manifest) != TYPE_DICTIONARY:
		return context["bridge_error"].call(400, request_id, "INVALID_MANIFEST", "Addon update requires a manifest object", {})
	var manifest_dict: Dictionary = manifest
	var manifest_files_value: Variant = manifest_dict.get("files", [])
	if typeof(manifest_files_value) != TYPE_ARRAY:
		return context["bridge_error"].call(400, request_id, "INVALID_MANIFEST", "Manifest files must be an array", {})
	var manifest_files: Array = manifest_files_value
	var files_value: Variant = params.get("files", [])
	if typeof(files_value) != TYPE_ARRAY:
		return context["bridge_error"].call(400, request_id, "INVALID_FILES", "Addon update files must be an array", {})
	var files: Array = files_value

	var allowed: Dictionary = {}
	for item in manifest_files:
		var rel: String = String(item)
		if not _is_safe_addon_path(rel):
			return context["bridge_error"].call(400, request_id, "INVALID_ADDON_PATH", "Manifest contains an unsafe file path", {"path": rel})
		allowed[rel] = true

	var backup: String = _backup_addon_files(manifest_files, context)
	var written: int = 0
	for file_item in files:
		if typeof(file_item) != TYPE_DICTIONARY:
			return context["bridge_error"].call(400, request_id, "INVALID_FILES", "Each addon update file must be an object", {})
		var file_dict: Dictionary = file_item
		var rel_path: String = String(file_dict.get("path", ""))
		if not _is_safe_addon_path(rel_path) or not allowed.has(rel_path):
			return context["bridge_error"].call(400, request_id, "INVALID_ADDON_PATH", "Addon update file path is not allowed", {"path": rel_path})
		var content_base64: String = String(file_dict.get("content_base64", ""))
		var bytes: PackedByteArray = Marshalls.base64_to_raw(content_base64)
		if bytes.is_empty() and content_base64 != "":
			return context["bridge_error"].call(400, request_id, "INVALID_FILE_CONTENT", "Addon update file content is not valid base64", {"path": rel_path})
		var write_err: Error = _write_addon_file(rel_path, bytes, context)
		if write_err != OK:
			return context["bridge_error"].call(500, request_id, "ADDON_WRITE_FAILED", "Failed to write addon file", {"path": rel_path, "error": error_string(write_err)})
		written += 1

	# Remove stale files: present in old_manifest but absent from new manifest.
	var removed: int = 0
	var old_manifest_value: Variant = params.get("old_manifest", null)
	if typeof(old_manifest_value) == TYPE_DICTIONARY:
		var old_manifest_dict: Dictionary = old_manifest_value
		var old_files_value: Variant = old_manifest_dict.get("files", [])
		if typeof(old_files_value) == TYPE_ARRAY:
			var old_files: Array = old_files_value
			var addon_root: String = String(context["addon_root"])
			for old_item in old_files:
				var old_rel: String = String(old_item)
				if not allowed.has(old_rel) and _is_safe_addon_path(old_rel):
					var stale_path: String = addon_root + old_rel
					if FileAccess.file_exists(stale_path):
						DirAccess.remove_absolute(ProjectSettings.globalize_path(stale_path))
						removed += 1

	return context["bridge_ok"].call(request_id, {
		"updated": true,
		"files_written": written,
		"files_removed": removed,
		"backup": backup,
		"reload_required": true,
	})


func _is_safe_addon_path(path: String) -> bool:
	if path == "":
		return false
	if path.begins_with("/") or path.begins_with("\\"):
		return false
	if path.find("..") != -1:
		return false
	if path.find(":") != -1:
		return false
	return true


func _backup_addon_files(files: Array, context: Dictionary) -> String:
	var addon_root: String = String(context["addon_root"])
	var backup_root_base: String = String(context["addon_backup_root"])
	var timestamp: String = Time.get_datetime_string_from_system(true).replace(":", "-")
	var backup_root: String = backup_root_base + timestamp + "/"
	var wrote_any: bool = false
	for item in files:
		var rel: String = String(item)
		if not _is_safe_addon_path(rel):
			continue
		var src: String = addon_root + rel
		if not FileAccess.file_exists(src):
			continue
		var data: PackedByteArray = FileAccess.get_file_as_bytes(src)
		var dst: String = backup_root + rel
		var dir: String = dst.get_base_dir()
		DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir))
		var file: FileAccess = FileAccess.open(dst, FileAccess.WRITE)
		if file:
			file.store_buffer(data)
			file.close()
			wrote_any = true
	if not wrote_any:
		return ""
	return backup_root


func _write_addon_file(path: String, bytes: PackedByteArray, context: Dictionary) -> Error:
	var addon_root: String = String(context["addon_root"])
	var dst: String = addon_root + path
	var dir: String = dst.get_base_dir()
	var dir_err: Error = DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir))
	if dir_err != OK:
		return dir_err
	var file: FileAccess = FileAccess.open(dst, FileAccess.WRITE)
	if file == null:
		return ERR_CANT_OPEN
	file.store_buffer(bytes)
	file.close()
	return OK
