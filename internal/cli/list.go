package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/db"
)

var (
	listType  string
	listTree  bool
	listCount bool
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List documentation paths in the database",
		Long: `List all documentation paths stored in the database.

Paths can be filtered by type and displayed in various formats.`,
		Example: `  documango list
  documango list -t go
  documango list --tree
  documango list --count`,
		RunE: runList,
	}

	cmd.Flags().StringVarP(&listType, "type", "t", "", "Filter by path prefix (e.g., go, atproto)")
	cmd.Flags().BoolVar(&listTree, "tree", false, "Display as tree structure")
	cmd.Flags().BoolVar(&listCount, "count", false, "Show only count of documents")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	dbPath, err := resolveDBPath()
	if err != nil {
		return err
	}

	store, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	ctx := context.Background()
	paths, err := listPaths(ctx, store)
	if err != nil {
		return err
	}

	prefix := listType
	if len(args) > 0 {
		if prefix != "" {
			prefix = prefix + "/" + args[0]
		} else {
			prefix = args[0]
		}
	}

	if prefix != "" {
		paths = filterPaths(paths, prefix)
	}

	if listCount {
		fmt.Fprintf(cmd.OutOrStdout(), "%d\n", len(paths))
		return nil
	}

	if listTree {
		return printTree(cmd, paths)
	}

	for _, path := range paths {
		fmt.Fprintln(cmd.OutOrStdout(), path)
	}

	return nil
}

func listPaths(ctx context.Context, store *db.Store) ([]string, error) {
	rows, err := store.DB().QueryContext(ctx, `SELECT path FROM documents ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	return paths, rows.Err()
}

func filterPaths(paths []string, prefix string) []string {
	var filtered []string
	for _, path := range paths {
		if strings.HasPrefix(path, prefix) {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

func printTree(cmd *cobra.Command, paths []string) error {
	root := &treeNode{name: "root"}

	for _, path := range paths {
		parts := strings.Split(path, "/")
		current := root
		for _, part := range parts {
			if part == "" {
				continue
			}
			found := false
			for _, child := range current.children {
				if child.name == part {
					current = child
					found = true
					break
				}
			}
			if !found {
				newNode := &treeNode{name: part}
				current.children = append(current.children, newNode)
				current = newNode
			}
		}
	}

	printNode(cmd.OutOrStdout(), root, "")
	return nil
}

type treeNode struct {
	name     string
	children []*treeNode
}

func printNode(w io.Writer, node *treeNode, prefix string) {
	if node.name != "root" {
		fmt.Fprintf(w, "%s%s\n", prefix, node.name)
	}

	sort.Slice(node.children, func(i, j int) bool {
		return node.children[i].name < node.children[j].name
	})

	for i, child := range node.children {
		isLast := i == len(node.children)-1
		var newPrefix string
		if node.name == "root" {
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}

		var connector string
		if node.name == "root" {
			connector = ""
		} else if isLast {
			connector = prefix + "└── "
		} else {
			connector = prefix + "├── "
		}

		if node.name != "root" {
			fmt.Fprintf(w, "%s%s\n", connector, child.name)
		}
		printNode(w, child, newPrefix)
	}
}
