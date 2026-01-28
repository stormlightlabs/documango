package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/ingest/golang"
)

func main() {
	ctx := context.Background()
	root := &cobra.Command{
		Use:   "documango",
		Short: "Documango is a terminal-first documentation browser",
	}
	root.SetUsageTemplate(usageTemplate())

	root.AddCommand(newDBCommand())
	root.AddCommand(newSearchCommand())
	root.AddCommand(newDocCommand())
	root.AddCommand(newIngestCommand())
	root.SilenceUsage = true
	root.SilenceErrors = true

	if err := fang.Execute(ctx, root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database operations",
	}
	cmd.AddCommand(newDBInitCommand())
	cmd.AddCommand(newDBAddCommand())
	return cmd
}

func newDBInitCommand() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize a .usde database",
		Example: "documango db init -d ./tmp/docs.usde",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := db.EnsureDir(dbPath); err != nil {
				return err
			}
			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.Init(ctx); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized %s\n", dbPath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&dbPath, "db", "d", "documango.usde", "Path to database")
	return cmd
}

func newDBAddCommand() *cobra.Command {
	var dbPath string
	var docPath string
	var filePath string
	var name string
	var entryType string
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Insert a markdown document into the database",
		Example: "documango db add -d ./tmp/docs.usde -p go/net/http -f ./doc.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if docPath == "" {
				return errors.New("--path is required")
			}
			if filePath == "" {
				return errors.New("--file is required")
			}
			content, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			compressed, err := codec.Compress(content)
			if err != nil {
				return err
			}

			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.EnsureSchema(ctx); err != nil {
				return err
			}

			if name == "" {
				name = filepath.Base(docPath)
			}
			if entryType == "" {
				entryType = "Doc"
			}

			docID, err := store.InsertDocument(ctx, db.Document{
				Path:   docPath,
				Format: "markdown",
				Body:   compressed,
				Hash:   db.HashBytes(content),
			})
			if err != nil {
				return err
			}

			body := string(content)
			if err := store.InsertSearchEntry(ctx, db.SearchEntry{
				Name:  name,
				Type:  entryType,
				Body:  body,
				DocID: docID,
			}); err != nil {
				return err
			}

			summary := firstLine(body)
			if err := store.InsertAgentContext(ctx, db.AgentContext{
				DocID:   docID,
				Symbol:  name,
				Summary: summary,
			}); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Inserted %s (%d)\n", docPath, docID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&dbPath, "db", "d", "documango.usde", "Path to database")
	cmd.Flags().StringVarP(&docPath, "path", "p", "", "Document path (e.g., go/net/http)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Markdown file to ingest")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Symbol name for search index")
	cmd.Flags().StringVarP(&entryType, "type", "t", "", "Symbol type for search index")
	return cmd
}

func newSearchCommand() *cobra.Command {
	var dbPath string
	var limit int
	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Search the FTS index",
		Args:    cobra.MinimumNArgs(1),
		Example: "documango search -d ./tmp/docs.usde -l 10 Client",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			results, err := store.Search(context.Background(), query, limit)
			if err != nil {
				return err
			}
			for _, res := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d\t%.4f\n", res.Name, res.Type, res.DocID, res.Score)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&dbPath, "db", "d", "documango.usde", "Path to database")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Max results")
	return cmd
}

func newDocCommand() *cobra.Command {
	var dbPath string
	var render bool
	var width int
	cmd := &cobra.Command{
		Use:     "doc <path>",
		Short:   "Print a document by path",
		Args:    cobra.ExactArgs(1),
		Example: "documango doc -d ./tmp/docs.usde -r -w 80 go/net/http",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			docRow, err := store.ReadDocument(context.Background(), path)
			if err != nil {
				return err
			}
			decompressed, err := codec.Decompress(docRow.Body)
			if err != nil {
				return err
			}
			if !render {
				_, err = cmd.OutOrStdout().Write(decompressed)
				return err
			}
			rendered, err := renderMarkdown(decompressed, width)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(rendered))
			return err
		},
	}
	cmd.Flags().StringVarP(&dbPath, "db", "d", "documango.usde", "Path to database")
	cmd.Flags().BoolVarP(&render, "render", "r", false, "Render markdown with glamour")
	cmd.Flags().IntVarP(&width, "width", "w", 0, "Render width (defaults to terminal width)")
	cmd.AddCommand(newDocSectionCommand())
	return cmd
}

func newIngestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest documentation sources",
	}
	cmd.AddCommand(newIngestGoCommand())
	return cmd
}

func newIngestGoCommand() *cobra.Command {
	var dbPath string
	var version string
	var stdlib bool
	var start string
	var maxPackages int
	cmd := &cobra.Command{
		Use:   "go [module]",
		Short: "Ingest Go docs (module via proxy.golang.org or stdlib)",
		Example: strings.TrimSpace(`
documango ingest go -d ./tmp/docs.usde golang.org/x/net
documango ingest go -d ./tmp/docs.usde --stdlib -s net/http -m 1
`),
		Args: func(cmd *cobra.Command, args []string) error {
			if stdlib {
				if len(args) != 0 {
					return errors.New("module argument not allowed with --stdlib")
				}
				return nil
			}
			if len(args) != 1 {
				return errors.New("module argument is required unless --stdlib is set")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := db.EnsureDir(dbPath); err != nil {
				return err
			}
			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.EnsureSchema(ctx); err != nil {
				return err
			}

			if stdlib {
				if err := golang.IngestStdlib(ctx, golang.StdlibOptions{
					DB:          store,
					Version:     version,
					Start:       start,
					MaxPackages: maxPackages,
				}); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Ingested Go standard library")
				return nil
			}
			modulePath := args[0]
			if err := golang.IngestModule(ctx, golang.Options{
				Module:  modulePath,
				Version: version,
				DB:      store,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Ingested %s\n", modulePath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&dbPath, "db", "d", "documango.usde", "Path to database (.usde)")
	cmd.Flags().StringVarP(&version, "version", "v", "", "Go module version (module mode) or Go toolchain tag (stdlib mode, e.g. go1.25.6)")
	cmd.Flags().BoolVar(&stdlib, "stdlib", false, "Use stdlib mode (no module argument)")
	cmd.Flags().StringVarP(&start, "start", "s", "", "Start at a specific stdlib package path (stdlib mode only)")
	cmd.Flags().IntVarP(&maxPackages, "max-packages", "m", 0, "Limit number of stdlib packages ingested (stdlib mode only)")
	return cmd
}

func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func renderMarkdown(input []byte, width int) (string, error) {
	if width <= 0 {
		width = 80
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return renderer.Render(string(input))
}

func usageTemplate() string {
	return strings.TrimSpace(`
{{with or .Long .Short }}{{. | trimTrailingWhitespaces}}{{end}}

Usage:
  {{.UseLine}}

Aliases:
  {{.Aliases}}

Examples:
{{.Example}}

Available Commands:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}

POSIX-style flags:
  Short flags are supported for primary options (e.g., -d for --db).
  Single-dash flags may be grouped when they take no argument.

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)
}

func newDocSectionCommand() *cobra.Command {
	var dbPath string
	var query string
	var render bool
	var width int
	var forceRG bool
	var forceGrep bool
	cmd := &cobra.Command{
		Use:     "section <path>",
		Short:   "Print a markdown section by heading match (uses ripgrep)",
		Args:    cobra.ExactArgs(1),
		Example: "documango doc section -d ./tmp/docs.usde -q \"type Client\" -r -w 80 go/net/http",
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return errors.New("--query is required")
			}
			if forceRG && forceGrep {
				return errors.New("choose only one of --rg or --gr")
			}
			path := args[0]
			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			docRow, err := store.ReadDocument(context.Background(), path)
			if err != nil {
				return err
			}
			decompressed, err := codec.Decompress(docRow.Body)
			if err != nil {
				return err
			}
			mode := "auto"
			if forceRG {
				mode = "rg"
			} else if forceGrep {
				mode = "grep"
			}
			section, err := extractSectionWithRG(decompressed, query, mode)
			if err != nil {
				return err
			}
			if render {
				rendered, err := renderMarkdown([]byte(section), width)
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write([]byte(rendered))
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(section))
			return err
		},
	}
	cmd.Flags().StringVarP(&dbPath, "db", "d", "documango.usde", "Path to database (.usde)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Heading match text (e.g., \"type writeRequest\")")
	cmd.Flags().BoolVarP(&render, "render", "r", false, "Render markdown with glamour")
	cmd.Flags().IntVarP(&width, "width", "w", 0, "Render width (defaults to terminal width)")
	cmd.Flags().BoolVarP(&forceRG, "rg", "R", false, "Force ripgrep usage")
	cmd.Flags().BoolVarP(&forceGrep, "gr", "G", false, "Force grep usage")
	return cmd
}

func extractSectionWithRG(markdown []byte, query string, mode string) (string, error) {
	tmp, err := os.CreateTemp("", "documango-md-*.md")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(markdown); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	defer os.Remove(tmp.Name())

	pattern := fmt.Sprintf(`^#{1,6} .*%s.*$`, regexp.QuoteMeta(query))
	line, err := findHeadingLine(tmp.Name(), pattern, mode)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(line, ":", 3)
	var lineStr string
	switch len(parts) {
	case 2:
		lineStr = parts[0]
	case 3:
		lineStr = parts[1]
	default:
		return "", errors.New("failed to parse ripgrep output")
	}
	lineNum, err := strconv.Atoi(lineStr)
	if err != nil || lineNum <= 0 {
		return "", errors.New("invalid line number from ripgrep")
	}
	lines := strings.Split(string(markdown), "\n")
	if lineNum-1 >= len(lines) {
		return "", errors.New("heading line out of range")
	}
	start := lineNum - 1
	level := headingLevel(lines[start])
	if level == 0 {
		return "", errors.New("matched line is not a heading")
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if lvl := headingLevel(lines[i]); lvl > 0 && lvl <= level {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), nil
}

func findHeadingLine(filename, pattern string, mode string) (string, error) {
	if mode == "rg" || mode == "auto" {
		if _, err := exec.LookPath("rg"); err == nil {
			cmd := exec.Command("rg", "--line-number", "--max-count", "1", pattern, filename)
			out, err := cmd.Output()
			if err == nil {
				return strings.TrimSpace(string(out)), nil
			}
			if mode == "rg" {
				return "", fmt.Errorf("no matching heading found for %q", pattern)
			}
		} else if mode == "rg" {
			return "", errors.New("ripgrep (rg) not found in PATH")
		}
	}
	if mode == "grep" || mode == "auto" {
		if _, err := exec.LookPath("grep"); err != nil {
			return "", errors.New("grep not found in PATH")
		}
		cmd := exec.Command("grep", "-n", "-m", "1", "-E", pattern, filename)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("no matching heading found for %q", pattern)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if mode != "auto" {
		return "", fmt.Errorf("unknown mode %q", mode)
	}
	return "", errors.New("ripgrep (rg) or grep not found in PATH")
}

func headingLevel(line string) int {
	if !strings.HasPrefix(line, "#") {
		return 0
	}
	count := 0
	for count < len(line) && line[count] == '#' {
		count++
	}
	if count == 0 || count >= len(line) || line[count] != ' ' {
		return 0
	}
	return count
}
