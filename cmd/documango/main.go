package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/stormlightlabs/documango/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	log.SetTimeFormat("2006-01-02 15:04:05")
	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(log.DebugLevel)
	}
	log.Debug("debug logging enabled")
}
