@tool
extends RefCounted


func handle_write(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "material.write", "Material write requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var material_path: String = String(params.get("path", ""))
	var shader_path: String = String(params.get("shader", ""))
	var path_error: Dictionary = _validate_material_path(material_path, context, request_id)
	if not path_error.is_empty():
		return path_error
	if shader_path == "" or not shader_path.begins_with("res://") or not shader_path.ends_with(".gdshader"):
		return context["bridge_error"].call(400, request_id, "SHADER_PATH_INVALID", "Shader path must be a res:// .gdshader path", {"path": shader_path})
	if not FileAccess.file_exists(shader_path):
		return context["bridge_error"].call(404, request_id, "SHADER_NOT_FOUND", "Shader does not exist", {"path": shader_path})

	var shader: Shader = ResourceLoader.load(shader_path, "Shader", ResourceLoader.CACHE_MODE_REPLACE) as Shader
	if shader == null:
		return context["bridge_error"].call(500, request_id, "SHADER_LOAD_FAILED", "Could not load shader resource", {"path": shader_path})
	var dir_err: Error = _ensure_resource_dir(material_path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "MATERIAL_DIR_FAILED", "Could not create material directory", {"path": material_path, "error": error_string(dir_err)})

	var material := ShaderMaterial.new()
	material.shader = shader
	var save_err: Error = ResourceSaver.save(material, material_path)
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "MATERIAL_SAVE_FAILED", "Could not save material", {"path": material_path, "error": error_string(save_err)})

	return context["bridge_ok"].call(request_id, {
		"path": material_path,
		"shader": shader_path,
		"written": true,
	})


func _validate_material_path(material_path: String, context: Dictionary, request_id: String) -> Dictionary:
	if material_path == "" or not material_path.begins_with("res://") or not material_path.ends_with(".tres"):
		return context["bridge_error"].call(400, request_id, "MATERIAL_PATH_INVALID", "Material path must be a res:// .tres path", {"path": material_path})
	return {}


func _ensure_resource_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
