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

func (c *Client) SetNodeResource(ctx context.Context, requestID, path, property, resourcePath string) (NodeSetResourceResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.set_resource",
		Params: map[string]any{
			"path":     path,
			"property": property,
			"resource": resourcePath,
		},
	}
	result, err := c.postEnvelope(ctx, "/node/set-resource", env)
	if err != nil {
		return NodeSetResourceResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeSetResourceResult{}, err
	}
	var out NodeSetResourceResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return NodeSetResourceResult{}, err
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

func (c *Client) CheckShader(ctx context.Context, requestID, path string) (ShaderCheckResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "shader.check",
		Params: map[string]any{
			"path": path,
		},
	}
	result, err := c.postEnvelope(ctx, "/shader/check", env)
	if err != nil {
		return ShaderCheckResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ShaderCheckResult{}, err
	}
	var out ShaderCheckResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return ShaderCheckResult{}, err
	}
	return out, nil
}

func (c *Client) WriteShader(ctx context.Context, requestID, path, body string) (ShaderWriteResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "shader.write",
		Params: map[string]any{
			"path": path,
			"body": body,
		},
	}
	result, err := c.postEnvelope(ctx, "/shader/write", env)
	if err != nil {
		return ShaderWriteResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ShaderWriteResult{}, err
	}
	var out ShaderWriteResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return ShaderWriteResult{}, err
	}
	return out, nil
}

func (c *Client) CreateResource(ctx context.Context, requestID, path, resourceType string, props map[string]any, shaderParams map[string]string) (ResourceCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "resource.create",
		Params: map[string]any{
			"path":          path,
			"type":          resourceType,
			"props":         props,
			"shader_params": shaderParams,
		},
	}
	result, err := c.postEnvelope(ctx, "/resource/create", env)
	if err != nil {
		return ResourceCreateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ResourceCreateResult{}, err
	}
	var out ResourceCreateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) WriteFileBytes(ctx context.Context, requestID, path, contentBase64 string) (FileWriteBytesResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "file.write_bytes",
		Params: map[string]any{
			"path":           path,
			"content_base64": contentBase64,
		},
	}
	result, err := c.postEnvelope(ctx, "/file/write-bytes", env)
	if err != nil {
		return FileWriteBytesResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return FileWriteBytesResult{}, err
	}
	var out FileWriteBytesResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return FileWriteBytesResult{}, err
	}
	return out, nil
}

func (c *Client) ScreenshotViewport(ctx context.Context, requestID, kind string, index int) (ViewportScreenshotResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "viewport.screenshot",
		Params: map[string]any{
			"kind":  kind,
			"index": index,
		},
	}
	result, err := c.postEnvelope(ctx, "/viewport/screenshot", env)
	if err != nil {
		return ViewportScreenshotResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ViewportScreenshotResult{}, err
	}
	var out ViewportScreenshotResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return ViewportScreenshotResult{}, err
	}
	return out, nil
}

func (c *Client) NodeGroupAdd(ctx context.Context, requestID, path, group string) (NodeGroupResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.group_add",
		Params:    map[string]any{"path": path, "group": group},
	}
	result, err := c.postEnvelope(ctx, "/node/group-add", env)
	if err != nil {
		return NodeGroupResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeGroupResult{}, err
	}
	var out NodeGroupResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) NodeGroupRemove(ctx context.Context, requestID, path, group string) (NodeGroupResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.group_remove",
		Params:    map[string]any{"path": path, "group": group},
	}
	result, err := c.postEnvelope(ctx, "/node/group-remove", env)
	if err != nil {
		return NodeGroupResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeGroupResult{}, err
	}
	var out NodeGroupResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) NodeGroupList(ctx context.Context, requestID, path string) (NodeGroupListResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.group_list",
		Params:    map[string]any{"path": path},
	}
	result, err := c.postEnvelope(ctx, "/node/group-list", env)
	if err != nil {
		return NodeGroupListResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeGroupListResult{}, err
	}
	var out NodeGroupListResult
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

func (c *Client) NodeDuplicate(ctx context.Context, requestID, path, name, parent string, dryRun bool) (NodeDuplicateResult, error) {
	params := map[string]any{"path": path, "name": name, "dry_run": dryRun}
	if parent != "" {
		params["parent"] = parent
	}
	env := RequestEnvelope{RequestID: requestID, Op: "node.duplicate", Params: params}
	result, err := c.postEnvelope(ctx, "/node/duplicate", env)
	if err != nil {
		return NodeDuplicateResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeDuplicateResult{}, err
	}
	var out NodeDuplicateResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) NodeListProperties(ctx context.Context, requestID, path string) (NodeListPropertiesResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "node.list_properties", Params: map[string]any{"path": path}}
	result, err := c.postEnvelope(ctx, "/node/list-properties", env)
	if err != nil {
		return NodeListPropertiesResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeListPropertiesResult{}, err
	}
	var out NodeListPropertiesResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) FileList(ctx context.Context, requestID, path string, recursive bool) (FileListResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.list", Params: map[string]any{"path": path, "recursive": recursive}}
	result, err := c.postEnvelope(ctx, "/file/list", env)
	if err != nil {
		return FileListResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return FileListResult{}, err
	}
	var out FileListResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) FileMkdir(ctx context.Context, requestID, path string) (FileMkdirResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.mkdir", Params: map[string]any{"path": path}}
	result, err := c.postEnvelope(ctx, "/file/mkdir", env)
	if err != nil {
		return FileMkdirResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return FileMkdirResult{}, err
	}
	var out FileMkdirResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) FileDelete(ctx context.Context, requestID, path string) (FileDeleteResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.delete", Params: map[string]any{"path": path}}
	result, err := c.postEnvelope(ctx, "/file/delete", env)
	if err != nil {
		return FileDeleteResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return FileDeleteResult{}, err
	}
	var out FileDeleteResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) FileExists(ctx context.Context, requestID, path string) (FileExistsResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.exists", Params: map[string]any{"path": path}}
	result, err := c.postEnvelope(ctx, "/file/exists", env)
	if err != nil {
		return FileExistsResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return FileExistsResult{}, err
	}
	var out FileExistsResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ImportSet(ctx context.Context, requestID, path string, params map[string]any) (ImportSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "import.set",
		Params:    map[string]any{"path": path, "params": params},
	}
	result, err := c.postEnvelope(ctx, "/import/set", env)
	if err != nil {
		return ImportSetResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ImportSetResult{}, err
	}
	var out ImportSetResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) SceneList(ctx context.Context, requestID, dir string, recursive bool) (SceneListResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.list",
		Params:    map[string]any{"dir": dir, "recursive": recursive},
	}
	result, err := c.postEnvelope(ctx, "/scene/list", env)
	if err != nil {
		return SceneListResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SceneListResult{}, err
	}
	var out SceneListResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) ResourceList(ctx context.Context, requestID, dir, ext string, recursive bool) (ResourceListResult, error) {
	params := map[string]any{"dir": dir, "recursive": recursive}
	if ext != "" {
		params["ext"] = ext
	}
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "resource.list",
		Params:    params,
	}
	result, err := c.postEnvelope(ctx, "/resource/list", env)
	if err != nil {
		return ResourceListResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return ResourceListResult{}, err
	}
	var out ResourceListResult
	return out, json.Unmarshal(encoded, &out)
}

func (c *Client) NavigationBake(ctx context.Context, requestID, path string) (NavigationBakeResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "navigation.bake", Params: map[string]any{"path": path}}
	result, err := c.postEnvelope(ctx, "/navigation/bake", env)
	if err != nil {
		return NavigationBakeResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NavigationBakeResult{}, err
	}
	var out NavigationBakeResult
	return out, json.Unmarshal(encoded, &out)
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
