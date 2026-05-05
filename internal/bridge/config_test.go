package bridge

import "testing"

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("GDCTL_BRIDGE_HOST", "example.local")
	t.Setenv("GDCTL_BRIDGE_PORT", "9001")
	t.Setenv("GDCTL_BRIDGE_TOKEN", "secret")

	cfg := ConfigFromEnv()
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

func TestConfigFromEnvDefaultsInvalidPort(t *testing.T) {
	t.Setenv("GDCTL_BRIDGE_PORT", "nope")

	cfg := ConfigFromEnv()
	if cfg.Host != DefaultHost {
		t.Fatalf("Host = %q", cfg.Host)
	}
	if cfg.Port != DefaultPort {
		t.Fatalf("Port = %d", cfg.Port)
	}
}
