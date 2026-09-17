package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationCreate_WithMigrationHistory tests incremental migration generation
// using existing migration files (100% offline, no DB connection).
func TestMigrationCreate_WithMigrationHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create migrations directory with existing migration
	migrationsDir := "migrations"
	err := os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)

	// Create initial migration (users table)
	initialMigration := `-- +goose Up
CREATE TABLE "User" (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL
);

-- +goose Down
DROP TABLE "User";
`
	err = os.WriteFile(filepath.Join(migrationsDir, "20240101000000_create_users.sql"), []byte(initialMigration), 0644)
	require.NoError(t, err)

	// Create schema.prisma with new column
	schemaDir := "prisma"
	err = os.MkdirAll(schemaDir, 0755)
	require.NoError(t, err)

	schemaContent := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    Int    @id @default(autoincrement())
  email String
  bio   String?
}
`
	schemaFile := filepath.Join(schemaDir, "schema.prisma")
	err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Create config (database URL not used for offline migration create)
	configContent := `database:
  provider: postgresql
  url: "postgresql://localhost:5432/unused"

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
  naming: timestamp
`
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Run migration create (100% offline)
	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "migration", "create", "add_bio"})
	require.NoError(t, err)

	// Verify migration was created
	files, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)
	assert.Len(t, files, 2, "should have 2 migrations (initial + new)")

	// Find the new migration
	var newMigrationContent string
	for _, file := range files {
		if strings.Contains(file.Name(), "add_bio") {
			content, err := os.ReadFile(filepath.Join(migrationsDir, file.Name()))
			require.NoError(t, err)
			newMigrationContent = string(content)
			break
		}
	}

	require.NotEmpty(t, newMigrationContent, "new migration should exist")

	// Verify migration contains ALTER, not CREATE
	assert.Contains(t, newMigrationContent, "ALTER TABLE", "should generate ALTER statement")
	assert.Contains(t, newMigrationContent, "ADD COLUMN", "should add new column")
	assert.Contains(t, newMigrationContent, "bio", "should reference bio column")
	assert.NotContains(t, newMigrationContent, "CREATE TABLE \"User\"", "should NOT recreate table")
}

// TestMigrationCreate_EmptyMigrations_FirstMigration tests first migration generation
// with empty migrations directory (100% offline).
func TestMigrationCreate_EmptyMigrations_FirstMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create empty migrations directory
	migrationsDir := "migrations"
	err := os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)

	// Create schema.prisma
	schemaDir := "prisma"
	err = os.MkdirAll(schemaDir, 0755)
	require.NoError(t, err)

	schemaContent := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    Int    @id @default(autoincrement())
  email String
}
`
	schemaFile := filepath.Join(schemaDir, "schema.prisma")
	err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Create config (database URL not used for offline migration create)
	configContent := `database:
  provider: postgresql
  url: "postgresql://localhost:5432/unused"

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
  naming: timestamp
`
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Run migration create (100% offline, first migration)
	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "migration", "create", "initial"})
	require.NoError(t, err)

	// Verify migration was created
	files, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)
	assert.Len(t, files, 1, "should have 1 migration")

	// Read migration content
	content, err := os.ReadFile(filepath.Join(migrationsDir, files[0].Name()))
	require.NoError(t, err)
	migrationContent := string(content)

	// Verify migration contains CREATE TABLE (full schema creation)
	assert.Contains(t, migrationContent, "CREATE TABLE", "should generate CREATE TABLE")
	assert.Contains(t, migrationContent, "User", "should create User table")
	assert.Contains(t, migrationContent, "email", "should include email column")
}

// TestMigrationCreate_NoDBConnection tests that migration create works 100% offline.
func TestMigrationCreate_NoDBConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Setup with invalid DATABASE_URL to ensure no connection attempted
	migrationsDir := "migrations"
	err := os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)

	// Create existing migration
	existingMigration := `-- +goose Up
CREATE TABLE "Product" (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL
);

-- +goose Down
DROP TABLE "Product";
`
	err = os.WriteFile(filepath.Join(migrationsDir, "20240101000000_create_products.sql"), []byte(existingMigration), 0644)
	require.NoError(t, err)

	// Create schema with new field
	schemaDir := "prisma"
	err = os.MkdirAll(schemaDir, 0755)
	require.NoError(t, err)

	schemaContent := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model Product {
  id          Int     @id @default(autoincrement())
  name        String
  description String?
}
`
	err = os.WriteFile(filepath.Join(schemaDir, "schema.prisma"), []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Create config with INVALID database URL (should not be used)
	configContent := `database:
  provider: postgresql
  url: "postgresql://invalid:invalid@nonexistent:9999/invalid"

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
  naming: timestamp
`
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Run migration create - should work without DB connection
	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "migration", "create", "add_description"})
	require.NoError(t, err, "should work offline without database connection")

	// Verify new migration was created
	files, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)
	assert.Len(t, files, 2, "should have 2 migrations")
}

// TestMigrationCreate_MultipleMigrations tests building schema from multiple migrations.
func TestMigrationCreate_MultipleMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create migrations directory
	migrationsDir := "migrations"
	err := os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)

	// Create first migration (users table)
	migration1 := `-- +goose Up
CREATE TABLE "User" (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL
);

-- +goose Down
DROP TABLE "User";
`
	err = os.WriteFile(filepath.Join(migrationsDir, "20240101000000_create_users.sql"), []byte(migration1), 0644)
	require.NoError(t, err)

	// Create second migration (add name column)
	migration2 := `-- +goose Up
ALTER TABLE "User" ADD COLUMN name TEXT;

-- +goose Down
ALTER TABLE "User" DROP COLUMN name;
`
	err = os.WriteFile(filepath.Join(migrationsDir, "20240102000000_add_name.sql"), []byte(migration2), 0644)
	require.NoError(t, err)

	// Create schema with additional column
	schemaDir := "prisma"
	err = os.MkdirAll(schemaDir, 0755)
	require.NoError(t, err)

	schemaContent := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    Int     @id @default(autoincrement())
  email String
  name  String?
  bio   String?
}
`
	err = os.WriteFile(filepath.Join(schemaDir, "schema.prisma"), []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Create config
	configContent := `database:
  provider: postgresql
  url: "postgresql://localhost:5432/unused"

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
  naming: timestamp
`
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	// Run migration create
	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "migration", "create", "add_bio"})
	require.NoError(t, err)

	// Verify migration was created
	files, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)
	assert.Len(t, files, 3, "should have 3 migrations")

	// Find the new migration
	var newMigrationContent string
	for _, file := range files {
		if strings.Contains(file.Name(), "add_bio") {
			content, err := os.ReadFile(filepath.Join(migrationsDir, file.Name()))
			require.NoError(t, err)
			newMigrationContent = string(content)
			break
		}
	}

	require.NotEmpty(t, newMigrationContent, "new migration should exist")

	// Should only add bio column (email and name already exist from previous migrations)
	assert.Contains(t, newMigrationContent, "ALTER TABLE", "should generate ALTER")
	assert.Contains(t, newMigrationContent, "bio", "should add bio column")
	assert.NotContains(t, newMigrationContent, "CREATE TABLE", "should not recreate table")
}
