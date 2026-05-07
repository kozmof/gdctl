@tool
extends RefCounted


func parse_request(buffer: PackedByteArray) -> Dictionary:
	var text: String = buffer.get_string_from_utf8()
	var header_end: int = text.find("\r\n\r\n")
	if header_end == -1:
		return {}

	var header_text: String = text.substr(0, header_end)
	var lines: PackedStringArray = header_text.split("\r\n")
	if lines.is_empty():
		return {}
	var request_line: PackedStringArray = lines[0].split(" ")
	if request_line.size() < 2:
		return {"method": "", "path": "", "headers": {}, "body": ""}

	var headers: Dictionary = {}
	var content_length: int = 0
	for i in range(1, lines.size()):
		var line: String = String(lines[i])
		var idx: int = line.find(":")
		if idx == -1:
			continue
		var name: String = line.substr(0, idx).strip_edges().to_lower()
		var value: String = line.substr(idx + 1).strip_edges()
		headers[name] = value
		if name == "content-length":
			content_length = int(value)

	var body_start: int = header_end + 4
	var body: String = text.substr(body_start)
	if body.length() < content_length:
		return {}
	body = body.substr(0, content_length)
	return {
		"method": String(request_line[0]),
		"path": String(request_line[1]),
		"headers": headers,
		"body": body,
	}


func json_body_or_error(request: Dictionary) -> Dictionary:
	var headers: Dictionary = request.get("headers", {})
	var content_type: String = String(headers.get("content-type", ""))
	if not content_type.to_lower().begins_with("application/json"):
		return {"error_response": bridge_error(415, "", "UNSUPPORTED_MEDIA_TYPE", "Mutation endpoints accept only application/json", {})}
	var parsed: Variant = JSON.parse_string(String(request.get("body", "")))
	if typeof(parsed) != TYPE_DICTIONARY:
		return {"error_response": bridge_error(400, "", "INVALID_JSON", "Request body must be a JSON object", {})}
	var parsed_dict: Dictionary = parsed
	return parsed_dict


func bridge_ok(request_id: String, result: Dictionary) -> Dictionary:
	return http_json(200, {
		"request_id": request_id,
		"ok": true,
		"result": result,
		"error": null,
	})


func bridge_error(status: int, request_id: String, code: String, message: String, detail: Dictionary) -> Dictionary:
	return http_json(status, {
		"request_id": request_id,
		"ok": false,
		"result": null,
		"error": {
			"code": code,
			"message": message,
			"detail": detail,
		},
	})


func http_json(status: int, body: Dictionary) -> Dictionary:
	return {"status": status, "body": body}


func write_response(peer: StreamPeerTCP, response: Dictionary) -> void:
	var status: int = int(response.get("status", 500))
	var body: String = JSON.stringify(response.get("body", {}))
	var reason: String = "OK"
	if status == 400:
		reason = "Bad Request"
	elif status == 401:
		reason = "Unauthorized"
	elif status == 404:
		reason = "Not Found"
	elif status == 409:
		reason = "Conflict"
	elif status == 415:
		reason = "Unsupported Media Type"
	elif status >= 500:
		reason = "Internal Server Error"

	var head: String = "HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n" % [status, reason, body.to_utf8_buffer().size()]
	peer.put_data((head + body).to_utf8_buffer())
