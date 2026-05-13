@tool
extends RefCounted


func handle_tileset_create(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "tilemap.tileset-create", "TileSet create requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var tileset_path: String = String(params.get("path", ""))
	var tile_width: int = int(params.get("tile_width", 16))
	var tile_height: int = int(params.get("tile_height", 16))
	var force: bool = bool(params.get("force", false))
	if not _valid_res_path(tileset_path):
		return context["bridge_error"].call(400, request_id, "TILESET_PATH_INVALID", "TileSet path must be a res:// path", {"path": tileset_path})
	if tile_width <= 0 or tile_height <= 0:
		return context["bridge_error"].call(400, request_id, "TILESET_SIZE_INVALID", "tile_width and tile_height must be positive", {"tile_width": tile_width, "tile_height": tile_height})
	if ResourceLoader.exists(tileset_path) and not force:
		return context["bridge_error"].call(409, request_id, "TILESET_ALREADY_EXISTS", "TileSet already exists", {"path": tileset_path})
	var dir_err: Error = _ensure_dir(tileset_path)
	if dir_err != OK:
		return context["bridge_error"].call(500, request_id, "TILESET_DIR_FAILED", "Could not create TileSet directory", {"path": tileset_path, "error": error_string(dir_err)})
	var tileset := TileSet.new()
	tileset.tile_size = Vector2i(tile_width, tile_height)
	var err: Error = ResourceSaver.save(tileset, tileset_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "TILESET_SAVE_FAILED", "Could not save TileSet", {"path": tileset_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": tileset_path, "created": true})


func handle_source_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "tilemap.source-add", "TileSet source-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var tileset_path: String = String(params.get("path", ""))
	var texture_path: String = String(params.get("texture", ""))
	var tile_width: int = int(params.get("tile_width", 16))
	var tile_height: int = int(params.get("tile_height", 16))
	if not _valid_res_path(tileset_path):
		return context["bridge_error"].call(400, request_id, "TILESET_PATH_INVALID", "TileSet path must be a res:// path", {"path": tileset_path})
	if texture_path == "" or not texture_path.begins_with("res://"):
		return context["bridge_error"].call(400, request_id, "TILESET_TEXTURE_INVALID", "Texture path must be a res:// path", {"texture": texture_path})
	var tileset: TileSet = _load_tileset(tileset_path, request_id, context)
	if tileset == null:
		return context["bridge_error"].call(404, request_id, "TILESET_NOT_FOUND", "TileSet does not exist", {"path": tileset_path})
	if not ResourceLoader.exists(texture_path):
		return context["bridge_error"].call(404, request_id, "TEXTURE_NOT_FOUND", "Texture does not exist", {"texture": texture_path})
	var texture: Texture2D = ResourceLoader.load(texture_path)
	if texture == null:
		return context["bridge_error"].call(400, request_id, "TEXTURE_LOAD_FAILED", "Could not load texture", {"texture": texture_path})
	var source := TileSetAtlasSource.new()
	source.texture = texture
	source.texture_region_size = Vector2i(tile_width, tile_height)
	tileset.add_source(source)
	var err: Error = ResourceSaver.save(tileset, tileset_path)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "TILESET_SAVE_FAILED", "Could not save TileSet", {"path": tileset_path, "error": error_string(err)})
	return context["bridge_ok"].call(request_id, {"path": tileset_path, "created": false})


func handle_cell_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "tilemap.cell-set", "TileMap cell-set requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path_str: String = String(params.get("node", ""))
	var layer: int = int(params.get("layer", 0))
	var x: int = int(params.get("x", 0))
	var y: int = int(params.get("y", 0))
	var source_id: int = int(params.get("source_id", 0))
	var atlas_x: int = int(params.get("atlas_x", 0))
	var atlas_y: int = int(params.get("atlas_y", 0))
	var node_result := _get_tilemap_node(node_path_str, request_id, context)
	if node_result.has("error"):
		return node_result["error"]
	var tilemap: TileMap = node_result["node"]
	if layer < 0 or layer >= tilemap.get_layers_count():
		return context["bridge_error"].call(400, request_id, "TILEMAP_LAYER_INVALID", "Layer index out of range", {"layer": layer, "layers": tilemap.get_layers_count()})
	tilemap.set_cell(layer, Vector2i(x, y), source_id, Vector2i(atlas_x, atlas_y))
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"node": node_path_str, "layer": layer, "x": x, "y": y, "applied": true})


func handle_cell_clear(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "tilemap.cell-clear", "TileMap cell-clear requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var node_path_str: String = String(params.get("node", ""))
	var layer: int = int(params.get("layer", 0))
	var x: int = int(params.get("x", 0))
	var y: int = int(params.get("y", 0))
	var node_result := _get_tilemap_node(node_path_str, request_id, context)
	if node_result.has("error"):
		return node_result["error"]
	var tilemap: TileMap = node_result["node"]
	if layer < 0 or layer >= tilemap.get_layers_count():
		return context["bridge_error"].call(400, request_id, "TILEMAP_LAYER_INVALID", "Layer index out of range", {"layer": layer, "layers": tilemap.get_layers_count()})
	tilemap.erase_cell(layer, Vector2i(x, y))
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"node": node_path_str, "layer": layer, "x": x, "y": y, "applied": true})


func _get_tilemap_node(node_path_str: String, request_id: String, context: Dictionary) -> Dictionary:
	if node_path_str == "":
		return {"error": context["bridge_error"].call(400, request_id, "TILEMAP_NODE_MISSING", "node path is required", {})}
	var root: Node = context["edited_scene_root"].call()
	if root == null:
		return {"error": context["bridge_error"].call(409, request_id, "NO_SCENE_OPEN", "No edited scene is open", {})}
	var node: Node = context["node_by_path"].call(node_path_str)
	if node == null:
		return {"error": context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Node not found", {"path": node_path_str})}
	if not node is TileMap:
		return {"error": context["bridge_error"].call(400, request_id, "TILEMAP_NODE_INVALID", "Node is not a TileMap", {"path": node_path_str})}
	return {"node": node as TileMap}


func _load_tileset(tileset_path: String, _request_id: String, _context: Dictionary) -> TileSet:
	if not ResourceLoader.exists(tileset_path):
		return null
	var res: Resource = ResourceLoader.load(tileset_path, "", ResourceLoader.CACHE_MODE_IGNORE)
	if res is TileSet:
		return res as TileSet
	return null


func _valid_res_path(path: String) -> bool:
	return path != "" and path.begins_with("res://")


func _ensure_dir(resource_path: String) -> Error:
	var dir_path: String = resource_path.get_base_dir()
	if dir_path == "" or dir_path == "res://":
		return OK
	return DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(dir_path))
