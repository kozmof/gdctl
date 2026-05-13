@tool
extends RefCounted


func handle_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "resource.create", "Resource create requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var resource_path: String = String(params.get("path", ""))
	var type_name: String = String(params.get("type", ""))
	var script_path: String = String(params.get("script", ""))
	if resource_path == "" or not resource_path.begins_with("res://") or not resource_path.ends_with(".tres"):
		return context["bridge_error"].call(400, request_id, "RESOURCE_PATH_INVALID", "Resource path must be a res:// .tres path", {"path": resource_path})
	if type_name == "" and script_path == "":
		return context["bridge_error"].call(400, request_id, "RESOURCE_TYPE_REQUIRED", "Resource type or script is required", {})

	var resource: Resource = null
	if script_path != "":
		if not script_path.begins_with("res://") or not script_path.ends_with(".gd"):
			return context["bridge_error"].call(400, request_id, "RESOURCE_SCRIPT_PATH_INVALID", "Resource script must be a res:// .gd path", {"script": script_path})
		if not ResourceLoader.exists(script_path):
			return context["bridge_error"].call(404, request_id, "RESOURCE_SCRIPT_NOT_FOUND", "Resource script does not exist", {"script": script_path})
		var script_resource: Resource = ResourceLoader.load(script_path, "", ResourceLoader.CACHE_MODE_REPLACE)
		if script_resource == null or not (script_resource is Script):
			return context["bridge_error"].call(400, request_id, "RESOURCE_SCRIPT_INVALID", "Resource script could not be loaded as a Script", {"script": script_path})
		var script: Script = script_resource as Script
		var instance: Variant = script.new()
		resource = instance as Resource
		if resource == null:
			return context["bridge_error"].call(400, request_id, "RESOURCE_SCRIPT_INVALID", "Resource script must extend Resource", {"script": script_path})
		if type_name == "":
			type_name = resource.get_class()
	else:
		if not ClassDB.class_exists(type_name):
			return context["bridge_error"].call(400, request_id, "RESOURCE_TYPE_UNKNOWN", "Unknown Godot class", {
				"type": type_name,
				"hint": "For custom Resource scripts, use resource create --script res://path/to/resource.gd"
			})
		if not ClassDB.is_parent_class(type_name, "Resource"):
			return context["bridge_error"].call(400, request_id, "RESOURCE_TYPE_INVALID", "Type must be a Resource subclass", {"type": type_name})
		resource = ClassDB.instantiate(type_name) as Resource
		if resource == null:
			return context["bridge_error"].call(500, request_id, "RESOURCE_INSTANTIATE_FAILED", "Could not instantiate resource", {"type": type_name})

	var props_value: Variant = params.get("props", {})
	if typeof(props_value) == TYPE_DICTIONARY:
		var props: Dictionary = props_value
		for prop_name_value in props.keys():
			var prop_name: String = String(prop_name_value)
			var decoded: Dictionary = context["typed_values"].decode(props[prop_name_value])
			if not bool(decoded.get("ok", false)):
				return context["bridge_error"].call(400, request_id, "RESOURCE_PROP_INVALID", "Invalid typed value for property", {"property": prop_name, "error": String(decoded.get("error", ""))})
			resource.set(prop_name, decoded["value"])

	var shader_params_value: Variant = params.get("shader_params", {})
	if typeof(shader_params_value) == TYPE_DICTIONARY:
		var shader_params: Dictionary = shader_params_value
		if not shader_params.is_empty():
			if not resource.has_method("set_shader_parameter"):
				return context["bridge_error"].call(400, request_id, "RESOURCE_TYPE_INVALID", "Resource type does not support shader parameters", {"type": type_name})
			for param_name_value in shader_params.keys():
				var param_name: String = String(param_name_value)
				var param_path: String = String(shader_params[param_name_value])
				if param_path == "" or not param_path.begins_with("res://"):
					return context["bridge_error"].call(400, request_id, "RESOURCE_PATH_INVALID", "Shader param resource must be a res:// path", {"name": param_name, "path": param_path})
				if not FileAccess.file_exists(param_path):
					return context["bridge_error"].call(404, request_id, "RESOURCE_NOT_FOUND", "Shader param resource does not exist", {"name": param_name, "path": param_path})
				var param_resource: Resource = ResourceLoader.load(param_path, "", ResourceLoader.CACHE_MODE_REPLACE)
				if param_resource == null:
					return context["bridge_error"].call(500, request_id, "RESOURCE_LOAD_FAILED", "Could not load shader param resource", {"name": param_name, "path": param_path})
				resource.set_shader_parameter(param_name, param_resource)

	var dir_err: Error = _ensure_resource_dir(resource_path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "RESOURCE_DIR_FAILED", "Could not create resource directory", {"path": resource_path, "error": error_string(dir_err)})

	var save_err: Error = ResourceSaver.save(resource, resource_path)
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "RESOURCE_SAVE_FAILED", "Could not save resource", {"path": resource_path, "error": error_string(save_err)})

	return context["bridge_ok"].call(request_id, {
		"path": resource_path,
		"type": type_name,
		"script": script_path,
		"created": true,
	})


func handle_list(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "resource.list", "Resource list requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var dir: String = String(params.get("dir", "res://"))
	var recursive: bool = bool(params.get("recursive", true))
	var ext_filter: String = String(params.get("ext", ""))
	if dir == "" or not dir.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "DIR_PATH_INVALID", "Directory must be a res:// path", {"dir": dir})
	if dir.find("..") != -1:
		return context["bridge_error"].call(400, request_id, "DIR_PATH_INVALID", "Directory must not contain ..", {"dir": dir})
	var abs_dir: String = ProjectSettings.globalize_path(dir)
	if not DirAccess.dir_exists_absolute(abs_dir):
		return context["bridge_error"].call(404, request_id, "DIR_NOT_FOUND", "Directory does not exist", {"dir": dir})

	var resources: Array = []
	_collect_resources(dir, resources, recursive, ext_filter)
	return context["bridge_ok"].call(request_id, {
		"dir": dir,
		"resources": resources,
	})


func _collect_resources(dir: String, out: Array, recursive: bool, ext_filter: String) -> void:
	for f: String in DirAccess.get_files_at(dir):
		var keep: bool = false
		if ext_filter != "":
			keep = f.ends_with(ext_filter)
		else:
			keep = f.ends_with(".tres") or f.ends_with(".res")
		if keep:
			out.append(dir.path_join(f))
	if recursive:
		for d: String in DirAccess.get_directories_at(dir):
			_collect_resources(dir.path_join(d), out, true, ext_filter)


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
