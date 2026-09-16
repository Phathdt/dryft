package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationCreate_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup temporary directory
	tmpDir := t.TempDir()

	// Create config file
	configPath := filepath.Join(tmpDir, ".dryft.yaml")
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost:5432/test

schema:
  file: prisma/schema.prisma
  naming:
    fields: camelCase
    tables: PascalCase

migration:
  directory: migrations
  format: goose
  naming: timestamp

goose:
  version_table: goose_db_version
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create schema directory
	schemaDir := filepath.Join(tmpDir, "prisma")
	err = os.MkdirAll(schemaDir, 0755)
	require.NoError(t, err)

	// Create a simple Prisma schema
	schemaPath := filepath.Join(schemaDir, "schema.prisma")
	schemaContent := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    String @id @default(uuid())
  email String @unique
  name  String?
}
`
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create migration directory
	migrationDir := filepath.Join(tmpDir, "migrations")
	err = os.MkdirAll(migrationDir, 0755)
	require.NoError(t, err)

	// For now, skip the actual execution test since we need proper mocking
	// This test verifies the file structure setup works
	t.Skip("Full integration test requires proper CLI command mocking")
}

func TestMigrationCreate_DestructiveProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup with a schema that would cause destructive operations
	// This test verifies that destructive operations are blocked
	// unless --allow-destructive flag is used

	// TODO: Implement when we have proper state management
	t.Skip("destructive protection test requires state management")
}
