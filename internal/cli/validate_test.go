package cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validConfig = `database:
  provider: postgresql
  url: postgresql://localhost/test

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
  naming: timestamp
`

func TestCmdValidate_ConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	app := NewApp()
	ctx := context.Background()

	// No .dryft.yaml file
	err := app.Run(ctx, []string{"dryft", "validate"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestCmdValidate_SchemaFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create config
	err := os.WriteFile(".dryft.yaml", []byte(validConfig), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	// Schema file doesn't exist
	err = app.Run(ctx, []string{"dryft", "validate"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCmdValidate_InvalidSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create config
	err := os.WriteFile(".dryft.yaml", []byte(validConfig), 0644)
	require.NoError(t, err)

	// Create schema directory
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	// Create invalid schema with syntax error
	invalidSchema := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id String @id
  // Missing closing brace
`
	err = os.WriteFile("prisma/schema.prisma", []byte(invalidSchema), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "validate"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestCmdValidate_ValidSchema(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create config
	err := os.WriteFile(".dryft.yaml", []byte(validConfig), 0644)
	require.NoError(t, err)

	// Create schema directory
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	// Valid schema with datasource and model
	validSchema := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    String @id @default(uuid())
  email String @unique
  name  String?
}
`
	err = os.WriteFile("prisma/schema.prisma", []byte(validSchema), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "validate"})

	assert.NoError(t, err)
}

func TestCmdValidate_MultipleModels(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create config
	err := os.WriteFile(".dryft.yaml", []byte(validConfig), 0644)
	require.NoError(t, err)

	// Create schema directory
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	// Valid schema with multiple models
	validSchema := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    String @id @default(uuid())
  email String @unique
  posts Post[]
}

model Post {
  id     String @id @default(uuid())
  title  String
  userId String
  user   User   @relation(fields: [userId], references: [id])
}
`
	err = os.WriteFile("prisma/schema.prisma", []byte(validSchema), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "validate"})

	assert.NoError(t, err)
}

func TestCmdValidate_WithEnums(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create config
	err := os.WriteFile(".dryft.yaml", []byte(validConfig), 0644)
	require.NoError(t, err)

	// Create schema directory
	err = os.MkdirAll("prisma", 0755)
	require.NoError(t, err)

	// Valid schema with enums
	validSchema := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

enum Role {
  ADMIN
  USER
  GUEST
}

model User {
  id   String @id @default(uuid())
  role Role   @default(USER)
}
`
	err = os.WriteFile("prisma/schema.prisma", []byte(validSchema), 0644)
	require.NoError(t, err)

	app := NewApp()
	ctx := context.Background()

	err = app.Run(ctx, []string{"dryft", "validate"})

	assert.NoError(t, err)
}
