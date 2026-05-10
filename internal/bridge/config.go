package bridge

import (
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultHost          = "127.0.0.1"
	DefaultPort      int = 7777
	DefaultProtocol      = "http"
	ProjectTokenFile     = ".godot-bridge-token"
)

type Config struct {
	Host       string
	Port       int
	Protocol   string
	Token      string
	Project    string
	GodotPath  string
}

func DefaultConfig() Config {
	return Config{
		Host:     DefaultHost,
		Port:     DefaultPort,
		Protocol: DefaultProtocol,
	}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	if host := os.Getenv("GDCTL_BRIDGE_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("GDCTL_BRIDGE_PORT"); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil && parsed > 0 {
			cfg.Port = parsed
		}
	}
	if token := os.Getenv("GDCTL_BRIDGE_TOKEN"); token != "" {
		cfg.Token = token
	}
	if godot := os.Getenv("GDCTL_GODOT_PATH"); godot != "" {
		cfg.GodotPath = godot
	}
	return cfg
}

func (c Config) WithProjectToken() (Config, error) {
	if c.Token != "" || c.Project == "" {
		return c, nil
	}
	data, err := os.ReadFile(filepath.Join(c.Project, ProjectTokenFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return c, err
	}
	c.Token = strings.TrimSpace(string(data))
	return c, nil
}

func (c Config) BaseURL() string {
	u := url.URL{
		Scheme: c.Protocol,
		Host:   c.hostPort(),
	}
	return u.String()
}

func (c Config) Address() string {
	return c.hostPort()
}

func (c Config) hostPort() string {
	if _, _, err := net.SplitHostPort(c.Host); err == nil {
		return c.Host
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}
