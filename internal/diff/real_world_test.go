package diff

import (
	"testing"

	"github.com/phathdt/dryft/internal/migration"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/stretchr/testify/require"
)

// TestRealWorldEnumDefault simulates the exact scenario from dryft-test
func TestRealWorldEnumDefault(t *testing.T) {
	// Step 1: Create enum
	enumSQL := `CREATE TYPE "UserRole" AS ENUM ('admin', 'user', 'guest')`
	parser := migration.NewParser()
	enumStmt, err := parser.Parse(enumSQL)
	require.NoError(t, err)

	// Step 2: Create table with enum default
	tableSQL := `CREATE TABLE users (
		id UUID NOT NULL DEFAULT gen_random_uuid(),
		email TEXT NOT NULL,
		name TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		role "UserRole" NOT NULL DEFAULT 'user',
		PRIMARY KEY (id),
		UNIQUE (email)
	)`

	tableStmt, err := parser.Parse(tableSQL)
	require.NoError(t, err)

	// Step 3: Build schema from migrations
	builder := migration.NewSchemaBuilder()
	err = builder.Apply(enumStmt)
	require.NoError(t, err)
	err = builder.Apply(tableStmt)
	require.NoError(t, err)

	migrationSchema := builder.Build()

	// Step 4: Parse Prisma schema
	prismaSchemaStr := `
enum UserRole {
  admin
  user
  guest
}

model Users {
  id        String   @id @default(uuid()) @db.Uuid
  email     String   @unique
  name      String?
  createdAt DateTime @default(now()) @db.Timestamptz @map("created_at")
  role      UserRole @default(user)

  @@map("users")
}
`

	prismaParser := prisma.NewParser(prismaSchemaStr)
	doc, err := prismaParser.ParseSchema()
	require.NoError(t, err)

	converter := prisma.NewConverter()
	prismaSchema, _ := converter.Convert(doc)

	// Step 5: Compare
	differ := NewDiffer(map[string]string{})
	ops, err := differ.Diff(migrationSchema, prismaSchema)
	require.NoError(t, err)

	// Debug: Print role column defaults
	migrationRoleCol := findColumn(*migrationSchema, "users", "role")
	prismaRoleCol := findColumn(*prismaSchema, "users", "role")

	t.Logf("Migration role default: Kind=%v, Literal=%q, Expression=%q",
		migrationRoleCol.Default.Kind,
		migrationRoleCol.Default.Literal,
		migrationRoleCol.Default.Expression)

	t.Logf("Prisma role default: Kind=%v, Literal=%q, Expression=%q",
		prismaRoleCol.Default.Kind,
		prismaRoleCol.Default.Literal,
		prismaRoleCol.Default.Expression)

	t.Logf("Normalized migration: %q", NormalizeLiteral(migrationRoleCol.Default.Literal))
	t.Logf("Normalized prisma: %q", NormalizeLiteral(prismaRoleCol.Default.Literal))

	// Should have no operations (no changes)
	require.Empty(t, ops, "Should detect no changes for enum default")
}

func findColumn(s schema.Schema, tableName, columnName string) schema.Column {
	for _, table := range s.Tables {
		if table.Name == tableName {
			for _, col := range table.Columns {
				if col.Name == columnName {
					return col
				}
			}
		}
	}
	panic("column not found")
}
