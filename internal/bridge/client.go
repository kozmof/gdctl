package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// DefaultRequestTimeout bounds how long a single bridge request may block when
// the caller has not supplied its own context deadline. It guards against a
// hung or unresponsive Godot editor holding the CLI open indefinitely.
const DefaultRequestTimeout = 60 * time.Second

type Client struct {
	cfg        Config
	httpClient *http.Client
	// requestTimeout is applied per request only when the incoming context has
	// no deadline of its own, so callers that set a shorter or longer deadline
	// keep full control.
	requestTimeout time.Duration
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 3 * time.Second,
				}).DialContext,
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 4,
				MaxConnsPerHost:     8,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		requestTimeout: DefaultRequestTimeout,
	}
}

func NewClientWithHTTP(cfg Config, httpClient *http.Client) *Client {
	return &Client{cfg: cfg, httpClient: httpClient, requestTimeout: DefaultRequestTimeout}
}

// requestContext derives a context carrying the client's default request
// timeout, but only when the caller has not already set a deadline. The
// returned cancel func must always be called.
func (c *Client) requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.requestTimeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.requestTimeout)
}

// Phase 4 client methods

func (c *Client) Dial(ctx context.Context) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", c.cfg.Address())
	if err != nil {
		return classifyTransportError(c.cfg.Address(), err)
	}
	return conn.Close()
}

func (c *Client) getJSON(ctx context.Context, path string, target any) error {
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL()+path, nil)
	if err != nil {
		return err
	}
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return classifyTransportError(c.cfg.Address(), err)
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeResponse(resp, target)
}

func (c *Client) postEnvelope(ctx context.Context, path string, env RequestEnvelope) (map[string]any, error) {
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	body, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL()+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, classifyTransportError(c.cfg.Address(), err)
	}
	defer func() { _ = resp.Body.Close() }()

	var out BridgeResponse[map[string]any]
	if err := decodeResponse(resp, &out); err != nil {
		return nil, err
	}
	if !out.OK {
		if out.Error != nil {
			return nil, out.Error
		}
		return nil, fmt.Errorf("bridge request failed")
	}
	return out.Result, nil
}

func callPost[T any](ctx context.Context, c *Client, path string, env RequestEnvelope) (T, error) {
	result, err := c.postEnvelope(ctx, path, env)
	if err != nil {
		var zero T
		return zero, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		var zero T
		return zero, err
	}
	var out T
	return out, json.Unmarshal(encoded, &out)
}

const maxResponseBytes = 100 * 1024 * 1024 // 100 MB

func decodeResponse(resp *http.Response, target any) error {
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var bridged BridgeResponse[any]
		if err := json.Unmarshal(data, &bridged); err == nil && bridged.Error != nil {
			return bridged.Error
		}
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(data)}
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode bridge response: %w", err)
	}
	return nil
}
