package bridge

import (
	"fmt"
	"io"
)

func RenderTree(w io.Writer, root NodeInfo) {
	renderTreeNode(w, root, "", true, true)
}

func renderTreeNode(w io.Writer, node NodeInfo, prefix string, last bool, root bool) {
	if root {
		fmt.Fprintf(w, "%s %s\n", node.Name, node.Type)
	} else {
		branch := "├── "
		childPrefix := "│   "
		if last {
			branch = "└── "
			childPrefix = "    "
		}
		fmt.Fprintf(w, "%s%s%s %s\n", prefix, branch, node.Name, node.Type)
		prefix += childPrefix
	}
	for i, child := range node.Children {
		renderTreeNode(w, child, prefix, i == len(node.Children)-1, false)
	}
}
