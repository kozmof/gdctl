@tool
extends RefCounted


func handle_screenshot(request: Dictionary, context: Dictionary) -> Dictionary:
	var body: Dictionary = context["json_body_or_error"].call(request)
	if body.has("error_response"):
		return body["error_response"]
	if not bool(context["authorized"].call(request)):
		return context["bridge_error"].call(401, body.get("request_id", ""), "UNAUTHORIZED", "Viewport screenshot requires bearer token", {})
	if body.get("op", "") != "viewport.screenshot":
		return context["bridge_error"].call(400, body.get("request_id", ""), "INVALID_OPERATION", "Expected viewport.screenshot operation", {})

	var params: Dictionary = context["params_or_empty"].call(body)
	var kind: String = String(params.get("kind", "2d"))
	var index: int = int(params.get("index", 0))
	if kind != "2d" and kind != "3d":
		return context["bridge_error"].call(400, body.get("request_id", ""), "VIEWPORT_KIND_INVALID", "Viewport kind must be 2d or 3d", {"kind": kind})
	if index < 0 or index > 3:
		return context["bridge_error"].call(400, body.get("request_id", ""), "VIEWPORT_INDEX_INVALID", "Viewport index must be between 0 and 3", {"index": index})
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(500, body.get("request_id", ""), "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})

	var job_id: String = String(context["queue_job"].call("viewport.screenshot", {
		"kind": kind,
		"index": index,
		"frames_remaining": 2,
		"request_id": body.get("request_id", ""),
	}))
	return context["bridge_ok"].call(body.get("request_id", ""), {
		"queued": true,
		"job_id": job_id,
		"kind": kind,
		"index": index,
	})
