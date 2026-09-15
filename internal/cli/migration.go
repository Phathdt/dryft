package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func MigrationCommand() *cli.Command {
	return &cli.Command{
		Name:  "migration",
		Usage: "Migration operations",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "Generate Goose migration",
				ArgsUsage: "<name>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "List migrations",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
		},
	}
}
