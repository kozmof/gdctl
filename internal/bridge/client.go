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

func (c *Client) Job(ctx context.Context, jobID string) (Job, error) {
	var out JobResponse
	if err := c.getJSON(ctx, "/jobs/"+jobID, &out); err != nil {
		return Job{}, err
	}
	if !out.OK {
		return Job{}, fmt.Errorf("job request failed")
	}
	return out.Job, nil
}

func (c *Client) SaveScene(ctx context.Context, requestID, path string) (SceneSaveResult, error) {
	params := map[string]any{}
	if path != "" {
		params["path"] = path
	}
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.save",
		Params:    params,
	}
	result, err := c.postEnvelope(ctx, "/scene/save", env)
	if err != nil {
		return SceneSaveResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SceneSaveResult{}, err
	}
	var out SceneSaveResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return SceneSaveResult{}, err
	}
	return out, nil
}

func (c *Client) CreateScene(ctx context.Context, requestID, path, rootType, rootName string, force bool) (SceneCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.create",
		Params: map[string]any{
			"path":      path,
			"root_type": rootType,
			"root_name": rootName,
			"force":     force,
		},
	}
	result, err := c.postEnvelope(ctx, "/scene/create", env)
	if err != nil {
		return SceneCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SceneCreateResult{}, err
	}
	var out SceneCreateResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return SceneCreateResult{}, err
	}
	return out, nil
}

func (c *Client) OpenScene(ctx context.Context, requestID, path string) (SceneOpenResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.open",
		Params: map[string]any{
			"path": path,
		},
	}
	result, err := c.postEnvelope(ctx, "/scene/open", env)
	if err != nil {
		return SceneOpenResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SceneOpenResult{}, err
	}
	var out SceneOpenResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return SceneOpenResult{}, err
	}
	return out, nil
}

func (c *Client) InstanceScene(ctx context.Context, requestID, parent, scenePath, name string) (SceneInstanceResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.instance",
		Params: map[string]any{
			"parent": parent,
			"scene":  scenePath,
			"name":   name,
		},
	}
	result, err := c.postEnvelope(ctx, "/scene/instance", env)
	if err != nil {
		return SceneInstanceResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SceneInstanceResult{}, err
	}
	var out SceneInstanceResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return SceneInstanceResult{}, err
	}
	return out, nil
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

func (c *Client) RenameNode(ctx context.Context, requestID, path, name string, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.rename",
		Params: map[string]any{
			"path":    path,
			"name":    name,
			"dry_run": dryRun,
		},
	}
	return c.postEnvelope(ctx, "/node/rename", env)
}

func (c *Client) MoveNode(ctx context.Context, requestID, path, parent string, index int, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.move",
		Params: map[string]any{
			"path":    path,
			"parent":  parent,
			"index":   index,
			"dry_run": dryRun,
		},
	}
	return c.postEnvelope(ctx, "/node/move", env)
}

func (c *Client) GetNodeProperty(ctx context.Context, requestID, path, property string) (NodePropertyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.get",
		Params: map[string]any{
			"path":     path,
			"property": property,
		},
	}
	result, err := c.postEnvelope(ctx, "/node/get", env)
	if err != nil {
		return NodePropertyResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodePropertyResult{}, err
	}
	var out NodePropertyResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return NodePropertyResult{}, err
	}
	return out, nil
}

func (c *Client) SetNodeProperty(ctx context.Context, requestID, path, property string, value any) (NodePropertyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.set",
		Params: map[string]any{
			"path":     path,
			"property": property,
			"value":    value,
		},
	}
	result, err := c.postEnvelope(ctx, "/node/set", env)
	if err != nil {
		return NodePropertyResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodePropertyResult{}, err
	}
	var out NodePropertyResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return NodePropertyResult{}, err
	}
	return out, nil
}

func (c *Client) AttachScript(ctx context.Context, requestID, path, scriptPath string) (NodeAttachScriptResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.attach_script",
		Params: map[string]any{
			"path":   path,
			"script": scriptPath,
		},
	}
	result, err := c.postEnvelope(ctx, "/node/attach-script", env)
	if err != nil {
		return NodeAttachScriptResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeAttachScriptResult{}, err
	}
	var out NodeAttachScriptResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return NodeAttachScriptResult{}, err
	}
	return out, nil
}

func (c *Client) CheckScript(ctx context.Context, requestID, path string) (ScriptCheckResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "script.check",
		Params: map[string]any{
			"path": path,
		},
	}
	result, err := c.postEnvelope(ctx, "/script/check", env)
	if err != nil {
		return ScriptCheckResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ScriptCheckResult{}, err
	}
	var out ScriptCheckResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return ScriptCheckResult{}, err
	}
	return out, nil
}

func (c *Client) CreateScript(ctx context.Context, requestID, path, extends string, force bool) (ScriptCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "script.create",
		Params: map[string]any{
			"path":    path,
			"extends": extends,
			"force":   force,
		},
	}
	result, err := c.postEnvelope(ctx, "/script/create", env)
	if err != nil {
		return ScriptCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ScriptCreateResult{}, err
	}
	var out ScriptCreateResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return ScriptCreateResult{}, err
	}
	return out, nil
}

func (c *Client) WriteScript(ctx context.Context, requestID, path, body string) (ScriptWriteResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "script.write",
		Params: map[string]any{
			"path": path,
			"body": body,
		},
	}
	result, err := c.postEnvelope(ctx, "/script/write", env)
	if err != nil {
		return ScriptWriteResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ScriptWriteResult{}, err
	}
	var out ScriptWriteResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return ScriptWriteResult{}, err
	}
	return out, nil
}

func (c *Client) UpdateAddon(ctx context.Context, requestID string, manifest map[string]any, files []AddonUpdateFile) (AddonUpdateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "addon.update",
		Params: map[string]any{
			"manifest": manifest,
			"files":    files,
		},
	}
	result, err := c.postEnvelope(ctx, "/addon/update", env)
	if err != nil {
		return AddonUpdateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return AddonUpdateResult{}, err
	}
	var out AddonUpdateResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return AddonUpdateResult{}, err
	}
	return out, nil
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
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
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
