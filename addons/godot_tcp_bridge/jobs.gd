@tool
extends RefCounted

var jobs: Dictionary = {}
var pending_jobs: Array[String] = []


func queue(kind: String, detail: Dictionary, context: Dictionary) -> String:
	var job_id: String = "%s-%d-%d" % [kind.replace(".", "-"), Time.get_ticks_msec(), randi()]
	jobs[job_id] = {
		"id": job_id,
		"kind": kind,
		"status": "pending",
		"created_at": Time.get_datetime_string_from_system(true),
		"updated_at": Time.get_datetime_string_from_system(true),
		"detail": detail,
		"result": {},
		"error": {},
	}
	pending_jobs.append(job_id)
	context["log"].call("info", "bridge.job", "Job queued", {"job_id": job_id, "kind": kind})
	return job_id


func status_response(job_id: String, protocol) -> Dictionary:
	if not jobs.has(job_id):
		return protocol.bridge_error(404, "", "JOB_NOT_FOUND", "Job does not exist", {"job_id": job_id})
	return protocol.http_json(200, {"ok": true, "job": jobs[job_id]})


func process(context: Dictionary) -> void:
	if pending_jobs.is_empty():
		return
	var job_id: String = pending_jobs.pop_front()
	if not jobs.has(job_id):
		return
	var job: Dictionary = jobs[job_id]
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	if String(job.get("kind", "")) == "scene.open":
		_run_scene_open_job(job_id, context)
	elif String(job.get("kind", "")) == "scene.save":
		_run_scene_save_job(job_id, context)
	elif String(job.get("kind", "")) == "viewport.screenshot":
		_run_viewport_screenshot_job(job_id, context)
	else:
		_finish_error(job_id, "JOB_KIND_UNKNOWN", "Unknown job kind", {"kind": job.get("kind", "")}, context)


func _run_scene_open_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	var scene_path: String = String(detail.get("path", ""))
	if scene_path == "" or not FileAccess.file_exists(scene_path):
		_finish_error(job_id, "SCENE_NOT_FOUND", "Scene does not exist", {"path": scene_path}, context)
		return
	var editor_plugin: EditorPlugin = context["editor_plugin"]
	if editor_plugin == null:
		_finish_error(job_id, "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {}, context)
		return
	editor_plugin.get_editor_interface().open_scene_from_path(scene_path)
	var root: Node = context["edited_scene_root"].call()
	var root_path := ""
	var root_name := ""
	var root_type := ""
	if root != null:
		root_path = context["logical_path"].call(root)
		root_name = String(root.name)
		root_type = root.get_class()
	_finish_ok(job_id, {
		"opened": true,
		"path": scene_path,
		"root": root_path,
		"root_name": root_name,
		"root_type": root_type,
	}, context)


func _run_scene_save_job(job_id: String, context: Dictionary) -> void:
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		_finish_error(job_id, "NO_SCENE_OPEN", "No edited scene is open", {}, context)
		return
	if root.scene_file_path == "":
		_finish_error(job_id, "SCENE_PATH_MISSING", "Edited scene has no path", {"root": context["logical_path"].call(root)}, context)
		return
	var editor_plugin: EditorPlugin = context["editor_plugin"]
	if editor_plugin == null:
		_finish_error(job_id, "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {}, context)
		return
	editor_plugin.get_editor_interface().save_scene()
	_finish_ok(job_id, {
		"saved": true,
		"path": root.scene_file_path,
		"root": context["logical_path"].call(root),
	}, context)


func _run_viewport_screenshot_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	var frames_remaining: int = int(detail.get("frames_remaining", 0))
	if frames_remaining > 0:
		detail["frames_remaining"] = frames_remaining - 1
		job["detail"] = detail
		job["status"] = "running"
		job["updated_at"] = Time.get_datetime_string_from_system(true)
		jobs[job_id] = job
		pending_jobs.append(job_id)
		return

	var editor_plugin: EditorPlugin = context["editor_plugin"]
	if editor_plugin == null:
		_finish_error(job_id, "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {}, context)
		return
	var editor_interface := editor_plugin.get_editor_interface()
	var kind: String = String(detail.get("kind", "2d"))
	var viewport: SubViewport = null
	if kind == "3d":
		viewport = editor_interface.get_editor_viewport_3d(int(detail.get("index", 0)))
	else:
		viewport = editor_interface.get_editor_viewport_2d()
	if viewport == null:
		_finish_error(job_id, "VIEWPORT_NOT_FOUND", "Editor viewport is unavailable", {"kind": kind, "index": detail.get("index", 0)}, context)
		return
	var texture := viewport.get_texture()
	if texture == null:
		_finish_error(job_id, "VIEWPORT_TEXTURE_MISSING", "Editor viewport texture is unavailable", {"kind": kind}, context)
		return
	var image: Image = texture.get_image()
	if image == null or image.is_empty():
		_finish_error(job_id, "VIEWPORT_IMAGE_EMPTY", "Editor viewport image is empty", {"kind": kind}, context)
		return
	var png: PackedByteArray = image.save_png_to_buffer()
	if png.is_empty():
		_finish_error(job_id, "VIEWPORT_PNG_EMPTY", "Could not encode viewport image as PNG", {"kind": kind}, context)
		return
	_finish_ok(job_id, {
		"format": "png",
		"kind": kind,
		"index": int(detail.get("index", 0)),
		"width": image.get_width(),
		"height": image.get_height(),
		"content_base64": Marshalls.raw_to_base64(png),
	}, context)


func _finish_ok(job_id: String, result: Dictionary, context: Dictionary) -> void:
	if not jobs.has(job_id):
		return
	var job: Dictionary = jobs[job_id]
	job["status"] = "succeeded"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	job["result"] = result
	jobs[job_id] = job
	context["log"].call("info", "bridge.job", "Job succeeded", {"job_id": job_id, "kind": job.get("kind", "")})


func _finish_error(job_id: String, code: String, message: String, detail: Dictionary, context: Dictionary) -> void:
	if not jobs.has(job_id):
		return
	var job: Dictionary = jobs[job_id]
	job["status"] = "failed"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	job["error"] = {"code": code, "message": message, "detail": detail}
	jobs[job_id] = job
	context["log"].call("error", "bridge.job", "Job failed", {"job_id": job_id, "kind": job.get("kind", ""), "error": job["error"]})
