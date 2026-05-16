package bridge

import (
	"encoding/json"
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

func TestBridgeErrorDebuggerInSuffix(t *testing.T) {
	err := &BridgeError{
		Code:    "RUNTIME_ERROR",
		Message: "Script error occurred",
		Detail: map[string]any{
			"path": "res://scripts/enemy.gd",
			"line": float64(10),
			"debugger": map[string]any{
				"paused": true,
				"file":   "res://scripts/enemy.gd",
				"line":   float64(10),
			},
		},
	}
	got := err.Error()
	// debugger context must appear inside the parenthesised suffix, not after it
	parenStart := strings.Index(got, "(")
	parenEnd := strings.Index(got, ")")
	if parenStart < 0 || parenEnd < 0 {
		t.Fatalf("expected parenthesised suffix in %q", got)
	}
	suffix := got[parenStart : parenEnd+1]
	if !strings.Contains(suffix, "debugger paused at") {
		t.Fatalf("debugger context not inside suffix parens: %q", got)
	}
}

func TestRunStatusResultUnmarshalJSONFlattened(t *testing.T) {
	raw := `{
		"running": true,
		"runtime_helper_present": true,
		"runtime_helper_autoload_configured": true,
		"runtime_helper_last_seen": "2024-01-01T00:00:00Z",
		"runtime_helper_error": "connection refused"
	}`
	var r RunStatusResult
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Running {
		t.Error("Running should be true")
	}
	if !r.RuntimeHelper.Present {
		t.Error("RuntimeHelper.Present should be true from flat field")
	}
	if !r.RuntimeHelper.AutoloadConfigured {
		t.Error("RuntimeHelper.AutoloadConfigured should be true from flat field")
	}
	if r.RuntimeHelper.LastSeen != "2024-01-01T00:00:00Z" {
		t.Errorf("RuntimeHelper.LastSeen = %q", r.RuntimeHelper.LastSeen)
	}
	if r.RuntimeHelper.Error != "connection refused" {
		t.Errorf("RuntimeHelper.Error = %q", r.RuntimeHelper.Error)
	}
}

func TestRunStatusResultUnmarshalJSONNestedWins(t *testing.T) {
	// When the nested runtime_helper struct is present, it takes precedence over flat fields.
	raw := `{
		"running": true,
		"runtime_helper": {"present": true, "last_seen": "nested-value"},
		"runtime_helper_present": false,
		"runtime_helper_last_seen": "flat-value"
	}`
	var r RunStatusResult
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if !r.RuntimeHelper.Present {
		t.Error("RuntimeHelper.Present should stay true from nested struct")
	}
	if r.RuntimeHelper.LastSeen != "nested-value" {
		t.Errorf("RuntimeHelper.LastSeen = %q, want nested-value", r.RuntimeHelper.LastSeen)
	}
}

func TestRunStatusResultUnmarshalJSONEmpty(t *testing.T) {
	raw := `{"running": false}`
	var r RunStatusResult
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.RuntimeHelper.Present || r.RuntimeHelper.AutoloadConfigured {
		t.Error("RuntimeHelper fields should be zero when absent")
	}
}
