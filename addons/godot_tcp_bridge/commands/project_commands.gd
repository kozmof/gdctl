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
