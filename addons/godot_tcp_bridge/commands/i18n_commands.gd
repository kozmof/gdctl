@tool
extends RefCounted

# Maps locale code -> Translation resource (kept in memory during editor session)
var _translations: Dictionary = {}


func handle_locale_set(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "i18n.locale-set", "i18n locale-set requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var locale: String = String(params.get("locale", ""))
	if locale == "":
		return context["bridge_error"].call(400, request_id, "I18N_LOCALE_MISSING", "locale is required", {})
	TranslationServer.set_locale(locale)
	var current: String = TranslationServer.get_locale()
	return context["bridge_ok"].call(request_id, {"locale": current, "applied": true})


func handle_string_add(request: Dictionary, context: Dictionary) -> Dictionary:
	var checked: Dictionary = context["request"].require_body(request, context, "i18n.string-add", "i18n string-add requires bearer token")
	if not bool(checked.get("ok", false)):
		return checked["error_response"]
	var params: Dictionary = checked["params"]
	var request_id: String = String(checked["request_id"])
	var key: String = String(params.get("key", ""))
	var locale: String = String(params.get("locale", ""))
	var text: String = String(params.get("text", ""))
	if key == "" or locale == "" or text == "":
		return context["bridge_error"].call(400, request_id, "I18N_PARAMS_MISSING", "key, locale, and text are required", {})
	var translation: Translation = null
	if _translations.has(locale):
		translation = _translations[locale]
	else:
		translation = Translation.new()
		translation.locale = locale
		_translations[locale] = translation
		TranslationServer.add_translation(translation)
	translation.add_message(key, text)
	return context["bridge_ok"].call(request_id, {"key": key, "locale": locale, "text": text, "added": true})
