package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPingUnreachableIsSentinel verifies that a bridge that refuses connections
// surfaces as ErrUnreachable so callers can render actionable guidance instead
// of a raw socket error.
func TestPingUnreachableIsSentinel(t *testing.T) {
	// Bind and immediately close a listener to obtain an address that is
	// guaranteed to refuse connections.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := server.Listener.Addr().String()
	server.Close()

	cfg := Config{Host: addr, Protocol: "http"}
	client := NewClient(cfg)

	_, err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error contacting closed bridge")
	}
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("errors.Is(err, ErrUnreachable) = false; err = %v", err)
	}

	var unreachable *UnreachableError
	if !errors.As(err, &unreachable) {
		t.Fatalf("errors.As(*UnreachableError) = false; err = %v", err)
	}
	if unreachable.Addr != addr {
		t.Fatalf("UnreachableError.Addr = %q, want %q", unreachable.Addr, addr)
	}
	// The message must name the address so users know what to check, and the
	// underlying cause must remain reachable via Unwrap.
	if !strings.Contains(unreachable.Error(), addr) {
		t.Fatalf("UnreachableError message missing address: %q", unreachable.Error())
	}
	if unreachable.Unwrap() == nil {
		t.Fatal("UnreachableError.Unwrap() = nil, want underlying cause")
	}
}

// TestDialUnreachable checks the low-level Dial path also classifies refusal.
func TestDialUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := server.Listener.Addr().String()
	server.Close()

	client := NewClient(Config{Host: addr, Protocol: "http"})
	if err := client.Dial(context.Background()); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("Dial errors.Is(ErrUnreachable) = false; err = %v", err)
	}
}

// TestHTTPErrorSurfacesStatusCode verifies a non-2xx response with no structured
// bridge error body is reported as *HTTPError carrying the status code, letting
// callers distinguish, e.g., auth failures from server faults.
func TestHTTPErrorSurfacesStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("nope"))
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.Listener.Addr().String(), Protocol: "http"})
	_, err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("errors.As(*HTTPError) = false; err = %v", err)
	}
	if httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("HTTPError.StatusCode = %d, want %d", httpErr.StatusCode, http.StatusUnauthorized)
	}
	if !strings.Contains(httpErr.Error(), "401") {
		t.Fatalf("HTTPError message missing status code: %q", httpErr.Error())
	}
}

// TestBridgeErrorBodyTakesPrecedence ensures a structured error body is returned
// as *BridgeError (preserving Code) rather than a generic *HTTPError.
func TestBridgeErrorBodyTakesPrecedence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(BridgeResponse[any]{
			OK:    false,
			Error: &BridgeError{Code: "invalid_params", Message: "bad node path"},
		})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.Listener.Addr().String(), Protocol: "http"})
	_, err := client.Ping(context.Background())
	var bridgeErr *BridgeError
	if !errors.As(err, &bridgeErr) {
		t.Fatalf("errors.As(*BridgeError) = false; err = %v", err)
	}
	if bridgeErr.Code != "invalid_params" {
		t.Fatalf("BridgeError.Code = %q, want invalid_params", bridgeErr.Code)
	}
}

// TestClassifyTransportErrorLeavesContextCancel confirms a user abort is not
// mislabeled as an unreachable bridge.
func TestClassifyTransportErrorLeavesContextCancel(t *testing.T) {
	got := classifyTransportError("127.0.0.1:7777", context.Canceled)
	if errors.Is(got, ErrUnreachable) {
		t.Fatalf("context.Canceled classified as unreachable: %v", got)
	}
	if !errors.Is(got, context.Canceled) {
		t.Fatalf("context.Canceled not preserved: %v", got)
	}
}
