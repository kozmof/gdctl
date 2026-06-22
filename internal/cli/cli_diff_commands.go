package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"gdctl/internal/bridge"
)

// runDiff compares a JSON desired-state file against the currently open scene tree.
// gdctl diff FILE --scene SCENE [--json]
func runDiff(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("diff")
	scene := fs.String("scene", "", "scene path to open for comparison")
	timeout := fs.Duration("timeout", 5*time.Second, "maximum time to wait for scene open")
	jsonOut := fs.Bool("json", false, "print result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	file := fs.Arg(0)
	if *scene == "" || file == "" {
		return fmt.Errorf("diff requires --scene <path> and <file>")
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	// The desired-state file uses the same format as scene apply:
	// a JSON object with a "nodes" array describing the desired tree.
	var desired map[string]any
	if err := json.Unmarshal(data, &desired); err != nil {
		return fmt.Errorf("diff: file must be JSON: %w", err)
	}

	openedPath, _, err := openSceneAndWait(ctx, client, *scene, *timeout)
	if err != nil {
		return err
	}

	current, err := client.SceneTree(ctx)
	if err != nil {
		return err
	}

	// Build path→type index from current tree
	currentIndex := map[string]string{}
	indexTree(current, currentIndex)

	// Build path→type index from desired tree
	desiredIndex := map[string]string{}
	if nodes, ok := desired["nodes"].([]any); ok {
		for _, n := range nodes {
			if node, ok := n.(map[string]any); ok {
				path, _ := node["path"].(string)
				nodeType, _ := node["type"].(string)
				if path != "" {
					desiredIndex[path] = nodeType
				}
			}
		}
	}

	type diffEntry struct {
		Path   string `json:"path"`
		Action string `json:"action"`
		Detail string `json:"detail,omitempty"`
	}
	var changes []diffEntry

	// Find nodes to add (in desired, not in current)
	for path, nodeType := range desiredIndex {
		if _, exists := currentIndex[path]; !exists {
			changes = append(changes, diffEntry{
				Path:   path,
				Action: "add",
				Detail: nodeType,
			})
		}
	}

	// Find type mismatches (in both but different type)
	for path, desiredType := range desiredIndex {
		if currentType, exists := currentIndex[path]; exists && desiredType != "" && currentType != desiredType {
			changes = append(changes, diffEntry{
				Path:   path,
				Action: "type-mismatch",
				Detail: fmt.Sprintf("%s → %s", currentType, desiredType),
			})
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"scene":          openedPath,
			"current_nodes":  len(currentIndex),
			"desired_nodes":  len(desiredIndex),
			"changes":        changes,
			"property_diff":  "not supported — requires per-node GetNodeProperty queries",
		})
	}

	fmt.Fprintf(stdout, "Diff: %s\n", openedPath)
	if len(changes) == 0 {
		fmt.Fprintln(stdout, "  No structural changes detected.")
		return nil
	}
	for _, c := range changes {
		switch c.Action {
		case "add":
			fmt.Fprintf(stdout, "  + %s (%s)\n", c.Path, c.Detail)
		case "type-mismatch":
			fmt.Fprintf(stdout, "  ~ %s [%s]\n", c.Path, c.Detail)
		}
	}
	fmt.Fprintf(stdout, "  %d change(s) (no scene was modified)\n", len(changes))
	fmt.Fprintln(stdout, "  Note: property-level diff is not supported at this time.")
	return nil
}

// indexTree recursively builds a path→type map from a NodeInfo tree.
func indexTree(node bridge.NodeInfo, index map[string]string) {
	index[node.Path] = node.Type
	for _, child := range node.Children {
		indexTree(child, index)
	}
}
