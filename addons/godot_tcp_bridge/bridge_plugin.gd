@tool
extends EditorPlugin

const BridgeServer := preload("res://addons/godot_tcp_bridge/bridge_server.gd")

var server

func _enter_tree() -> void:
	server = BridgeServer.new()
	server.editor_plugin = self
	server.start()
	set_process(true)


func _exit_tree() -> void:
	if server:
		server.stop()
		server = null


func _process(_delta: float) -> void:
	if server:
		server.poll()
