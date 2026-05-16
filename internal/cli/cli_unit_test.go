package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	addonPkg "gdctl/internal/addon"
	"gdctl/internal/bridge"
)

// ---------------------------------------------------------------------------
// resolveAssertPredicate
// ---------------------------------------------------------------------------

func TestResolveAssertPredicateExpr(t *testing.T) {
	out, err := resolveAssertPredicate("key>=1", "", "", "")
	if err != nil || out != "key>=1" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestResolveAssertPredicateSplit(t *testing.T) {
	out, err := resolveAssertPredicate("", "health", ">=", "50")
	if err != nil || out != "health>=50" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestResolveAssertPredicateSplitAllOps(t *testing.T) {
	for _, op := range []string{">=", "<=", "==", "!=", ">", "<"} {
		out, err := resolveAssertPredicate("", "k", op, "v")
		if err != nil {
			t.Errorf("op %s: %v", op, err)
		}
		if !strings.Contains(out, op) {
			t.Errorf("op %s: output %q missing op", op, out)
		}
	}
}

func TestResolveAssertPredicateBothError(t *testing.T) {
	_, err := resolveAssertPredicate("key>=1", "key", ">=", "1")
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("expected combined error, got %v", err)
	}
}

func TestResolveAssertPredicateIncompleteError(t *testing.T) {
	_, err := resolveAssertPredicate("", "key", ">=", "")
	if err == nil || !strings.Contains(err.Error(), "together") {
		t.Fatalf("expected together error, got %v", err)
	}
}

func TestResolveAssertPredicateInvalidOp(t *testing.T) {
	_, err := resolveAssertPredicate("", "key", "~=", "1")
	if err == nil || !strings.Contains(err.Error(), "--assert-op") {
		t.Fatalf("expected assert-op error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// evalPredicate
// ---------------------------------------------------------------------------

func TestEvalPredicateFloat(t *testing.T) {
	detail := map[string]any{"health": float64(75)}
	cases := []struct {
		op   string
		val  string
		want bool
	}{
		{">=", "50", true},
		{"<=", "50", false},
		{">", "74", true},
		{"<", "74", false},
		{"==", "75", true},
		{"!=", "75", false},
	}
	for _, tc := range cases {
		got := evalPredicate(detail, "health", tc.op, tc.val)
		if got != tc.want {
			t.Errorf("op %s %s: got %v, want %v", tc.op, tc.val, got, tc.want)
		}
	}
}

func TestEvalPredicateIntValue(t *testing.T) {
	detail := map[string]any{"count": int(5)}
	if !evalPredicate(detail, "count", "==", "5") {
		t.Error("int==5 should be true")
	}
}

func TestEvalPredicateInt64Value(t *testing.T) {
	detail := map[string]any{"count": int64(10)}
	if !evalPredicate(detail, "count", ">", "9") {
		t.Error("int64>9 should be true")
	}
}

func TestEvalPredicateBoolValue(t *testing.T) {
	detail := map[string]any{"alive": true}
	if !evalPredicate(detail, "alive", "==", "1") {
		t.Error("bool true == 1 should be true")
	}
}

func TestEvalPredicateStringComparison(t *testing.T) {
	detail := map[string]any{"name": "player"}
	if !evalPredicate(detail, "name", "==", "player") {
		t.Error("string == should match")
	}
	if !evalPredicate(detail, "name", "!=", "enemy") {
		t.Error("string != should not match")
	}
}

func TestEvalPredicateMissingKey(t *testing.T) {
	detail := map[string]any{}
	if evalPredicate(detail, "missing", "==", "x") {
		t.Error("missing key should return false")
	}
}

func TestEvalPredicateUnknownTypeNotNumeric(t *testing.T) {
	detail := map[string]any{"val": []int{1, 2, 3}}
	// val is a slice — not numeric, string comparison fallback: "[1 2 3]" != "1"
	got := evalPredicate(detail, "val", "==", "1")
	if got {
		t.Error("slice value should not equal '1'")
	}
}

// ---------------------------------------------------------------------------
// parseColorValue
// ---------------------------------------------------------------------------

func TestParseColorValueRGB(t *testing.T) {
	c, err := parseColorValue("1,0,0")
	if err != nil {
		t.Fatal(err)
	}
	if c[0] != 1 || c[1] != 0 || c[2] != 0 || c[3] != 1 {
		t.Fatalf("unexpected color: %v", c)
	}
}

func TestParseColorValueRGBA(t *testing.T) {
	c, err := parseColorValue("1,0,0,0.5")
	if err != nil {
		t.Fatal(err)
	}
	if c[3] != 0.5 {
		t.Fatalf("unexpected alpha: %v", c[3])
	}
}

func TestParseColorValueHex6(t *testing.T) {
	c, err := parseColorValue("ff0000")
	if err != nil {
		t.Fatal(err)
	}
	if c[0] != 1 || c[1] != 0 || c[2] != 0 {
		t.Fatalf("unexpected hex color: %v", c)
	}
}

func TestParseColorValueHex8(t *testing.T) {
	c, err := parseColorValue("ff000080")
	if err != nil {
		t.Fatal(err)
	}
	if c[3] < 0.49 || c[3] > 0.51 {
		t.Fatalf("unexpected alpha: %v", c[3])
	}
}

func TestParseColorValueHexPrefix(t *testing.T) {
	_, err := parseColorValue("#ff0000")
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseColorValueInvalidCSV(t *testing.T) {
	_, err := parseColorValue("1,0")
	if err == nil {
		t.Fatal("expected error for 2-component color")
	}
}

func TestParseColorValueInvalidHex(t *testing.T) {
	_, err := parseColorValue("ggrrbb")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestParseColorValueInvalidHexLength(t *testing.T) {
	_, err := parseColorValue("fff")
	if err == nil {
		t.Fatal("expected error for wrong hex length")
	}
}

func TestParseColorValueInvalidComponent(t *testing.T) {
	_, err := parseColorValue("red,green,blue")
	if err == nil {
		t.Fatal("expected error for non-numeric component")
	}
}

// ---------------------------------------------------------------------------
// floatFromResult
// ---------------------------------------------------------------------------

func TestFloatFromResult(t *testing.T) {
	if floatFromResult(float64(3.14)) != 3.14 {
		t.Error("float64 case")
	}
	if floatFromResult(float32(2.5)) != float64(float32(2.5)) {
		t.Error("float32 case")
	}
	if floatFromResult("string") != 0 {
		t.Error("unknown type should be 0")
	}
}

// ---------------------------------------------------------------------------
// modeValue
// ---------------------------------------------------------------------------

func TestModeValue(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"off", 0},
		{"blur", 0.33},
		{"bleed", 0.66},
		{"preserve", 1.0},
		{"ink-preserve", 1.0},
		{"crisp", 1.0},
		{"unknown", 0},
	}
	for _, tc := range cases {
		got := modeValue(tc.input)
		if got != tc.want {
			t.Errorf("modeValue(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// byteFromUnit
// ---------------------------------------------------------------------------

func TestByteFromUnit(t *testing.T) {
	if byteFromUnit(0) != 0 {
		t.Error("0 should be 0")
	}
	if byteFromUnit(1) != 255 {
		t.Error("1 should be 255")
	}
	if byteFromUnit(-5) != 0 {
		t.Error("negative should clamp to 0")
	}
	if byteFromUnit(5) != 255 {
		t.Error(">1 should clamp to 255")
	}
}

// ---------------------------------------------------------------------------
// autoload list variants
// ---------------------------------------------------------------------------

func TestAutoloadListWithAutoloads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.AutoloadListResult]{
			OK: true,
			Result: bridge.AutoloadListResult{
				Autoloads: []bridge.AutoloadResult{
					{Name: "GameManager", Path: "res://autoloads/game_manager.gd"},
					{Name: "AudioManager", Path: "res://autoloads/audio.gd"},
				},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "autoload", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GameManager") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestAutoloadListJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bridge.BridgeResponse[bridge.AutoloadListResult]{
			OK: true,
			Result: bridge.AutoloadListResult{
				Autoloads: []bridge.AutoloadResult{
					{Name: "GameManager", Path: "res://autoloads/game_manager.gd"},
				},
			},
		})
	}))
	defer server.Close()
	out, err := runCmd(t, server, "autoload", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected JSON, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// run logs variants
// ---------------------------------------------------------------------------

func makeLogsServer(entries []bridge.LogEntry) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run/logs":
			_ = json.NewEncoder(w).Encode(bridge.LogsResponse{OK: true, Entries: entries})
		default:
			// clear logs endpoint
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
}

func TestRunLogsWithEntries(t *testing.T) {
	entries := []bridge.LogEntry{
		{Time: "2024-01-01T00:00:00Z", Level: "info", Source: "engine", Message: "started"},
	}
	server := makeLogsServer(entries)
	defer server.Close()
	out, err := runCmd(t, server, "run", "logs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "started") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunLogsJSON(t *testing.T) {
	entries := []bridge.LogEntry{
		{Time: "2024-01-01T00:00:00Z", Level: "info", Source: "engine", Message: "hello"},
	}
	server := makeLogsServer(entries)
	defer server.Close()
	out, err := runCmd(t, server, "run", "logs", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("expected JSON, got: %s", out)
	}
}

func TestRunLogsWithDetail(t *testing.T) {
	entries := []bridge.LogEntry{
		{Time: "2024-01-01T00:00:00Z", Level: "info", Source: "engine", Message: "event",
			Detail: map[string]any{"key": "value"}},
	}
	server := makeLogsServer(entries)
	defer server.Close()
	out, err := runCmd(t, server, "run", "logs")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "key") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestRunLogsClear(t *testing.T) {
	entries := []bridge.LogEntry{
		{Time: "2024-01-01T00:00:00Z", Level: "info", Source: "engine", Message: "msg"},
	}
	server := makeLogsServer(entries)
	defer server.Close()
	_, err := runCmd(t, server, "run", "logs", "--clear")
	if err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// run probe subcommands
// ---------------------------------------------------------------------------

func TestRunProbeRequiresSubcommand(t *testing.T) {
	err := Run(context.Background(), []string{"run", "probe"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunProbeUnknownSubcommand(t *testing.T) {
	err := Run(context.Background(), []string{"run", "probe", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown run probe") {
		t.Fatalf("expected unknown probe error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// input subcommand dispatching
// ---------------------------------------------------------------------------

func TestInputActionUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"input", "action", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown input action") {
		t.Fatalf("expected unknown input action error, got %v", err)
	}
}

func TestInputEventUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"input", "event", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown input event") {
		t.Fatalf("expected unknown input event error, got %v", err)
	}
}

func TestInputEventRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"input", "event"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInputActionRequiresSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"input", "action"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInputUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"input", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown input subcommand") {
		t.Fatalf("expected unknown input subcommand error, got %v", err)
	}
}

func TestInputEventAddKey(t *testing.T) {
	server := singleHandler("/input/event-add-key", map[string]any{"action": "jump", "event_added": true})
	defer server.Close()
	out, err := runCmd(t, server, "input", "event", "add-key", "--action", "jump", "--key", "Space")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "jump") {
		t.Fatalf("stdout: %s", out)
	}
}

func TestInputActionAdd(t *testing.T) {
	server := singleHandler("/input/action-add", map[string]any{"action": "jump", "deadzone": 0.5})
	defer server.Close()
	out, err := runCmd(t, server, "input", "action", "add", "--name", "jump")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "jump") {
		t.Fatalf("stdout: %s", out)
	}
}

// ---------------------------------------------------------------------------
// runAddon missing branches
// ---------------------------------------------------------------------------

func TestAddonInstall(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"addon", "install", "--project", project}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddonRemove(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	// Install first, then remove
	mgr := newAddonManager()
	if _, err := mgr.Install(addonPkg.InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"addon", "remove", "--project", project}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddonEnable(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	mgr := newAddonManager()
	if _, err := mgr.Install(addonPkg.InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"addon", "enable", "--project", project}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddonDisable(t *testing.T) {
	useTestAddon(t)
	project := newCLIProject(t)
	mgr := newAddonManager()
	if _, err := mgr.Install(addonPkg.InstallOptions{ProjectPath: project}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Enable(project); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"addon", "disable", "--project", project}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddonRollbackRequiresProject(t *testing.T) {
	err := Run(context.Background(), []string{"addon", "rollback"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("expected --project error, got %v", err)
	}
}

func TestAddonUnknownSubcmd(t *testing.T) {
	err := Run(context.Background(), []string{"addon", "unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown addon command") {
		t.Fatalf("expected unknown addon command error, got %v", err)
	}
}
