package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
)

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

	decompressed, err := codec.Decompress(docRow.Body)
	if err != nil {
		return err
	}

	symbolCount, err := getDocumentSymbols(ctx, store, docRow.Path)
	if err != nil {
		return err
	}

	p.PrintListItem("Path", p.FormatPath(docRow.Path))
	p.PrintListItem("Format", docRow.Format)
	p.PrintListItem("Size", fmt.Sprintf("%d bytes (compressed: %d bytes)", len(decompressed), len(docRow.Body)))
	p.PrintListItem("Hash", docRow.Hash)
	p.PrintListItem("Symbols", fmt.Sprintf("%d", symbolCount))

	return nil
}

func getDocumentSymbols(ctx context.Context, store *db.Store, path string) (int, error) {
	var count int
	if err := store.DB().QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT symbol)
		 FROM agent_context
		 WHERE doc_id = (SELECT id FROM documents WHERE path = ?)`,
		path,
	).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
