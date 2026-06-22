package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("GDCTL_BRIDGE_HOST", "example.local")
	t.Setenv("GDCTL_BRIDGE_PORT", "9001")
	t.Setenv("GDCTL_BRIDGE_TOKEN", "secret")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.local" {
		t.Fatalf("Host = %q", cfg.Host)
	}
	if cfg.Port != 9001 {
		t.Fatalf("Port = %d", cfg.Port)
	}
	if cfg.Token != "secret" {
		t.Fatalf("Token = %q", cfg.Token)
	}
	if cfg.BaseURL() != "http://example.local:9001" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL())
	}
}

func TestConfigFromEnvInvalidPortReturnsError(t *testing.T) {
	t.Setenv("GDCTL_BRIDGE_PORT", "nope")

	_, err := ConfigFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid port, got nil")
	}
}

func TestWithProjectTokenLoadsTokenWhenUnset(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ProjectTokenFile), []byte("project-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Config{Project: dir}.WithProjectToken()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "project-secret" {
		t.Fatalf("Token = %q", cfg.Token)
	}
}

func TestWithProjectTokenPreservesExplicitToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ProjectTokenFile), []byte("project-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Config{Project: dir, Token: "explicit-secret"}.WithProjectToken()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "explicit-secret" {
		t.Fatalf("Token = %q", cfg.Token)
	}
}
