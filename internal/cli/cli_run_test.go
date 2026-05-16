package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"gdctl/internal/bridge"
)

// filterLogEntries tests

func logEntry(source, msg string) bridge.LogEntry {
	return bridge.LogEntry{Source: source, Message: msg, Time: "2024-01-01T00:00:00Z", Level: "info"}
}

func TestFilterLogEntriesLatestKeepsNewest(t *testing.T) {
	entries := []bridge.LogEntry{
		logEntry("gdscript", "first"),
		logEntry("gdscript", "second"),
		logEntry("engine", "only"),
		logEntry("gdscript", "third"),
	}
	got := filterLogEntries(entries, "", true, false)
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d: %#v", len(got), got)
	}
	// slices.Reverse restores original relative order: engine before gdscript
	if got[0].Source != "engine" || got[0].Message != "only" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].Source != "gdscript" || got[1].Message != "third" {
		t.Errorf("got[1] = %+v", got[1])
	}
}

func TestFilterLogEntriesLatestPreservesOrder(t *testing.T) {
	entries := []bridge.LogEntry{
		logEntry("a", "1"),
		logEntry("b", "2"),
		logEntry("a", "3"),
		logEntry("c", "4"),
		logEntry("b", "5"),
	}
	got := filterLogEntries(entries, "", true, false)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	// last occurrences: a@3, c@4, b@5 — after reverse they should appear in that order
	if got[0].Source != "a" || got[1].Source != "c" || got[2].Source != "b" {
		t.Errorf("order wrong: %v %v %v", got[0].Source, got[1].Source, got[2].Source)
	}
}

func TestFilterLogEntriesSourceFilter(t *testing.T) {
	entries := []bridge.LogEntry{
		logEntry("engine", "e1"),
		logEntry("gdscript", "g1"),
		logEntry("engine", "e2"),
	}
	got := filterLogEntries(entries, "engine", false, false)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	for _, e := range got {
		if e.Source != "engine" {
			t.Errorf("unexpected source: %s", e.Source)
		}
	}
}

func TestFilterLogEntriesSinceStart(t *testing.T) {
	entries := []bridge.LogEntry{
		{Source: "engine", Message: "before", Time: "2024-01-01T00:00:01Z", Level: "info"},
		{Source: "run.start", Message: "started", Time: "2024-01-01T00:00:02Z", Level: "info"},
		{Source: "engine", Message: "after", Time: "2024-01-01T00:00:03Z", Level: "info"},
	}
	got := filterLogEntries(entries, "", false, true)
	if len(got) != 1 || got[0].Message != "after" {
		t.Fatalf("want [after], got %+v", got)
	}
}

func TestFilterLogEntriesNoLatestKeepsAll(t *testing.T) {
	entries := []bridge.LogEntry{
		logEntry("src", "a"),
		logEntry("src", "b"),
		logEntry("src", "c"),
	}
	got := filterLogEntries(entries, "", false, false)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
}

// waitForJob tests

func clientForServer(server *httptest.Server) *bridge.Client {
	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.SplitN(hostPort, ":", 2)
	port := bridge.DefaultPort
	if len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}
	return bridge.NewClient(bridge.Config{
		Host:     parts[0],
		Port:     port,
		Protocol: "http",
	})
}

func TestWaitForJobSucceeds(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := "pending"
		if calls >= 2 {
			status = "succeeded"
		}
		_ = json.NewEncoder(w).Encode(bridge.JobResponse{
			OK: true,
			Job: bridge.Job{
				ID:     "job-1",
				Status: status,
				Result: map[string]any{"path": "res://main.tscn"},
			},
		})
	}))
	defer server.Close()

	client := clientForServer(server)
	job, err := waitForJob(context.Background(), client, "job-1", 5*time.Second, "test")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "succeeded" {
		t.Fatalf("status = %q", job.Status)
	}
}

func TestWaitForJobContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.JobResponse{
			OK:  true,
			Job: bridge.Job{ID: "job-x", Status: "pending"},
		})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	client := clientForServer(server)
	_, err := waitForJob(ctx, client, "job-x", 10*time.Second, "test")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if err != context.Canceled {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestWaitForJobTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.JobResponse{
			OK:  true,
			Job: bridge.Job{ID: "job-y", Status: "pending"},
		})
	}))
	defer server.Close()

	client := clientForServer(server)
	_, err := waitForJob(context.Background(), client, "job-y", 50*time.Millisecond, "test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout in error: %v", err)
	}
}

func TestWaitForJobFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.JobResponse{
			OK: true,
			Job: bridge.Job{
				ID:     "job-z",
				Status: "failed",
				Error:  &bridge.BridgeError{Code: "FAIL", Message: "it broke"},
			},
		})
	}))
	defer server.Close()

	client := clientForServer(server)
	_, err := waitForJob(context.Background(), client, "job-z", 5*time.Second, "test")
	if err == nil {
		t.Fatal("expected error for failed job")
	}
	if !strings.Contains(err.Error(), "FAIL") {
		t.Fatalf("expected FAIL in error: %v", err)
	}
}
