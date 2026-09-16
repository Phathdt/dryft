package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phathdt/dryft/internal/config"
	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/migration"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	"github.com/phathdt/dryft/internal/sql/postgres"
	"github.com/urfave/cli/v3"
)

// MigrationCommand creates the migration command for migration operations.
func MigrationCommand() *cli.Command {
	return &cli.Command{
		Name:  "migration",
		Usage: "Migration operations",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "Generate Goose migration from schema diff",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "allow-destructive",
						Usage: "Allow destructive operations (DROP TABLE, DROP COLUMN, etc.)",
					},
				},
				Action: migrationCreateAction,
			},
			{
				Name:  "status",
				Usage: "List migrations",
				Action: func(_ context.Context, _ *cli.Command) error {
					fmt.Println("Not implemented yet")
					return nil
				},
			},
		},
	}
}

func migrationCreateAction(_ context.Context, cmd *cli.Command) error {
	// 1. Get migration name
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("migration name required")
	}
	name := cmd.Args().Get(0)

	// 2. Load config
	cfg, err := config.Load(".dryft.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// 3. Check if schema file exists
	schemaPath := cfg.Schema.File
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s\nRun 'dryft db pull' first", schemaPath)
	}

	// 4. Parse current schema
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	parseResult, err := prisma.Parse(string(schemaContent))
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}
	currentSchema := parseResult.Schema

	// Print parser warnings if any
	if len(parseResult.Warnings) > 0 {
		fmt.Println("⚠ Parser warnings:")
		for _, warning := range parseResult.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
		fmt.Println()
	}

	// 5. TODO: Load previous schema state (for now, assume empty schema as "before")
	// In real implementation, this would come from:
	// - Last migration state in .dryft/state.json, OR
	// - Introspect current database
	previousSchema := &schema.Schema{
		Tables: []schema.Table{},
		Enums:  []schema.Enum{},
	}

	// 6. Run diff
	differ := diff.NewDiffer(nil) // No rename hints for now
	operations, err := differ.Diff(previousSchema, currentSchema)
	if err != nil {
		return fmt.Errorf("failed to diff schemas: %w", err)
	}

	if len(operations) == 0 {
		fmt.Println("✓ No schema changes detected")
		return nil
	}

	// 7. Plan operations
	planner := diff.NewPlanner()
	plan, err := planner.Plan(operations)
	if err != nil {
		return fmt.Errorf("failed to plan operations: %w", err)
	}

	// 8. Check for destructive operations
	allowDestructive := cmd.Bool("allow-destructive")
	if len(plan.Destructive) > 0 && !allowDestructive {
		fmt.Println("⚠ Destructive operations detected:")
		for _, op := range plan.Destructive {
			fmt.Printf("  - %s (%s)\n", op.Description(), op.IsDestructive())
		}
		fmt.Println("\nUse --allow-destructive to proceed")
		return fmt.Errorf("destructive operations not allowed")
	}

	// 9. Generate SQL
	generator := postgres.NewGenerator(sql.GeneratorOptions{})
	upStatements, err := generator.Generate(plan.Operations)
	if err != nil {
		return fmt.Errorf("failed to generate SQL: %w", err)
	}

	downStatements, warnings, err := generator.GenerateReverse(plan.Operations)
	if err != nil {
		return fmt.Errorf("failed to generate reverse SQL: %w", err)
	}

	// Combine plan warnings with generator warnings
	allWarnings := append(plan.Warnings, warnings...)

	// 10. Format as Goose migration
	migrationContent := migration.FormatGoose(name, upStatements, downStatements, allWarnings)
	filename := migration.GenerateFilename(name)

	// 11. Write migration file
	migrationDir := cfg.Migration.Directory
	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory: %w", err)
	}

	migrationPath := filepath.Join(migrationDir, filename)
	if err := os.WriteFile(migrationPath, []byte(migrationContent), 0644); err != nil {
		return fmt.Errorf("failed to write migration: %w", err)
	}

	// 12. Print summary
	fmt.Printf("✓ Created migration: %s\n\n", filename)
	fmt.Printf("Operations:\n")
	for _, op := range plan.Operations {
		fmt.Printf("  - %s\n", op.Description())
	}

	if len(allWarnings) > 0 {
		fmt.Printf("\n⚠ Warnings:\n")
		for _, warning := range allWarnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review the migration: %s\n", migrationPath)
	fmt.Printf("  2. Apply it: goose -dir %s postgres $DATABASE_URL up\n", migrationDir)

	return nil
}
