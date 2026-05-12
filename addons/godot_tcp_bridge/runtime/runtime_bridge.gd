extends Node

const REQUESTS_DIR := "res://.gdctl_runtime/requests/"
const RESULTS_DIR := "res://.gdctl_runtime/results/"

var active: Dictionary = {}


func _ready() -> void:
	_ensure_dir(REQUESTS_DIR)
	_ensure_dir(RESULTS_DIR)


func _process(_delta: float) -> void:
	_load_requests()
	_process_requests()


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
				"frames": int(request.get("frames", 2)),
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
		_capture(id)
		_remove_file(String(request.get("path", "")))
		active.erase(id)


func _capture(id: String) -> void:
	var texture := get_viewport().get_texture()
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


func _ensure_dir(path: String) -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(path))


func _remove_file(path: String) -> void:
	if path != "" and FileAccess.file_exists(path):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(path))
