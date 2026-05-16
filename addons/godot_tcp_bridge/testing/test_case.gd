@tool
extends RefCounted

var _gdctl_current_test := ""
var _gdctl_failures: Array[Dictionary] = []


func _gdctl_begin_test(name: String) -> void:
	_gdctl_current_test = name
	_gdctl_failures.clear()


func _gdctl_end_test() -> Array[Dictionary]:
	var out: Array[Dictionary] = []
	for failure in _gdctl_failures:
		out.append(failure.duplicate(true))
	return out


func assert_true(value: Variant, message: String = "") -> void:
	if not bool(value):
		fail(_message_or_default(message, "Expected value to be true"))


func assert_false(value: Variant, message: String = "") -> void:
	if bool(value):
		fail(_message_or_default(message, "Expected value to be false"))


func assert_eq(actual: Variant, expected: Variant, message: String = "") -> void:
	if actual != expected:
		fail(_message_or_default(message, "Expected %s to equal %s" % [var_to_str(actual), var_to_str(expected)]))


func assert_ne(actual: Variant, expected: Variant, message: String = "") -> void:
	if actual == expected:
		fail(_message_or_default(message, "Expected %s to not equal %s" % [var_to_str(actual), var_to_str(expected)]))


func fail(message: String = "Test failed") -> void:
	_gdctl_failures.append({
		"test": _gdctl_current_test,
		"message": String(message),
	})


func _message_or_default(message: String, default_message: String) -> String:
	if message.strip_edges() != "":
		return message
	return default_message
