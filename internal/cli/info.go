package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/db"
)

// newInfoCommand creates the info command.
func newInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <path>",
		Short: "Show metadata for a document",
		Long: `Display metadata and statistics for a document stored in the database.

This includes symbol count, size, hash, and other context information.`,
		Example: `  documango info go/net/http
  documango info atproto/lexicon/com.atproto.repo.createRecord`,
		Args:              cobra.ExactArgs(1),
		RunE:              runInfo,
		ValidArgsFunction: readPathCompletion,
	}

	return cmd
}

func runInfo(cmd *cobra.Command, args []string) error {
	path := args[0]
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
	docRow, err := store.ReadDocument(ctx, path)
	if err != nil {
		return err
	}

	symbolCount, size, err := getDocumentInfo(ctx, store, docRow.Path)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Path:   %s\n", docRow.Path)
	fmt.Fprintf(cmd.OutOrStdout(), "Format: %s\n", docRow.Format)
	fmt.Fprintf(cmd.OutOrStdout(), "Size:   %d bytes (compressed: %d bytes)\n", size, len(docRow.Body))
	fmt.Fprintf(cmd.OutOrStdout(), "Hash:   %s\n", docRow.Hash)
	fmt.Fprintf(cmd.OutOrStdout(), "Symbols: %d\n", symbolCount)

	return nil
}

func getDocumentInfo(ctx context.Context, store *db.Store, path string) (int, int64, error) {
	rows, err := store.DB().QueryContext(ctx,
		`SELECT COUNT(DISTINCT symbol), SUM(LENGTH(signature)) FROM agent_context WHERE doc_id = (SELECT id FROM documents WHERE path = ?)`,
		path,
	)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, 0, nil
	}

	var count int
	var size int64
	if err := rows.Scan(&count, &size); err != nil {
		return 0, 0, err
	}

	return count, size, rows.Err()
}
