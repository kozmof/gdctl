package bridge

import (
	"context"
)

func (c *Client) CheckScript(ctx context.Context, requestID, path string) (ScriptCheckResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "script.check", Params: map[string]any{"path": path}}
	return callPost[ScriptCheckResult](ctx, c, "/script/check", env)
}

func (c *Client) CreateScript(ctx context.Context, requestID, path, extends string, force bool) (ScriptCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "script.create",
		Params:    map[string]any{"path": path, "extends": extends, "force": force},
	}
	return callPost[ScriptCreateResult](ctx, c, "/script/create", env)
}

func (c *Client) WriteScript(ctx context.Context, requestID, path, body string, allowMissingPreloads bool, allowMissingAutoloads bool) (ScriptWriteResult, error) {
	params := map[string]any{"path": path, "body": body}
	if allowMissingPreloads {
		params["allow_missing_preloads"] = true
	}
	if allowMissingAutoloads {
		params["allow_missing_autoloads"] = true
	}
	env := RequestEnvelope{RequestID: requestID, Op: "script.write", Params: params}
	return callPost[ScriptWriteResult](ctx, c, "/script/write", env)
}

func (c *Client) TestGDScript(ctx context.Context, requestID, path, dir string) (GDScriptTestQueuedResult, error) {
	params := map[string]any{}
	if path != "" {
		params["path"] = path
	}
	if dir != "" {
		params["dir"] = dir
	}
	env := RequestEnvelope{RequestID: requestID, Op: "test.gdscript", Params: params}
	return callPost[GDScriptTestQueuedResult](ctx, c, "/test/gdscript", env)
}

func (c *Client) CheckShader(ctx context.Context, requestID, path string) (ShaderCheckResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "shader.check", Params: map[string]any{"path": path}}
	return callPost[ShaderCheckResult](ctx, c, "/shader/check", env)
}

func (c *Client) WriteShader(ctx context.Context, requestID, path, body string) (ShaderWriteResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "shader.write",
		Params:    map[string]any{"path": path, "body": body},
	}
	return callPost[ShaderWriteResult](ctx, c, "/shader/write", env)
}

func (c *Client) CreateResource(ctx context.Context, requestID, path, resourceType, scriptPath string, props map[string]any, shaderParams map[string]string) (ResourceCreateResult, error) {
	params := map[string]any{"path": path, "type": resourceType, "props": props, "shader_params": shaderParams}
	if scriptPath != "" {
		params["script"] = scriptPath
	}
	env := RequestEnvelope{RequestID: requestID, Op: "resource.create", Params: params}
	return callPost[ResourceCreateResult](ctx, c, "/resource/create", env)
}

func (c *Client) WriteFileBytes(ctx context.Context, requestID, path, contentBase64 string) (FileWriteBytesResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "file.write_bytes",
		Params:    map[string]any{"path": path, "content_base64": contentBase64},
	}
	return callPost[FileWriteBytesResult](ctx, c, "/file/write-bytes", env)
}

func (c *Client) FileList(ctx context.Context, requestID, path string, recursive bool) (FileListResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.list", Params: map[string]any{"path": path, "recursive": recursive}}
	return callPost[FileListResult](ctx, c, "/file/list", env)
}

func (c *Client) FileMkdir(ctx context.Context, requestID, path string) (FileMkdirResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.mkdir", Params: map[string]any{"path": path}}
	return callPost[FileMkdirResult](ctx, c, "/file/mkdir", env)
}

func (c *Client) FileDelete(ctx context.Context, requestID, path string) (FileDeleteResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.delete", Params: map[string]any{"path": path}}
	return callPost[FileDeleteResult](ctx, c, "/file/delete", env)
}

func (c *Client) FileExists(ctx context.Context, requestID, path string) (FileExistsResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "file.exists", Params: map[string]any{"path": path}}
	return callPost[FileExistsResult](ctx, c, "/file/exists", env)
}

func (c *Client) ImportSet(ctx context.Context, requestID, path string, params map[string]any) (ImportSetResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "import.set",
		Params:    map[string]any{"path": path, "params": params},
	}
	return callPost[ImportSetResult](ctx, c, "/import/set", env)
}

func (c *Client) ResourceList(ctx context.Context, requestID, dir, ext string, recursive bool) (ResourceListResult, error) {
	params := map[string]any{"dir": dir, "recursive": recursive}
	if ext != "" {
		params["ext"] = ext
	}
	env := RequestEnvelope{RequestID: requestID, Op: "resource.list", Params: params}
	return callPost[ResourceListResult](ctx, c, "/resource/list", env)
}

func (c *Client) NavigationBake(ctx context.Context, requestID, path string) (NavigationBakeResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "navigation.bake", Params: map[string]any{"path": path}}
	return callPost[NavigationBakeResult](ctx, c, "/navigation/bake", env)
}

func (c *Client) UpdateAddon(ctx context.Context, requestID string, manifest map[string]any, oldManifest map[string]any, files []AddonUpdateFile) (AddonUpdateResult, error) {
	params := map[string]any{"manifest": manifest, "files": files}
	if len(oldManifest) > 0 {
		params["old_manifest"] = oldManifest
	}
	env := RequestEnvelope{RequestID: requestID, Op: "addon.update", Params: params}
	return callPost[AddonUpdateResult](ctx, c, "/addon/update", env)
}
