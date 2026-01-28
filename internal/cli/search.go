package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/db"
)

var (
	searchLimit   int
	searchType    string
	searchFormat  string
	searchFirst   bool
	searchPackage string
)

func newSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the documentation",
		Long: `Search the full-text index for documentation matching the query.

Results are ranked by relevance using BM25 ranking with exact matches
receiving a boost.`,
		Example: `  documango search "http.Client"
  documango search -l 50 -t Func "Write"
  documango search -f json "net/http"`,
		Args: cobra.ExactArgs(1),
		RunE: runSearch,
	}

	cmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Maximum number of results")
	cmd.Flags().StringVarP(&searchType, "type", "t", "", "Filter by symbol type (e.g., Func, Type, Package)")
	cmd.Flags().StringVarP(&searchFormat, "format", "f", "table", "Output format (table, json, paths)")
	cmd.Flags().BoolVarP(&searchFirst, "first", "1", false, "Return only the top result")
	cmd.Flags().StringVarP(&searchPackage, "package", "p", "", "Filter by package path prefix")

	return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
	dbPath, err := resolveDBPath()
	if err != nil {
		return err
	}

	store, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	query := strings.Join(args, " ")
	if searchType != "" {
		query = fmt.Sprintf("type:%s %s", searchType, query)
	}

	limit := searchLimit
	if searchFirst {
		limit = 1
	}

	packagePrefix := searchPackage
	if packagePrefix != "" && !strings.HasPrefix(packagePrefix, "go/") {
		packagePrefix = "go/" + packagePrefix
	}

	ctx := context.Background()
	results, err := store.SearchPackage(ctx, query, packagePrefix, limit)
	if err != nil {
		return err
	}

	if len(results) == 0 && !quiet {
		p.PrintError("No results found")
		return nil
	}

	switch searchFormat {
	case "json":
		return outputSearchJSON(cmd, results)
	case "paths":
		return outputSearchPaths(cmd, results)
	default:
		return outputSearchTable(cmd, results)
	}
}

func outputSearchTable(cmd *cobra.Command, results []db.SearchResult) error {
	if len(results) == 0 {
		return nil
	}

	maxScore := results[0].Score
	for _, res := range results {
		if res.Score > maxScore {
			maxScore = res.Score
		}
	}

	t := table.NewWriter()
	t.SetOutputMirror(cmd.OutOrStdout())
	t.AppendHeader(table.Row{"Name", "Type", "Doc ID", "Score (BM25 Relevance)"})

	for _, res := range results {
		t.AppendRow(table.Row{res.Name, res.Type, res.DocID, fmt.Sprintf("%.4f", res.Score)})
	}

	t.SetStyle(table.StyleRounded)
	t.Render()
	return nil
}

func outputSearchJSON(cmd *cobra.Command, results []db.SearchResult) error {
	fmt.Fprint(cmd.OutOrStdout(), "[")
	for i, res := range results {
		if i > 0 {
			fmt.Fprint(cmd.OutOrStdout(), ",")
		}
		fmt.Fprintf(cmd.OutOrStdout(), `{"name":"%s","type":"%s","doc_id":%d,"score":%.4f}`, res.Name, res.Type, res.DocID, res.Score)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "]")
	return nil
}

func outputSearchPaths(cmd *cobra.Command, results []db.SearchResult) error {
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
	for _, res := range results {
		doc, err := store.ReadDocumentByID(ctx, res.DocID)
		if err != nil {
			continue
		}
		fmt.Fprintln(cmd.OutOrStdout(), doc.Path)
	}
	return nil
}
