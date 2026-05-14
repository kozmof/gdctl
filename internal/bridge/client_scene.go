package bridge

import (
	"context"
	"encoding/json"
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

func (c *Client) ApplyScene(ctx context.Context, requestID string, tree any, dryRun bool) (SceneApplyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.apply",
		Params: map[string]any{
			"tree":    tree,
			"dry_run": dryRun,
		},
	}
	result, err := c.postEnvelope(ctx, "/scene/apply", env)
	if err != nil {
		return SceneApplyResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SceneApplyResult{}, err
	}
	var out SceneApplyResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return SceneApplyResult{}, err
	}
	return out, nil
}

func (c *Client) ApplyBlueprint(ctx context.Context, requestID, scenePath, blueprint string, props map[string]any, dryRun bool) (SceneApplyBlueprintResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "scene.apply.blueprint",
		Params: map[string]any{
			"path":      scenePath,
			"blueprint": blueprint,
			"props":     props,
			"dry_run":   dryRun,
		},
	}
	result, err := c.postEnvelope(ctx, "/scene/apply/blueprint", env)
	if err != nil {
		return SceneApplyBlueprintResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SceneApplyBlueprintResult{}, err
	}
	var out SceneApplyBlueprintResult
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
