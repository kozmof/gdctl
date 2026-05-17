@tool
extends RefCounted


func handle_action_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "input.action_add", "Input action add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var action: String = String(params.get("action", ""))
	var deadzone: float = float(params.get("deadzone", 0.5))
	var valid_error: String = _validate_action(action)
	if valid_error != "":
		return context["bridge_error"].call(400, request_id, "INPUT_ACTION_INVALID", valid_error, {"action": action})
	if not InputMap.has_action(action):
		InputMap.add_action(action, deadzone)
	else:
		InputMap.action_set_deadzone(action, deadzone)
	var save_result: Dictionary = _save_action(action)
	if not bool(save_result.get("ok", false)):
		return context["bridge_error"].call(500, request_id, "INPUT_SAVE_FAILED", String(save_result.get("error", "Failed to save input map")), {})
	return context["bridge_ok"].call(request_id, _action_payload(action, {"added": true}))


func handle_action_remove(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "input.action_remove", "Input action remove requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var action: String = String(params.get("action", ""))
	var valid_error: String = _validate_action(action)
	if valid_error != "":
		return context["bridge_error"].call(400, request_id, "INPUT_ACTION_INVALID", valid_error, {"action": action})
	if not InputMap.has_action(action):
		return context["bridge_error"].call(404, request_id, "INPUT_ACTION_NOT_FOUND", "Input action does not exist", {"action": action})
	InputMap.erase_action(action)
	ProjectSettings.clear("input/" + action)
	var save_err: Error = ProjectSettings.save()
	if save_err != OK:
		return context["bridge_error"].call(500, request_id, "INPUT_SAVE_FAILED", "Failed to save project settings", {"error": error_string(save_err)})
	return context["bridge_ok"].call(request_id, {
		"action": action,
		"removed": true,
	})


func handle_action_list(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "input.action_list", "Input action list requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var include_builtin: bool = bool(params.get("include_builtin", false))
	var actions: Array = []
	for action_name: StringName in InputMap.get_actions():
		var action: String = String(action_name)
		var is_project_action: bool = ProjectSettings.has_setting("input/" + action)
		if include_builtin or is_project_action:
			actions.append(_action_payload(action, {"project": is_project_action}))
	actions.sort_custom(func(a: Dictionary, b: Dictionary) -> bool: return String(a["action"]) < String(b["action"]))
	return context["bridge_ok"].call(request_id, {
		"actions": actions,
	})


func handle_event_add_key(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "input.event_add_key", "Input key event add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var action: String = String(params.get("action", ""))
	var key_name: String = String(params.get("key", ""))
	var physical: bool = bool(params.get("physical", true))
	var valid_error: String = _validate_action(action)
	if valid_error != "":
		return context["bridge_error"].call(400, request_id, "INPUT_ACTION_INVALID", valid_error, {"action": action})
	if key_name == "":
		return context["bridge_error"].call(400, request_id, "INPUT_KEY_INVALID", "Key is required", {})
	if not InputMap.has_action(action):
		InputMap.add_action(action)
	var keycode: int = OS.find_keycode_from_string(key_name)
	if keycode == KEY_NONE:
		return context["bridge_error"].call(400, request_id, "INPUT_KEY_INVALID", "Unknown key name", {"key": key_name})
	var event := InputEventKey.new()
	if physical:
		event.physical_keycode = keycode
	else:
		event.keycode = keycode
	for old_event: InputEvent in InputMap.action_get_events(action):
		if old_event is InputEventKey:
			var old_key := old_event as InputEventKey
			if physical and old_key.physical_keycode == keycode:
				return context["bridge_ok"].call(request_id, _action_payload(action, {"event_added": false, "key": key_name, "physical": physical}))
			if not physical and old_key.keycode == keycode:
				return context["bridge_ok"].call(request_id, _action_payload(action, {"event_added": false, "key": key_name, "physical": physical}))
	InputMap.action_add_event(action, event)
	var save_result: Dictionary = _save_action(action)
	if not bool(save_result.get("ok", false)):
		return context["bridge_error"].call(500, request_id, "INPUT_SAVE_FAILED", String(save_result.get("error", "Failed to save input map")), {})
	return context["bridge_ok"].call(request_id, _action_payload(action, {"event_added": true, "key": key_name, "physical": physical}))


func handle_event_add_mouse_button(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "input.event_add_mouse_button", "Input mouse button event add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var action: String = String(params.get("action", ""))
	var button_name: String = String(params.get("button", ""))
	var valid_error: String = _validate_action(action)
	if valid_error != "":
		return context["bridge_error"].call(400, request_id, "INPUT_ACTION_INVALID", valid_error, {"action": action})
	if button_name == "":
		return context["bridge_error"].call(400, request_id, "INPUT_MOUSE_BUTTON_INVALID", "button is required (use left, right, or middle)", {})
	var button: int = _mouse_button_from_name(button_name)
	if button == 0:
		return context["bridge_error"].call(400, request_id, "INPUT_MOUSE_BUTTON_INVALID", "Unknown mouse button name (use left, right, or middle)", {"button": button_name})
	if not InputMap.has_action(action):
		InputMap.add_action(action)
	for old_event: InputEvent in InputMap.action_get_events(action):
		if old_event is InputEventMouseButton:
			var old_btn := old_event as InputEventMouseButton
			if int(old_btn.button_index) == button:
				return context["bridge_ok"].call(request_id, _action_payload(action, {"event_added": false, "button": button_name}))
	var event := InputEventMouseButton.new()
	event.button_index = button
	InputMap.action_add_event(action, event)
	var save_result: Dictionary = _save_action(action)
	if not bool(save_result.get("ok", false)):
		return context["bridge_error"].call(500, request_id, "INPUT_SAVE_FAILED", String(save_result.get("error", "Failed to save input map")), {})
	return context["bridge_ok"].call(request_id, _action_payload(action, {"event_added": true, "button": button_name}))


func _mouse_button_from_name(name: String) -> int:
	match name.to_lower():
		"left", "1":
			return MOUSE_BUTTON_LEFT
		"right", "2":
			return MOUSE_BUTTON_RIGHT
		"middle", "3":
			return MOUSE_BUTTON_MIDDLE
	return 0


func handle_event_add_joypad(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "input.event_add_joypad", "Input joypad event add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var action: String = String(params.get("action", ""))
	var button: int = int(params.get("button", -1))
	var axis: int = int(params.get("axis", -1))
	var axis_value: float = float(params.get("axis_value", 1.0))
	var device: int = int(params.get("device", -1))
	var valid_error: String = _validate_action(action)
	if valid_error != "":
		return context["bridge_error"].call(400, request_id, "INPUT_ACTION_INVALID", valid_error, {"action": action})
	if button < 0 and axis < 0:
		return context["bridge_error"].call(400, request_id, "INPUT_JOYPAD_PARAMS_MISSING", "Either --button or --axis is required", {})
	if not InputMap.has_action(action):
		InputMap.add_action(action)
	if button >= 0:
		var event := InputEventJoypadButton.new()
		event.button_index = button
		if device >= 0:
			event.device = device
		InputMap.action_add_event(action, event)
	else:
		var event := InputEventJoypadMotion.new()
		event.axis = axis
		event.axis_value = axis_value
		if device >= 0:
			event.device = device
		InputMap.action_add_event(action, event)
	var save_result: Dictionary = _save_action(action)
	if not bool(save_result.get("ok", false)):
		return context["bridge_error"].call(500, request_id, "INPUT_SAVE_FAILED", String(save_result.get("error", "Failed to save input map")), {})
	return context["bridge_ok"].call(request_id, _action_payload(action, {"event_added": true}))


func _validate_action(action: String) -> String:
	if action == "":
		return "Input action is required"
	if action.find("/") != -1 or action.find("..") != -1:
		return "Input action must not contain path separators"
	return ""


func _save_action(action: String) -> Dictionary:
	ProjectSettings.set_setting("input/" + action, {
		"deadzone": InputMap.action_get_deadzone(action),
		"events": InputMap.action_get_events(action),
	})
	var save_err: Error = ProjectSettings.save()
	if save_err != OK:
		return {"ok": false, "error": error_string(save_err)}
	return {"ok": true}


func _action_payload(action: String, extra: Dictionary = {}) -> Dictionary:
	var events: Array = []
	if InputMap.has_action(action):
		for event: InputEvent in InputMap.action_get_events(action):
			events.append(_event_payload(event))
	var payload: Dictionary = {
		"action": action,
		"deadzone": InputMap.action_get_deadzone(action) if InputMap.has_action(action) else 0.0,
		"events": events,
	}
	for key: Variant in extra.keys():
		payload[key] = extra[key]
	return payload


func _event_payload(event: InputEvent) -> Dictionary:
	if event is InputEventKey:
		var key_event := event as InputEventKey
		var keycode: int = key_event.physical_keycode
		var physical: bool = true
		if keycode == KEY_NONE:
			keycode = key_event.keycode
			physical = false
		return {
			"type": "key",
			"keycode": keycode,
			"key": OS.get_keycode_string(keycode),
			"physical": physical,
		}
	if event is InputEventMouseButton:
		var btn_event := event as InputEventMouseButton
		var btn_name := ""
		match int(btn_event.button_index):
			MOUSE_BUTTON_LEFT: btn_name = "left"
			MOUSE_BUTTON_RIGHT: btn_name = "right"
			MOUSE_BUTTON_MIDDLE: btn_name = "middle"
		return {
			"type": "mouse_button",
			"button": int(btn_event.button_index),
			"button_name": btn_name,
		}
	if event is InputEventJoypadButton:
		var btn_event := event as InputEventJoypadButton
		return {
			"type": "joypad_button",
			"button": int(btn_event.button_index),
			"device": btn_event.device,
		}
	if event is InputEventJoypadMotion:
		var motion_event := event as InputEventJoypadMotion
		return {
			"type": "joypad_motion",
			"axis": int(motion_event.axis),
			"axis_value": motion_event.axis_value,
			"device": motion_event.device,
		}
	return {
		"type": event.get_class(),
		"text": event.as_text(),
	}
