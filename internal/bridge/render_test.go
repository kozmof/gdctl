package bridge

import (
	"bytes"
	"testing"
)

func TestRenderTree(t *testing.T) {
	root := NodeInfo{
		Name: "Main",
		Type: "Node2D",
		Children: []NodeInfo{
			{
				Name: "Player",
				Type: "CharacterBody2D",
				Children: []NodeInfo{
					{Name: "Camera2D", Type: "Camera2D"},
				},
			},
		},
	}
	var out bytes.Buffer
	RenderTree(&out, root)
	want := "Main Node2D\n└── Player CharacterBody2D\n    └── Camera2D Camera2D\n"
	if out.String() != want {
		t.Fatalf("tree:\n%s\nwant:\n%s", out.String(), want)
	}
}
