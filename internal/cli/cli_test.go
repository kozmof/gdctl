package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gdctl/internal/bridge"
)

func TestRunPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.PingResponse{
			OK:            true,
			Engine:        "Godot",
			EngineVersion: "4.4.1",
			PluginVersion: "0.1.0",
			ProjectName:   "my-game",
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "ping"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Godot bridge: ok", "Engine: Godot 4.4.1", "Project: my-game", "Plugin: 0.1.0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunSceneTree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.SceneTreeResponse{
			OK: true,
			Root: bridge.NodeInfo{
				Name: "Main",
				Type: "Node2D",
				Children: []bridge.NodeInfo{
					{Name: "Player", Type: "CharacterBody2D"},
				},
			},
		})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "scene", "tree"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "└── Player CharacterBody2D") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestNodeAddRequiresFlagsBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "add", "--parent", "/root/Main"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--parent, --type, and --name") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeRemoveRequiresPathBeforeNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"node", "remove"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("err = %v", err)
	}
}

func serverArgs(server *httptest.Server) []string {
	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(hostPort, ":")
	return []string{"--host", parts[0], "--port", parts[1]}
}
