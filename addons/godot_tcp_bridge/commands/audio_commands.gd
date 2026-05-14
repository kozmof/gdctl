@tool
extends RefCounted


func handle_bus_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "audio.bus-add", "Audio bus-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var bus_name: String = String(params.get("name", ""))
	var if_missing: bool = bool(params.get("if_missing", false))
	if bus_name == "":
		return context["bridge_error"].call(400, request_id, "AUDIO_BUS_NAME_MISSING", "Bus name is required", {})
	if AudioServer.get_bus_index(bus_name) >= 0:
		if if_missing:
			return context["bridge_ok"].call(request_id, {"bus": bus_name, "applied": false, "created": false})
		return context["bridge_error"].call(409, request_id, "AUDIO_BUS_ALREADY_EXISTS", "Audio bus already exists", {"bus": bus_name})
	var idx: int = AudioServer.get_bus_count()
	AudioServer.add_bus(idx)
	AudioServer.set_bus_name(idx, bus_name)
	return context["bridge_ok"].call(request_id, {"bus": bus_name, "applied": true, "created": true})


func handle_bus_volume_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "audio.bus-volume-set", "Audio bus-volume-set requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var bus_name: String = String(params.get("name", ""))
	var volume_db: float = float(params.get("volume_db", 0.0))
	if bus_name == "":
		return context["bridge_error"].call(400, request_id, "AUDIO_BUS_NAME_MISSING", "Bus name is required", {})
	var idx: int = AudioServer.get_bus_index(bus_name)
	if idx < 0:
		return context["bridge_error"].call(404, request_id, "AUDIO_BUS_NOT_FOUND", "Audio bus does not exist", {"bus": bus_name})
	AudioServer.set_bus_volume_db(idx, volume_db)
	return context["bridge_ok"].call(request_id, {"bus": bus_name, "applied": true})


func handle_bus_effect_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "audio.bus-effect-add", "Audio bus-effect-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var bus_name: String = String(params.get("name", ""))
	var effect_type: String = String(params.get("effect_type", ""))
	if bus_name == "" or effect_type == "":
		return context["bridge_error"].call(400, request_id, "AUDIO_EFFECT_PARAMS_MISSING", "Bus name and effect_type are required", {})
	var idx: int = AudioServer.get_bus_index(bus_name)
	if idx < 0:
		return context["bridge_error"].call(404, request_id, "AUDIO_BUS_NOT_FOUND", "Audio bus does not exist", {"bus": bus_name})
	if not ClassDB.class_exists(effect_type) or not ClassDB.is_parent_class(effect_type, "AudioEffect"):
		return context["bridge_error"].call(400, request_id, "AUDIO_EFFECT_TYPE_INVALID", "effect_type must be a valid AudioEffect subclass", {"effect_type": effect_type})
	var effect: AudioEffect = ClassDB.instantiate(effect_type)
	if effect == null:
		return context["bridge_error"].call(500, request_id, "AUDIO_EFFECT_INSTANTIATE_FAILED", "Could not instantiate audio effect", {"effect_type": effect_type})
	AudioServer.add_bus_effect(idx, effect)
	return context["bridge_ok"].call(request_id, {"bus": bus_name, "applied": true})


func handle_listener_make_current(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "audio.listener-make-current", "Audio listener-make-current requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path: String = String(params.get("path", ""))
	if node_path == "":
		return context["bridge_error"].call(400, request_id, "AUDIO_LISTENER_PATH_MISSING", "path is required", {})
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})
	var node: Node = context["node_by_path"].call(node_path)
	if node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node not found", {"path": node_path})
	if node is AudioListener3D:
		var listener3d: AudioListener3D = node as AudioListener3D
		listener3d.make_current()
		context["mark_scene_dirty"].call()
		return context["bridge_ok"].call(request_id, {"path": node_path, "type": "AudioListener3D", "applied": true})
	if node is AudioListener2D:
		var listener2d: AudioListener2D = node as AudioListener2D
		listener2d.make_current()
		context["mark_scene_dirty"].call()
		return context["bridge_ok"].call(request_id, {"path": node_path, "type": "AudioListener2D", "applied": true})
	return context["bridge_error"].call(400, request_id, "AUDIO_LISTENER_NODE_INVALID", "Node is not an AudioListener3D or AudioListener2D", {"path": node_path})
