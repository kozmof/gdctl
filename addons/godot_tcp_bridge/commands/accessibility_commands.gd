@tool
extends RefCounted

var _tts_pitch: float = 1.0
var _tts_rate: float = 1.0
var _tts_volume: int = 50
var _tts_voice: String = ""


func handle_tts_speak(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "accessibility.tts-speak", "TTS speak requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var text: String = String(params.get("text", ""))
	var interrupt: bool = bool(params.get("interrupt", false))
	if text == "":
		return context["bridge_error"].call(400, request_id, "TTS_TEXT_MISSING", "text is required", {})
	if not DisplayServer.has_feature(DisplayServer.FEATURE_TEXT_TO_SPEECH):
		return context["bridge_error"].call(503, request_id, "TTS_NOT_SUPPORTED", "TTS is not supported on this platform", {})
	var voices: Array[Dictionary] = DisplayServer.tts_get_voices()
	var voice_id: String = _tts_voice
	if voice_id == "" and not voices.is_empty():
		# pick first matching language or just first available
		voice_id = String(voices[0].get("id", ""))
	DisplayServer.tts_speak(text, voice_id, _tts_volume, _tts_rate, _tts_pitch, 0, interrupt)
	return context["bridge_ok"].call(request_id, {"text": text, "voice": voice_id, "interrupt": interrupt, "spoken": true})


func handle_tts_configure(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "accessibility.tts-configure", "TTS configure requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	if params.has("pitch"):
		_tts_pitch = float(params["pitch"])
	if params.has("rate"):
		_tts_rate = float(params["rate"])
	if params.has("volume"):
		_tts_volume = int(params["volume"])
	if params.has("voice"):
		_tts_voice = String(params["voice"])
	return context["bridge_ok"].call(request_id, {"pitch": _tts_pitch, "rate": _tts_rate, "volume": _tts_volume, "voice": _tts_voice, "applied": true})


func handle_tts_stop(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "accessibility.tts-stop", "TTS stop requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var request_id: String = String(checked["request_id"])
	if not DisplayServer.has_feature(DisplayServer.FEATURE_TEXT_TO_SPEECH):
		return context["bridge_ok"].call(request_id, {"stopped": false, "note": "TTS not supported on this platform"})
	DisplayServer.tts_stop()
	return context["bridge_ok"].call(request_id, {"stopped": true})
