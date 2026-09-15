package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func DiffCommand() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "Show schema changes",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("Not implemented yet")
			return nil
		},
	}
}
