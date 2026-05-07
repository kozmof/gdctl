@tool
extends RefCounted


func handle_list(request: Dictionary, context: Dictionary) -> Dictionary:
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, "", "UNAUTHORIZED", "Bridge logs require bearer token", {})
	var log_buffer = context["log_buffer"]
	return context["http_json"].call(200, {"ok": true, "entries": log_buffer.list()})


func handle_clear(request: Dictionary, context: Dictionary) -> Dictionary:
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, "", "UNAUTHORIZED", "Bridge log clearing requires bearer token", {})
	var log_buffer = context["log_buffer"]
	log_buffer.clear()
	log_buffer.add("info", "bridge.logs", "Logs cleared", {})
	return context["bridge_ok"].call("", {"cleared": true})
