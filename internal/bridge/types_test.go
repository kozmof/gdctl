package bridge

import (
	"strings"
	"testing"
)

func TestBridgeErrorFormatsScriptDiagnostic(t *testing.T) {
	err := &BridgeError{
		Code:    "SCRIPT_SYNTAX_INVALID",
		Message: "Script did not pass Godot syntax check",
		Detail: map[string]any{
			"path":       "res://scripts/main.gd",
			"line":       float64(24),
			"diagnostic": "The variable type is being inferred from a Variant value.",
			"source": []any{
				map[string]any{"line": float64(22), "text": "func _ready() -> void:", "error": false},
				map[string]any{"line": float64(23), "text": "\tvar data = {}", "error": false},
				map[string]any{"line": float64(24), "text": "\tvar node := data[\"node\"]", "error": true},
			},
		},
	}

	got := err.Error()
	for _, want := range []string{
		"SCRIPT_SYNTAX_INVALID: Script did not pass Godot syntax check",
		"res://scripts/main.gd:24",
		"The variable type is being inferred from a Variant value.",
		">   24 | \tvar node := data[\"node\"]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Error() missing %q:\n%s", want, got)
		}
	}
}
