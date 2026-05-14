package bridge

import (
	"context"
	"encoding/json"
)

func (c *Client) AddNode(ctx context.Context, requestID, parent, nodeType, name string, props map[string]any, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.add",
		Params: map[string]any{
			"parent":  parent,
			"type":    nodeType,
			"name":    name,
			"props":   props,
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

func (c *Client) SetNodeProperties(ctx context.Context, requestID, path string, properties map[string]any) (NodeSetManyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.set_many",
		Params: map[string]any{
			"path":       path,
			"properties": properties,
		},
	}
	result, err := c.postEnvelope(ctx, "/node/set-many", env)
	if err != nil {
		return NodeSetManyResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return NodeSetManyResult{}, err
	}
	var out NodeSetManyResult
	if err := json.Unmarshal(encoded, &out); err != nil {
		return NodeSetManyResult{}, err
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
