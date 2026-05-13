@tool
extends RefCounted


func handle_setting_get(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "project.setting_get", "Project setting read requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var key: String = String(params.get("key", ""))
	if key == "":
		return context["bridge_error"].call(400, request_id, "SETTING_KEY_INVALID", "Setting key is required", {})
	if not ProjectSettings.has_setting(key):
		return context["bridge_error"].call(404, request_id, "SETTING_NOT_FOUND", "Project setting does not exist", {"key": key})
	var typed_values: RefCounted = context["typed_values"]
	return context["bridge_ok"].call(request_id, {
		"key": key,
		"value": typed_values.encode(ProjectSettings.get_setting(key)),
	})


func handle_setting_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "project.setting_set", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var key: String = String(params.get("key", ""))
	if key == "":
		return context["bridge_error"].call(400, request_id, "SETTING_KEY_INVALID", "Setting key is required", {})
	if not params.has("value"):
		return context["bridge_error"].call(400, request_id, "VALUE_MISSING", "Typed value is required", {})
	var typed_values: RefCounted = context["typed_values"]
	var decoded: Dictionary = typed_values.decode(params.get("value"))
	if not bool(decoded.get("ok", false)):
		return context["bridge_error"].call(400, request_id, "VALUE_INVALID", String(decoded.get("error", "Invalid typed value")), {})
	ProjectSettings.set_setting(key, decoded.get("value"))
	var save_err: Error = ProjectSettings.save()
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "SETTING_SAVE_FAILED", "Failed to save project settings", {"error": error_string(save_err)})
	return context["bridge_ok"].call(request_id, {
		"key": key,
		"value": typed_values.encode(ProjectSettings.get_setting(key)),
		"set": true,
	})


func handle_autoload_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "autoload.add", "Autoload add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var name: String = String(params.get("name", ""))
	var path: String = String(params.get("path", ""))
	var valid_name_error: String = _validate_autoload_name(name)
	if valid_name_error != "":
		return context["bridge_error"].call(400, request_id, "AUTOLOAD_NAME_INVALID", valid_name_error, {"name": name})
	if path == "" or not path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "AUTOLOAD_PATH_INVALID", "Autoload path must be a res:// path", {"path": path})
	if not ResourceLoader.exists(path):
		return context["bridge_error"].call(404, request_id, "AUTOLOAD_RESOURCE_NOT_FOUND", "Autoload resource does not exist", {"path": path})
	var key: String = "autoload/" + name
	ProjectSettings.set_setting(key, "*" + path)
	var save_err: Error = ProjectSettings.save()
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "AUTOLOAD_SAVE_FAILED", "Failed to save project settings", {"error": error_string(save_err)})
	return context["bridge_ok"].call(request_id, {
		"name": name,
		"path": path,
		"key": key,
		"added": true,
	})


func handle_autoload_remove(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "autoload.remove", "Autoload remove requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var name: String = String(params.get("name", ""))
	var valid_name_error: String = _validate_autoload_name(name)
	if valid_name_error != "":
		return context["bridge_error"].call(400, request_id, "AUTOLOAD_NAME_INVALID", valid_name_error, {"name": name})
	var key: String = "autoload/" + name
	if not ProjectSettings.has_setting(key):
		return context["bridge_error"].call(404, request_id, "AUTOLOAD_NOT_FOUND", "Autoload does not exist", {"name": name})
	var old_path: String = _autoload_path(ProjectSettings.get_setting(key))
	ProjectSettings.clear(key)
	var save_err: Error = ProjectSettings.save()
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "AUTOLOAD_SAVE_FAILED", "Failed to save project settings", {"error": error_string(save_err)})
	return context["bridge_ok"].call(request_id, {
		"name": name,
		"path": old_path,
		"key": key,
		"removed": true,
	})


func handle_autoload_list(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "autoload.list", "Autoload list requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var request_id: String = String(checked["request_id"])
	var autoloads: Array = []
	for property: Dictionary in ProjectSettings.get_property_list():
		var key: String = String(property.get("name", ""))
		if not key.begins_with("autoload/"):
			continue
		var name: String = key.trim_prefix("autoload/")
		autoloads.append({
			"name": name,
			"path": _autoload_path(ProjectSettings.get_setting(key)),
			"key": key,
		})
	autoloads.sort_custom(func(a: Dictionary, b: Dictionary) -> bool: return String(a["name"]) < String(b["name"]))
	return context["bridge_ok"].call(request_id, {
		"autoloads": autoloads,
	})


func _validate_autoload_name(name: String) -> String:
	if name == "":
		return "Autoload name is required"
	if not name.is_valid_identifier():
		return "Autoload name must be a valid GDScript identifier"
	return ""


func _autoload_path(value: Variant) -> String:
	var path: String = String(value)
	if path.begins_with("*"):
		return path.substr(1)
	return path
