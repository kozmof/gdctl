package bridge

import (
	"net"
	"net/url"
	"os"
	"strconv"
)

const (
	DefaultHost         = "host.docker.internal"
	DefaultPort     int = 7777
	DefaultProtocol     = "http"
)

type Config struct {
	Host     string
	Port     int
	Protocol string
	Token    string
	Project  string
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
	return cfg
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
