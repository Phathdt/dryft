package diff

import (
	"testing"

	"github.com/phathdt/dryft/internal/migration"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultValueIntegration tests the full flow:
// SQL migration -> parsed default -> compared with Prisma default
func TestDefaultValueIntegration(t *testing.T) {
	// Parse migration SQL
	migrationSQL := `CREATE TABLE users (
		role "UserRole" NOT NULL DEFAULT 'user'
	)`

	parser := migration.NewParser()
	stmt, err := parser.Parse(migrationSQL)
	require.NoError(t, err)

	createTable := stmt.(*migration.CreateTable)
	require.Len(t, createTable.Columns, 1)
	roleCol := createTable.Columns[0]

	t.Logf("Migration parsed default: %q", *roleCol.Default)

	// Parse Prisma schema
	prismaSchema := `
enum UserRole {
  admin
  user
  guest
}

model Users {
  role UserRole @default(user)
  @@map("users")
}
`

	prismaParser := prisma.NewParser(prismaSchema)
	doc, err := prismaParser.ParseSchema()
	require.NoError(t, err)

	converter := prisma.NewConverter()
	sch, warnings := converter.Convert(doc)
	require.Empty(t, warnings)
	require.Len(t, sch.Tables, 1)
	require.Len(t, sch.Tables[0].Columns, 1)

	roleField := sch.Tables[0].Columns[0]
	require.NotNil(t, roleField.Default)

	t.Logf("Prisma converted default: %q", roleField.Default.Literal)

	// Compare using normalization
	migrationDefault := NormalizeLiteral(*roleCol.Default)
	prismaDefault := NormalizeLiteral(roleField.Default.Literal)

	t.Logf("Migration normalized: %q", migrationDefault)
	t.Logf("Prisma normalized: %q", prismaDefault)

	assert.Equal(t, migrationDefault, prismaDefault,
		"Migration and Prisma defaults should match after normalization")
}

// TestDefaultValueWithCast tests enum default with type cast
func TestDefaultValueWithCast(t *testing.T) {
	// Parse migration SQL with type cast
	migrationSQL := `CREATE TABLE users (
		role "UserRole" NOT NULL DEFAULT 'admin'::user_role
	)`

	parser := migration.NewParser()
	stmt, err := parser.Parse(migrationSQL)
	require.NoError(t, err)

	createTable := stmt.(*migration.CreateTable)
	roleCol := createTable.Columns[0]

	t.Logf("Migration parsed default with cast: %q", *roleCol.Default)

	// Should capture the full cast
	assert.Contains(t, *roleCol.Default, "::", "Default should include type cast")

	// Normalize should strip the cast
	normalized := NormalizeLiteral(*roleCol.Default)
	assert.Equal(t, "admin", normalized, "Normalized should strip cast and quotes")
}

// TestDifferDefaultComparison tests the Differ.defaultsEqual method
func TestDifferDefaultComparison(t *testing.T) {
	differ := NewDiffer(map[string]string{})

	tests := []struct {
		name     string
		a        *schema.DefaultValue
		b        *schema.DefaultValue
		expected bool
	}{
		{
			name: "same literal - quoted enum",
			a: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'user'",
			},
			b: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'user'",
			},
			expected: true,
		},
		{
			name: "enum with vs without cast",
			a: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'user'::user_role",
			},
			b: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'user'",
			},
			expected: true,
		},
		{
			name: "different values",
			a: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'user'",
			},
			b: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'admin'",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := differ.defaultsEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
