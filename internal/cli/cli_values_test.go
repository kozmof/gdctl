package cli

import (
	"flag"
	"strings"
	"testing"
)

func newFlags(t *testing.T) (*flag.FlagSet, *typedValueFlags) {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	return fs, newTypedValueFlags(fs, "test")
}

func parseFlags(t *testing.T, args []string) *typedValueFlags {
	t.Helper()
	fs, f := newFlags(t)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("flag parse: %v", err)
	}
	return f
}

func TestTypedValueFlagsStringValue(t *testing.T) {
	f := parseFlags(t, []string{"--string", "hello"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", v)
	}
	if m["kind"] != "String" || m["value"] != "hello" {
		t.Fatalf("unexpected value: %#v", m)
	}
}

func TestTypedValueFlagsIntValue(t *testing.T) {
	f := parseFlags(t, []string{"--int", "42"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "int" || m["value"] != 42 {
		t.Fatalf("unexpected value: %#v", m)
	}
}

func TestTypedValueFlagsIntInvalid(t *testing.T) {
	f := parseFlags(t, []string{"--int", "notanint"})
	_, err := f.Value()
	if err == nil || !strings.Contains(err.Error(), "--int") {
		t.Fatalf("expected --int error, got %v", err)
	}
}

func TestTypedValueFlagsFloatValue(t *testing.T) {
	f := parseFlags(t, []string{"--float", "3.14"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "float" {
		t.Fatalf("unexpected kind: %v", m["kind"])
	}
	fv, ok := m["value"].(float64)
	if !ok || fv < 3.13 || fv > 3.15 {
		t.Fatalf("unexpected float value: %v", m["value"])
	}
}

func TestTypedValueFlagsBoolValue(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
	} {
		f := parseFlags(t, []string{"--bool", tc.input})
		v, err := f.Value()
		if err != nil {
			t.Fatalf("input %q: %v", tc.input, err)
		}
		m := v.(map[string]any)
		if m["value"] != tc.want {
			t.Fatalf("input %q: got %v, want %v", tc.input, m["value"], tc.want)
		}
	}
}

func TestTypedValueFlagsVector2(t *testing.T) {
	f := parseFlags(t, []string{"--vector2", "100,200"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Vector2" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsVector3(t *testing.T) {
	f := parseFlags(t, []string{"--vector3", "1,2,3"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Vector3" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsColor(t *testing.T) {
	for _, input := range []string{"1,0,0", "1,0,0,1"} {
		f := parseFlags(t, []string{"--color", input})
		v, err := f.Value()
		if err != nil {
			t.Fatalf("input %q: %v", input, err)
		}
		m := v.(map[string]any)
		if m["kind"] != "Color" {
			t.Fatalf("input %q: kind = %v", input, m["kind"])
		}
	}
}

func TestTypedValueFlagsColorInvalidComponents(t *testing.T) {
	f := parseFlags(t, []string{"--color", "1,0"})
	_, err := f.Value()
	if err == nil || !strings.Contains(err.Error(), "--color") {
		t.Fatalf("expected --color error, got %v", err)
	}
}

func TestTypedValueFlagsNodePath(t *testing.T) {
	f := parseFlags(t, []string{"--node-path", "/root/Main/Player"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "NodePath" || m["value"] != "/root/Main/Player" {
		t.Fatalf("unexpected value: %#v", m)
	}
}

func TestTypedValueFlagsResource(t *testing.T) {
	f := parseFlags(t, []string{"--resource", "res://textures/stone.png"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Resource" || m["value"] != "res://textures/stone.png" {
		t.Fatalf("unexpected value: %#v", m)
	}
}

func TestTypedValueFlagsAABB(t *testing.T) {
	f := parseFlags(t, []string{"--aabb", "1,2,3,4,5,6"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "AABB" {
		t.Fatalf("kind = %v", m["kind"])
	}
	inner := m["value"].(map[string]any)
	if inner["position"] == nil || inner["size"] == nil {
		t.Fatalf("missing position/size in AABB: %#v", inner)
	}
}

func TestTypedValueFlagsArrayString(t *testing.T) {
	f := parseFlags(t, []string{"--array-string", "a;b;c"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Array[String]" {
		t.Fatalf("kind = %v", m["kind"])
	}
	vals := m["value"].([]string)
	if len(vals) != 3 || vals[0] != "a" || vals[2] != "c" {
		t.Fatalf("unexpected values: %v", vals)
	}
}

func TestTypedValueFlagsArrayInt(t *testing.T) {
	f := parseFlags(t, []string{"--array-int", "1;2;3"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Array[int]" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsArrayFloat(t *testing.T) {
	f := parseFlags(t, []string{"--array-float", "1.1;2.2"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Array[float]" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsArrayBool(t *testing.T) {
	f := parseFlags(t, []string{"--array-bool", "true;false;true"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Array[bool]" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsArrayVector2(t *testing.T) {
	f := parseFlags(t, []string{"--array-vector2", "1,2;3,4"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Array[Vector2]" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsArrayVector3(t *testing.T) {
	f := parseFlags(t, []string{"--array-vector3", "1,2,3;4,5,6"})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Array[Vector3]" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsValueJSON(t *testing.T) {
	f := parseFlags(t, []string{"--value", `{"kind":"Vector2","value":[200,400]}`})
	v, err := f.Value()
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["kind"] != "Vector2" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestTypedValueFlagsValueJSONInvalid(t *testing.T) {
	f := parseFlags(t, []string{"--value", "not json"})
	_, err := f.Value()
	if err == nil || !strings.Contains(err.Error(), "--value") {
		t.Fatalf("expected --value error, got %v", err)
	}
}

func TestTypedValueFlagsRequiresAtLeastOne(t *testing.T) {
	_, f := newFlags(t)
	_, err := f.Value()
	if err == nil || !strings.Contains(err.Error(), "requires a value flag") {
		t.Fatalf("expected requires error, got %v", err)
	}
}

func TestTypedValueFlagsRejectsMutuallyExclusive(t *testing.T) {
	fs, f := newFlags(t)
	if err := fs.Parse([]string{"--string", "hello", "--int", "5"}); err != nil {
		t.Fatal(err)
	}
	_, err := f.Value()
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("expected mutual-exclusion error, got %v", err)
	}
}

// parseNameJSONPairs tests

func TestParseNameJSONPairsValid(t *testing.T) {
	out, err := parseNameJSONPairs([]string{`position={"kind":"Vector3","value":[1,2,3]}`, `visible=true`})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["position"]; !ok {
		t.Error("missing position key")
	}
	if out["visible"] != true {
		t.Errorf("visible = %v", out["visible"])
	}
}

func TestParseNameJSONPairsMissingEquals(t *testing.T) {
	_, err := parseNameJSONPairs([]string{"noequals"})
	if err == nil || !strings.Contains(err.Error(), "--prop") {
		t.Fatalf("expected --prop error, got %v", err)
	}
}

func TestParseNameJSONPairsInvalidJSON(t *testing.T) {
	_, err := parseNameJSONPairs([]string{"key=notjson"})
	if err == nil || !strings.Contains(err.Error(), "typed JSON") {
		t.Fatalf("expected typed JSON error, got %v", err)
	}
}

func TestParseNameRawJSONPairsUsesParamLabel(t *testing.T) {
	_, err := parseNameRawJSONPairs([]string{"noequals"})
	if err == nil || !strings.Contains(err.Error(), "--param") {
		t.Fatalf("expected --param error, got %v", err)
	}
}
