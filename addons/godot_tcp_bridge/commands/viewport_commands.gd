@tool
extends RefCounted


func handle_screenshot(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "viewport.screenshot", "Viewport screenshot requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var kind: String = String(params.get("kind", "2d"))
	var index: int = int(params.get("index", 0))
	if kind != "2d" and kind != "3d":
		return context["bridge_error"].call(400, request_id, "VIEWPORT_KIND_INVALID", "Viewport kind must be 2d or 3d", {"kind": kind})
	if index < 0 or index > 3:
		return context["bridge_error"].call(400, request_id, "VIEWPORT_INDEX_INVALID", "Viewport index must be between 0 and 3", {"index": index})
	if not bool(context["editor_plugin_available"].call()):
		return context["bridge_error"].call(500, request_id, "EDITOR_PLUGIN_UNAVAILABLE", "Editor plugin is unavailable", {})

	var job_id: String = String(context["queue_job"].call("viewport.screenshot", {
		"kind": kind,
		"index": index,
		"frames_remaining": 2,
		"request_id": request_id,
	}))
	return context["bridge_ok"].call(request_id, {
		"queued": true,
		"job_id": job_id,
		"kind": kind,
		"index": index,
	})
