@tool
extends RefCounted


func handle_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation.create", "Animation create requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var library_path: String = String(params.get("path", ""))
	var anim_name: String = String(params.get("name", ""))
	var length: float = float(params.get("length", 1.0))
	var loop: bool = bool(params.get("loop", false))
	if not _valid_library_path(library_path):
		return context["bridge_error"].call(400, request_id, "ANIMATION_PATH_INVALID", "Animation library path must be a res:// .tres or .res path", {"path": library_path})
	if anim_name == "" or not anim_name.is_valid_identifier():
		return context["bridge_error"].call(400, request_id, "ANIMATION_NAME_INVALID", "Animation name must be a valid identifier", {"name": anim_name})
	var dir_err: Error = _ensure_dir(library_path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "ANIMATION_DIR_FAILED", "Could not create animation library directory", {"path": library_path, "error": error_string(dir_err)})
	var library: AnimationLibrary
	if ResourceLoader.exists(library_path):
		var res: Resource = ResourceLoader.load(library_path, "", ResourceLoader.CACHE_MODE_IGNORE)
		if not res is AnimationLibrary:
			return context["bridge_error"].call(409, request_id, "ANIMATION_LIBRARY_TYPE_MISMATCH", "Resource at path is not an AnimationLibrary", {"path": library_path})
		library = res as AnimationLibrary
	else:
		library = AnimationLibrary.new()
	var animation := Animation.new()
	animation.length = length
	if loop:
		animation.loop_mode = Animation.LOOP_LINEAR
	else:
		animation.loop_mode = Animation.LOOP_NONE
	library.add_animation(anim_name, animation)
	var err: Error = ResourceSaver.save(library, library_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "ANIMATION_SAVE_FAILED", "Could not save animation library", {"path": library_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": library_path, "name": anim_name, "created": true})


func handle_track_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation.track-add", "Animation track-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var library_path: String = String(params.get("path", ""))
	var anim_name: String = String(params.get("animation", ""))
	var node_path: String = String(params.get("node_path", ""))
	var property: String = String(params.get("property", ""))
	if not _valid_library_path(library_path):
		return context["bridge_error"].call(400, request_id, "ANIMATION_PATH_INVALID", "Animation library path must be a res:// .tres or .res path", {"path": library_path})
	if anim_name == "" or node_path == "" or property == "":
		return context["bridge_error"].call(400, request_id, "ANIMATION_PARAMS_MISSING", "animation, node_path, and property are required", {})
	var library_result := _load_library(library_path, anim_name, request_id, context)
	if library_result.has("error"):
		return library_result["error"]
	var animation: Animation = library_result["animation"]
	var track_path := NodePath(node_path + ":" + property)
	var track_idx: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(track_idx, track_path)
	animation.value_track_set_update_mode(track_idx, Animation.UPDATE_DISCRETE)
	var library: AnimationLibrary = library_result["library"]
	var err: Error = ResourceSaver.save(library, library_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "ANIMATION_SAVE_FAILED", "Could not save animation library", {"path": library_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": library_path, "animation": anim_name, "track_idx": track_idx})


func handle_keyframe_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation.keyframe-add", "Animation keyframe-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var library_path: String = String(params.get("path", ""))
	var anim_name: String = String(params.get("animation", ""))
	var track_idx: int = int(params.get("track_idx", -1))
	var time_pos: float = float(params.get("time", 0.0))
	var value: Variant = params.get("value", null)
	if not _valid_library_path(library_path):
		return context["bridge_error"].call(400, request_id, "ANIMATION_PATH_INVALID", "Animation library path must be a res:// .tres or .res path", {"path": library_path})
	if anim_name == "" or track_idx < 0:
		return context["bridge_error"].call(400, request_id, "ANIMATION_PARAMS_MISSING", "animation and track_idx are required", {})
	var library_result := _load_library(library_path, anim_name, request_id, context)
	if library_result.has("error"):
		return library_result["error"]
	var animation: Animation = library_result["animation"]
	if track_idx >= animation.get_track_count():
		return context["bridge_error"].call(400, request_id, "ANIMATION_TRACK_INVALID", "Track index out of range", {"track_idx": track_idx, "track_count": animation.get_track_count()})
	animation.track_insert_key(track_idx, time_pos, value)
	var library: AnimationLibrary = library_result["library"]
	var err: Error = ResourceSaver.save(library, library_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "ANIMATION_SAVE_FAILED", "Could not save animation library", {"path": library_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": library_path, "animation": anim_name, "track_idx": track_idx, "time": time_pos, "added": true})


func handle_length_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation.length-set", "Animation length-set requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var library_path: String = String(params.get("path", ""))
	var anim_name: String = String(params.get("animation", ""))
	var length: float = float(params.get("length", 1.0))
	if not _valid_library_path(library_path):
		return context["bridge_error"].call(400, request_id, "ANIMATION_PATH_INVALID", "Animation library path must be a res:// .tres or .res path", {"path": library_path})
	if length <= 0.0:
		return context["bridge_error"].call(400, request_id, "ANIMATION_LENGTH_INVALID", "Animation length must be positive", {"length": length})
	var library_result := _load_library(library_path, anim_name, request_id, context)
	if library_result.has("error"):
		return library_result["error"]
	var animation: Animation = library_result["animation"]
	animation.length = length
	var library: AnimationLibrary = library_result["library"]
	var err: Error = ResourceSaver.save(library, library_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "ANIMATION_SAVE_FAILED", "Could not save animation library", {"path": library_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": library_path, "name": anim_name, "created": false})


func handle_player_play(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation.player-play", "Animation player-play requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path_str: String = String(params.get("node_path", ""))
	var anim_name: String = String(params.get("animation", ""))
	if node_path_str == "":
		return context["bridge_error"].call(400, request_id, "ANIMATION_NODE_PATH_MISSING", "node_path is required", {})
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})
	var node: Node = context["node_by_path"].call(node_path_str)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node not found", {"path": node_path_str})
	if not node is AnimationPlayer:
		return context["bridge_error"].call(400, request_id, "ANIMATION_NODE_NOT_PLAYER", "Node is not an AnimationPlayer", {"path": node_path_str})
	var player: AnimationPlayer = node as AnimationPlayer
	if anim_name != "":
		if not player.has_animation(anim_name):
			return context["bridge_error"].call(404, request_id, "ANIMATION_NOT_FOUND", "Animation not found on player", {"animation": anim_name, "path": node_path_str})
		player.play(anim_name)
	else:
		player.play()
	return context["bridge_ok"].call(request_id, {"path": node_path_str, "name": anim_name, "created": false})


func _load_library(library_path: String, anim_name: String, request_id: String, context: Dictionary) -> Dictionary:
	if not ResourceLoader.exists(library_path):
		return {"error": context["bridge_error"].call(404, request_id, "ANIMATION_LIBRARY_NOT_FOUND", "Animation library does not exist", {"path": library_path})}
	var res: Resource = ResourceLoader.load(library_path, "", ResourceLoader.CACHE_MODE_IGNORE)
	if not res is AnimationLibrary:
		return {"error": context["bridge_error"].call(409, request_id, "ANIMATION_LIBRARY_TYPE_MISMATCH", "Resource at path is not an AnimationLibrary", {"path": library_path})}
	var library: AnimationLibrary = res as AnimationLibrary
	if not library.has_animation(anim_name):
		return {"error": context["bridge_error"].call(404, request_id, "ANIMATION_NOT_FOUND", "Animation not found in library", {"path": library_path, "animation": anim_name})}
	return {"library": library, "animation": library.get_animation(anim_name)}


func _valid_library_path(path: String) -> bool:
	return path != "" and path.begins_with("res://") and (path.ends_with(".tres") or path.ends_with(".res"))


func _ensure_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
