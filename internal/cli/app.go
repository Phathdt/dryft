// Package cli provides command-line interface components for dryft.
package cli

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

var (
	// Version is the semantic version
	Version = "dev"
	// Commit is the git commit hash
	Commit = "unknown"
	// BuildDate is the build timestamp
	BuildDate = "unknown"
	// Arch is the target architecture
	Arch = "unknown"
)

// NewApp creates and configures the main CLI application.
func NewApp() *cli.Command {
	version := fmt.Sprintf("Version: %s\nGit Commit: %s\nBuild Date: %s\nArch: %s", Version, Commit, BuildDate, Arch)
	return &cli.Command{
		Name:    "dryft",
		Version: version,
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
