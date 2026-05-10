@tool
extends RefCounted


func decode(encoded: Variant) -> Dictionary:
	if typeof(encoded) != TYPE_DICTIONARY:
		return {"ok": false, "error": "Value must be an object with kind and value fields"}
	var encoded_dict: Dictionary = encoded
	var kind: String = String(encoded_dict.get("kind", "")).to_lower()
	var raw: Variant = encoded_dict.get("value", null)
	match kind:
		"nil", "null":
			return {"ok": true, "value": null}
		"string":
			return {"ok": true, "value": String(raw)}
		"bool", "boolean":
			return {"ok": true, "value": bool(raw)}
		"int", "integer":
			return {"ok": true, "value": int(raw)}
		"float", "number":
			return {"ok": true, "value": float(raw)}
		"vector2":
			var vector2_value: Variant = _array_to_vector2(raw)
			if typeof(vector2_value) != TYPE_VECTOR2:
				return {"ok": false, "error": "Vector2 value must be [x, y]"}
			return {"ok": true, "value": vector2_value}
		"vector3":
			var vector3_value: Variant = _array_to_vector3(raw)
			if typeof(vector3_value) != TYPE_VECTOR3:
				return {"ok": false, "error": "Vector3 value must be [x, y, z]"}
			return {"ok": true, "value": vector3_value}
		"color":
			var color_value: Variant = _array_to_color(raw)
			if typeof(color_value) != TYPE_COLOR:
				return {"ok": false, "error": "Color value must be [r, g, b] or [r, g, b, a]"}
			return {"ok": true, "value": color_value}
		"packedvector2array", "vector2array":
			var vector2_array_value: Variant = _array_to_packed_vector2_array(raw)
			if typeof(vector2_array_value) != TYPE_PACKED_VECTOR2_ARRAY:
				return {"ok": false, "error": "PackedVector2Array value must be [[x, y], ...]"}
			return {"ok": true, "value": vector2_array_value}
		"packedstringarray", "stringarray":
			if typeof(raw) != TYPE_ARRAY:
				return {"ok": false, "error": "PackedStringArray value must be an array"}
			var raw_strings: Array = raw
			var string_array: PackedStringArray = PackedStringArray()
			for item in raw_strings:
				string_array.append(String(item))
			return {"ok": true, "value": string_array}
		"resource":
			var resource_path: String = String(raw)
			if resource_path == "" or not resource_path.begins_with("res://"):
				return {"ok": false, "error": "Resource value must be a res:// path"}
			if not FileAccess.file_exists(resource_path):
				return {"ok": false, "error": "Resource file does not exist: " + resource_path}
			var res: Resource = ResourceLoader.load(resource_path, "", ResourceLoader.CACHE_MODE_REPLACE)
			if res == null:
				return {"ok": false, "error": "Could not load resource: " + resource_path}
			return {"ok": true, "value": res}
	return {"ok": false, "error": "Unsupported value kind: " + kind}


func encode(value: Variant) -> Dictionary:
	match typeof(value):
		TYPE_NIL:
			return {"kind": "Nil", "value": null}
		TYPE_STRING, TYPE_STRING_NAME, TYPE_NODE_PATH:
			return {"kind": "String", "value": String(value)}
		TYPE_BOOL:
			return {"kind": "bool", "value": bool(value)}
		TYPE_INT:
			return {"kind": "int", "value": int(value)}
		TYPE_FLOAT:
			return {"kind": "float", "value": float(value)}
		TYPE_VECTOR2:
			var vector2_value: Vector2 = value
			return {"kind": "Vector2", "value": [vector2_value.x, vector2_value.y]}
		TYPE_VECTOR3:
			var vector3_value: Vector3 = value
			return {"kind": "Vector3", "value": [vector3_value.x, vector3_value.y, vector3_value.z]}
		TYPE_COLOR:
			var color_value: Color = value
			return {"kind": "Color", "value": [color_value.r, color_value.g, color_value.b, color_value.a]}
		TYPE_PACKED_VECTOR2_ARRAY:
			var vector2_array: PackedVector2Array = value
			var encoded_vectors: Array = []
			for point in vector2_array:
				encoded_vectors.append([point.x, point.y])
			return {"kind": "PackedVector2Array", "value": encoded_vectors}
		TYPE_PACKED_STRING_ARRAY:
			var packed_strings: PackedStringArray = value
			var encoded_strings: Array = []
			for item in packed_strings:
				encoded_strings.append(String(item))
			return {"kind": "PackedStringArray", "value": encoded_strings}
	return {"kind": type_string(typeof(value)), "value": str(value)}


func _array_to_vector2(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_ARRAY:
		return null
	var items: Array = raw
	if items.size() != 2:
		return null
	return Vector2(float(items[0]), float(items[1]))


func _array_to_vector3(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_ARRAY:
		return null
	var items: Array = raw
	if items.size() != 3:
		return null
	return Vector3(float(items[0]), float(items[1]), float(items[2]))


func _array_to_color(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_ARRAY:
		return null
	var items: Array = raw
	if items.size() == 3:
		return Color(float(items[0]), float(items[1]), float(items[2]), 1.0)
	if items.size() == 4:
		return Color(float(items[0]), float(items[1]), float(items[2]), float(items[3]))
	return null


func _array_to_packed_vector2_array(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_ARRAY:
		return null
	var items: Array = raw
	var points: PackedVector2Array = PackedVector2Array()
	for item in items:
		var point: Variant = _array_to_vector2(item)
		if typeof(point) != TYPE_VECTOR2:
			return null
		var point_vector: Vector2 = point
		points.append(point_vector)
	return points
