package bridge

import (
	"context"
	"fmt"
)

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
	env := RequestEnvelope{RequestID: requestID, Op: "scene.save", Params: params}
	return callPost[SceneSaveResult](ctx, c, "/scene/save", env)
}

func (c *Client) CreateScene(ctx context.Context, requestID, path, rootType, rootName string, force bool) (SceneCreateResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.create",
		Params:    map[string]any{"path": path, "root_type": rootType, "root_name": rootName, "force": force},
	}
	return callPost[SceneCreateResult](ctx, c, "/scene/create", env)
}

func (c *Client) OpenScene(ctx context.Context, requestID, path string) (SceneOpenResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.open",
		Params:    map[string]any{"path": path},
	}
	return callPost[SceneOpenResult](ctx, c, "/scene/open", env)
}

func (c *Client) InstanceScene(ctx context.Context, requestID, parent, scenePath, name string) (SceneInstanceResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.instance",
		Params:    map[string]any{"parent": parent, "scene": scenePath, "name": name},
	}
	return callPost[SceneInstanceResult](ctx, c, "/scene/instance", env)
}

func (c *Client) ApplyScene(ctx context.Context, requestID string, tree any, dryRun bool) (SceneApplyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.apply",
		Params:    map[string]any{"tree": tree, "dry_run": dryRun},
	}
	return callPost[SceneApplyResult](ctx, c, "/scene/apply", env)
}

func (c *Client) ApplyBlueprint(ctx context.Context, requestID, scenePath, blueprint string, props map[string]any, dryRun bool) (SceneApplyBlueprintResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.apply.blueprint",
		Params:    map[string]any{"path": scenePath, "blueprint": blueprint, "props": props, "dry_run": dryRun},
	}
	return callPost[SceneApplyBlueprintResult](ctx, c, "/scene/apply/blueprint", env)
}

func (c *Client) SceneList(ctx context.Context, requestID, dir string, recursive bool) (SceneListResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.list",
		Params:    map[string]any{"dir": dir, "recursive": recursive},
	}
	return callPost[SceneListResult](ctx, c, "/scene/list", env)
}
