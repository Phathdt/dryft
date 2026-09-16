package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdDbPull_Integration_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.NoError(t, err)

	// Check that schema file was created
	_, err = os.Stat("prisma/schema.prisma")
	assert.NoError(t, err)

	// Check file content has datasource
	content, err := os.ReadFile("prisma/schema.prisma")
	require.NoError(t, err)
	assert.Contains(t, string(content), "datasource")
}

func TestCmdDbPull_Integration_WithTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	// Create a table in the database
	err = container.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255)
		)
	`)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.NoError(t, err)

	// Check schema content
	content, err := os.ReadFile("prisma/schema.prisma")
	require.NoError(t, err)
	schemaStr := string(content)

	// Should contain Users model (PascalCase from users table)
	assert.Contains(t, schemaStr, "model Users")
	assert.Contains(t, schemaStr, "email")
	assert.Contains(t, schemaStr, "name")
}

func TestCmdDbPull_Integration_WithEnums(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	// Create enum type
	err = container.ExecuteSQL(ctx, `
		CREATE TYPE user_role AS ENUM ('admin', 'user', 'guest')
	`)
	require.NoError(t, err)

	// Create table using enum
	err = container.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			role user_role NOT NULL DEFAULT 'user'
		)
	`)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.NoError(t, err)

	// Check schema content
	content, err := os.ReadFile("prisma/schema.prisma")
	require.NoError(t, err)
	schemaStr := string(content)

	// Should contain enum
	assert.Contains(t, schemaStr, "enum")
	assert.Contains(t, schemaStr, "admin")
}

func TestCmdDbPull_Integration_WithConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	// Create tables with foreign key
	err = container.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE
		);

		CREATE TABLE posts (
			id UUID PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id)
		)
	`)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.NoError(t, err)

	// Check schema content
	content, err := os.ReadFile("prisma/schema.prisma")
	require.NoError(t, err)
	schemaStr := string(content)

	// Should contain both models
	assert.Contains(t, schemaStr, "model Users")
	assert.Contains(t, schemaStr, "model Posts")
}

func TestCmdDbInspect_Integration_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	var outBuf bytes.Buffer

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.Run(ctx, []string{"dryft", "db", "inspect"})

	_ = w.Close()
	os.Stdout = oldStdout

	// Read output
	_, _ = outBuf.ReadFrom(r)

	assert.NoError(t, err)
	output := outBuf.String()
	assert.Contains(t, output, "Tables")
}

func TestCmdDbInspect_Integration_WithTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	// Create a table
	err = container.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL
		)
	`)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	var outBuf bytes.Buffer

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.Run(ctx, []string{"dryft", "db", "inspect"})

	_ = w.Close()
	os.Stdout = oldStdout

	// Read output
	_, _ = outBuf.ReadFrom(r)

	assert.NoError(t, err)
	output := outBuf.String()
	// JSON output should contain tables
	assert.Contains(t, output, "Tables")
}

func TestCmdDbPull_Integration_DirectoryCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config with nested schema path
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

schema:
  file: deeply/nested/prisma/schema.prisma
  naming:
    fields: camelCase
    tables: PascalCase

migration:
  directory: migrations
  format: goose
  naming: timestamp
`
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	assert.NoError(t, err)

	// Check that nested directories were created
	_, err = os.Stat("deeply/nested/prisma/schema.prisma")
	assert.NoError(t, err)
}

func TestCmdDbPull_OutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	// Create tables
	err = container.ExecuteSQL(ctx, `
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE posts (id UUID PRIMARY KEY);
	`)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()
	var outBuf bytes.Buffer

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.Run(ctx, []string{"dryft", "db", "pull"})

	_ = w.Close()
	os.Stdout = oldStdout

	// Read output
	_, _ = outBuf.ReadFrom(r)

	assert.NoError(t, err)
	output := outBuf.String()
	// Should contain summary with model count
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Schema written")
	assert.True(t, strings.Contains(output, "Models:") || strings.Contains(output, "2"), "should show model count")
}

func TestCmdDbInspect_OutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	// Create table
	err = container.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255)
		)
	`)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create valid config
	configContent := `database:
  provider: postgresql
  url: ` + container.ConnString + `

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
	err = os.WriteFile(".dryft.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	app := NewApp()

	err = app.Run(ctx, []string{"dryft", "db", "inspect"})

	assert.NoError(t, err)
}
