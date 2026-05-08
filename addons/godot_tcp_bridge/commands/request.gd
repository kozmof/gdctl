@tool
extends RefCounted


func require_body(request: Dictionary, context: Dictionary, expected_op: String, unauthorized_message: String) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return {"ok": false, "error_response": body["error_response"]}

	var request_id: String = String(body.get("request_id", ""))
	if not bool(context["authorized"].call(request)):
		return {
			"ok": false,
			"error_response": context["bridge_error"].call(401, request_id, "UNAUTHORIZED", unauthorized_message, {}),
		}
	if String(body.get("op", "")) != expected_op:
		return {
			"ok": false,
			"error_response": context["bridge_error"].call(400, request_id, "INVALID_OPERATION", "Expected %s operation" % expected_op, {}),
		}

	return {
		"ok": true,
		"body": body,
		"params": context["params_or_empty"].call(body),
		"request_id": request_id,
	}
