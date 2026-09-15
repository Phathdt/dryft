package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func ValidateCommand() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate schema + config",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("Not implemented yet")
			return nil
		},
	}
}
