@tool
extends RefCounted

const DEFAULT_LIMIT := 200

var entries: Array[Dictionary] = []
var limit: int = DEFAULT_LIMIT


func add(level: String, source: String, message: String, detail: Dictionary = {}) -> void:
	entries.append({
		"time": Time.get_datetime_string_from_system(true),
		"level": level,
		"source": source,
		"message": message,
		"detail": detail,
	})
	while entries.size() > limit:
		entries.pop_front()


func list() -> Array[Dictionary]:
	return entries.duplicate(true)


func clear() -> void:
	entries.clear()
