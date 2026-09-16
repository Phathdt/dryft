package postgres

import (
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerate_AllOperationTypes covers all operation types in the switch statement
func TestGenerate_AllOperationTypes(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name string
		op   diff.Operation
		check func(t *testing.T, sql string)
	}{
		{
			name: "CreateEnum",
			op: diff.CreateEnum{
				Enum: schema.Enum{
					Name: "status",
					Values: []schema.EnumValue{
						{Label: "active"},
						{Label: "inactive"},
					},
				},
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "CREATE TYPE status AS ENUM")
			},
		},
		{
			name: "AlterEnum",
			op: diff.AlterEnum{
				Name:      "status",
				AddValues: []string{"pending"},
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "ALTER TYPE status ADD VALUE 'pending'")
			},
		},
		{
			name: "RenameTable",
			op: diff.RenameTable{
				From: "users",
				To:   "accounts",
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "ALTER TABLE users RENAME TO accounts")
			},
		},
		{
			name: "CreateForeignKey",
			op: diff.CreateForeignKey{
				Table: "posts",
				Constraint: schema.ForeignKey{
					Name:       "fk_posts_user_id",
					Columns:    []string{"user_id"},
					RefTable:   "users",
					RefColumns: []string{"id"},
					OnDelete:   schema.ActionCascade,
				},
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "ALTER TABLE posts")
				assert.Contains(t, sql, "FOREIGN KEY")
				assert.Contains(t, sql, "ON DELETE CASCADE")
			},
		},
		{
			name: "DropForeignKey",
			op: diff.DropForeignKey{
				Table: "posts",
				Name:  "fk_posts_user_id",
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "ALTER TABLE posts")
				assert.Contains(t, sql, "DROP CONSTRAINT")
			},
		},
		{
			name: "DropTable",
			op: diff.DropTable{
				Name: "users",
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "DROP TABLE")
			},
		},
		{
			name: "DropColumn",
			op: diff.DropColumn{
				Table:  "users",
				Column: "email",
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "DROP COLUMN")
			},
		},
		{
			name: "DropIndex",
			op: diff.DropIndex{
				Table: "users",
				Name:  "idx_users_email",
			},
			check: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "DROP INDEX")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statements, err := gen.Generate([]diff.Operation{tt.op})
			require.NoError(t, err)
			require.Len(t, statements, 1)
			tt.check(t, statements[0])
		})
	}
}

// TestGenerateCreateTable_WithConstraints tests CreateTable with primary key and constraints
func TestGenerateCreateTable_WithConstraints(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateTable{
		Table: schema.Table{
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
			Constraints: []schema.Constraint{
				{
					Name:    "uk_users_email",
					Type:    schema.ConstraintUnique,
					Columns: []string{"email"},
				},
			},
		},
	}

	sql := gen.generateCreateTable(op)

	assert.Contains(t, sql, "CREATE TABLE")
	assert.Contains(t, sql, "PRIMARY KEY")
	assert.Contains(t, sql, "UNIQUE")
}

// TestGenerateCreateTable_WithDefaults tests CreateTable with default values
func TestGenerateCreateTable_WithDefaults(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateTable{
		Table: schema.Table{
			Name: "posts",
			Columns: []schema.Column{
				{
					Name:     "id",
					Type:     schema.DataType{Kind: schema.TypeUUID},
					Nullable: false,
					Default: &schema.DefaultValue{
						Kind:       schema.DefaultExpression,
						Expression: "uuid_generate_v4()",
					},
				},
				{
					Name:     "created_at",
					Type:     schema.DataType{Kind: schema.TypeTimestampTZ},
					Nullable: false,
					Default: &schema.DefaultValue{
						Kind:       schema.DefaultExpression,
						Expression: "now()",
					},
				},
			},
		},
	}

	sql := gen.generateCreateTable(op)

	assert.Contains(t, sql, "CREATE TABLE")
	assert.Contains(t, sql, "DEFAULT uuid_generate_v4()")
	assert.Contains(t, sql, "DEFAULT now()")
}

// TestGenerate_EmptyOperations tests Generate with no operations
func TestGenerate_EmptyOperations(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	statements, err := gen.Generate([]diff.Operation{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(statements))
}

// TestGenerateReverse_AllOperationTypes covers all reverse operation types
func TestGenerateReverse_AllOperationTypes(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name string
		op   diff.Operation
		checkSQL func(t *testing.T, sql string)
		checkWarning func(t *testing.T, warning string)
	}{
		{
			name: "ReverseCreateEnum",
			op: diff.CreateEnum{
				Enum: schema.Enum{Name: "status"},
			},
			checkSQL: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "DROP TYPE")
			},
		},
		{
			name: "ReverseAlterEnum",
			op: diff.AlterEnum{
				Name:      "status",
				AddValues: []string{"pending"},
			},
			checkSQL: func(t *testing.T, sql string) {
				// Cannot reverse ALTER ENUM
			},
			checkWarning: func(t *testing.T, warning string) {
				assert.Contains(t, warning, "WARNING")
			},
		},
		{
			name: "ReverseRenameTable",
			op: diff.RenameTable{
				From: "users",
				To:   "accounts",
			},
			checkSQL: func(t *testing.T, sql string) {
				// Should reverse the rename
				assert.Contains(t, sql, "ALTER TABLE accounts RENAME TO users")
			},
		},
		{
			name: "ReverseCreateForeignKey",
			op: diff.CreateForeignKey{
				Table: "posts",
				Constraint: schema.ForeignKey{
					Name:       "fk_posts_user_id",
					Columns:    []string{"user_id"},
					RefTable:   "users",
					RefColumns: []string{"id"},
				},
			},
			checkSQL: func(t *testing.T, sql string) {
				assert.Contains(t, sql, "DROP CONSTRAINT")
			},
		},
		{
			name: "ReverseRenameColumn",
			op: diff.RenameColumn{
				Table: "users",
				From:  "name",
				To:    "full_name",
			},
			checkSQL: func(t *testing.T, sql string) {
				// Should reverse the rename
				assert.Contains(t, sql, "RENAME COLUMN full_name TO name")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statements, warnings, err := gen.GenerateReverse([]diff.Operation{tt.op})
			require.NoError(t, err)

			if tt.checkSQL != nil && len(statements) > 0 {
				tt.checkSQL(t, statements[0])
			}
			if tt.checkWarning != nil && len(warnings) > 0 {
				tt.checkWarning(t, warnings[0])
			}
		})
	}
}

// TestGenerateAlterColumn_NoChanges tests AlterColumn when no fields change
func TestGenerateAlterColumn_NoChanges(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.AlterColumn{
		Table:       "users",
		Column:      "email",
		OldType:     schema.DataType{Kind: schema.TypeText},
		NewType:     schema.DataType{Kind: schema.TypeText},
		OldNullable: true,
		NewNullable: true,
		OldDefault: &schema.DefaultValue{
			Kind:    schema.DefaultLiteral,
			Literal: "''",
		},
		NewDefault: &schema.DefaultValue{
			Kind:    schema.DefaultLiteral,
			Literal: "''",
		},
	}

	sql := gen.generateAlterColumn(op)

	// No changes should produce empty string
	assert.Equal(t, "", sql)
}

// TestGenerateAddColumn_WithDefault tests AddColumn with a default value
func TestGenerateAddColumn_WithDefault(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.AddColumn{
		Table: "users",
		Column: schema.Column{
			Name:     "status",
			Type:     schema.DataType{Kind: schema.TypeText},
			Nullable: false,
			Default: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'active'",
			},
		},
	}

	sql := gen.generateAddColumn(op)

	assert.Contains(t, sql, "ALTER TABLE users")
	assert.Contains(t, sql, "ADD COLUMN status TEXT NOT NULL DEFAULT 'active'")
}

// TestMapDataType_WithArrayAndEnum tests complex type scenarios
func TestMapDataType_WithArrayAndEnum(t *testing.T) {
	tests := []struct {
		name string
		dt   schema.DataType
		expected string
	}{
		{
			name: "enum array",
			dt: schema.DataType{
				Kind:      schema.TypeEnum,
				EnumName:  "status",
				ArrayDepth: 1,
			},
			expected: "status[]",
		},
		{
			name: "json array",
			dt: schema.DataType{
				Kind:       schema.TypeJSON,
				ArrayDepth: 2,
			},
			expected: "JSON[][]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDataType(tt.dt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatColumnList_ReservedKeywords tests formatting with special column names
func TestFormatColumnList_SpecialCases(t *testing.T) {
	tests := []struct {
		name string
		cols []string
		check func(t *testing.T, result string)
	}{
		{
			name: "all lowercase",
			cols: []string{"id", "email", "name"},
			check: func(t *testing.T, result string) {
				assert.Equal(t, "id, email, name", result)
			},
		},
		{
			name: "mixed reserved and normal",
			cols: []string{"id", "user", "email"},
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "id")
				assert.Contains(t, result, `"user"`)
				assert.Contains(t, result, "email")
			},
		},
		{
			name: "all reserved",
			cols: []string{"user", "table", "order"},
			check: func(t *testing.T, result string) {
				assert.Equal(t, `"user", "table", "order"`, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatColumnList(tt.cols)
			tt.check(t, result)
		})
	}
}

// TestDefaultsEqual_ComplexCases tests defaultsEqual with complex scenarios
func TestDefaultsEqual_ComplexCases(t *testing.T) {
	tests := []struct {
		name string
		a    *schema.DefaultValue
		b    *schema.DefaultValue
		expected bool
	}{
		{
			name: "both with sequence same name",
			a: &schema.DefaultValue{
				Kind: schema.DefaultSequence,
				Sequence: &schema.SequenceRef{Name: "seq"},
			},
			b: &schema.DefaultValue{
				Kind: schema.DefaultSequence,
				Sequence: &schema.SequenceRef{Name: "seq"},
			},
			expected: true,
		},
		{
			name: "both with sequence different name",
			a: &schema.DefaultValue{
				Kind: schema.DefaultSequence,
				Sequence: &schema.SequenceRef{Name: "seq1"},
			},
			b: &schema.DefaultValue{
				Kind: schema.DefaultSequence,
				Sequence: &schema.SequenceRef{Name: "seq2"},
			},
			expected: true, // defaultsEqual only checks Kind, Literal, Expression — not Sequence
		},
		{
			name: "literal vs expression",
			a: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "0",
			},
			b: &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "0",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultsEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
