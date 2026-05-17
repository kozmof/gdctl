@tool
extends RefCounted


func decode(encoded: Variant) -> Dictionary:
	if typeof(encoded) != TYPE_DICTIONARY:
		return {"ok": false, "error": "Value must be an object with kind and value fields"}
	var encoded_dict: Dictionary = encoded
	var kind: String = String(encoded_dict.get("kind", encoded_dict.get("type", ""))).to_lower()
	var raw: Variant = encoded_dict.get("value", null)
	if kind.begins_with("array[") and kind.ends_with("]"):
		return _decode_typed_array(kind.substr(6, kind.length() - 7), raw)
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
		"nodepath", "node_path":
			return {"ok": true, "value": NodePath(String(raw))}
		"aabb":
			var aabb_value: Variant = _dict_to_aabb(raw)
			if typeof(aabb_value) != TYPE_AABB:
				return {"ok": false, "error": "AABB value must be {\"position\":[x,y,z],\"size\":[w,h,d]}"}
			return {"ok": true, "value": aabb_value}
		"resource":
			return _decode_resource(raw)
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
		TYPE_AABB:
			var aabb_value: AABB = value
			return {"kind": "AABB", "value": {
				"position": [aabb_value.position.x, aabb_value.position.y, aabb_value.position.z],
				"size": [aabb_value.size.x, aabb_value.size.y, aabb_value.size.z],
			}}
		TYPE_ARRAY:
			var array_value: Array = value
			var encoded_array: Array = []
			for item in array_value:
				encoded_array.append(encode(item).get("value"))
			return {"kind": "Array", "value": encoded_array}
	return {"kind": type_string(typeof(value)), "value": str(value)}


func _decode_typed_array(element_kind: String, raw: Variant) -> Dictionary:
	if typeof(raw) != TYPE_ARRAY:
		return {"ok": false, "error": "Array[%s] value must be an array" % element_kind}
	var items: Array = raw
	match element_kind:
		"vector2":
			var out_vector2: Array[Vector2] = []
			for item in items:
				var value: Variant = _array_to_vector2(item)
				if typeof(value) != TYPE_VECTOR2:
					return {"ok": false, "error": "Array[Vector2] value must be [[x, y], ...]"}
				out_vector2.append(value)
			return {"ok": true, "value": out_vector2}
		"vector3":
			var out_vector3: Array[Vector3] = []
			for item in items:
				var value: Variant = _array_to_vector3(item)
				if typeof(value) != TYPE_VECTOR3:
					return {"ok": false, "error": "Array[Vector3] value must be [[x, y, z], ...]"}
				out_vector3.append(value)
			return {"ok": true, "value": out_vector3}
		"color":
			var out_color: Array[Color] = []
			for item in items:
				var value: Variant = _array_to_color(item)
				if typeof(value) != TYPE_COLOR:
					return {"ok": false, "error": "Array[Color] value must be [[r, g, b], ...] or [[r, g, b, a], ...]"}
				out_color.append(value)
			return {"ok": true, "value": out_color}
		"string":
			var out_string: Array[String] = []
			for item in items:
				out_string.append(String(item))
			return {"ok": true, "value": out_string}
		"int", "integer":
			var out_int: Array[int] = []
			for item in items:
				out_int.append(int(item))
			return {"ok": true, "value": out_int}
		"float", "number":
			var out_float: Array[float] = []
			for item in items:
				out_float.append(float(item))
			return {"ok": true, "value": out_float}
		"bool", "boolean":
			var out_bool: Array[bool] = []
			for item in items:
				out_bool.append(bool(item))
			return {"ok": true, "value": out_bool}
	return {"ok": false, "error": "Unsupported typed array kind: Array[%s]" % element_kind}


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


func _dict_to_aabb(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_DICTIONARY:
		return null
	var spec: Dictionary = raw
	var pos: Variant = _array_to_vector3(spec.get("position", null))
	var sz: Variant = _array_to_vector3(spec.get("size", null))
	if typeof(pos) != TYPE_VECTOR3 or typeof(sz) != TYPE_VECTOR3:
		return null
	return AABB(pos, sz)


func _decode_resource(raw: Variant) -> Dictionary:
	if typeof(raw) == TYPE_STRING:
		var resource_path: String = String(raw)
		if resource_path == "" or not resource_path.begins_with("res://"):
			return {"ok": false, "error": "Resource value must be a res:// path"}
		if not FileAccess.file_exists(resource_path):
			return {"ok": false, "error": "Resource file does not exist: " + resource_path}
		var res: Resource = ResourceLoader.load(resource_path, "", ResourceLoader.CACHE_MODE_REPLACE)
		if res == null:
			return {"ok": false, "error": "Could not load resource: " + resource_path}
		return {"ok": true, "value": res}
	if typeof(raw) != TYPE_DICTIONARY:
		return {"ok": false, "error": "Resource value must be a res:// path or inline resource object"}
	var spec: Dictionary = raw
	var type_name := String(spec.get("type", ""))
	if type_name == "" or not ClassDB.can_instantiate(type_name):
		return {"ok": false, "error": "Inline resource type cannot be instantiated: " + type_name}
	if not ClassDB.is_parent_class(type_name, "Resource") and type_name != "Resource":
		return {"ok": false, "error": "Inline resource type must inherit Resource: " + type_name}
	var resource: Resource = ClassDB.instantiate(type_name) as Resource
	if resource == null:
		return {"ok": false, "error": "Could not instantiate inline resource: " + type_name}
	var props_value: Variant = spec.get("properties", {})
	if typeof(props_value) != TYPE_DICTIONARY:
		return {"ok": false, "error": "Inline resource properties must be an object"}
	var props: Dictionary = props_value
	for property in props.keys():
		var decoded := decode(props[property])
		if not bool(decoded.get("ok", false)):
			return {"ok": false, "error": "%s.%s: %s" % [type_name, String(property), String(decoded.get("error", "Invalid typed value"))]}
		resource.set(String(property), decoded.get("value"))
	return {"ok": true, "value": resource}
