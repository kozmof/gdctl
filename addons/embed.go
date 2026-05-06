package addons

import "embed"

// FS carries the Godot bridge addon with the gdctl binary.
//
//go:embed godot_tcp_bridge/**
var FS embed.FS
