@tool
extends RefCounted


func handle_gdscript(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "test.gdscript", "GDScript tests require bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var path: String = String(params.get("path", ""))
	var dir: String = String(params.get("dir", ""))
	var has_path := path != ""
	var has_dir := dir != ""
	if has_path == has_dir:
		return context["bridge_error"].call(400, request_id, "TEST_SELECTOR_INVALID", "GDScript tests require exactly one of path or dir", {})
	if has_path:
		if not path.begins_with("res://") or not path.ends_with(".gd"):
			return context["bridge_error"].call(400, request_id, "TEST_PATH_INVALID", "Test path must be a res:// .gd path", {"path": path})
		if not FileAccess.file_exists(path):
			return context["bridge_error"].call(404, request_id, "TEST_NOT_FOUND", "Test script does not exist", {"path": path})
	if has_dir:
		if not dir.begins_with("res://"):
			return context["bridge_error"].call(400, request_id, "TEST_DIR_INVALID", "Test dir must be a res:// path", {"dir": dir})
		if DirAccess.open(dir) == null:
			return context["bridge_error"].call(404, request_id, "TEST_DIR_NOT_FOUND", "Test dir does not exist", {"dir": dir})
	var job_id: String = String(context["queue_job"].call("test.gdscript", {"path": path, "dir": dir}))
	return context["bridge_ok"].call(request_id, {
		"queued": true,
		"job_id": job_id,
		"path": path,
		"dir": dir,
	})
