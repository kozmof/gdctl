package bridge

import (
	"context"
)

func (c *Client) AddNode(ctx context.Context, requestID, parent, nodeType, name string, props map[string]any, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.add",
		Params:    map[string]any{"parent": parent, "type": nodeType, "name": name, "props": props, "dry_run": dryRun},
	}
	return c.postEnvelope(ctx, "/node/add", env)
}

func (c *Client) RemoveNode(ctx context.Context, requestID, path string, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.remove",
		Params:    map[string]any{"path": path, "dry_run": dryRun},
	}
	return c.postEnvelope(ctx, "/node/remove", env)
}

func (c *Client) RenameNode(ctx context.Context, requestID, path, name string, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.rename",
		Params:    map[string]any{"path": path, "name": name, "dry_run": dryRun},
	}
	return c.postEnvelope(ctx, "/node/rename", env)
}

func (c *Client) MoveNode(ctx context.Context, requestID, path, parent string, index int, dryRun bool) (map[string]any, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.move",
		Params:    map[string]any{"path": path, "parent": parent, "index": index, "dry_run": dryRun},
	}
	return c.postEnvelope(ctx, "/node/move", env)
}

func (c *Client) GetNodeProperty(ctx context.Context, requestID, path, property string) (NodePropertyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.get",
		Params:    map[string]any{"path": path, "property": property},
	}
	return callPost[NodePropertyResult](ctx, c, "/node/get", env)
}

func (c *Client) SetNodeProperty(ctx context.Context, requestID, path, property string, value any) (NodePropertyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.set",
		Params:    map[string]any{"path": path, "property": property, "value": value},
	}
	return callPost[NodePropertyResult](ctx, c, "/node/set", env)
}

func (c *Client) SetNodeProperties(ctx context.Context, requestID, path string, properties map[string]any) (NodeSetManyResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.set_many",
		Params:    map[string]any{"path": path, "properties": properties},
	}
	return callPost[NodeSetManyResult](ctx, c, "/node/set-many", env)
}

func (c *Client) SetNodeResource(ctx context.Context, requestID, path, property, resourcePath string) (NodeSetResourceResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.set_resource",
		Params:    map[string]any{"path": path, "property": property, "resource": resourcePath},
	}
	return callPost[NodeSetResourceResult](ctx, c, "/node/set-resource", env)
}

func (c *Client) AttachScript(ctx context.Context, requestID, path, scriptPath string) (NodeAttachScriptResult, error) {
	env := RequestEnvelope{
		RequestID: requestID,
		Op:        "node.attach_script",
		Params:    map[string]any{"path": path, "script": scriptPath},
	}
	return callPost[NodeAttachScriptResult](ctx, c, "/node/attach-script", env)
}

func (c *Client) NodeGroupAdd(ctx context.Context, requestID, path, group string) (NodeGroupResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "node.group_add", Params: map[string]any{"path": path, "group": group}}
	return callPost[NodeGroupResult](ctx, c, "/node/group-add", env)
}

func (c *Client) NodeGroupRemove(ctx context.Context, requestID, path, group string) (NodeGroupResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "node.group_remove", Params: map[string]any{"path": path, "group": group}}
	return callPost[NodeGroupResult](ctx, c, "/node/group-remove", env)
}

func (c *Client) NodeGroupList(ctx context.Context, requestID, path string) (NodeGroupListResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "node.group_list", Params: map[string]any{"path": path}}
	return callPost[NodeGroupListResult](ctx, c, "/node/group-list", env)
}

func (c *Client) NodeDuplicate(ctx context.Context, requestID, path, name, parent string, dryRun bool) (NodeDuplicateResult, error) {
	params := map[string]any{"path": path, "name": name, "dry_run": dryRun}
	if parent != "" {
		params["parent"] = parent
	}
	env := RequestEnvelope{RequestID: requestID, Op: "node.duplicate", Params: params}
	return callPost[NodeDuplicateResult](ctx, c, "/node/duplicate", env)
}

func (c *Client) NodeListProperties(ctx context.Context, requestID, path string) (NodeListPropertiesResult, error) {
	env := RequestEnvelope{RequestID: requestID, Op: "node.list_properties", Params: map[string]any{"path": path}}
	return callPost[NodeListPropertiesResult](ctx, c, "/node/list-properties", env)
}
