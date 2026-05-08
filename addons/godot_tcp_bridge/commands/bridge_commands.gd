@tool
extends RefCounted


func handle_ping(_request: Dictionary, context: Dictionary) -> Dictionary:
	var root: Node = context["edited_scene_root"].call()
	return context["http_json"].call(200, {
		"ok": true,
		"service": "godot-bridge",
		"engine": "Godot",
		"engine_version": Engine.get_version_info().get("string", ""),
		"plugin_version": String(context["plugin_version"]),
		"project_name": ProjectSettings.get_setting("application/config/name", ""),
		"project_path": ProjectSettings.globalize_path("res://"),
		"scene_open": root != null,
		"auth_enabled": bool(context["auth_enabled"]),
		"host": String(context["host"]),
		"port": int(context["port"]),
		"protocol_version": String(context["protocol_version"]),
		"capabilities": context["capabilities"],
	})
