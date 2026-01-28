package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
)

var (
	readRender    bool
	readWidth     int
	readSection   string
	readPager     bool
	readNoPager   bool
	readForceRG   bool
	readForceGrep bool
)

func newReadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <path>",
		Short: "Read documentation by path",
		Long: `Read and display documentation from the database.

The path can be a document path (e.g., go/net/http) or a subcommand
for specific reading modes.`,
		Example: `  documango read go/net/http
  documango read -r -w 100 go/golang.org/x/net/http2
  documango read section -q "type Client" go/net/http`,
		Args:              cobra.MinimumNArgs(1),
		RunE:              runRead,
		ValidArgsFunction: readPathCompletion,
	}

	cmd.Flags().BoolVarP(&readRender, "render", "r", false, "Render markdown with glamour")
	cmd.Flags().IntVarP(&readWidth, "width", "w", 0, "Render width (defaults to terminal width)")
	cmd.Flags().StringVarP(&readSection, "section", "s", "", "Extract a section by heading match")
	cmd.Flags().BoolVarP(&readPager, "pager", "P", false, "Enable pager")
	cmd.Flags().BoolVarP(&readNoPager, "no-pager", "p", false, "Disable pager")

	cmd.AddCommand(newReadSectionCommand())
	return cmd
}

func runRead(cmd *cobra.Command, args []string) error {
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

	var output []byte
	if readRender {
		rendered, err := renderMarkdown(decompressed, readWidth)
		if err != nil {
			return err
		}
		output = []byte(rendered)
	} else {
		output = decompressed
	}

	usePager := shouldUsePager(len(output))
	if usePager {
		return pageOutput(cmd, output)
	}

	_, err = cmd.OutOrStdout().Write(output)
	return err
}

func newReadSectionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "section <path>",
		Short: "Extract a markdown section by heading match",
		Long: `Extract and display a section of markdown by finding a matching heading.

This command uses ripgrep (rg) if available, falling back to grep.
The heading is matched as a substring against heading lines.`,
		Example: `  documango read section -q "type Client" go/net/http
  documango read section -q "func ListenAndServe" -r go/net/http`,
		Args: cobra.ExactArgs(1),
		RunE: runReadSection,
	}

	cmd.Flags().StringVarP(&readSection, "query", "q", "", "Heading match text (required)")
	cmd.Flags().BoolVarP(&readRender, "render", "r", false, "Render markdown with glamour")
	cmd.Flags().IntVarP(&readWidth, "width", "w", 0, "Render width (defaults to terminal width)")
	cmd.Flags().BoolVarP(&readForceRG, "rg", "R", false, "Force ripgrep usage")
	cmd.Flags().BoolVarP(&readForceGrep, "gr", "G", false, "Force grep usage")

	return cmd
}

func runReadSection(cmd *cobra.Command, args []string) error {
	if readSection == "" {
		return errors.New("--query is required")
	}
	if readForceRG && readForceGrep {
		return errors.New("choose only one of --rg or --gr")
	}

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

	mode := "auto"
	if readForceRG {
		mode = "rg"
	} else if readForceGrep {
		mode = "grep"
	}

	section, err := extractSectionWithRG(decompressed, readSection, mode)
	if err != nil {
		return err
	}

	var output []byte
	if readRender {
		rendered, err := renderMarkdown([]byte(section), readWidth)
		if err != nil {
			return err
		}
		output = []byte(rendered)
	} else {
		output = []byte(section)
	}

	usePager := shouldUsePager(len(output))
	if usePager {
		return pageOutput(cmd, output)
	}

	_, err = cmd.OutOrStdout().Write(output)
	return err
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

func shouldUsePager(contentSize int) bool {
	if readNoPager {
		return false
	}
	if readPager {
		return true
	}
	if !isTerminal() {
		return false
	}
	return contentSize > 4096
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func pageOutput(cmd *cobra.Command, data []byte) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}

	pagerCmd := exec.Command(pager)
	pagerCmd.Stdin = cmd.InOrStdin()
	pagerCmd.Stdout = cmd.OutOrStdout()
	pagerCmd.Stderr = cmd.OutOrStderr()

	stdin, err := pagerCmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := pagerCmd.Start(); err != nil {
		return err
	}

	if _, err := stdin.Write(data); err != nil {
		return err
	}
	_ = stdin.Close()

	return pagerCmd.Wait()
}

func readPathCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
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

	line, err := findHeadingLine(tmp.Name(), query, mode)
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
		return "", errors.New("failed to parse search output")
	}

	lineNum, err := strconv.Atoi(lineStr)
	if err != nil || lineNum <= 0 {
		return "", errors.New("invalid line number from search")
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

func findHeadingLine(filename, query, mode string) (string, error) {
	pattern := fmt.Sprintf(`^#{1,6} .*%s.*$`, regexp.QuoteMeta(query))

	if mode == "rg" || mode == "auto" {
		if _, err := exec.LookPath("rg"); err == nil {
			cmd := exec.Command("rg", "--line-number", "--max-count", "1", pattern, filename)
			out, err := cmd.Output()
			if err == nil {
				return strings.TrimSpace(string(out)), nil
			}
			if mode == "rg" {
				return "", fmt.Errorf("no matching heading found for %q", query)
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
			return "", fmt.Errorf("no matching heading found for %q", query)
		}
		return strings.TrimSpace(string(out)), nil
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
