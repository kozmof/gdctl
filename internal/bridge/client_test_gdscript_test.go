package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientTestGDScriptRequest(t *testing.T) {
	var got RequestEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test/gdscript" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing auth header")
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(BridgeResponse[GDScriptTestQueuedResult]{
			OK: true,
			Result: GDScriptTestQueuedResult{
				Queued: true,
				JobID:  "test-1",
			},
		})
	}))
	defer server.Close()

	result, err := NewClient(Config{Host: server.Listener.Addr().String(), Protocol: "http", Token: "secret"}).TestGDScript(context.Background(), "cli-test", "res://tests/test_math.gd", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.JobID != "test-1" || !result.Queued {
		t.Fatalf("result = %#v", result)
	}
	if got.Op != "test.gdscript" || got.Params["path"] != "res://tests/test_math.gd" {
		t.Fatalf("envelope = %#v", got)
	}
}

func TestGDScriptTestResultUnmarshal(t *testing.T) {
	data := []byte(`{"passed":false,"total":1,"passed_count":0,"failed_count":1,"duration_ms":12,"files":[{"path":"res://tests/test_fail.gd","passed":false,"total":1,"passed_count":0,"failed_count":1,"duration_ms":12,"tests":[{"name":"test_fail","status":"failed","duration_ms":1,"failures":[{"message":"boom"}]}]}]}`)
	var result GDScriptTestResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Passed || result.FailedCount != 1 || result.Files[0].Tests[0].Failures[0].Message != "boom" {
		t.Fatalf("result = %#v", result)
	}
}
