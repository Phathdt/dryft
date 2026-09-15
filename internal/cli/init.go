package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize dryft in current directory",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runInit()
		},
	}
}

func runInit() error {
	configPath := ".dryft.yaml"

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("dryft already initialized: %s exists", configPath)
	}

	if err := os.MkdirAll("prisma", 0755); err != nil {
		return fmt.Errorf("failed to create prisma directory: %w", err)
	}

	if err := os.MkdirAll("migrations", 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	if err := os.MkdirAll(".dryft", 0755); err != nil {
		return fmt.Errorf("failed to create .dryft directory: %w", err)
	}

	config := map[string]interface{}{
		"database": map[string]interface{}{
			"provider": "postgresql",
			"url":      "${DATABASE_URL}",
		},
		"schema": map[string]interface{}{
			"file": "prisma/schema.prisma",
			"naming": map[string]interface{}{
				"fields": "camelCase",
				"tables": "PascalCase",
			},
		},
		"migration": map[string]interface{}{
			"directory": "migrations",
			"format":    "goose",
			"naming":    "timestamp",
		},
		"goose": map[string]interface{}{
			"version_table": "goose_db_version",
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println("✓ Initialized dryft")
	fmt.Println()
	fmt.Println("Created:")
	fmt.Println("  - .dryft.yaml (config)")
	fmt.Println("  - prisma/ (schemas)")
	fmt.Println("  - migrations/ (migration files)")
	fmt.Println("  - .dryft/ (state directory)")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Set DATABASE_URL in your environment")
	fmt.Println("  2. Run 'dryft db pull' to introspect your database")
	fmt.Println("  3. Run 'dryft migration create <name>' to generate migrations")

	return nil
}
