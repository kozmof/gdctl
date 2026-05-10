@tool
extends RefCounted


func handle_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "import.set", "Import set requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var asset_path: String = String(params.get("path", ""))
	if asset_path == "" or not asset_path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "IMPORT_PATH_INVALID", "Asset path must be a res:// path", {"path": asset_path})
	if asset_path.find("..") != -1:
		return context["bridge_error"].call(400, request_id, "IMPORT_PATH_INVALID", "Asset path must not contain ..", {"path": asset_path})

	var import_path: String = asset_path + ".import"
	if not FileAccess.file_exists(import_path):
		return context["bridge_error"].call(404, request_id, "IMPORT_FILE_NOT_FOUND", "Import file does not exist for asset", {"path": asset_path, "import_path": import_path})

	var import_params_value: Variant = params.get("params", {})
	if typeof(import_params_value) != TYPE_DICTIONARY:
		return context["bridge_error"].call(400, request_id, "IMPORT_PARAMS_INVALID", "Params must be a dictionary", {})
	var import_params: Dictionary = import_params_value

	var cfg := ConfigFile.new()
	var load_err: Error = cfg.load(ProjectSettings.globalize_path(import_path))
	if load_err != OK:
		return context["bridge_error"].call(500, request_id, "IMPORT_LOAD_FAILED", "Could not load import file", {"path": import_path, "error": error_string(load_err)})

	var applied: int = 0
	for key_value in import_params.keys():
		var key: String = String(key_value)
		cfg.set_value("params", key, import_params[key_value])
		applied += 1

	var save_err: Error = cfg.save(ProjectSettings.globalize_path(import_path))
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "IMPORT_SAVE_FAILED", "Could not save import file", {"path": import_path, "error": error_string(save_err)})

	if context["editor_plugin_available"].call():
		context["reimport_files"].call(PackedStringArray([asset_path]))

	return context["bridge_ok"].call(request_id, {
		"path": asset_path,
		"params": applied,
		"applied": true,
	})
