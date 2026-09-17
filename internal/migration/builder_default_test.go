package migration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder_EnumDefaultValue(t *testing.T) {
	// Build schema from migration SQL
	builder := NewSchemaBuilder()

	// Create enum first
	enumSQL := `CREATE TYPE "UserRole" AS ENUM ('admin', 'user', 'guest')`
	parser := NewParser()

	enumStmt, err := parser.Parse(enumSQL)
	require.NoError(t, err)
	err = builder.Apply(enumStmt)
	require.NoError(t, err)

	// Create table with enum column
	tableSQL := `CREATE TABLE users (
		role "UserRole" NOT NULL DEFAULT 'user'
	)`

	tableStmt, err := parser.Parse(tableSQL)
	require.NoError(t, err)
	err = builder.Apply(tableStmt)
	require.NoError(t, err)

	// Check built schema
	schema := builder.Build()
	require.Len(t, schema.Tables, 1)
	require.Len(t, schema.Tables[0].Columns, 1)

	roleCol := schema.Tables[0].Columns[0]
	require.NotNil(t, roleCol.Default)

	t.Logf("Built schema default kind: %v", roleCol.Default.Kind)
	t.Logf("Built schema default literal: %q", roleCol.Default.Literal)

	// This should match what Prisma generates
	require.Equal(t, "'user'", roleCol.Default.Literal,
		"Builder should preserve default literal as-is from SQL")
}
