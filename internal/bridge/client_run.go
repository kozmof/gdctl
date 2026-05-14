package bridge

import (
	"context"
	"encoding/json"
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
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "logs.clear",
		Params:    map[string]any{},
	}
	_, err := c.postEnvelope(ctx, "/logs/clear", env)
	return err
}

func (c *Client) RunStart(ctx context.Context, requestID, scene string, main, clearLogs bool) (RunStartResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.start",
		Params: map[string]any{
			"scene":      scene,
			"main":       main,
			"clear_logs": clearLogs,
		},
	}
	result, err := c.postEnvelope(ctx, "/run/start", env)
	if err != nil {
		return RunStartResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return RunStartResult{}, err
	}
	var out RunStartResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return RunStartResult{}, err
	}
	return out, nil
}

func (c *Client) RunStatus(ctx context.Context, requestID string) (RunStatusResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.status",
		Params:    map[string]any{},
	}
	result, err := c.postEnvelope(ctx, "/run/status", env)
	if err != nil {
		return RunStatusResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return RunStatusResult{}, err
	}
	var out RunStatusResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return RunStatusResult{}, err
	}
	return out, nil
}

func (c *Client) RunStop(ctx context.Context, requestID string) (RunStopResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.stop",
		Params:    map[string]any{},
	}
	result, err := c.postEnvelope(ctx, "/run/stop", env)
	if err != nil {
		return RunStopResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return RunStopResult{}, err
	}
	var out RunStopResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return RunStopResult{}, err
	}
	return out, nil
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
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.logs.clear",
		Params:    map[string]any{},
	}
	_, err := c.postEnvelope(ctx, "/run/logs/clear", env)
	return err
}

func (c *Client) RunScreenshot(ctx context.Context, requestID, source string, screen int) (RunScreenshotResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.screenshot",
		Params: map[string]any{
			"source": source,
			"screen": screen,
		},
	}
	result, err := c.postEnvelope(ctx, "/run/screenshot", env)
	if err != nil {
		return RunScreenshotResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return RunScreenshotResult{}, err
	}
	var out RunScreenshotResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return RunScreenshotResult{}, err
	}
	return out, nil
}

func (c *Client) RunInput(ctx context.Context, requestID string, steps []any) (RunInputResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.input",
		Params: map[string]any{
			"steps": steps,
		},
	}
	result, err := c.postEnvelope(ctx, "/run/input", env)
	if err != nil {
		return RunInputResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return RunInputResult{}, err
	}
	var out RunInputResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return RunInputResult{}, err
	}
	return out, nil
}

func (c *Client) RunProbeNode(ctx context.Context, requestID, path string, properties []string) (RunProbeNodeResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "run.probe.node",
		Params: map[string]any{
			"path":       path,
			"properties": properties,
		},
	}
	result, err := c.postEnvelope(ctx, "/run/probe/node", env)
	if err != nil {
		return RunProbeNodeResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return RunProbeNodeResult{}, err
	}
	var out RunProbeNodeResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return RunProbeNodeResult{}, err
	}
	return out, nil
}
