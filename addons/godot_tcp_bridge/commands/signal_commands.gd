@tool
extends RefCounted


func handle_connect(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "signal.connect", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var from_path: String = String(params.get("from", ""))
	var signal_name: String = String(params.get("signal", ""))
	var to_path: String = String(params.get("to", ""))
	var method: String = String(params.get("method", ""))
	if signal_name == "":
		return context["bridge_error"].call(400, request_id, "SIGNAL_NAME_INVALID", "Signal name is required", {})
	if method == "":
		return context["bridge_error"].call(400, request_id, "METHOD_NAME_INVALID", "Method name is required", {})
	var from_node: Node = context["node_by_path"].call(from_path)
	if from_node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Source node does not exist", {"path": from_path})
	var to_node: Node = context["node_by_path"].call(to_path)
	if to_node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Target node does not exist", {"path": to_path})
	if not from_node.has_signal(signal_name):
		return context["bridge_error"].call(400, request_id, "SIGNAL_NOT_FOUND", "Node does not have signal", {"path": from_path, "signal": signal_name})
	var callable := Callable(to_node, method)
	if from_node.is_connected(signal_name, callable):
		return context["bridge_error"].call(409, request_id, "SIGNAL_ALREADY_CONNECTED", "Signal is already connected", {"signal": signal_name, "method": method})
	var err: Error = from_node.connect(signal_name, callable)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "SIGNAL_CONNECT_FAILED", "Failed to connect signal", {"error": error_string(err)})
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"from": context["logical_path"].call(from_node),
		"signal": signal_name,
		"to": context["logical_path"].call(to_node),
		"method": method,
		"connected": true,
	})


func handle_disconnect(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "signal.disconnect", "Mutation endpoint requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var from_path: String = String(params.get("from", ""))
	var signal_name: String = String(params.get("signal", ""))
	var to_path: String = String(params.get("to", ""))
	var method: String = String(params.get("method", ""))
	if signal_name == "":
		return context["bridge_error"].call(400, request_id, "SIGNAL_NAME_INVALID", "Signal name is required", {})
	if method == "":
		return context["bridge_error"].call(400, request_id, "METHOD_NAME_INVALID", "Method name is required", {})
	var from_node: Node = context["node_by_path"].call(from_path)
	if from_node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Source node does not exist", {"path": from_path})
	var to_node: Node = context["node_by_path"].call(to_path)
	if to_node == null:
		return context["bridge_error"].call(404, request_id, "NODE_NOT_FOUND", "Target node does not exist", {"path": to_path})
	if not from_node.has_signal(signal_name):
		return context["bridge_error"].call(400, request_id, "SIGNAL_NOT_FOUND", "Node does not have signal", {"path": from_path, "signal": signal_name})
	var callable := Callable(to_node, method)
	if not from_node.is_connected(signal_name, callable):
		return context["bridge_error"].call(404, request_id, "SIGNAL_NOT_CONNECTED", "Signal is not connected", {"signal": signal_name, "method": method})
	from_node.disconnect(signal_name, callable)
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {
		"from": context["logical_path"].call(from_node),
		"signal": signal_name,
		"to": context["logical_path"].call(to_node),
		"method": method,
		"disconnected": true,
	})
