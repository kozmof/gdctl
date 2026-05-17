package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdctl/internal/bridge"
)

func TestRunInputAcceptsMouseMotionRelative(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "input.json")
	body := `{"steps":[{"type":"mouse_motion","relative":[180,-220]},{"type":"wait","ms":0}]}`
	if err := os.WriteFile(inputPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var gotEnvelope bridge.RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/input":
			if err := json.NewDecoder(r.Body).Decode(&gotEnvelope); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{
				OK:     true,
				Result: map[string]any{"queued": true, "job_id": "input-1"},
			})
		case "/jobs/input-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK:  true,
				Job: bridge.Job{ID: "input-1", Kind: "run.input", Status: "succeeded", Result: map[string]any{"steps": 2}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "run", "input", "--file", inputPath), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	steps, ok := gotEnvelope.Params["steps"].([]any)
	if !ok || len(steps) != 2 {
		t.Fatalf("steps = %#v", gotEnvelope.Params["steps"])
	}
}

func TestRunInputValidationRejectsInvalidStepsBeforeNetwork(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "mouse motion dx dy",
			body: `{"steps":[{"type":"mouse_motion","dx":0,"dy":-140}]}`,
			want: "run input step 0 mouse_motion requires relative: [x, y]; got dx/dy",
		},
		{
			name: "mouse motion missing relative",
			body: `{"steps":[{"type":"mouse_motion"}]}`,
			want: "run input step 0 mouse_motion requires relative: [x, y]",
		},
		{
			name: "mouse motion malformed relative",
			body: `{"steps":[{"type":"mouse_motion","relative":[1]}]}`,
			want: "run input step 0 mouse_motion relative must be [x, y]",
		},
		{
			name: "mouse motion non numeric relative",
			body: `{"steps":[{"type":"mouse_motion","relative":[1,"up"]}]}`,
			want: "run input step 0 mouse_motion relative[1] must be numeric",
		},
		{
			name: "invalid key action",
			body: `{"steps":[{"type":"key","key":"W","action":"hold"}]}`,
			want: "run input step 0 key action must be tap, press, or release",
		},
		{
			name: "invalid mouse button action",
			body: `{"steps":[{"type":"mouse_button","button":"left","action":"hold"}]}`,
			want: "run input step 0 mouse_button action must be tap, press, or release",
		},
		{
			name: "invalid action mode",
			body: `{"steps":[{"type":"action","action":"jump","mode":"hold"}]}`,
			want: "run input step 0 action mode must be tap, press, or release",
		},
		{
			name: "negative wait",
			body: `{"steps":[{"type":"wait","ms":-1}]}`,
			want: "run input step 0 wait ms must be non-negative",
		},
		{
			name: "negative duration",
			body: `{"steps":[{"type":"key","key":"W","duration_ms":-1}]}`,
			want: "run input step 0 key duration_ms must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputPath := filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(inputPath, []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/run/input" {
					called = true
				}
				_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[map[string]any]{OK: true, Result: map[string]any{}})
			}))
			defer server.Close()

			var stdout, stderr bytes.Buffer
			err := Run(context.Background(), append(serverArgs(server), "run", "input", "--file", inputPath), &stdout, &stderr)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if called {
				t.Fatal("/run/input was called despite validation failure")
			}
		})
	}
}
