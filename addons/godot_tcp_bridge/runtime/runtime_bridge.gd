extends Node

const REQUESTS_DIR := "res://.gdctl_runtime/requests/"
const RESULTS_DIR := "res://.gdctl_runtime/results/"
const LOGS_DIR := "res://.gdctl_runtime/logs/"
const LOG_PATH := "res://.gdctl_runtime/logs/runtime.jsonl"
const STATUS_PATH := "res://.gdctl_runtime/logs/helper_status.json"

var active: Dictionary = {}
var heartbeat_elapsed := 0.0


func _ready() -> void:
	_ensure_dir(REQUESTS_DIR)
	_ensure_dir(RESULTS_DIR)
	_ensure_dir(LOGS_DIR)
	info("gdctl_helper", "ready", {
		"helper_present": 1,
		"requests_dir": REQUESTS_DIR,
		"results_dir": RESULTS_DIR,
	})
	_write_helper_status("ready")


func log_event(level: String, source: String, message: String, detail: Dictionary = {}) -> void:
	var normalized_level := level.strip_edges().to_lower()
	if normalized_level == "":
		normalized_level = "info"
	var normalized_source := source.strip_edges()
	if normalized_source == "":
		normalized_source = "runtime.game"
	elif not normalized_source.begins_with("runtime."):
		normalized_source = "runtime." + normalized_source
	var entry := {
		"time": Time.get_datetime_string_from_system(true),
		"level": normalized_level,
		"source": normalized_source,
		"message": message,
		"detail": _json_safe(detail),
	}
	_append_log(entry)


func info(source: String, message: String, detail: Dictionary = {}) -> void:
	log_event("info", source, message, detail)


func warn(source: String, message: String, detail: Dictionary = {}) -> void:
	log_event("warn", source, message, detail)


func error(source: String, message: String, detail: Dictionary = {}) -> void:
	log_event("error", source, message, detail)


func probe(source: String, message: String, detail: Dictionary = {}) -> void:
	log_event("info", source, message, detail)


func clear_logs() -> void:
	_remove_file(LOG_PATH)


func _process(delta: float) -> void:
	heartbeat_elapsed += delta
	if heartbeat_elapsed >= 1.0:
		heartbeat_elapsed = 0.0
		_write_helper_status("heartbeat")
	_load_requests()
	_process_requests()


func _write_helper_status(message: String) -> void:
	var file := FileAccess.open(STATUS_PATH, FileAccess.WRITE)
	if file == null:
		return
	file.store_string(JSON.stringify({
		"time": Time.get_datetime_string_from_system(true),
		"source": "runtime.gdctl_helper",
		"message": message,
		"detail": {
			"helper_present": 1,
			"active_requests": active.size(),
		},
	}))
	file.close()


func _load_requests() -> void:
	var dir := DirAccess.open(REQUESTS_DIR)
	if dir == null:
		return
	dir.list_dir_begin()
	while true:
		var file_name := dir.get_next()
		if file_name == "":
			break
		if dir.current_is_dir() or not file_name.ends_with(".json"):
			continue
		var path := REQUESTS_DIR + file_name
		var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(path))
		if typeof(parsed) != TYPE_DICTIONARY:
			_write_error(file_name.get_basename(), "Invalid runtime request JSON")
			_remove_file(path)
			continue
		var request: Dictionary = parsed
		var id := String(request.get("id", file_name.get_basename()))
		if not active.has(id):
			active[id] = {
				"id": id,
				"path": path,
				"kind": String(request.get("kind", "screenshot")),
				"request": request,
				"frames": int(request.get("frames", 2)),
				"step_index": 0,
				"wait_remaining": 0.0,
				"pending_release": {},
				"started_ticks": Time.get_ticks_msec(),
			}
	dir.list_dir_end()


func _process_requests() -> void:
	for id in active.keys():
		var request: Dictionary = active[id]
		var frames: int = int(request.get("frames", 0))
		if frames > 0:
			request["frames"] = frames - 1
			active[id] = request
			continue
		var kind := String(request.get("kind", "screenshot"))
		if kind == "input":
			var done := _process_input(id, request)
			if done:
				_remove_file(String(request.get("path", "")))
				active.erase(id)
			else:
				active[id] = request
			continue
		if kind == "raycast":
			_capture_raycast(id)
			_remove_file(String(request.get("path", "")))
			active.erase(id)
			continue
		if kind == "node_probe":
			_probe_node(id, request.get("request", {}))
			_remove_file(String(request.get("path", "")))
			active.erase(id)
			continue
		if kind == "instantiate":
			_instantiate(id, request.get("request", {}))
			_remove_file(String(request.get("path", "")))
			active.erase(id)
			continue
		if kind == "scene_reload":
			_scene_reload(id)
			_remove_file(String(request.get("path", "")))
			active.erase(id)
			continue
		_capture(id, request.get("request", {}))
		_remove_file(String(request.get("path", "")))
		active.erase(id)


func _capture(id: String, request: Dictionary = {}) -> void:
	var viewport_path := String(request.get("viewport_path", ""))
	var texture: ViewportTexture = null
	if viewport_path != "":
		var node := get_node_or_null(NodePath(viewport_path))
		if node == null:
			_write_error(id, "Viewport node not found: " + viewport_path)
			return
		if not node is SubViewport:
			_write_error(id, "Node is not a SubViewport: " + viewport_path)
			return
		texture = (node as SubViewport).get_texture()
	else:
		texture = get_viewport().get_texture()
	if texture == null:
		_write_error(id, "Game viewport texture is unavailable")
		return
	var image: Image = texture.get_image()
	if image == null or image.is_empty():
		_write_error(id, "Game viewport image is empty")
		return
	var png: PackedByteArray = image.save_png_to_buffer()
	if png.is_empty():
		_write_error(id, "Could not encode game viewport image as PNG")
		return
	var result := {
		"ok": true,
		"format": "png",
		"source": "game",
		"width": image.get_width(),
		"height": image.get_height(),
		"content_base64": Marshalls.raw_to_base64(png),
	}
	_write_result(id, result)


func _instantiate(id: String, request: Dictionary) -> void:
	var scene_path := String(request.get("scene", ""))
	var parent_path := String(request.get("parent", ""))
	var node_name := String(request.get("name", ""))
	if scene_path == "" or parent_path == "":
		_write_error(id, "Instantiate requires scene and parent")
		return
	var packed: PackedScene = load(scene_path)
	if packed == null:
		_write_error(id, "Could not load scene: " + scene_path)
		return
	var parent := get_node_or_null(NodePath(parent_path))
	if parent == null:
		_write_error(id, "Parent node not found: " + parent_path)
		return
	var instance := packed.instantiate()
	if node_name != "":
		instance.name = node_name
	parent.add_child(instance)
	_write_result(id, {
		"ok": true,
		"source": "game",
		"scene": scene_path,
		"parent": parent_path,
		"name": String(instance.name),
		"path": str(instance.get_path()),
		"instanced": true,
	})


func _scene_reload(id: String) -> void:
	var scene_path := get_tree().current_scene.scene_file_path if get_tree().current_scene != null else ""
	_write_result(id, {
		"ok": true,
		"source": "game",
		"scene": scene_path,
		"reloaded": true,
	})
	get_tree().reload_current_scene()


func _capture_raycast(id: String) -> void:
	var viewport := get_viewport()
	if viewport == null:
		_write_error(id, "Game viewport is unavailable")
		return
	var camera: Camera3D = viewport.get_camera_3d()
	if camera == null:
		_write_error(id, "No active Camera3D found in game viewport")
		return
	var center := viewport.get_visible_rect().size * 0.5
	var ray_origin: Vector3 = camera.project_ray_origin(center)
	var ray_direction: Vector3 = camera.project_ray_normal(center)
	var ray_length: float = 1000.0
	var space_state: PhysicsDirectSpaceState3D = viewport.find_world_3d().direct_space_state
	if space_state == null:
		_write_error(id, "Could not access PhysicsDirectSpaceState3D")
		return
	var query := PhysicsRayQueryParameters3D.create(ray_origin, ray_origin + ray_direction * ray_length)
	var result: Dictionary = space_state.intersect_ray(query)
	if result.is_empty():
		_write_result(id, {
			"ok": true,
			"hit": false,
			"camera_path": str(camera.get_path()),
			"ray_origin": [ray_origin.x, ray_origin.y, ray_origin.z],
			"ray_direction": [ray_direction.x, ray_direction.y, ray_direction.z],
		})
		return
	var hit_pos: Vector3 = result.get("position", Vector3.ZERO)
	var hit_normal: Vector3 = result.get("normal", Vector3.ZERO)
	var collider: Object = result.get("collider", null)
	var collider_path := ""
	if collider is Node:
		collider_path = str((collider as Node).get_path())
	var hit_distance: float = ray_origin.distance_to(hit_pos)
	_write_result(id, {
		"ok": true,
		"hit": true,
		"camera_path": str(camera.get_path()),
		"ray_origin": [ray_origin.x, ray_origin.y, ray_origin.z],
		"ray_direction": [ray_direction.x, ray_direction.y, ray_direction.z],
		"hit_collider": collider_path,
		"hit_position": [hit_pos.x, hit_pos.y, hit_pos.z],
		"hit_normal": [hit_normal.x, hit_normal.y, hit_normal.z],
		"hit_distance": hit_distance,
	})


func _probe_node(id: String, request: Dictionary) -> void:
	var path := String(request.get("node_path", ""))
	if path == "":
		_write_error(id, "Node probe requires node_path")
		return
	var node := get_node_or_null(NodePath(path))
	if node == null:
		_write_error(id, "Node not found: " + path)
		return
	var properties_value: Variant = request.get("properties", [])
	if typeof(properties_value) != TYPE_ARRAY:
		_write_error(id, "Node probe properties must be an array")
		return
	var properties: Array = properties_value
	var values := {}
	for property_value in properties:
		var property := String(property_value)
		if property == "":
			continue
		values[property] = _json_safe(node.get(property))
	_write_result(id, {
		"ok": true,
		"source": "game",
		"path": path,
		"type": node.get_class(),
		"properties": values,
	})


func _process_input(id: String, active_request: Dictionary) -> bool:
	var runtime_request: Dictionary = active_request.get("request", {})
	var steps_value: Variant = runtime_request.get("steps", [])
	if typeof(steps_value) != TYPE_ARRAY:
		_write_error(id, "Input request steps must be an array")
		return true
	var steps: Array = steps_value
	var wait_remaining := float(active_request.get("wait_remaining", 0.0))
	if wait_remaining > 0.0:
		wait_remaining -= get_process_delta_time() * 1000.0
		active_request["wait_remaining"] = maxf(0.0, wait_remaining)
		return false
	var pending_release: Dictionary = active_request.get("pending_release", {})
	if not pending_release.is_empty():
		if String(pending_release.get("kind", "")) == "key":
			_send_key(int(pending_release.get("keycode", 0)), false)
		elif String(pending_release.get("kind", "")) == "mouse_button":
			_send_mouse_button(int(pending_release.get("button", 0)), false)
		active_request["pending_release"] = {}
		active_request["step_index"] = int(active_request.get("step_index", 0)) + 1
		return false
	var index := int(active_request.get("step_index", 0))
	if index >= steps.size():
		_write_result(id, {
			"ok": true,
			"source": "game",
			"steps": steps.size(),
			"duration_ms": Time.get_ticks_msec() - int(active_request.get("started_ticks", Time.get_ticks_msec())),
		})
		return true
	var step_value: Variant = steps[index]
	if typeof(step_value) != TYPE_DICTIONARY:
		_write_error(id, "Input step %d must be an object" % index)
		return true
	var step: Dictionary = step_value
	var result := _execute_input_step(step)
	if not bool(result.get("ok", false)):
		_write_error(id, "Input step %d failed: %s" % [index, String(result.get("error", "invalid input step"))])
		return true
	active_request["wait_remaining"] = float(result.get("wait_ms", 0.0))
	if result.has("pending_release"):
		active_request["pending_release"] = result["pending_release"]
	else:
		active_request["step_index"] = index + 1
	return false


func _execute_input_step(step: Dictionary) -> Dictionary:
	var step_type := String(step.get("type", ""))
	if step_type == "wait":
		return {"ok": true, "wait_ms": float(step.get("ms", 0))}
	if step_type == "key":
		return _execute_key_step(step)
	if step_type == "mouse_button":
		return _execute_mouse_button_step(step)
	if step_type == "mouse_motion":
		var relative := _array_to_vector2(step.get("relative", [0, 0]))
		var event := InputEventMouseMotion.new()
		event.relative = relative
		Input.parse_input_event(event)
		return {"ok": true}
	return {"ok": false, "error": "Unsupported input step type: " + step_type}


func _execute_key_step(step: Dictionary) -> Dictionary:
	var key_name := String(step.get("key", ""))
	var keycode := OS.find_keycode_from_string(key_name)
	if keycode == KEY_NONE:
		return {"ok": false, "error": "Unknown key: " + key_name}
	var action := String(step.get("action", "tap"))
	var duration := float(step.get("duration_ms", 50))
	if action == "tap":
		_send_key(keycode, true)
		return {"ok": true, "wait_ms": duration, "pending_release": {"kind": "key", "keycode": keycode}}
	if action == "press":
		_send_key(keycode, true)
		return {"ok": true}
	if action == "release":
		_send_key(keycode, false)
		return {"ok": true}
	return {"ok": false, "error": "Unsupported key action: " + action}


func _execute_mouse_button_step(step: Dictionary) -> Dictionary:
	var button := _mouse_button(String(step.get("button", "")))
	if button == 0:
		return {"ok": false, "error": "Unknown mouse button: " + String(step.get("button", ""))}
	var action := String(step.get("action", "tap"))
	var duration := float(step.get("duration_ms", 50))
	if action == "tap":
		_send_mouse_button(button, true)
		return {"ok": true, "wait_ms": duration, "pending_release": {"kind": "mouse_button", "button": button}}
	if action == "press":
		_send_mouse_button(button, true)
		return {"ok": true}
	if action == "release":
		_send_mouse_button(button, false)
		return {"ok": true}
	return {"ok": false, "error": "Unsupported mouse button action: " + action}


func _send_key(keycode: int, pressed: bool) -> void:
	var event := InputEventKey.new()
	event.keycode = keycode
	event.physical_keycode = keycode
	event.pressed = pressed
	Input.parse_input_event(event)


func _send_mouse_button(button: int, pressed: bool) -> void:
	var event := InputEventMouseButton.new()
	event.button_index = button
	event.pressed = pressed
	Input.parse_input_event(event)


func _mouse_button(button: String) -> int:
	match button.to_lower():
		"left", "1":
			return MOUSE_BUTTON_LEFT
		"right", "2":
			return MOUSE_BUTTON_RIGHT
		"middle", "3":
			return MOUSE_BUTTON_MIDDLE
	return 0


func _array_to_vector2(raw: Variant) -> Vector2:
	if typeof(raw) != TYPE_ARRAY:
		return Vector2.ZERO
	var items: Array = raw
	if items.size() != 2:
		return Vector2.ZERO
	return Vector2(float(items[0]), float(items[1]))


func _write_error(id: String, message: String) -> void:
	_write_result(id, {
		"ok": false,
		"source": "game",
		"error": message,
	})


func _write_result(id: String, result: Dictionary) -> void:
	_ensure_dir(RESULTS_DIR)
	var file := FileAccess.open(RESULTS_DIR + id + ".json", FileAccess.WRITE)
	if file == null:
		return
	file.store_string(JSON.stringify(result))
	file.close()


func _append_log(entry: Dictionary) -> void:
	_ensure_dir(LOGS_DIR)
	var file := FileAccess.open(LOG_PATH, FileAccess.READ_WRITE)
	if file == null:
		file = FileAccess.open(LOG_PATH, FileAccess.WRITE)
	if file == null:
		return
	file.seek_end()
	file.store_line(JSON.stringify(entry))
	file.close()


func _json_safe(value: Variant) -> Variant:
	match typeof(value):
		TYPE_DICTIONARY:
			var out := {}
			for key in value.keys():
				out[str(key)] = _json_safe(value[key])
			return out
		TYPE_ARRAY:
			var out: Array = []
			for item in value:
				out.append(_json_safe(item))
			return out
		TYPE_VECTOR2:
			return [value.x, value.y]
		TYPE_VECTOR3:
			return [value.x, value.y, value.z]
		TYPE_VECTOR4:
			return [value.x, value.y, value.z, value.w]
		TYPE_COLOR:
			return [value.r, value.g, value.b, value.a]
		TYPE_NODE_PATH:
			return str(value)
		TYPE_OBJECT:
			if value == null:
				return null
			if value is Node:
				return (value as Node).get_path()
			return str(value)
		_:
			return value


func _ensure_dir(path: String) -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(path))


func _remove_file(path: String) -> void:
	if path != "" and FileAccess.file_exists(path):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(path))
