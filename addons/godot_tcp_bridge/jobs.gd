@tool
extends RefCounted

const RUNTIME_ROOT := "res://.gdctl_runtime/"
const RUNTIME_REQUESTS := "res://.gdctl_runtime/requests/"
const RUNTIME_RESULTS := "res://.gdctl_runtime/results/"
const RUNTIME_SCREENSHOT_TIMEOUT_MS := 8000
const RUNTIME_INPUT_TIMEOUT_MS := 15000

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
	elif String(job.get("kind", "")) == "run.screenshot":
		_run_run_screenshot_job(job_id, context)
	elif String(job.get("kind", "")) == "run.input":
		_run_run_input_job(job_id, context)
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


func _run_run_screenshot_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	var source: String = String(detail.get("source", "game"))
	if source == "screen":
		_run_screen_screenshot_job(job_id, context)
		return
	if source != "game":
		_finish_error(job_id, "RUN_SCREENSHOT_SOURCE_INVALID", "Run screenshot source must be game or screen", {"source": source}, context)
		return
	_run_game_screenshot_job(job_id, context)


func _run_screen_screenshot_job(job_id: String, context: Dictionary) -> void:
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

	if not ClassDB.class_has_method("DisplayServer", "screen_get_image"):
		_finish_error(job_id, "RUN_SCREENSHOT_UNSUPPORTED", "DisplayServer.screen_get_image is unavailable in this Godot build", {}, context)
		return
	var screen: int = int(detail.get("screen", 0))
	if screen < 0 or screen >= DisplayServer.get_screen_count():
		_finish_error(job_id, "RUN_SCREEN_INVALID", "Screen index is out of range", {"screen": screen}, context)
		return
	var image: Image = DisplayServer.screen_get_image(screen)
	if image == null or image.is_empty():
		_finish_error(job_id, "RUN_SCREENSHOT_EMPTY", "Host screen image is empty", {"screen": screen}, context)
		return
	var png: PackedByteArray = image.save_png_to_buffer()
	if png.is_empty():
		_finish_error(job_id, "RUN_SCREENSHOT_PNG_EMPTY", "Could not encode run screenshot as PNG", {"screen": screen}, context)
		return
	_finish_ok(job_id, {
		"format": "png",
		"source": "screen",
		"screen": screen,
		"width": image.get_width(),
		"height": image.get_height(),
		"content_base64": Marshalls.raw_to_base64(png),
	}, context)


func _run_game_screenshot_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	if not bool(detail.get("requested", false)):
		var dir_err := _ensure_runtime_dirs()
		if dir_err != OK:
			_finish_error(job_id, "RUN_SCREENSHOT_REQUEST_FAILED", "Could not create runtime screenshot exchange directory", {"error": error_string(dir_err)}, context)
			return
		var request_path := RUNTIME_REQUESTS + job_id + ".json"
		var request := {
			"id": job_id,
			"kind": "screenshot",
			"frames": int(detail.get("frames_remaining", 2)),
			"created_at": Time.get_datetime_string_from_system(true),
		}
		var file := FileAccess.open(request_path, FileAccess.WRITE)
		if file == null:
			_finish_error(job_id, "RUN_SCREENSHOT_REQUEST_FAILED", "Could not write runtime screenshot request", {"path": request_path}, context)
			return
		file.store_string(JSON.stringify(request))
		file.close()
		detail["requested"] = true
		detail["started_ticks"] = Time.get_ticks_msec()
		job["detail"] = detail
		job["status"] = "running"
		job["updated_at"] = Time.get_datetime_string_from_system(true)
		jobs[job_id] = job
		pending_jobs.append(job_id)
		return

	var result_path := RUNTIME_RESULTS + job_id + ".json"
	if FileAccess.file_exists(result_path):
		var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(result_path))
		_remove_runtime_file(result_path)
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		if typeof(parsed) != TYPE_DICTIONARY:
			_finish_error(job_id, "RUN_SCREENSHOT_RESULT_INVALID", "Runtime screenshot result is invalid JSON", {"path": result_path}, context)
			return
		var result: Dictionary = parsed
		if not bool(result.get("ok", false)):
			_finish_error(job_id, "RUN_SCREENSHOT_FAILED", String(result.get("error", "Runtime helper failed to capture screenshot")), {"path": result_path}, context)
			return
		_finish_ok(job_id, {
			"format": String(result.get("format", "png")),
			"source": "game",
			"width": int(result.get("width", 0)),
			"height": int(result.get("height", 0)),
			"content_base64": String(result.get("content_base64", "")),
			"request_id": job_id,
		}, context)
		return

	var started_ticks: int = int(detail.get("started_ticks", Time.get_ticks_msec()))
	if Time.get_ticks_msec() - started_ticks > RUNTIME_SCREENSHOT_TIMEOUT_MS:
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		var timeout_detail := {"request_id": job_id}
		var debugger: Dictionary = context["debugger_state"].call()
		if bool(debugger.get("paused", false)):
			timeout_detail["debugger"] = debugger
		_finish_error(job_id, "RUN_SCREENSHOT_HELPER_TIMEOUT", "Runtime helper did not return a game viewport screenshot. Restart the scene with gdctl run start, or use run screenshot --source screen.", timeout_detail, context)
		return
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	pending_jobs.append(job_id)


func _run_run_input_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	if not bool(detail.get("requested", false)):
		var dir_err := _ensure_runtime_dirs()
		if dir_err != OK:
			_finish_error(job_id, "RUN_INPUT_REQUEST_FAILED", "Could not create runtime input exchange directory", {"error": error_string(dir_err)}, context)
			return
		var request_path := RUNTIME_REQUESTS + job_id + ".json"
		var request := {
			"id": job_id,
			"kind": "input",
			"frames": 1,
			"steps": detail.get("steps", []),
			"created_at": Time.get_datetime_string_from_system(true),
		}
		var file := FileAccess.open(request_path, FileAccess.WRITE)
		if file == null:
			_finish_error(job_id, "RUN_INPUT_REQUEST_FAILED", "Could not write runtime input request", {"path": request_path}, context)
			return
		file.store_string(JSON.stringify(request))
		file.close()
		detail["requested"] = true
		detail["started_ticks"] = Time.get_ticks_msec()
		job["detail"] = detail
		job["status"] = "running"
		job["updated_at"] = Time.get_datetime_string_from_system(true)
		jobs[job_id] = job
		pending_jobs.append(job_id)
		return

	var result_path := RUNTIME_RESULTS + job_id + ".json"
	if FileAccess.file_exists(result_path):
		var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(result_path))
		_remove_runtime_file(result_path)
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		if typeof(parsed) != TYPE_DICTIONARY:
			_finish_error(job_id, "RUN_INPUT_RESULT_INVALID", "Runtime input result is invalid JSON", {"path": result_path}, context)
			return
		var result: Dictionary = parsed
		if not bool(result.get("ok", false)):
			_finish_error(job_id, "RUN_INPUT_FAILED", String(result.get("error", "Runtime helper failed to execute input")), {"path": result_path}, context)
			return
		_finish_ok(job_id, {
			"source": "game",
			"steps": int(result.get("steps", 0)),
			"duration_ms": int(result.get("duration_ms", 0)),
			"request_id": job_id,
		}, context)
		return

	var started_ticks: int = int(detail.get("started_ticks", Time.get_ticks_msec()))
	if Time.get_ticks_msec() - started_ticks > RUNTIME_INPUT_TIMEOUT_MS:
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		_finish_error(job_id, "RUN_INPUT_HELPER_TIMEOUT", "Runtime helper did not complete input playback. Restart the scene with gdctl run start.", {"request_id": job_id}, context)
		return
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	pending_jobs.append(job_id)


func _ensure_runtime_dirs() -> Error:
	for path in [RUNTIME_ROOT, RUNTIME_REQUESTS, RUNTIME_RESULTS]:
		var err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(path))
		if err != OK:
			return err
	return OK


func _remove_runtime_file(path: String) -> void:
	if FileAccess.file_exists(path):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(path))


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
