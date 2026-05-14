package bridge

import (
	"context"
	"encoding/json"
)

func (c *Client) AutoloadAdd(ctx context.Context, requestID, name, path string) (AutoloadResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "autoload.add",
		Params:    map[string]any{"name": name, "path": path},
	}
	result, err := c.postEnvelope(ctx, "/autoload/add", env)
	if err != nil {
		return AutoloadResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AutoloadResult{}, err
	}
	var out AutoloadResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AutoloadRemove(ctx context.Context, requestID, name string) (AutoloadResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "autoload.remove",
		Params:    map[string]any{"name": name},
	}
	result, err := c.postEnvelope(ctx, "/autoload/remove", env)
	if err != nil {
		return AutoloadResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AutoloadResult{}, err
	}
	var out AutoloadResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) AutoloadList(ctx context.Context, requestID string) (AutoloadListResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "autoload.list",
		Params:    map[string]any{},
	}
	result, err := c.postEnvelope(ctx, "/autoload/list", env)
	if err != nil {
		return AutoloadListResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AutoloadListResult{}, err
	}
	var out AutoloadListResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) InputActionAdd(ctx context.Context, requestID, action string, deadzone float64) (InputActionResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "input.action_add",
		Params:    map[string]any{"action": action, "deadzone": deadzone},
	}
	result, err := c.postEnvelope(ctx, "/input/action-add", env)
	if err != nil {
		return InputActionResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return InputActionResult{}, err
	}
	var out InputActionResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) InputActionRemove(ctx context.Context, requestID, action string) (InputActionResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "input.action_remove",
		Params:    map[string]any{"action": action},
	}
	result, err := c.postEnvelope(ctx, "/input/action-remove", env)
	if err != nil {
		return InputActionResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return InputActionResult{}, err
	}
	var out InputActionResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) InputActionList(ctx context.Context, requestID string, includeBuiltin bool) (InputActionListResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "input.action_list",
		Params:    map[string]any{"include_builtin": includeBuiltin},
	}
	result, err := c.postEnvelope(ctx, "/input/action-list", env)
	if err != nil {
		return InputActionListResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return InputActionListResult{}, err
	}
	var out InputActionListResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) InputEventAddKey(ctx context.Context, requestID, action, key string, physical bool) (InputActionResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "input.event_add_key",
		Params:    map[string]any{"action": action, "key": key, "physical": physical},
	}
	result, err := c.postEnvelope(ctx, "/input/event-add-key", env)
	if err != nil {
		return InputActionResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return InputActionResult{}, err
	}
	var out InputActionResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) SignalConnect(ctx context.Context, requestID, fromPath, signalName, toPath, method string) (SignalConnectResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "signal.connect",
		Params: map[string]any{
			"from":   fromPath,
			"signal": signalName,
			"to":     toPath,
			"method": method,
		},
	}
	result, err := c.postEnvelope(ctx, "/signal/connect", env)
	if err != nil {
		return SignalConnectResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SignalConnectResult{}, err
	}
	var out SignalConnectResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) SignalDisconnect(ctx context.Context, requestID, fromPath, signalName, toPath, method string) (SignalDisconnectResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "signal.disconnect",
		Params: map[string]any{
			"from":   fromPath,
			"signal": signalName,
			"to":     toPath,
			"method": method,
		},
	}
	result, err := c.postEnvelope(ctx, "/signal/disconnect", env)
	if err != nil {
		return SignalDisconnectResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SignalDisconnectResult{}, err
	}
	var out SignalDisconnectResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ProjectSettingGet(ctx context.Context, requestID, key string) (ProjectSettingResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "project.setting_get",
		Params:    map[string]any{"key": key},
	}
	result, err := c.postEnvelope(ctx, "/project/setting-get", env)
	if err != nil {
		return ProjectSettingResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ProjectSettingResult{}, err
	}
	var out ProjectSettingResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ProjectSettingSet(ctx context.Context, requestID, key string, value any) (ProjectSettingResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "project.setting_set",
		Params:    map[string]any{"key": key, "value": value},
	}
	result, err := c.postEnvelope(ctx, "/project/setting-set", env)
	if err != nil {
		return ProjectSettingResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ProjectSettingResult{}, err
	}
	var out ProjectSettingResult
	return out, json.Unmarshal(encoded, &out)
}
