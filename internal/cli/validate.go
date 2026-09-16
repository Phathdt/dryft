package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/phathdt/dryft/internal/config"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/urfave/cli/v3"
)

// ValidateCommand creates the validate command for schema and config validation.
func ValidateCommand() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate schema + config",
		Action: func(_ context.Context, _ *cli.Command) error {
			return cmdValidate()
		},
	}
}

// cmdValidate validates the Prisma schema and dryft configuration.
func cmdValidate() error {
	// Load configuration
	cfg, err := config.Load(".dryft.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate database URL
	if cfg.Database.URL == "" {
		return fmt.Errorf("database.url not configured in .dryft.yaml")
	}

	// Check if schema file exists
	schemaPath := cfg.Schema.File
	if schemaPath == "" {
		schemaPath = "prisma/schema.prisma"
	}

	if _, statErr := os.Stat(schemaPath); os.IsNotExist(statErr) {
		return fmt.Errorf("schema file not found: %s", schemaPath)
	}

	// Read schema file
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	// Parse schema
	parseResult, err := prisma.Parse(string(schemaContent))
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Note: Datasource validation happens during parsing.
	// The schema object only contains Tables and Enums.
	if parseResult.Schema == nil {
		return fmt.Errorf("schema parsing resulted in nil schema")
	}

	// Print warnings if any
	if len(parseResult.Warnings) > 0 {
		fmt.Println("⚠ Warnings:")
		for _, warning := range parseResult.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
		fmt.Println()
	}

	// Print validation results
	fmt.Println("✓ Schema is valid")
	fmt.Printf("  Models: %d\n", len(parseResult.Schema.Tables))
	fmt.Printf("  Enums: %d\n", len(parseResult.Schema.Enums))

	return nil
}
