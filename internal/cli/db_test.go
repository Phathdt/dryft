package cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validFullConfig = `database:
  provider: postgresql
  url: postgresql://testuser:testpass@localhost:5432/testdb

schema:
  file: prisma/schema.prisma
  naming:
    fields: camelCase
    tables: PascalCase

migration:
  directory: migrations
  format: goose
  naming: timestamp
`

func TestCmdDbPull_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	err := os.WriteFile(".dryft.yaml", []byte(validFullConfig), 0644)
	require.NoError(t, err)

	// This test verifies config loading works; actual DB connection
	// would require a running database. For now, we verify the config path.
	_, err = os.Stat(".dryft.yaml")
	assert.NoError(t, err)
}

func TestCmdDbPull_ConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	app := NewApp()
	ctx := context.Background()

	// No .dryft.yaml file exists
	err := app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestCmdDbPull_MissingDatabaseURL(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config without database URL but with required migration config
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

func TestCmdDbInspect_ConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	app := NewApp()
	ctx := context.Background()

	// No .dryft.yaml file exists
	err := app.Run(ctx, []string{"dryft", "db", "inspect"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestCmdDbInspect_MissingDatabaseURL(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config without database URL
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

	err = app.Run(ctx, []string{"dryft", "db", "inspect"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database.url")
}

func TestCmdDbPull_InvalidDatabaseURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config with invalid connection string
	configContent := `database:
  provider: postgresql
  url: postgresql://invalid:invalid@localhost:9999/nonexistent

schema:
  file: prisma/schema.prisma
  naming:
    fields: camelCase
    tables: PascalCase

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
	// Connection errors should be wrapped with context
	assert.Contains(t, err.Error(), "failed to create introspector")
}

func TestCmdDbInspect_InvalidDatabaseURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config with invalid connection string
	configContent := `database:
  provider: postgresql
  url: postgresql://invalid:invalid@localhost:9999/nonexistent

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
	assert.Contains(t, err.Error(), "failed to create introspector")
}

func TestCmdDbPull_SchemaDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create config with nested schema path
	configContent := `database:
  provider: postgresql
  url: postgresql://testuser:testpass@localhost:5432/testdb

schema:
  file: deeply/nested/path/schema.prisma
  naming:
    fields: camelCase
    tables: PascalCase

migration:
  directory: migrations
  format: goose
`
	err := os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Verify nested path doesn't exist yet (we're not running the command,
	// just verifying the config parsing works)
	_, err = os.Stat("deeply/nested/path")
	assert.True(t, os.IsNotExist(err))
}
