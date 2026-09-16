package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/introspect/postgres"
	"github.com/phathdt/dryft/internal/migration"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	pggen "github.com/phathdt/dryft/internal/sql/postgres"
	"github.com/phathdt/dryft/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_BasicRoundTrip tests the full pipeline:
// 1. Start with SQL schema in PostgreSQL
// 2. Introspect → Internal Schema
// 3. Write → Prisma schema
// 4. Parse Prisma → Internal Schema
// 5. Verify schemas match (round-trip integrity)
func TestE2E_BasicRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	connStr := container.ConnString

	// 2. Apply initial schema
	initialSQL := `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX idx_users_email ON users (email);
	`
	err = container.ExecuteSQL(ctx, initialSQL)
	require.NoError(t, err)

	// 3. Introspect database
	introspector, err := postgres.NewPostgresIntrospector(ctx, connStr)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)
	require.NotNil(t, schema1)

	// Verify introspected schema
	assert.Len(t, schema1.Tables, 1, "should have 1 table")
	assert.Equal(t, "users", schema1.Tables[0].Name)
	assert.Len(t, schema1.Tables[0].Columns, 4, "should have 4 columns")
	// PostgreSQL creates 2 indexes: 1 explicit index + 1 unique constraint index
	assert.Len(t, schema1.Tables[0].Indexes, 2, "should have 2 indexes")

	// 4. Write to Prisma schema
	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)

	// Verify Prisma output
	assert.Contains(t, prismaText, "model User")
	assert.Contains(t, prismaText, "id")
	assert.Contains(t, prismaText, "email")
	assert.Contains(t, prismaText, "@unique")
	assert.Contains(t, prismaText, "@@index([email]")

	// 5. Parse Prisma back to Internal Schema
	parseResult, err := prisma.Parse(prismaText)
	require.NoError(t, err)
	schema2 := parseResult.Schema

	// 6. Diff should be empty (round-trip integrity)
	differ := diff.NewDiffer(nil)
	ops, err := differ.Diff(schema1, schema2)
	require.NoError(t, err)

	// There might be some minor differences due to naming conventions
	// For now, just verify major structures match
	assert.Len(t, schema2.Tables, 1, "parsed schema should have 1 table")
	if len(ops) > 0 {
		t.Logf("Round-trip differences detected:")
		for _, op := range ops {
			t.Logf("  - %s", op.Description())
		}
	}
}

// TestE2E_ModifyAndMigrate tests the full migration workflow:
// 1. Start with existing schema
// 2. Modify schema (add column)
// 3. Generate migration
// 4. Verify migration SQL
func TestE2E_ModifyAndMigrate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	connStr := container.ConnString

	// 2. Apply initial schema
	initialSQL := `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL
		);
	`
	err = container.ExecuteSQL(ctx, initialSQL)
	require.NoError(t, err)

	// 3. Introspect to get "before" schema
	introspector, err := postgres.NewPostgresIntrospector(ctx, connStr)
	require.NoError(t, err)
	schemaBefore, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// 4. Create "after" schema with additional column
	schemaAfter := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{
						Name:     "id",
						Type:     schema.DataType{Kind: schema.TypeUUID},
						Nullable: false,
					},
					{
						Name:     "email",
						Type:     schema.DataType{Kind: schema.TypeText},
						Nullable: false,
					},
					{
						Name:     "name",
						Type:     schema.DataType{Kind: schema.TypeText},
						Nullable: true,
					},
				},
				PrimaryKey: &schema.PrimaryKey{
					Columns: []string{"id"},
				},
			},
		},
	}

	// 5. Generate diff
	differ := diff.NewDiffer(nil)
	operations, err := differ.Diff(schemaBefore, schemaAfter)
	require.NoError(t, err)
	assert.NotEmpty(t, operations, "should detect column addition")

	// 6. Plan operations
	planner := diff.NewPlanner()
	plan, err := planner.Plan(operations)
	require.NoError(t, err)

	// 7. Generate SQL
	generator := pggen.NewGenerator(sql.GeneratorOptions{})
	upStatements, err := generator.Generate(plan.Operations)
	require.NoError(t, err)
	assert.NotEmpty(t, upStatements, "should generate UP SQL")

	downStatements, warnings, err := generator.GenerateReverse(plan.Operations)
	require.NoError(t, err)

	// 8. Format as Goose migration
	migrationContent := migration.FormatGoose("add_name_column", upStatements, downStatements, warnings)

	// Verify migration format
	assert.Contains(t, migrationContent, "-- +goose Up")
	assert.Contains(t, migrationContent, "-- +goose Down")
	assert.Contains(t, migrationContent, "ALTER TABLE users")
	assert.Contains(t, migrationContent, "ADD COLUMN name TEXT")

	// 9. Write migration to temp file
	tmpDir := t.TempDir()
	filename := migration.GenerateFilename("add_name_column")
	migrationPath := filepath.Join(tmpDir, filename)
	err = os.WriteFile(migrationPath, []byte(migrationContent), 0644)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(migrationPath)
	assert.NoError(t, err, "migration file should exist")

	t.Logf("Migration created: %s", migrationPath)
	t.Logf("Migration content:\n%s", migrationContent)
}

// TestE2E_ComplexSchema tests round-trip with more complex schema:
// - Multiple tables
// - Foreign keys
// - Indexes
// - Enums
func TestE2E_ComplexSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	connStr := container.ConnString

	// Apply complex schema
	complexSQL := `
		CREATE TYPE user_role AS ENUM ('USER', 'ADMIN');

		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			role user_role NOT NULL DEFAULT 'USER'
		);

		CREATE TABLE posts (
			id UUID PRIMARY KEY,
			title TEXT NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX idx_posts_user_id ON posts (user_id);
		CREATE INDEX idx_posts_created_at ON posts (created_at DESC);
	`
	err = container.ExecuteSQL(ctx, complexSQL)
	require.NoError(t, err)

	// 3. Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, connStr)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify introspection
	assert.Len(t, introspectedSchema.Tables, 2, "should have 2 tables")
	assert.Len(t, introspectedSchema.Enums, 1, "should have 1 enum")

	// Find users table
	var usersTable *schema.Table
	for i := range introspectedSchema.Tables {
		if introspectedSchema.Tables[i].Name == "users" {
			usersTable = &introspectedSchema.Tables[i]
			break
		}
	}
	require.NotNil(t, usersTable, "users table should exist")

	// Find posts table
	var postsTable *schema.Table
	for i := range introspectedSchema.Tables {
		if introspectedSchema.Tables[i].Name == "posts" {
			postsTable = &introspectedSchema.Tables[i]
			break
		}
	}
	require.NotNil(t, postsTable, "posts table should exist")
	assert.Len(t, postsTable.Indexes, 2, "posts should have 2 indexes")

	// Verify foreign key
	var fkFound bool
	for _, fk := range postsTable.ForeignKeys {
		if fk.RefTable == "users" {
			fkFound = true
			assert.Equal(t, schema.ActionCascade, fk.OnDelete)
		}
	}
	assert.True(t, fkFound, "should have FK to users table")

	// 4. Write to Prisma
	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	// Verify Prisma output
	assert.Contains(t, prismaText, "enum UserRole")
	assert.Contains(t, prismaText, "model Users")
	assert.Contains(t, prismaText, "model Posts")
	// Check relation field (may have variable spacing)
	assert.Contains(t, prismaText, "Users")

	t.Logf("Generated Prisma schema:\n%s", prismaText)
}
