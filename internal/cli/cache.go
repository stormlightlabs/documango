package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/cache"
)

var (
	cachePruneAge int
	cacheType     string
)

func newCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the documentation cache",
		Long: `Manage the local cache of downloaded documentation sources.

The cache stores downloaded Go modules, stdlib packages, and git repositories
to avoid re-fetching on every ingestion.`,
	}

	cmd.AddCommand(newCacheStatusCommand())
	cmd.AddCommand(newCacheListCommand())
	cmd.AddCommand(newCachePruneCommand())
	cmd.AddCommand(newCacheClearCommand())

	return cmd
}

func newCacheStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show cache statistics",
		RunE:  runCacheStatus,
	}

	return cmd
}

func runCacheStatus(cmd *cobra.Command, args []string) error {
	cacheDir, err := cache.CacheDir()
	if err != nil {
		return err
	}

	c, err := cache.New(cacheDir)
	if err != nil {
		return err
	}

	totalSize := c.Size()
	fileCount := 0
	for _, entry := range c.List("") {
		if !entry.IsExpired() && entry.Path != "_git_meta" {
			fileCount++
		}
	}

	gitCache, err := cache.NewGitCache(cacheDir)
	if err == nil {
		gitCount := gitCache.Count()

		p.PrintListItem("Cache Directory", p.FormatPath(cacheDir))
		p.PrintListItem("File Cache", fmt.Sprintf("%s (%d files)", formatBytes(totalSize), fileCount))
		if gitCount > 0 {
			p.PrintListItem("Git Repos", fmt.Sprintf("%d tracked", gitCount))
		}
	} else {
		p.PrintListItem("Cache Directory", p.FormatPath(cacheDir))
		p.PrintListItem("Total Size", formatBytes(totalSize))
		p.PrintListItem("Entries", fmt.Sprintf("%d", fileCount))
	}

	return nil
}

func newCacheListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [prefix]",
		Short: "List cached items",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCacheList,
	}

	cmd.Flags().StringVar(&cacheType, "type", "", "Filter by type prefix")

	return cmd
}

func runCacheList(cmd *cobra.Command, args []string) error {
	cacheDir, err := cache.CacheDir()
	if err != nil {
		return err
	}

	c, err := cache.New(cacheDir)
	if err != nil {
		return err
	}

	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}
	if cacheType != "" {
		prefix = cacheType + "/" + prefix
	}

	if cacheType == "" || cacheType == "atproto" {
		gitCache, err := cache.NewGitCache(cacheDir)
		if err == nil {
			commits := gitCache.ListCommits()
			if len(commits) > 0 && (cacheType == "" || cacheType == "atproto") {
				fmt.Fprintln(cmd.OutOrStdout(), "Git Repositories:")
				for key, commit := range commits {
					if cacheType == "" || strings.HasPrefix(key, cacheType+"/") {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", key)
						fmt.Fprintf(cmd.OutOrStdout(), "    Commit:   %s\n", commit)
						fmt.Fprintln(cmd.OutOrStdout())
					}
				}
			}
		}
	}

	entries := c.List(prefix)
	fileEntries := make([]*cache.CacheEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Path != "_git_meta" {
			fileEntries = append(fileEntries, entry)
		}
	}

	if len(fileEntries) == 0 {
		return nil
	}

	if cacheType == "" || cacheType != "atproto" {
		for _, entry := range fileEntries {
			age := time.Since(entry.FetchedAt)
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", entry.Source)
			fmt.Fprintf(cmd.OutOrStdout(), "  Path:      %s\n", entry.Path)
			fmt.Fprintf(cmd.OutOrStdout(), "  Size:      %s\n", formatBytes(entry.Size))
			fmt.Fprintf(cmd.OutOrStdout(), "  Age:       %s\n", formatDuration(age))
			fmt.Fprintf(cmd.OutOrStdout(), "  Checksum:  %s\n", entry.Checksum)
			if !entry.ExpiresAt.IsZero() {
				fmt.Fprintf(cmd.OutOrStdout(), "  Expires:   %s\n", entry.ExpiresAt.Format(time.RFC3339))
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return nil
}

func newCachePruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove old or expired cache entries",
		RunE:  runCachePrune,
	}

	cmd.Flags().IntVar(&cachePruneAge, "age", 0, "Prune entries older than N days")
	cmd.Flags().StringVar(&cacheType, "type", "", "Only prune specific type")

	return cmd
}

func runCachePrune(cmd *cobra.Command, args []string) error {
	cacheDir, err := cache.CacheDir()
	if err != nil {
		return err
	}

	c, err := cache.New(cacheDir)
	if err != nil {
		return err
	}

	maxAge := time.Duration(0)
	if cachePruneAge > 0 {
		maxAge = time.Duration(cachePruneAge) * 24 * time.Hour
	}

	count, err := c.Prune(maxAge)
	if err != nil {
		return err
	}

	if !quiet {
		p.PrintSuccess(fmt.Sprintf("Pruned %d cache entries", count))
	}

	return nil
}

func newCacheClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all cache entries",
		RunE:  runCacheClear,
	}

	cmd.Flags().StringVar(&cacheType, "type", "", "Only clear specific type")

	return cmd
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	cacheDir, err := cache.CacheDir()
	if err != nil {
		return err
	}

	c, err := cache.New(cacheDir)
	if err != nil {
		return err
	}

	if cacheType != "" {
		entries := c.List(cacheType + "/")
		count := 0
		for _, entry := range entries {
			if err := c.Delete(filepath.Join(cacheDir, entry.Path)); err == nil {
				count++
			}
		}
		if !quiet {
			p.PrintSuccess(fmt.Sprintf("Cleared %d cache entries", count))
		}
		return nil
	}

	if err := c.Clear(); err != nil {
		return err
	}

	if !quiet {
		p.PrintSuccess("Cache cleared")
	}

	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}
