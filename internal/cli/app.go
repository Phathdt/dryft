// Package cli provides command-line interface components for dryft.
package cli

import (
	"github.com/urfave/cli/v3"
)

// NewApp creates and configures the main CLI application.
func NewApp() *cli.Command {
	return &cli.Command{
		Name:    "dryft",
		Version: "v0.1.0",
		Usage:   "Schema in. Migration out.",
		Commands: []*cli.Command{
			InitCommand(),
			DBCommand(),
			MigrationCommand(),
			DiffCommand(),
			ValidateCommand(),
		},
	}
}
