package cli

import (
	"reflect"
	"testing"

	"gdctl/internal/bridge"
	"gdctl/internal/policy"
)

func TestExtractPositionalArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPos  string
		wantRest []string
	}{
		{"empty", nil, "", nil},
		{"leading positional", []string{"main", "--flag"}, "main", []string{"--flag"}},
		{"only dashed tokens", []string{"--flag", "--other"}, "", []string{"--flag", "--other"}},
		{"first of several positionals", []string{"a", "b"}, "a", []string{"b"}},
		// The helper is intentionally naive: it does not know which flags take a
		// value, so the first non-dash token wins even when it is a flag argument.
		{"flag value taken as positional", []string{"--scene", "x", "main"}, "x", []string{"--scene", "main"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPos, gotRest := extractPositionalArg(tt.args)
			if gotPos != tt.wantPos {
				t.Errorf("positional = %q, want %q", gotPos, tt.wantPos)
			}
			if len(gotRest) != len(tt.wantRest) {
				t.Fatalf("rest = %v, want %v", gotRest, tt.wantRest)
			}
			for i := range gotRest {
				if gotRest[i] != tt.wantRest[i] {
					t.Errorf("rest[%d] = %q, want %q", i, gotRest[i], tt.wantRest[i])
				}
			}
		})
	}
}

func TestExtractPositionalArgDoesNotMutateInput(t *testing.T) {
	args := []string{"--scene", "x", "main", "--flag"}
	orig := append([]string(nil), args...)
	extractPositionalArg(args)
	if !reflect.DeepEqual(args, orig) {
		t.Errorf("input mutated: got %v, want %v", args, orig)
	}
}

func TestCategorizeFiles(t *testing.T) {
	files := []string{
		"b.gd", "a.gd",
		"scene.tscn", "packed.scn",
		"tex.PNG", "icon.svg",
		"mat.tres", "data.res",
		"world.gdshader",
		"beep.wav", "music.ogg",
		"notes.txt", "noext",
	}
	got := categorizeFiles(files)

	if want := []string{"a.gd", "b.gd"}; !reflect.DeepEqual(got["scripts"], want) {
		t.Errorf("scripts = %v, want %v (should be sorted)", got["scripts"], want)
	}
	if len(got["scenes"]) != 2 {
		t.Errorf("scenes = %v, want 2 entries", got["scenes"])
	}
	if len(got["textures"]) != 2 { // .PNG must match case-insensitively
		t.Errorf("textures = %v, want 2 entries", got["textures"])
	}
	if len(got["resources"]) != 2 {
		t.Errorf("resources = %v, want 2 entries", got["resources"])
	}
	if len(got["shaders"]) != 1 {
		t.Errorf("shaders = %v, want 1 entry", got["shaders"])
	}
	if len(got["audio"]) != 2 {
		t.Errorf("audio = %v, want 2 entries", got["audio"])
	}
	if want := []string{"noext", "notes.txt"}; !reflect.DeepEqual(got["other"], want) {
		t.Errorf("other = %v, want %v", got["other"], want)
	}
}

func TestExtensionLabel(t *testing.T) {
	for _, cat := range []string{"scripts", "scenes", "textures", "resources", "shaders", "audio", "other"} {
		if extensionLabel(cat) == "" {
			t.Errorf("extensionLabel(%q) returned empty", cat)
		}
	}
	if got := extensionLabel("unknown"); got != "unknown" {
		t.Errorf("extensionLabel fallback = %q, want the input echoed back", got)
	}
}

func TestEnvironmentBackgroundModeInt(t *testing.T) {
	valid := map[string]int{
		"clear": 0, "clear_color": 0, "COLOR": 1, "sky": 2,
		"canvas": 3, "keep": 4, "camera_feed": 5,
	}
	for mode, want := range valid {
		got, err := environmentBackgroundModeInt(mode)
		if err != nil {
			t.Errorf("environmentBackgroundModeInt(%q) unexpected error: %v", mode, err)
		}
		if got != want {
			t.Errorf("environmentBackgroundModeInt(%q) = %d, want %d", mode, got, want)
		}
	}
	if _, err := environmentBackgroundModeInt("bogus"); err == nil {
		t.Error("environmentBackgroundModeInt(bogus) = nil error, want error")
	}
}

func TestLightNodeTypes(t *testing.T) {
	if got := light3DNodeType("omni"); got != "OmniLight3D" {
		t.Errorf("light3DNodeType(omni) = %q", got)
	}
	if got := light3DNodeType("SPOT"); got != "SpotLight3D" {
		t.Errorf("light3DNodeType(SPOT) = %q", got)
	}
	if got := light3DNodeType("anything"); got != "DirectionalLight3D" {
		t.Errorf("light3DNodeType default = %q, want DirectionalLight3D", got)
	}
	if got := light2DNodeType("point"); got != "PointLight2D" {
		t.Errorf("light2DNodeType(point) = %q", got)
	}
	if got := light2DNodeType("x"); got != "DirectionalLight2D" {
		t.Errorf("light2DNodeType default = %q, want DirectionalLight2D", got)
	}
}

func TestParseFloatComponents(t *testing.T) {
	got, err := parseFloatComponents("1.5, -2, 0", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []float64{1.5, -2, 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := parseFloatComponents("1,2", 3); err == nil {
		t.Error("wrong count = nil error, want error")
	}
	if _, err := parseFloatComponents("1,x,3", 3); err == nil {
		t.Error("non-numeric = nil error, want error")
	}
}

func TestIndexTree(t *testing.T) {
	tree := bridge.NodeInfo{
		Path: ".", Type: "Node3D",
		Children: []bridge.NodeInfo{
			{Path: "Car", Type: "VehicleBody3D", Children: []bridge.NodeInfo{
				{Path: "Car/Camera", Type: "Camera3D"},
			}},
			{Path: "Ground", Type: "StaticBody3D"},
		},
	}
	index := map[string]string{}
	indexTree(tree, index)

	want := map[string]string{
		".":          "Node3D",
		"Car":        "VehicleBody3D",
		"Car/Camera": "Camera3D",
		"Ground":     "StaticBody3D",
	}
	if !reflect.DeepEqual(index, want) {
		t.Errorf("index = %v, want %v", index, want)
	}
}

func TestViolationsToAny(t *testing.T) {
	got := violationsToAny([]policy.Violation{
		{Rule: "no-orphans", Path: "Foo", Message: "orphan node"},
	})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	want := map[string]string{"rule": "no-orphans", "path": "Foo", "message": "orphan node"}
	if !reflect.DeepEqual(got[0], want) {
		t.Errorf("got %v, want %v", got[0], want)
	}

	if got := violationsToAny(nil); len(got) != 0 {
		t.Errorf("nil input = %v, want empty", got)
	}
}
