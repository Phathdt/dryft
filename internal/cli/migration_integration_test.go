package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/introspect/postgres"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	sqlpostgres "github.com/phathdt/dryft/internal/sql/postgres"
	"github.com/phathdt/dryft/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestMigrationCreate_DatabaseIntrospection verifies the fix:
// When a table already exists in the database, migration create should generate
// ALTER TABLE statements, not CREATE TABLE statements.
func TestMigrationCreate_DatabaseIntrospection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup test database with a table
	pgContainer, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	dbName := "test_migration_fix"
	setupConn, err := pgx.Connect(ctx, pgContainer.ConnString)
	require.NoError(t, err)
	_, _ = setupConn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
	_, err = setupConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	setupConn.Close(ctx)
	require.NoError(t, err)

	testDBConnStr := strings.Replace(pgContainer.ConnString, "/testdb", "/"+dbName, 1)

	t.Cleanup(func() {
		conn, _ := pgx.Connect(ctx, pgContainer.ConnString)
		if conn != nil {
			conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
			conn.Close(ctx)
		}
	})

	// Create initial table
	conn, err := pgx.Connect(ctx, testDBConnStr)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `
		CREATE TABLE posts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL
		)
	`)
	conn.Close(ctx)
	require.NoError(t, err)

	// Scenario 1: Old buggy behavior - using empty schema as previousSchema
	emptySchema := &schema.Schema{
		Tables: []schema.Table{},
		Enums:  []schema.Enum{},
	}

	// Scenario 2: Fixed behavior - introspect database for previousSchema
	intr, err := postgres.NewPostgresIntrospector(ctx, testDBConnStr)
	require.NoError(t, err)
	defer intr.Close()

	actualSchema, err := intr.Introspect(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, actualSchema.Tables)
	require.Equal(t, "posts", actualSchema.Tables[0].Name)

	// Create a modified schema: add bio column
	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaSchema, err := writer.Write(actualSchema)
	require.NoError(t, err)

	// Insert bio column after title line - find the line with title field and add bio after it
	lines := strings.Split(prismaSchema, "\n")
	var modifiedLines []string
	for _, line := range lines {
		modifiedLines = append(modifiedLines, line)
		if strings.Contains(line, "title") && strings.Contains(line, "String") {
			// Add bio field with same indentation
			modifiedLines = append(modifiedLines, "  bio   String? @db.Text")
		}
	}
	modifiedPrisma := strings.Join(modifiedLines, "\n")

	parseResult, err := prisma.Parse(modifiedPrisma)
	require.NoError(t, err)
	modifiedSchema := parseResult.Schema

	// Diff with empty (buggy) vs actual (fixed)
	differ := diff.NewDiffer(nil)

	opsWithEmpty, err := differ.Diff(emptySchema, modifiedSchema)
	require.NoError(t, err)

	opsWithActual, err := differ.Diff(actualSchema, modifiedSchema)
	require.NoError(t, err)

	// Plan operations
	planner := diff.NewPlanner()
	planWithEmpty, _ := planner.Plan(opsWithEmpty)
	planWithActual, _ := planner.Plan(opsWithActual)

	// Generate SQL
	gen := sqlpostgres.NewGenerator(sql.GeneratorOptions{})
	sqlEmpty, _ := gen.Generate(planWithEmpty.Operations)
	sqlActual, _ := gen.Generate(planWithActual.Operations)

	sqlEmptyStr := strings.Join(sqlEmpty, "\n")
	sqlActualStr := strings.Join(sqlActual, "\n")

	// Key assertions: the fix should eliminate CREATE TABLE for existing tables
	require.Contains(t, sqlEmptyStr, "CREATE TABLE", "buggy behavior: uses empty schema, creates table")
	require.NotContains(t, sqlActualStr, "CREATE TABLE posts", "fixed behavior: introspects DB, only alters")
	require.Contains(t, sqlActualStr, "ALTER TABLE", "fixed behavior: generates ALTER statements")
	require.Contains(t, sqlActualStr, "bio", "fixed behavior: adds the bio column")
}

func TestMigrationCreate_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	pgContainer, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	dbName := "test_empty_migration"
	setupConn, err := pgx.Connect(ctx, pgContainer.ConnString)
	require.NoError(t, err)
	_, _ = setupConn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
	_, err = setupConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	setupConn.Close(ctx)
	require.NoError(t, err)

	testDBConnStr := strings.Replace(pgContainer.ConnString, "/testdb", "/"+dbName, 1)

	t.Cleanup(func() {
		conn, _ := pgx.Connect(ctx, pgContainer.ConnString)
		if conn != nil {
			conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
			conn.Close(ctx)
		}
	})

	// Introspect empty database
	intr, err := postgres.NewPostgresIntrospector(ctx, testDBConnStr)
	require.NoError(t, err)
	defer intr.Close()

	emptySchema, err := intr.Introspect(ctx)
	require.NoError(t, err)

	require.Empty(t, emptySchema.Tables)
	require.Empty(t, emptySchema.Enums)
}

func TestMigrationCreate_DestructiveProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// TODO: Implement when we have proper state management
	t.Skip("destructive protection test requires state management")
}
