package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/phathdt/dryft/internal/config"
	"github.com/phathdt/dryft/internal/introspect/postgres"
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
					return cmdDbInspect(ctx, cmd)
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

// cmdDbInspect introspects the database and prints the schema as JSON.
func cmdDbInspect(ctx context.Context, cmd *cli.Command) error {
	// Load configuration
	cfg, err := config.Load(".dryft.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate database URL
	if cfg.Database.URL == "" {
		return fmt.Errorf("database.url not configured in .dryft.yaml")
	}

	// Create introspector
	intr, err := postgres.NewPostgresIntrospector(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to create introspector: %w", err)
	}
	defer intr.Close()

	// Introspect schema
	schema, err := intr.Introspect(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect schema: %w", err)
	}

	// Encode as JSON and print to stdout
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		return fmt.Errorf("failed to encode schema as JSON: %w", err)
	}

	return nil
}
