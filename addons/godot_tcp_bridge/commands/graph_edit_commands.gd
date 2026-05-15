@tool
extends RefCounted


func handle_node_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "graph-edit.node-add", "GraphEdit node-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var graph_path: String = String(params.get("path", ""))
	var node_name: String = String(params.get("name", ""))
	if graph_path == "" or node_name == "":
		return context["bridge_error"].call(400, request_id, "GRAPH_EDIT_PARAMS_MISSING", "path and name are required", {})
	var scene_root: Node = context["edited_scene_root"].call()
	if scene_root == null:
		return context["bridge_error"].call(503, request_id, "NO_EDITED_SCENE", "No scene is currently open in the editor", {})
	var graph: Node = context["node_by_path"].call(graph_path)
	if graph == null:
		return context["bridge_error"].call(404, request_id, "GRAPH_EDIT_NOT_FOUND", "GraphEdit node not found", {"path": graph_path})
	if not graph is GraphEdit:
		return context["bridge_error"].call(400, request_id, "GRAPH_EDIT_NODE_INVALID", "Node is not a GraphEdit", {"path": graph_path})
	var ge: GraphEdit = graph as GraphEdit
	var gn := GraphNode.new()
	gn.name = node_name
	gn.title = node_name
	var pos_raw = params.get("position", null)
	if pos_raw is Array and pos_raw.size() >= 2:
		gn.position_offset = Vector2(float(pos_raw[0]), float(pos_raw[1]))
	ge.add_child(gn)
	gn.owner = scene_root
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"graph": graph_path, "name": node_name, "added": true})


func handle_connection_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "graph-edit.connection-add", "GraphEdit connection-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var graph_path: String = String(params.get("graph", ""))
	var from_node: String = String(params.get("from", ""))
	var to_node: String = String(params.get("to", ""))
	var from_port: int = int(params.get("from_port", 0))
	var to_port: int = int(params.get("to_port", 0))
	if graph_path == "" or from_node == "" or to_node == "":
		return context["bridge_error"].call(400, request_id, "GRAPH_EDIT_PARAMS_MISSING", "graph, from, and to are required", {})
	var graph: Node = context["node_by_path"].call(graph_path)
	if graph == null:
		return context["bridge_error"].call(404, request_id, "GRAPH_EDIT_NOT_FOUND", "GraphEdit node not found", {"path": graph_path})
	if not graph is GraphEdit:
		return context["bridge_error"].call(400, request_id, "GRAPH_EDIT_NODE_INVALID", "Node is not a GraphEdit", {"path": graph_path})
	var ge: GraphEdit = graph as GraphEdit
	var err: Error = ge.connect_node(from_node, from_port, to_node, to_port)
	if err != OK:
		return context["bridge_error"].call(500, request_id, "GRAPH_EDIT_CONNECT_FAILED", "Could not connect graph nodes", {"from": from_node, "to": to_node, "error": error_string(err)})
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"graph": graph_path, "from": from_node, "from_port": from_port, "to": to_node, "to_port": to_port, "connected": true})


func handle_node_remove(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "graph-edit.node-remove", "GraphEdit node-remove requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var graph_path: String = String(params.get("path", ""))
	var node_name: String = String(params.get("name", ""))
	if graph_path == "" or node_name == "":
		return context["bridge_error"].call(400, request_id, "GRAPH_EDIT_PARAMS_MISSING", "path and name are required", {})
	var graph: Node = context["node_by_path"].call(graph_path)
	if graph == null:
		return context["bridge_error"].call(404, request_id, "GRAPH_EDIT_NOT_FOUND", "GraphEdit node not found", {"path": graph_path})
	if not graph is GraphEdit:
		return context["bridge_error"].call(400, request_id, "GRAPH_EDIT_NODE_INVALID", "Node is not a GraphEdit", {"path": graph_path})
	var ge: GraphEdit = graph as GraphEdit
	var child: Node = ge.get_node_or_null(node_name)
	if child == null:
		return context["bridge_error"].call(404, request_id, "GRAPH_NODE_NOT_FOUND", "GraphNode not found in GraphEdit", {"name": node_name})
	child.queue_free()
	context["mark_scene_dirty"].call()
	return context["bridge_ok"].call(request_id, {"graph": graph_path, "name": node_name, "removed": true})
