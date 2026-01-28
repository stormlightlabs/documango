package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/config"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage the documango configuration file.

Configuration is stored in TOML format in the XDG config directory.`,
	}

	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigEditCommand())
	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigGetCommand())
	cmd.AddCommand(newConfigPathCommand())

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		RunE:  runConfigShow,
	}

	return cmd
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	configPath, err := config.ConfigDir()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "# Configuration file: %s/config.toml\n\n", configPath)
	fmt.Fprintf(cmd.OutOrStdout(), "[database]\n")
	fmt.Fprintf(cmd.OutOrStdout(), "default = %q\n\n", cfg.Database.Default)
	fmt.Fprintf(cmd.OutOrStdout(), "[cache]\n")
	fmt.Fprintf(cmd.OutOrStdout(), "max_size_bytes = %d\n", cfg.Cache.MaxSizeBytes)
	fmt.Fprintf(cmd.OutOrStdout(), "max_age_days = %d\n", cfg.Cache.MaxAgeDays)
	fmt.Fprintf(cmd.OutOrStdout(), "ttl_seconds = %d\n\n", cfg.Cache.TTLSeconds)
	fmt.Fprintf(cmd.OutOrStdout(), "[search]\n")
	fmt.Fprintf(cmd.OutOrStdout(), "default_limit = %d\n\n", cfg.Search.DefaultLimit)
	fmt.Fprintf(cmd.OutOrStdout(), "[display]\n")
	fmt.Fprintf(cmd.OutOrStdout(), "width = %d\n", cfg.Display.Width)
	fmt.Fprintf(cmd.OutOrStdout(), "use_pager = %v\n", cfg.Display.UsePager)
	fmt.Fprintf(cmd.OutOrStdout(), "render_markdown = %v\n", cfg.Display.RenderMarkdown)
	if cfg.Display.ColorOutput != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "color_output = %v\n", *cfg.Display.ColorOutput)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "color_output = auto\n")
	}

	return nil
}

func newConfigEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open configuration in editor",
		RunE:  runConfigEdit,
	}

	return cmd
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		cfg := config.DefaultConfig()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	editCmd := exec.Command(editor, configPath)
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = os.Stdout
	editCmd.Stderr = os.Stderr

	return editCmd.Run()
}

func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	}

	return cmd
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "database.default":
		cfg.Database.Default = value
	case "cache.max_size_bytes":
		var size int64
		if _, err := fmt.Sscanf(value, "%d", &size); err != nil {
			return fmt.Errorf("invalid size: %s", value)
		}
		cfg.Cache.MaxSizeBytes = size
	case "cache.max_age_days":
		var days int
		if _, err := fmt.Sscanf(value, "%d", &days); err != nil {
			return fmt.Errorf("invalid days: %s", value)
		}
		cfg.Cache.MaxAgeDays = days
	case "search.default_limit":
		var limit int
		if _, err := fmt.Sscanf(value, "%d", &limit); err != nil {
			return fmt.Errorf("invalid limit: %s", value)
		}
		cfg.Search.DefaultLimit = limit
	case "display.width":
		var width int
		if _, err := fmt.Sscanf(value, "%d", &width); err != nil {
			return fmt.Errorf("invalid width: %s", value)
		}
		cfg.Display.Width = width
	case "display.use_pager":
		var usePager bool
		switch value {
		case "true":
			usePager = true
		case "false":
			usePager = false
		default:
			return fmt.Errorf("invalid boolean: %s (use true/false)", value)
		}
		cfg.Display.UsePager = usePager
	case "display.render_markdown":
		var render bool
		switch value {
		case "true":
			render = true
		case "false":
			render = false
		default:
			return fmt.Errorf("invalid boolean: %s (use true/false)", value)
		}
		cfg.Display.RenderMarkdown = render
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	if !quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
	}

	return nil
}

func newConfigGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigGet,
	}

	return cmd
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "database.default":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Database.Default)
	case "cache.max_size_bytes":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Cache.MaxSizeBytes)
	case "cache.max_age_days":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Cache.MaxAgeDays)
	case "cache.ttl_seconds":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Cache.TTLSeconds)
	case "search.default_limit":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Search.DefaultLimit)
	case "display.width":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Display.Width)
	case "display.use_pager":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Display.UsePager)
	case "display.render_markdown":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Display.RenderMarkdown)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func newConfigPathCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print configuration file path",
		RunE:  runConfigPath,
	}

	return cmd
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), configPath)
	return nil
}

func configFilePath() (string, error) {
	configDir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/config.toml", configDir), nil
}
