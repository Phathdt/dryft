package migration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultValueComparison tests that default values are correctly parsed
// and compared between migrations and Prisma schema.
func TestDefaultValueComparison(t *testing.T) {
	tests := []struct {
		name           string
		migrationSQL   string
		prismaDefault  string
		shouldMatch    bool
		description    string
	}{
		{
			name:          "enum with cast - exact match",
			migrationSQL:  "role \"UserRole\" NOT NULL DEFAULT 'user'::user_role",
			prismaDefault: "user",
			shouldMatch:   true,
			description:   "Prisma 'user' should match SQL 'user'::user_role cast",
		},
		{
			name:          "enum with cast - normalized",
			migrationSQL:  "role \"UserRole\" NOT NULL DEFAULT 'admin'::user_role",
			prismaDefault: "admin",
			shouldMatch:   true,
			description:   "Prisma 'admin' should match SQL 'admin'::user_role cast",
		},
		{
			name:          "simple string defaults",
			migrationSQL:  "status TEXT DEFAULT 'active'",
			prismaDefault: "active",
			shouldMatch:   true,
			description:   "Simple string defaults should match",
		},
		{
			name:          "function call - NOW()",
			migrationSQL:  "created_at TIMESTAMPTZ DEFAULT NOW()",
			prismaDefault: "now()",
			shouldMatch:   true,
			description:   "Function calls should match case-insensitively",
		},
		{
			name:          "function call - gen_random_uuid()",
			migrationSQL:  "id UUID DEFAULT gen_random_uuid()",
			prismaDefault: "uuid()",
			shouldMatch:   true,
			description:   "UUID generation functions should be normalized",
		},
		{
			name:          "numeric defaults",
			migrationSQL:  "count INTEGER DEFAULT 0",
			prismaDefault: "0",
			shouldMatch:   true,
			description:   "Numeric defaults should match",
		},
		{
			name:          "boolean defaults",
			migrationSQL:  "active BOOLEAN DEFAULT true",
			prismaDefault: "true",
			shouldMatch:   true,
			description:   "Boolean defaults should match",
		},
		{
			name:          "different values",
			migrationSQL:  "role \"UserRole\" DEFAULT 'user'::user_role",
			prismaDefault: "admin",
			shouldMatch:   false,
			description:   "Different enum values should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse migration SQL to extract default value
			parser := NewParser()

			// Create a simple CREATE TABLE statement with the column
			sql := "CREATE TABLE test_table (" + tt.migrationSQL + ")"
			stmt, err := parser.Parse(sql)
			require.NoError(t, err, "Failed to parse SQL")

			createTable, ok := stmt.(*CreateTable)
			require.True(t, ok, "Expected CreateTable statement")
			require.Len(t, createTable.Columns, 1, "Expected 1 column")

			column := createTable.Columns[0]

			// Normalize both defaults for comparison
			migrationDefault := normalizeDefaultValue(column.Default)
			prismaDefault := normalizeDefaultValue(&tt.prismaDefault)

			if tt.shouldMatch {
				assert.Equal(t, migrationDefault, prismaDefault,
					"Defaults should match: %s\nMigration: %v\nPrisma: %v",
					tt.description, migrationDefault, prismaDefault)
			} else {
				assert.NotEqual(t, migrationDefault, prismaDefault,
					"Defaults should NOT match: %s", tt.description)
			}
		})
	}
}

// normalizeDefaultValue normalizes a default value for comparison.
// This should strip casts (::type), normalize case, handle NULL vs nil, etc.
func normalizeDefaultValue(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := *value

	// Strip type casts (e.g., 'user'::user_role → 'user')
	if idx := strings.Index(normalized, "::"); idx != -1 {
		normalized = normalized[:idx]
	}

	// Trim whitespace
	normalized = strings.TrimSpace(normalized)

	// Remove surrounding single quotes
	normalized = strings.Trim(normalized, "'")

	// Normalize function calls to lowercase
	normalized = strings.ToLower(normalized)

	// Map common function aliases
	if normalized == "gen_random_uuid()" {
		normalized = "uuid()"
	}

	return &normalized
}

// TestEnumDefaultWithCast specifically tests the bug report:
// Migration has: DEFAULT 'user'::user_role
// Prisma has: @default(user)
// These should be treated as equivalent
func TestEnumDefaultWithCast(t *testing.T) {
	parser := NewParser()

	// Parse the actual migration SQL from the bug report
	sql := `CREATE TABLE users (
		id UUID NOT NULL DEFAULT gen_random_uuid(),
		email TEXT NOT NULL,
		name TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		role "UserRole" NOT NULL DEFAULT 'user'::user_role,
		PRIMARY KEY (id),
		UNIQUE (email)
	)`

	stmt, err := parser.Parse(sql)
	require.NoError(t, err)

	createTable, ok := stmt.(*CreateTable)
	require.True(t, ok)

	// Find the role column
	var roleColumn *ColumnDef
	for i := range createTable.Columns {
		if createTable.Columns[i].Name == "role" {
			roleColumn = &createTable.Columns[i]
			break
		}
	}
	require.NotNil(t, roleColumn, "role column not found")

	// Check that default was parsed
	require.NotNil(t, roleColumn.Default, "role column should have default")

	t.Logf("Parsed default value: %q", *roleColumn.Default)

	// The default should be normalized to just the value without cast
	// 'user'::user_role should normalize to 'user'
	normalized := normalizeDefaultValue(roleColumn.Default)
	require.NotNil(t, normalized)

	// After normalization, it should match Prisma's "user"
	prismaDefault := "user"
	assert.Equal(t, prismaDefault, *normalized,
		"Enum default 'user'::user_role should normalize to 'user'")
}
