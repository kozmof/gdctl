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

func TestTestShorthandDir(t *testing.T) {
	var got bridge.RequestEnvelope
	server := gdscriptTestServer(t, func(r *http.Request, env bridge.RequestEnvelope) {
		got = env
	})
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "test", "--dir", "res://tests"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if got.Op != "test.gdscript" || got.Params["dir"] != "res://tests" {
		t.Fatalf("envelope = %#v", got)
	}
	if !strings.Contains(stdout.String(), "PASS gdscript tests") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestTestRequiresGDScriptSelector(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"test"}, &stdout, &stderr); err == nil {
		t.Fatal("Run succeeded, want selector error")
	}
}

func TestTestGoIsNotUserCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"test", "go"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unknown test command: go") {
		t.Fatalf("err = %v, want unknown test command", err)
	}
}

func TestGDScriptTestPath(t *testing.T) {
	var got bridge.RequestEnvelope
	server := gdscriptTestServer(t, func(r *http.Request, env bridge.RequestEnvelope) {
		got = env
	})
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "test", "gdscript", "--path", "res://tests/test_math.gd"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if got.Op != "test.gdscript" || got.Params["path"] != "res://tests/test_math.gd" {
		t.Fatalf("envelope = %#v", got)
	}
	if !strings.Contains(stdout.String(), "PASS gdscript tests: 1 passed, 0 failed, 1 total") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestGDScriptTestDir(t *testing.T) {
	var got bridge.RequestEnvelope
	server := gdscriptTestServer(t, func(r *http.Request, env bridge.RequestEnvelope) {
		got = env
	})
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "test", "gdscript", "--dir", "res://tests"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if got.Op != "test.gdscript" || got.Params["dir"] != "res://tests" {
		t.Fatalf("envelope = %#v", got)
	}
}

func TestGDScriptTestValidation(t *testing.T) {
	tests := [][]string{
		{"test", "gdscript"},
		{"test", "gdscript", "--path", "res://tests/test_a.gd", "--dir", "res://tests"},
		{"test", "gdscript", "--path", "local.gd"},
		{"test", "gdscript", "--dir", "local"},
	}
	for _, args := range tests {
		var stdout, stderr bytes.Buffer
		if err := Run(context.Background(), args, &stdout, &stderr); err == nil {
			t.Fatalf("Run(%v) succeeded, want error", args)
		}
	}
}

func TestGDScriptTestFailureOutput(t *testing.T) {
	server := gdscriptTestServerWithResult(t, bridge.GDScriptTestResult{
		Passed:      false,
		Total:       1,
		FailedCount: 1,
		Files: []bridge.GDScriptTestFile{{
			Path:        "res://tests/test_fail.gd",
			Passed:      false,
			Total:       1,
			FailedCount: 1,
			Tests: []bridge.GDScriptTestCase{{
				Name:   "test_nope",
				Status: "failed",
				Failures: []bridge.GDScriptTestFailure{{
					Message: "Expected 1 to equal 2",
				}},
			}},
		}},
	})
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "test", "gdscript", "--path", "res://tests/test_fail.gd"), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected failing tests to return error")
	}
	for _, want := range []string{"FAIL gdscript tests", "res://tests/test_fail.gd", "test_nope", "Expected 1 to equal 2"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestGDScriptTestJSON(t *testing.T) {
	server := gdscriptTestServer(t, nil)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), append(serverArgs(server), "--token", "secret", "test", "gdscript", "--path", "res://tests/test_math.gd", "--json"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	var result bridge.GDScriptTestResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !result.Passed || result.Total != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func gdscriptTestServer(t *testing.T, capture func(*http.Request, bridge.RequestEnvelope)) *httptest.Server {
	t.Helper()
	return gdscriptTestServerWithResult(t, bridge.GDScriptTestResult{
		Passed:      true,
		Total:       1,
		PassedCount: 1,
		Files: []bridge.GDScriptTestFile{{
			Path:        "res://tests/test_math.gd",
			Passed:      true,
			Total:       1,
			PassedCount: 1,
			Tests:       []bridge.GDScriptTestCase{{Name: "test_add", Status: "passed"}},
		}},
	}, capture)
}

func gdscriptTestServerWithResult(t *testing.T, result bridge.GDScriptTestResult, captures ...func(*http.Request, bridge.RequestEnvelope)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test/gdscript":
			var env bridge.RequestEnvelope
			_ = json.NewDecoder(r.Body).Decode(&env)
			for _, capture := range captures {
				if capture != nil {
					capture(r, env)
				}
			}
			_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.GDScriptTestQueuedResult]{
				OK: true,
				Result: bridge.GDScriptTestQueuedResult{
					Queued: true,
					JobID:  "test-1",
				},
			})
		case "/jobs/test-1":
			_ = json.NewEncoder(w).Encode(bridge.JobResponse{
				OK: true,
				Job: bridge.Job{
					ID:     "test-1",
					Kind:   "test.gdscript",
					Status: "succeeded",
					Result: mustMap(t, result),
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
