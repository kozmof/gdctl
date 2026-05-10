@tool
extends RefCounted


func handle_start(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "run.start", "Run start requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(503, String(checked["request_id"]), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var scene: String = String(params.get("scene", ""))
	var main: bool = bool(params.get("main", false))
	var clear_logs: bool = bool(params.get("clear_logs", true))
	var editor_interface = context["editor_plugin"].get_editor_interface()
	if clear_logs:
		context["log_buffer"].clear()
		context["log"].call("info", "run.logs", "Runtime logs cleared", {})
	if editor_interface.is_playing_scene():
		return context["bridge_error"].call(409, request_id, "RUN_ALREADY_PLAYING", "A scene is already running", {"playing_scene": editor_interface.get_playing_scene()})
	if scene != "":
		if not ResourceLoader.exists(scene):
			return context["bridge_error"].call(404, request_id, "RUN_SCENE_NOT_FOUND", "Scene does not exist", {"scene": scene})
		editor_interface.play_custom_scene(scene)
	elif main:
		editor_interface.play_main_scene()
	else:
		editor_interface.play_current_scene()
	context["log"].call("info", "run.start", "Started editor run", {"scene": scene, "main": main})
	return context["bridge_ok"].call(request_id, {
		"running": true,
		"scene": scene,
		"playing_scene": editor_interface.get_playing_scene(),
	})


func handle_status(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "run.status", "Run status requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(503, String(checked["request_id"]), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	var editor_interface = context["editor_plugin"].get_editor_interface()
	return context["bridge_ok"].call(String(checked["request_id"]), {
		"running": editor_interface.is_playing_scene(),
		"playing_scene": editor_interface.get_playing_scene(),
	})


func handle_stop(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "run.stop", "Run stop requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(503, String(checked["request_id"]), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})
	var editor_interface = context["editor_plugin"].get_editor_interface()
	var was_running := editor_interface.is_playing_scene()
	var playing_scene := editor_interface.get_playing_scene()
	if was_running:
		editor_interface.stop_playing_scene()
		context["log"].call("info", "run.stop", "Stopped editor run", {"playing_scene": playing_scene})
	return context["bridge_ok"].call(String(checked["request_id"]), {
		"stopped": was_running,
		"running": false,
		"playing_scene": playing_scene,
	})


func handle_logs(request: Dictionary, context: Dictionary) -> Dictionary:
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, "", "UNAUTHORIZED", "Run logs require bearer token", {})
	var entries: Array[Dictionary] = []
	for entry in context["log_buffer"].list():
		var source := String(entry.get("source", ""))
		if source.begins_with("run.") or source.begins_with("runtime."):
			entries.append(entry)
	return context["http_json"].call(200, {"ok": true, "entries": entries})
