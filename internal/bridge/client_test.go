package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientAddNodeRequest(t *testing.T) {
	var gotAuth string
	var gotEnvelope RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/add" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(BridgeResponse[map[string]any]{
			OK:     true,
			Result: map[string]any{"path": "/root/Main/Marker"},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}
	client := NewClient(cfg)
	result, err := client.AddNode(context.Background(), "cli-test", "/root/Main", "Node2D", "Marker", true)
	if err != nil {
		t.Fatal(err)
	}

	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotEnvelope.RequestID != "cli-test" || gotEnvelope.Op != "node.add" {
		t.Fatalf("envelope = %#v", gotEnvelope)
	}
	if gotEnvelope.Params["dry_run"] != true {
		t.Fatalf("dry_run = %#v", gotEnvelope.Params["dry_run"])
	}
	if result["path"] != "/root/Main/Marker" {
		t.Fatalf("result path = %#v", result["path"])
	}
}

func TestClientBridgeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(BridgeResponse[any]{
			OK: false,
			Error: &BridgeError{
				Code:    "NODE_PARENT_NOT_FOUND",
				Message: "Parent node does not exist",
			},
		})
	}))
	defer server.Close()

	cfg := Config{Host: server.Listener.Addr().String(), Protocol: "http"}
	client := NewClient(cfg)
	_, err := client.RemoveNode(context.Background(), "cli-test", "/root/Main/Missing", false)
	if err == nil {
		t.Fatal("expected error")
	}
	bridgeErr, ok := err.(*BridgeError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if bridgeErr.Code != "NODE_PARENT_NOT_FOUND" {
		t.Fatalf("code = %s", bridgeErr.Code)
	}
}
