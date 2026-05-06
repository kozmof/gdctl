@tool
extends EditorPlugin

const BridgeServer := preload("res://addons/godot_tcp_bridge/bridge_server.gd")

var server
var dock: VBoxContainer
var status_label: Label
var token_edit: LineEdit
var command_edit: LineEdit
var show_token_button: CheckButton

func _enter_tree() -> void:
	server = BridgeServer.new()
	server.editor_plugin = self
	server.start()
	_create_dock()
	_refresh_dock()
	set_process(true)


func _exit_tree() -> void:
	if dock:
		remove_control_from_docks(dock)
		dock.queue_free()
		dock = null
	if server:
		server.stop()
		server = null


func _process(_delta: float) -> void:
	if server:
		server.poll()
		_refresh_status()


func _create_dock() -> void:
	dock = VBoxContainer.new()
	dock.name = "gdctl"
	dock.custom_minimum_size = Vector2(320, 0)

	var title := Label.new()
	title.text = "gdctl Bridge"
	title.add_theme_font_size_override("font_size", 16)
	dock.add_child(title)

	status_label = Label.new()
	dock.add_child(status_label)

	var token_label := Label.new()
	token_label.text = "Mutation token"
	dock.add_child(token_label)

	var token_row := HBoxContainer.new()
	token_edit = LineEdit.new()
	token_edit.editable = false
	token_edit.secret = true
	token_edit.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	token_row.add_child(token_edit)

	show_token_button = CheckButton.new()
	show_token_button.text = "Show"
	show_token_button.toggled.connect(_on_show_token_toggled)
	token_row.add_child(show_token_button)
	dock.add_child(token_row)

	var button_row := HBoxContainer.new()
	var copy_token_button := Button.new()
	copy_token_button.text = "Copy Token"
	copy_token_button.pressed.connect(_on_copy_token_pressed)
	button_row.add_child(copy_token_button)

	var copy_command_button := Button.new()
	copy_command_button.text = "Copy Export"
	copy_command_button.pressed.connect(_on_copy_command_pressed)
	button_row.add_child(copy_command_button)

	var reset_button := Button.new()
	reset_button.text = "Reset Token"
	reset_button.pressed.connect(_on_reset_token_pressed)
	button_row.add_child(reset_button)
	dock.add_child(button_row)

	var command_label := Label.new()
	command_label.text = "Devcontainer command"
	dock.add_child(command_label)

	command_edit = LineEdit.new()
	command_edit.editable = false
	command_edit.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	dock.add_child(command_edit)

	add_control_to_dock(DOCK_SLOT_RIGHT_UL, dock)


func _refresh_dock() -> void:
	if not dock or not server:
		return
	var token: String = server.get_token()
	token_edit.text = token
	command_edit.text = "export GDCTL_BRIDGE_TOKEN=%s" % token
	_refresh_status()


func _refresh_status() -> void:
	if not status_label or not server:
		return
	var auth_text := "enabled" if server.auth_enabled else "disabled"
	var state_text := "listening" if server.running else "stopped"
	status_label.text = "%s on %s:%d, auth %s" % [state_text, server.host, server.port, auth_text]


func _on_show_token_toggled(enabled: bool) -> void:
	if token_edit:
		token_edit.secret = not enabled


func _on_copy_token_pressed() -> void:
	if server:
		DisplayServer.clipboard_set(server.get_token())


func _on_copy_command_pressed() -> void:
	if command_edit:
		DisplayServer.clipboard_set(command_edit.text)


func _on_reset_token_pressed() -> void:
	if server:
		server.reset_token()
		server.restart()
		_refresh_dock()
