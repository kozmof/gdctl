package bridge

import (
	"context"
)

func (c *Client) AutoloadAdd(ctx context.Context, requestID, name, path string) (AutoloadResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "autoload.add", Params: map[string]any{"name": name, "path": path}}
	return callPost[AutoloadResult](ctx, c, "/autoload/add", env)
}

func (c *Client) AutoloadRemove(ctx context.Context, requestID, name string) (AutoloadResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "autoload.remove", Params: map[string]any{"name": name}}
	return callPost[AutoloadResult](ctx, c, "/autoload/remove", env)
}

func (c *Client) AutoloadList(ctx context.Context, requestID string) (AutoloadListResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "autoload.list", Params: map[string]any{}}
	return callPost[AutoloadListResult](ctx, c, "/autoload/list", env)
}

func (c *Client) InputActionAdd(ctx context.Context, requestID, action string, deadzone float64) (InputActionResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "input.action_add", Params: map[string]any{"action": action, "deadzone": deadzone}}
	return callPost[InputActionResult](ctx, c, "/input/action-add", env)
}

func (c *Client) InputActionRemove(ctx context.Context, requestID, action string) (InputActionResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "input.action_remove", Params: map[string]any{"action": action}}
	return callPost[InputActionResult](ctx, c, "/input/action-remove", env)
}

func (c *Client) InputActionList(ctx context.Context, requestID string, includeBuiltin bool) (InputActionListResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "input.action_list", Params: map[string]any{"include_builtin": includeBuiltin}}
	return callPost[InputActionListResult](ctx, c, "/input/action-list", env)
}

func (c *Client) InputEventAddKey(ctx context.Context, requestID, action, key string, physical bool) (InputActionResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "input.event_add_key", Params: map[string]any{"action": action, "key": key, "physical": physical}}
	return callPost[InputActionResult](ctx, c, "/input/event-add-key", env)
}

func (c *Client) InputEventAddMouseButton(ctx context.Context, requestID, action, button string) (InputActionResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "input.event_add_mouse_button", Params: map[string]any{"action": action, "button": button}}
	return callPost[InputActionResult](ctx, c, "/input/event-add-mouse-button", env)
}

func (c *Client) InputEventAddJoypad(ctx context.Context, requestID, action string, button, axis int, axisValue float64, device int) (InputActionResult, error) {
	params := map[string]any{"action": action, "button": button, "axis": axis}
	if axisValue != 0 {
		params["axis_value"] = axisValue
	}
	if device >= 0 {
		params["device"] = device
	}
	env := RequestEnvelope{RequestID: requestID, Op: "input.event_add_joypad", Params: params}
	return callPost[InputActionResult](ctx, c, "/input/event-add-joypad", env)
}

func (c *Client) SignalConnect(ctx context.Context, requestID, fromPath, signalName, toPath, method string) (SignalConnectResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "signal.connect",
		Params:    map[string]any{"from": fromPath, "signal": signalName, "to": toPath, "method": method},
	}
	return callPost[SignalConnectResult](ctx, c, "/signal/connect", env)
}

func (c *Client) SignalDisconnect(ctx context.Context, requestID, fromPath, signalName, toPath, method string) (SignalDisconnectResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "signal.disconnect",
		Params:    map[string]any{"from": fromPath, "signal": signalName, "to": toPath, "method": method},
	}
	return callPost[SignalDisconnectResult](ctx, c, "/signal/disconnect", env)
}

func (c *Client) ProjectSettingGet(ctx context.Context, requestID, key string) (ProjectSettingResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "project.setting_get", Params: map[string]any{"key": key}}
	return callPost[ProjectSettingResult](ctx, c, "/project/setting-get", env)
}

func (c *Client) ProjectSettingSet(ctx context.Context, requestID, key string, value any) (ProjectSettingResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "project.setting_set", Params: map[string]any{"key": key, "value": value}}
	return callPost[ProjectSettingResult](ctx, c, "/project/setting-set", env)
}
