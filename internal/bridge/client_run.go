package bridge

import (
	"context"
	"fmt"
)

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

func (c *Client) Logs(ctx context.Context) ([]LogEntry, error) {
	var out LogsResponse
	if err := c.getJSON(ctx, "/logs", &out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("logs request failed")
	}
	return out.Entries, nil
}

func (c *Client) ClearLogs(ctx context.Context, requestID string) error {
	env := RequestEnvelope{RequestID: requestID, Op: "logs.clear", Params: map[string]any{}}
	_, err := c.postEnvelope(ctx, "/logs/clear", env)
	return err
}

func (c *Client) RunStart(ctx context.Context, requestID, scene string, main, clearLogs bool) (RunStartResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.start",
		Params:    map[string]any{"scene": scene, "main": main, "clear_logs": clearLogs},
	}
	return callPost[RunStartResult](ctx, c, "/run/start", env)
}

func (c *Client) RunStatus(ctx context.Context, requestID string) (RunStatusResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "run.status", Params: map[string]any{}}
	return callPost[RunStatusResult](ctx, c, "/run/status", env)
}

func (c *Client) RunStop(ctx context.Context, requestID string) (RunStopResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "run.stop", Params: map[string]any{}}
	return callPost[RunStopResult](ctx, c, "/run/stop", env)
}

func (c *Client) RunLogs(ctx context.Context) ([]LogEntry, error) {
	var out LogsResponse
	if err := c.getJSON(ctx, "/run/logs", &out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("run logs request failed")
	}
	return out.Entries, nil
}

func (c *Client) ClearRunLogs(ctx context.Context, requestID string) error {
	env := RequestEnvelope{RequestID: requestID, Op: "run.logs.clear", Params: map[string]any{}}
	_, err := c.postEnvelope(ctx, "/run/logs/clear", env)
	return err
}

func (c *Client) RunScreenshot(ctx context.Context, requestID, source string, screen int, viewportPath string) (RunScreenshotResult, error) {
	params := map[string]any{"source": source, "screen": screen}
	if viewportPath != "" {
		params["viewport_path"] = viewportPath
	}
	env := RequestEnvelope{RequestID: requestID, Op: "run.screenshot", Params: params}
	return callPost[RunScreenshotResult](ctx, c, "/run/screenshot", env)
}

func (c *Client) RunInstantiate(ctx context.Context, requestID, scene, parent, name string) (RunInstantiateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.instantiate",
		Params:    map[string]any{"scene": scene, "parent": parent, "name": name},
	}
	return callPost[RunInstantiateResult](ctx, c, "/run/instantiate", env)
}

func (c *Client) RunSceneReload(ctx context.Context, requestID string) (RunSceneReloadResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "run.scene-reload", Params: map[string]any{}}
	return callPost[RunSceneReloadResult](ctx, c, "/run/scene-reload", env)
}

func (c *Client) RunInput(ctx context.Context, requestID string, steps []any) (RunInputResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "run.input", Params: map[string]any{"steps": steps}}
	return callPost[RunInputResult](ctx, c, "/run/input", env)
}

func (c *Client) RunProbeNode(ctx context.Context, requestID, path string, properties []string) (RunProbeNodeResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.probe.node",
		Params:    map[string]any{"path": path, "properties": properties},
	}
	return callPost[RunProbeNodeResult](ctx, c, "/run/probe/node", env)
}
