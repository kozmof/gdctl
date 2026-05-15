@tool
extends RefCounted


func handle_add_state(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation-tree.add-state", "AnimationTree add-state requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var tree_path: String = String(params.get("tree_path", ""))
	var state_name: String = String(params.get("name", ""))
	var anim_name: String = String(params.get("animation", ""))
	if tree_path == "" or state_name == "":
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_PARAMS_MISSING", "tree_path and name are required", {})
	var tree_node: Node = context["node_by_path"].call(tree_path)
	if tree_node == null or not tree_node is AnimationTree:
		return context["bridge_error"].call(404, request_id, "ANIM_TREE_NOT_FOUND", "AnimationTree node not found", {"path": tree_path})
	var anim_tree: AnimationTree = tree_node as AnimationTree
	var state_machine: AnimationNodeStateMachine = anim_tree.tree_root as AnimationNodeStateMachine
	if state_machine == null:
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_NOT_STATE_MACHINE", "AnimationTree.tree_root is not an AnimationNodeStateMachine", {})
	if state_machine.has_node(state_name):
		return context["bridge_ok"].call(request_id, {"tree_path": tree_path, "name": state_name, "created": false, "note": "state already exists"})
	var anim_node := AnimationNodeAnimation.new()
	if anim_name != "":
		anim_node.animation = anim_name
	state_machine.add_node(state_name, anim_node)
	return context["bridge_ok"].call(request_id, {"tree_path": tree_path, "name": state_name, "animation": anim_name, "created": true})


func handle_add_transition(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation-tree.add-transition", "AnimationTree add-transition requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var tree_path: String = String(params.get("tree_path", ""))
	var from_state: String = String(params.get("from", ""))
	var to_state: String = String(params.get("to", ""))
	var condition: String = String(params.get("condition", ""))
	if tree_path == "" or from_state == "" or to_state == "":
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_PARAMS_MISSING", "tree_path, from, and to are required", {})
	var tree_node: Node = context["node_by_path"].call(tree_path)
	if tree_node == null or not tree_node is AnimationTree:
		return context["bridge_error"].call(404, request_id, "ANIM_TREE_NOT_FOUND", "AnimationTree node not found", {"path": tree_path})
	var anim_tree: AnimationTree = tree_node as AnimationTree
	var state_machine: AnimationNodeStateMachine = anim_tree.tree_root as AnimationNodeStateMachine
	if state_machine == null:
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_NOT_STATE_MACHINE", "AnimationTree.tree_root is not an AnimationNodeStateMachine", {})
	var transition := AnimationNodeStateMachineTransition.new()
	if condition != "":
		transition.advance_condition = condition
	state_machine.add_transition(from_state, to_state, transition)
	return context["bridge_ok"].call(request_id, {"tree_path": tree_path, "from": from_state, "to": to_state, "condition": condition, "created": true})


func handle_blend_space_2d_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation-tree.blend-space-2d-add", "AnimationTree blend-space-2d-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var tree_path: String = String(params.get("tree_path", ""))
	var state_name: String = String(params.get("state", ""))
	var blend_x: String = String(params.get("blend_x", ""))
	var blend_y: String = String(params.get("blend_y", ""))
	if tree_path == "" or state_name == "":
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_PARAMS_MISSING", "tree_path and state are required", {})
	var tree_node: Node = context["node_by_path"].call(tree_path)
	if tree_node == null or not tree_node is AnimationTree:
		return context["bridge_error"].call(404, request_id, "ANIM_TREE_NOT_FOUND", "AnimationTree node not found", {"path": tree_path})
	var anim_tree: AnimationTree = tree_node as AnimationTree
	var state_machine: AnimationNodeStateMachine = anim_tree.tree_root as AnimationNodeStateMachine
	if state_machine == null:
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_NOT_STATE_MACHINE", "AnimationTree.tree_root is not an AnimationNodeStateMachine", {})
	var blend_space := AnimationNodeBlendSpace2D.new()
	if state_machine.has_node(state_name):
		state_machine.remove_node(state_name)
	state_machine.add_node(state_name, blend_space)
	return context["bridge_ok"].call(request_id, {"tree_path": tree_path, "state": state_name, "blend_x": blend_x, "blend_y": blend_y, "created": true})


func handle_set_param(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "animation-tree.set-param", "AnimationTree set-param requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var tree_path: String = String(params.get("tree_path", ""))
	var param: String = String(params.get("param", ""))
	if tree_path == "" or param == "":
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_PARAMS_MISSING", "tree_path and param are required", {})
	var tree_node: Node = context["node_by_path"].call(tree_path)
	if tree_node == null or not tree_node is AnimationTree:
		return context["bridge_error"].call(404, request_id, "ANIM_TREE_NOT_FOUND", "AnimationTree node not found", {"path": tree_path})
	var anim_tree: AnimationTree = tree_node as AnimationTree
	var value = null
	if params.has("vector2"):
		var v2 = params["vector2"]
		if v2 is Array and v2.size() >= 2:
			value = Vector2(float(v2[0]), float(v2[1]))
		elif v2 is Dictionary:
			value = Vector2(float(v2.get("x", 0.0)), float(v2.get("y", 0.0)))
	elif params.has("float"):
		value = float(params["float"])
	elif params.has("bool"):
		value = bool(params["bool"])
	elif params.has("int"):
		value = int(params["int"])
	if value == null:
		return context["bridge_error"].call(400, request_id, "ANIM_TREE_VALUE_MISSING", "One of vector2, float, bool, or int is required", {})
	anim_tree.set(param, value)
	return context["bridge_ok"].call(request_id, {"tree_path": tree_path, "param": param, "set": true})


