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

func writeRunStatusResponse(t *testing.T, w http.ResponseWriter, running bool, scene string, helper bridge.RuntimeHelperStatus) {
	t.Helper()
	_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.RunStatusResult]{
		OK: true,
		Result: bridge.RunStatusResult{
			Running:       running,
			PlayingScene:  scene,
			RuntimeHelper: helper,
		},
	})
}

func TestRunInputPreflightFailsBeforeInputWhenHelperAbsent(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{"steps":[{"type":"wait","ms":1}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, true, "res://SpiderGame.tscn", bridge.RuntimeHelperStatus{
				AutoloadConfigured: true,
				Error:              "runtime helper has not checked in",
			})
		case "/run/input":
			t.Fatal("run input should fail before posting /run/input")
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "run", "input", "--file", inputPath), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected helper preflight error")
	}
	for _, want := range []string{"run input requires the gdctl runtime helper", "run is running (res://SpiderGame.tscn)", "runtime helper has not checked in", "restart/reload Godot"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err missing %q:\n%v", want, err)
		}
	}
}

func TestRunWaitProbePreflightFailsBeforeLogsWhenHelperAbsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, false, "", bridge.RuntimeHelperStatus{Error: "runtime helper has not checked in"})
		case "/run/logs":
			t.Fatal("run wait-probe should fail before polling /run/logs")
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "run", "wait-probe", "--source", "runtime.game", "--assert", "score>=1"), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected helper preflight error")
	}
	for _, want := range []string{"run wait-probe requires the gdctl runtime helper", "run is stopped", "runtime helper has not checked in", "gdctl run start --scene <scene>"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err missing %q:\n%v", want, err)
		}
	}
}

func TestRunScreenshotGamePreflightFailsBeforeScreenshotWhenHelperAbsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/status":
			writeRunStatusResponse(t, w, true, "res://SpiderGame.tscn", bridge.RuntimeHelperStatus{Error: "runtime helper has not checked in"})
		case "/run/screenshot":
			t.Fatal("game screenshot should fail before posting /run/screenshot")
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "run", "screenshot", "--source", "game", "--out", filepath.Join(t.TempDir(), "shot.png")), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected helper preflight error")
	}
	for _, want := range []string{"run screenshot requires the gdctl runtime helper", "run is running (res://SpiderGame.tscn)", "runtime helper has not checked in"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err missing %q:\n%v", want, err)
		}
	}
}

func TestRunStatusPlainSaysRunningWithoutRuntimeHelper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		writeRunStatusResponse(t, w, true, "res://SpiderGame.tscn", bridge.RuntimeHelperStatus{Error: "runtime helper has not checked in"})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), append(serverArgs(server), "run", "status"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Run status: running without runtime helper (res://SpiderGame.tscn)") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}
