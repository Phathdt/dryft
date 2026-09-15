package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func DBCommand() *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "Database operations",
		Commands: []*cli.Command{
			{
				Name:  "pull",
				Usage: "Introspect DB → schema.prisma",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
			{
				Name:  "inspect",
				Usage: "Print schema without writes",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
			{
				Name:  "baseline",
				Usage: "Mark current DB as baseline",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
		},
	}
}
