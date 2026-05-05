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

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func NewClientWithHTTP(cfg Config, httpClient *http.Client) *Client {
	return &Client{cfg: cfg, httpClient: httpClient}
}

func (c *Client) Ping(ctx context.Context) (PingResponse, error) {
	var out PingResponse
	err := c.getJSON(ctx, "/ping", &out)
	return out, err
}

func (c *Client) SceneTree(ctx context.Context) (NodeInfo, error) {
	var out SceneTreeResponse
	if err := c.getJSON(ctx, "/scene/tree", &out); err != nil {
		return NodeInfo{}, err
	}
	if !out.OK {
		return NodeInfo{}, fmt.Errorf("scene tree request failed")
	}
	return out.Root, nil
}

func (c *Client) AddNode(ctx context.Context, requestID, parent, nodeType, name string, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.add",
		Params: map[string]any{
			"parent":  parent,
			"type":    nodeType,
			"name":    name,
			"dry_run": dryRun,
		},
	}
	return c.postEnvelope(ctx, "/node/add", env)
}

func (c *Client) RemoveNode(ctx context.Context, requestID, path string, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.remove",
		Params: map[string]any{
			"path":    path,
			"dry_run": dryRun,
		},
	}
	return c.postEnvelope(ctx, "/node/remove", env)
}

func (c *Client) Dial(ctx context.Context) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", c.cfg.Address())
	if err != nil {
		return err
	}
	return conn.Close()
}

func (c *Client) getJSON(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL()+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, target)
}

func (c *Client) postEnvelope(ctx context.Context, path string, env RequestEnvelope) (map[string]any, error) {
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
		return nil, err
	}
	defer resp.Body.Close()

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

func decodeResponse(resp *http.Response, target any) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var bridged BridgeResponse[any]
		if err := json.Unmarshal(data, &bridged); err == nil && bridged.Error != nil {
			return bridged.Error
		}
		return fmt.Errorf("bridge returned HTTP %d: %s", resp.StatusCode, string(data))
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode bridge response: %w", err)
	}
	return nil
}
