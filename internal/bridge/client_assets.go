package bridge

import (
	"context"
	"encoding/json"
)

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

func (c *Client) WriteScript(ctx context.Context, requestID, path, body string, allowMissingPreloads bool) (ScriptWriteResult, error) {
	params := map[string]any{
		"path": path,
		"body": body,
	}
	if allowMissingPreloads {
		params["allow_missing_preloads"] = true
	}
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "script.write",
		Params:    params,
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

func (c *Client) CreateResource(ctx context.Context, requestID, path, resourceType, scriptPath string, props map[string]any, shaderParams map[string]string) (ResourceCreateResult, error) {
	params := map[string]any{
		"path":          path,
		"type":          resourceType,
		"props":         props,
		"shader_params": shaderParams,
	}
	if scriptPath != "" {
		params["script"] = scriptPath
	}
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "resource.create",
		Params:    params,
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

func (c *Client) UpdateAddon(ctx context.Context, requestID string, manifest map[string]any, oldManifest map[string]any, files []AddonUpdateFile) (AddonUpdateResult, error) {
	params := map[string]any{
		"manifest": manifest,
		"files":    files,
	}
	if len(oldManifest) > 0 {
		params["old_manifest"] = oldManifest
	}
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "addon.update",
		Params:    params,
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
