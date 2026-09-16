package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// DiffCommand creates the diff command for showing schema changes.
func DiffCommand() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "Show schema changes",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println("Not implemented yet")
			return nil
		},
	}
}
