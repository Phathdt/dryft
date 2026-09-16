package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phathdt/dryft/internal/config"
	"github.com/phathdt/dryft/internal/introspect/postgres"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/urfave/cli/v3"
)

// DBCommand creates the db command for database operations.
func DBCommand() *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "Database operations",
		Commands: []*cli.Command{
			{
				Name:  "pull",
				Usage: "Introspect DB → schema.prisma",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return cmdDbPull(ctx)
				},
			},
			{
				Name:  "inspect",
				Usage: "Print schema without writes",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return cmdDbInspect(ctx)
				},
			},
			{
				Name:  "baseline",
				Usage: "Mark current DB as baseline",
				Action: func(_ context.Context, _ *cli.Command) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
		},
	}
}

// cmdDbPull introspects the database and writes schema.prisma file.
func cmdDbPull(ctx context.Context) error {
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
	defer func() {
		if err := intr.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close introspector: %v\n", err)
		}
	}()

	// Introspect schema
	schema, err := intr.Introspect(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect schema: %w", err)
	}

	// Create Prisma writer with default naming convention
	writer := prisma.NewWriter(prisma.DefaultNamingConvention())

	// Convert to Prisma schema
	prismaSchema, err := writer.Write(schema)
	if err != nil {
		return fmt.Errorf("failed to generate Prisma schema: %w", err)
	}

	// Determine output file path
	schemaFile := cfg.Schema.File
	if schemaFile == "" {
		schemaFile = "prisma/schema.prisma"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(schemaFile), 0755); err != nil {
		return fmt.Errorf("failed to create schema directory: %w", err)
	}

	// Write schema to file
	if err := os.WriteFile(schemaFile, []byte(prismaSchema), 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	fmt.Printf("✓ Schema written to %s\n", schemaFile)
	fmt.Printf("  Models: %d\n", len(schema.Tables))
	fmt.Printf("  Enums: %d\n", len(schema.Enums))

	return nil
}

// cmdDbInspect introspects the database and prints the schema as JSON.
func cmdDbInspect(ctx context.Context) error {
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
	defer func() {
		if err := intr.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close introspector: %v\n", err)
		}
	}()

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
