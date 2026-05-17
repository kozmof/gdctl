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
	elif String(job.get("kind", "")) == "run.probe.raycast":
		_run_probe_raycast_job(job_id, context)
	elif String(job.get("kind", "")) == "run.probe.node":
		_run_probe_node_job(job_id, context)
	elif String(job.get("kind", "")) == "run.instantiate":
		_run_run_instantiate_job(job_id, context)
	elif String(job.get("kind", "")) == "run.scene-reload":
		_run_run_scene_reload_job(job_id, context)
	elif String(job.get("kind", "")) == "run.profile":
		_run_run_profile_job(job_id, context)
	elif String(job.get("kind", "")) == "test.gdscript":
		_run_gdscript_test_job(job_id, context)
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
			"viewport_path": String(detail.get("viewport_path", "")),
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


func _run_probe_raycast_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	if not bool(detail.get("requested", false)):
		var dir_err := _ensure_runtime_dirs()
		if dir_err != OK:
			_finish_error(job_id, "RUN_PROBE_REQUEST_FAILED", "Could not create runtime probe exchange directory", {"error": error_string(dir_err)}, context)
			return
		var request_path := RUNTIME_REQUESTS + job_id + ".json"
		var request := {
			"id": job_id,
			"kind": "raycast",
			"frames": 1,
			"created_at": Time.get_datetime_string_from_system(true),
		}
		var file := FileAccess.open(request_path, FileAccess.WRITE)
		if file == null:
			_finish_error(job_id, "RUN_PROBE_REQUEST_FAILED", "Could not write runtime raycast request", {"path": request_path}, context)
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
			_finish_error(job_id, "RUN_PROBE_RESULT_INVALID", "Runtime raycast result is invalid JSON", {"path": result_path}, context)
			return
		var result: Dictionary = parsed
		if not bool(result.get("ok", false)):
			_finish_error(job_id, "RUN_PROBE_RAYCAST_FAILED", String(result.get("error", "Runtime raycast probe failed")), {"path": result_path}, context)
			return
		_finish_ok(job_id, {
			"hit": bool(result.get("hit", false)),
			"camera_path": String(result.get("camera_path", "")),
			"ray_origin": result.get("ray_origin", []),
			"ray_direction": result.get("ray_direction", []),
			"hit_collider": String(result.get("hit_collider", "")),
			"hit_position": result.get("hit_position", []),
			"hit_normal": result.get("hit_normal", []),
			"hit_distance": float(result.get("hit_distance", 0.0)),
		}, context)
		return

	var started_ticks: int = int(detail.get("started_ticks", Time.get_ticks_msec()))
	if Time.get_ticks_msec() - started_ticks > RUNTIME_INPUT_TIMEOUT_MS:
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		_finish_error(job_id, "RUN_PROBE_RAYCAST_TIMEOUT", "Runtime helper did not return a raycast result. Make sure GdctlRuntimeBridge autoload is active.", {"request_id": job_id}, context)
		return
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	pending_jobs.append(job_id)


func _run_probe_node_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	if not bool(detail.get("requested", false)):
		var dir_err := _ensure_runtime_dirs()
		if dir_err != OK:
			_finish_error(job_id, "RUN_PROBE_REQUEST_FAILED", "Could not create runtime probe exchange directory", {"error": error_string(dir_err)}, context)
			return
		var request_path := RUNTIME_REQUESTS + job_id + ".json"
		var request := {
			"id": job_id,
			"kind": "node_probe",
			"frames": 1,
			"node_path": String(detail.get("path", "")),
			"properties": detail.get("properties", []),
			"created_at": Time.get_datetime_string_from_system(true),
		}
		var file := FileAccess.open(request_path, FileAccess.WRITE)
		if file == null:
			_finish_error(job_id, "RUN_PROBE_REQUEST_FAILED", "Could not write runtime node probe request", {"path": request_path}, context)
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
			_finish_error(job_id, "RUN_PROBE_RESULT_INVALID", "Runtime node probe result is invalid JSON", {"path": result_path}, context)
			return
		var result: Dictionary = parsed
		if not bool(result.get("ok", false)):
			_finish_error(job_id, "RUN_PROBE_NODE_FAILED", String(result.get("error", "Runtime node probe failed")), {"path": result_path}, context)
			return
		_finish_ok(job_id, {
			"source": "game",
			"path": String(result.get("path", "")),
			"type": String(result.get("type", "")),
			"properties": result.get("properties", {}),
		}, context)
		return

	var started_ticks: int = int(detail.get("started_ticks", Time.get_ticks_msec()))
	if Time.get_ticks_msec() - started_ticks > RUNTIME_INPUT_TIMEOUT_MS:
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		_finish_error(job_id, "RUN_PROBE_NODE_TIMEOUT", "Runtime helper did not return a node probe result. Make sure GdctlRuntimeBridge autoload is active.", {"request_id": job_id}, context)
		return
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	pending_jobs.append(job_id)


func _run_run_instantiate_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	if not bool(detail.get("requested", false)):
		var dir_err := _ensure_runtime_dirs()
		if dir_err != OK:
			_finish_error(job_id, "RUN_INSTANTIATE_REQUEST_FAILED", "Could not create runtime exchange directory", {"error": error_string(dir_err)}, context)
			return
		var request_path := RUNTIME_REQUESTS + job_id + ".json"
		var request := {
			"id": job_id,
			"kind": "instantiate",
			"frames": 1,
			"scene": String(detail.get("scene", "")),
			"parent": String(detail.get("parent", "")),
			"name": String(detail.get("name", "")),
			"created_at": Time.get_datetime_string_from_system(true),
		}
		var file := FileAccess.open(request_path, FileAccess.WRITE)
		if file == null:
			_finish_error(job_id, "RUN_INSTANTIATE_REQUEST_FAILED", "Could not write runtime instantiate request", {"path": request_path}, context)
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
			_finish_error(job_id, "RUN_INSTANTIATE_RESULT_INVALID", "Runtime instantiate result is invalid JSON", {"path": result_path}, context)
			return
		var result: Dictionary = parsed
		if not bool(result.get("ok", false)):
			_finish_error(job_id, "RUN_INSTANTIATE_FAILED", String(result.get("error", "Runtime helper failed to instantiate scene")), {"path": result_path}, context)
			return
		_finish_ok(job_id, {
			"source": "game",
			"scene": String(result.get("scene", "")),
			"parent": String(result.get("parent", "")),
			"name": String(result.get("name", "")),
			"path": String(result.get("path", "")),
			"instanced": bool(result.get("instanced", false)),
		}, context)
		return

	var started_ticks: int = int(detail.get("started_ticks", Time.get_ticks_msec()))
	if Time.get_ticks_msec() - started_ticks > RUNTIME_INPUT_TIMEOUT_MS:
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		_finish_error(job_id, "RUN_INSTANTIATE_HELPER_TIMEOUT", "Runtime helper did not complete instantiate. Make sure GdctlRuntimeBridge autoload is active.", {"request_id": job_id}, context)
		return
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	pending_jobs.append(job_id)


func _run_run_scene_reload_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	if not bool(detail.get("requested", false)):
		var dir_err := _ensure_runtime_dirs()
		if dir_err != OK:
			_finish_error(job_id, "RUN_SCENE_RELOAD_REQUEST_FAILED", "Could not create runtime exchange directory", {"error": error_string(dir_err)}, context)
			return
		var request_path := RUNTIME_REQUESTS + job_id + ".json"
		var request := {
			"id": job_id,
			"kind": "scene_reload",
			"frames": 1,
			"created_at": Time.get_datetime_string_from_system(true),
		}
		var file := FileAccess.open(request_path, FileAccess.WRITE)
		if file == null:
			_finish_error(job_id, "RUN_SCENE_RELOAD_REQUEST_FAILED", "Could not write runtime scene-reload request", {"path": request_path}, context)
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
			_finish_error(job_id, "RUN_SCENE_RELOAD_RESULT_INVALID", "Runtime scene-reload result is invalid JSON", {"path": result_path}, context)
			return
		var result: Dictionary = parsed
		if not bool(result.get("ok", false)):
			_finish_error(job_id, "RUN_SCENE_RELOAD_FAILED", String(result.get("error", "Runtime helper failed to reload scene")), {"path": result_path}, context)
			return
		_finish_ok(job_id, {
			"reloaded": true,
			"scene": String(result.get("scene", "")),
		}, context)
		return

	var started_ticks: int = int(detail.get("started_ticks", Time.get_ticks_msec()))
	if Time.get_ticks_msec() - started_ticks > RUNTIME_INPUT_TIMEOUT_MS:
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		_finish_error(job_id, "RUN_SCENE_RELOAD_HELPER_TIMEOUT", "Runtime helper did not complete scene reload. Make sure GdctlRuntimeBridge autoload is active.", {"request_id": job_id}, context)
		return
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	pending_jobs.append(job_id)


func _run_run_profile_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	if not bool(detail.get("requested", false)):
		var dir_err := _ensure_runtime_dirs()
		if dir_err != OK:
			_finish_error(job_id, "RUN_PROFILE_REQUEST_FAILED", "Could not create runtime exchange directory", {"error": error_string(dir_err)}, context)
			return
		var request_path := RUNTIME_REQUESTS + job_id + ".json"
		var metrics: Array = Array(detail.get("metrics", ["fps"]))
		var duration_ms: float = float(detail.get("duration_ms", 5000))
		var request := {
			"id": job_id,
			"kind": "profile",
			"frames": 1,
			"metrics": metrics,
			"duration_ms": duration_ms,
			"created_at": Time.get_datetime_string_from_system(true),
		}
		var file := FileAccess.open(request_path, FileAccess.WRITE)
		if file == null:
			_finish_error(job_id, "RUN_PROFILE_REQUEST_FAILED", "Could not write runtime profile request", {"path": request_path}, context)
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
			_finish_error(job_id, "RUN_PROFILE_RESULT_INVALID", "Runtime profile result is invalid JSON", {"path": result_path}, context)
			return
		var result: Dictionary = parsed
		if not bool(result.get("ok", false)):
			_finish_error(job_id, "RUN_PROFILE_FAILED", String(result.get("error", "Runtime helper failed to profile")), {"path": result_path}, context)
			return
		_finish_ok(job_id, result, context)
		return
	var started_ticks: int = int(detail.get("started_ticks", Time.get_ticks_msec()))
	var duration_ms: float = float(detail.get("duration_ms", 5000))
	# Allow duration + 5 seconds for result to appear
	var timeout_ms: int = int(duration_ms) + 5000
	if Time.get_ticks_msec() - started_ticks > timeout_ms:
		_remove_runtime_file(RUNTIME_REQUESTS + job_id + ".json")
		_finish_error(job_id, "RUN_PROFILE_HELPER_TIMEOUT", "Runtime helper did not return profile results. Make sure GdctlRuntimeBridge autoload is active.", {"request_id": job_id}, context)
		return
	job["status"] = "running"
	job["updated_at"] = Time.get_datetime_string_from_system(true)
	jobs[job_id] = job
	pending_jobs.append(job_id)



func _run_gdscript_test_job(job_id: String, context: Dictionary) -> void:
	var job: Dictionary = jobs[job_id]
	var detail: Dictionary = job.get("detail", {})
	var path: String = String(detail.get("path", ""))
	var dir: String = String(detail.get("dir", ""))
	var files: Array[String] = []
	if path != "":
		if not FileAccess.file_exists(path):
			_finish_error(job_id, "TEST_NOT_FOUND", "Test script does not exist", {"path": path}, context)
			return
		files.append(path)
	else:
		var discovered: Dictionary = _discover_gdscript_tests(dir)
		if not bool(discovered.get("ok", false)):
			_finish_error(job_id, String(discovered.get("code", "TEST_DISCOVERY_FAILED")), String(discovered.get("message", "Could not discover tests")), Dictionary(discovered.get("detail", {})), context)
			return
		var discovered_files: Array = discovered.get("files", [])
		for discovered_file in discovered_files:
			files.append(String(discovered_file))
	if files.is_empty():
		_finish_error(job_id, "TESTS_NOT_FOUND", "No GDScript test files found", {"path": path, "dir": dir}, context)
		return

	var started := Time.get_ticks_msec()
	var suite := {
		"passed": true,
		"total": 0,
		"passed_count": 0,
		"failed_count": 0,
		"duration_ms": 0,
		"files": [],
	}
	for file_path in files:
		var file_result: Dictionary = _run_gdscript_test_file(file_path)
		suite["files"].append(file_result)
		suite["total"] = int(suite["total"]) + int(file_result.get("total", 0))
		suite["passed_count"] = int(suite["passed_count"]) + int(file_result.get("passed_count", 0))
		suite["failed_count"] = int(suite["failed_count"]) + int(file_result.get("failed_count", 0))
		if not bool(file_result.get("passed", false)):
			suite["passed"] = false
	suite["duration_ms"] = Time.get_ticks_msec() - started
	if int(suite["total"]) == 0:
		_finish_error(job_id, "TESTS_NOT_FOUND", "No GDScript test methods found", {"path": path, "dir": dir, "files": files}, context)
		return
	_finish_ok(job_id, suite, context)


func _discover_gdscript_tests(dir_path: String) -> Dictionary:
	if dir_path == "" or not dir_path.begins_with("res://"):
		return {"ok": false, "code": "TEST_DIR_INVALID", "message": "Test dir must be a res:// path", "detail": {"dir": dir_path}}
	var dir := DirAccess.open(dir_path)
	if dir == null:
		return {"ok": false, "code": "TEST_DIR_NOT_FOUND", "message": "Test dir does not exist", "detail": {"dir": dir_path}}
	var files: Array[String] = []
	_collect_gdscript_test_files(dir_path.trim_suffix("/"), files)
	files.sort()
	return {"ok": true, "files": files}


func _collect_gdscript_test_files(dir_path: String, files: Array[String]) -> void:
	var dir := DirAccess.open(dir_path)
	if dir == null:
		return
	dir.list_dir_begin()
	var name := dir.get_next()
	while name != "":
		if name == "." or name == "..":
			name = dir.get_next()
			continue
		var child_path := dir_path + "/" + name
		if dir.current_is_dir():
			_collect_gdscript_test_files(child_path, files)
		elif name.begins_with("test_") and name.ends_with(".gd") and name != "test_case.gd":
			files.append(child_path)
		name = dir.get_next()
	dir.list_dir_end()


func _run_gdscript_test_file(path: String) -> Dictionary:
	var started := Time.get_ticks_msec()
	var result := {
		"path": path,
		"passed": true,
		"total": 0,
		"passed_count": 0,
		"failed_count": 0,
		"duration_ms": 0,
		"tests": [],
	}
	var script := ResourceLoader.load(path, "GDScript", ResourceLoader.CACHE_MODE_REPLACE)
	if not (script is GDScript):
		return _gdscript_file_load_error(result, started, "TEST_SCRIPT_LOAD_FAILED", "Could not load test script", {})
	var reload_err: Error = (script as GDScript).reload()
	if reload_err != OK:
		return _gdscript_file_load_error(result, started, "TEST_SCRIPT_INVALID", "Test script did not pass Godot syntax check", {"error": error_string(reload_err)})
	var instance: Object = (script as GDScript).new()
	if instance == null:
		return _gdscript_file_load_error(result, started, "TEST_SCRIPT_INSTANTIATE_FAILED", "Could not instantiate test script", {})
	if not instance.has_method("_gdctl_begin_test") or not instance.has_method("_gdctl_end_test"):
		return _gdscript_file_load_error(result, started, "TEST_CASE_INVALID", "Test script must extend res://addons/godot_tcp_bridge/testing/test_case.gd", {})
	var methods := _gdscript_test_methods(instance)
	if methods.is_empty():
		result["duration_ms"] = Time.get_ticks_msec() - started
		return result
	var before_all_result: Dictionary = _run_gdscript_hook(instance, "before_all")
	if not before_all_result.is_empty():
		result["tests"].append(before_all_result)
		result["total"] = int(result["total"]) + 1
		result["failed_count"] = int(result["failed_count"]) + 1
		result["passed"] = false
		result["duration_ms"] = Time.get_ticks_msec() - started
		return result
	for method in methods:
		var test_result: Dictionary = _run_gdscript_test_method(instance, method)
		result["tests"].append(test_result)
		result["total"] = int(result["total"]) + 1
		if String(test_result.get("status", "")) == "passed":
			result["passed_count"] = int(result["passed_count"]) + 1
		else:
			result["failed_count"] = int(result["failed_count"]) + 1
			result["passed"] = false
	var after_all_result: Dictionary = _run_gdscript_hook(instance, "after_all")
	if not after_all_result.is_empty():
		result["tests"].append(after_all_result)
		result["total"] = int(result["total"]) + 1
		result["failed_count"] = int(result["failed_count"]) + 1
		result["passed"] = false
	result["duration_ms"] = Time.get_ticks_msec() - started
	return result


func _gdscript_file_load_error(result: Dictionary, started: int, code: String, message: String, detail: Dictionary) -> Dictionary:
	var failure := {"message": message, "code": code}
	for key in detail.keys():
		failure[key] = detail[key]
	result["passed"] = false
	result["total"] = 1
	result["failed_count"] = 1
	result["duration_ms"] = Time.get_ticks_msec() - started
	result["tests"] = [{"name": "<load>", "status": "failed", "duration_ms": int(result["duration_ms"]), "failures": [failure]}]
	return result


func _gdscript_test_methods(instance: Object) -> Array[String]:
	var methods: Array[String] = []
	for method in instance.get_method_list():
		var name := String(method.get("name", ""))
		var args: Array = method.get("args", [])
		if name.begins_with("test_") and args.is_empty():
			methods.append(name)
	methods.sort()
	return methods


func _run_gdscript_test_method(instance: Object, method: String) -> Dictionary:
	var started := Time.get_ticks_msec()
	instance.call("_gdctl_begin_test", method)
	_call_optional(instance, "before_each")
	instance.call(method)
	_call_optional(instance, "after_each")
	var failures: Array = instance.call("_gdctl_end_test")
	return {
		"name": method,
		"status": "passed" if failures.is_empty() else "failed",
		"duration_ms": Time.get_ticks_msec() - started,
		"failures": failures,
	}


func _run_gdscript_hook(instance: Object, method: String) -> Dictionary:
	if not instance.has_method(method):
		return {}
	var started := Time.get_ticks_msec()
	instance.call("_gdctl_begin_test", method)
	instance.call(method)
	var failures: Array = instance.call("_gdctl_end_test")
	if failures.is_empty():
		return {}
	return {
		"name": method,
		"status": "failed",
		"duration_ms": Time.get_ticks_msec() - started,
		"failures": failures,
	}


func _call_optional(instance: Object, method: String) -> void:
	if instance.has_method(method):
		instance.call(method)


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
