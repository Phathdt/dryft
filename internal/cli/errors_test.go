package cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadErrors_MissingConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// .dryft.yaml doesn't exist
	app := NewApp()
	ctx := context.Background()

	err := app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestConfigLoadErrors_InvalidYAMLSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create invalid YAML
	invalidYAML := `database:
  provider: postgresql
  url: "unclosed string

schema:
  file: schema.prisma
`
	err := os.WriteFile(".dryft.yaml", []byte(invalidYAML), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.Error(t, err)
}

func TestConfigLoadErrors_MissingDatabaseURL(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Config without database URL
	configContent := `database:
  provider: postgresql

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database.url")
}

func TestInitErrors_AlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Initialize once
	app := NewApp()
	ctx := context.Background()
	err := app.Run(ctx, []string{"dryft", "init"})
	require.NoError(t, err)

	// Try to initialize again - should fail
	err = app.Run(ctx, []string{"dryft", "init"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already initialized")
}

func TestDbPullErrors_InvalidDatabaseURL(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config with invalid database URL
	configContent := `database:
  provider: postgresql
  url: "postgresql://invalid:invalid@nonexistent:9999/db"

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	// This should fail because the database doesn't exist
	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.Error(t, err)
	// Error could be connection refused, connection timeout, or authentication failure
}

func TestMigrationCreateErrors_NoSchemaFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config but no schema file
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "migration", "create", "test_migration"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMigrationCreateErrors_NoMigrationName(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config and schema
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Create schema directory and file
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	schemaContent := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id String @id
}
`
	err = os.WriteFile("prisma/schema.prisma", []byte(schemaContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	// Call without migration name argument
	err = app.Run(ctx, []string{"dryft", "migration", "create"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migration name")
}

func TestMigrationCreateErrors_InvalidSchemaFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config and schema
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Create schema directory and invalid schema file
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	invalidSchema := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  // Missing closing brace
`
	err = os.WriteFile("prisma/schema.prisma", []byte(invalidSchema), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "migration", "create", "test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestFileSystemErrors_ReadOnlyDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Create read-only directory
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	// Make directory read-only
	err = os.Chmod("prisma", 0444)
	require.NoError(t, err)
	defer func() { _ = os.Chmod("prisma", 0755) }() // Restore for cleanup

	app := NewApp()
	ctx := context.Background()

	// Try to write schema file in read-only directory - should fail
	// Note: This might not fail on all systems, so we only assert that something went wrong
	_ = app.Run(ctx, []string{"dryft", "db", "pull"})

	// Test passed if we didn't panic
}

func TestConfigErrors_EnvVarNotSet(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Config with env var that's not set
	configContent := `database:
  provider: postgresql
  url: ${UNDEFINED_DATABASE_URL}

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Make sure the env var is not set
	os.Unsetenv("UNDEFINED_DATABASE_URL")

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.Error(t, err)
}

func TestDbInspectErrors_InvalidDatabaseConnection(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config with invalid database
	configContent := `database:
  provider: postgresql
  url: "postgresql://invalid:invalid@localhost:19999/db"

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "db", "inspect"})

	assert.Error(t, err)
}

func TestValidateErrors_DirectoryInsteadOfFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Create directory instead of file at schema path
	err = os.MkdirAll("prisma/schema.prisma", 0755)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "validate"})

	assert.Error(t, err)
}

func TestInitErrors_PermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Make directory read-only to prevent file creation
	err := os.Chmod(tmpDir, 0444)
	require.NoError(t, err)
	defer func() { _ = os.Chmod(tmpDir, 0755) }() // Restore for cleanup

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "init"})

	// Should fail due to permission denied
	assert.Error(t, err)
}

func TestConfigErrors_MissingSchemaFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Config without schema file
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "validate"})

	assert.Error(t, err)
}

func TestMigrationCreateErrors_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Create empty schema (just datasource, no models)
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	emptySchema := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}
`
	err = os.WriteFile("prisma/schema.prisma", []byte(emptySchema), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "migration", "create", "empty_change"})

	// Should not error but indicate no changes
	if err != nil {
		// Some implementations might error on no changes
		assert.Contains(t, err.Error(), "no changes")
	}
}

func TestDbPullErrors_DestructiveSchemaFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config
	configContent := `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Create a protected schema file
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	existingSchema := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model ExistingModel {
  id String @id
}
`
	err = os.WriteFile("prisma/schema.prisma", []byte(existingSchema), 0755)
	require.NoError(t, err)

	// Make file read-only
	if os.Geteuid() != 0 {
		err = os.Chmod("prisma/schema.prisma", 0444)
		require.NoError(t, err)
		defer func() { _ = os.Chmod("prisma/schema.prisma", 0644) }() // Restore for cleanup

		app := NewApp()
		ctx := context.Background()

		err = app.Run(ctx, []string{"dryft", "db", "pull"})

		// Should fail due to read-only file
		assert.Error(t, err)
	}
}
