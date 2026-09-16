package postgres

import (
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateReverse_CreateTable(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name           string
		op             diff.Operation
		shouldHaveSQL  string
		shouldHaveWarn string
	}{
		{
			name: "reverse CREATE TABLE simple",
			op: diff.CreateTable{
				Table: schema.Table{Name: "users"},
			},
			shouldHaveSQL: "DROP TABLE",
		},
		{
			name: "reverse CREATE TABLE with reserved keyword",
			op: diff.CreateTable{
				Table: schema.Table{Name: "order"},
			},
			shouldHaveSQL: `DROP TABLE "order"`,
		},
		{
			name: "reverse CREATE TABLE with uppercase",
			op: diff.CreateTable{
				Table: schema.Table{Name: "Users"},
			},
			shouldHaveSQL: `DROP TABLE "Users"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, warning := gen.generateReverseOperation(tt.op)
			if tt.shouldHaveWarn != "" {
				assert.Contains(t, warning, tt.shouldHaveWarn)
			} else {
				assert.Equal(t, "", warning)
			}
			assert.Contains(t, sql, tt.shouldHaveSQL)
		})
	}
}

func TestGenerateReverse_DropTable(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.DropTable{Name: "users"}
	sql, warning := gen.generateReverseOperation(op)

	// DropTable is irreversible - should have warning
	assert.Empty(t, sql)
	assert.Contains(t, warning, "Cannot reverse DROP TABLE")
}

func TestGenerateReverse_AddColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name          string
		op            diff.AddColumn
		shouldHaveSQL string
	}{
		{
			name: "reverse ADD COLUMN",
			op: diff.AddColumn{
				Table: "users",
				Column: schema.Column{
					Name: "email",
					Type: schema.DataType{Kind: schema.TypeText},
				},
			},
			shouldHaveSQL: "DROP COLUMN",
		},
		{
			name: "reverse ADD COLUMN with reserved keyword",
			op: diff.AddColumn{
				Table: "users",
				Column: schema.Column{
					Name: "order",
					Type: schema.DataType{Kind: schema.TypeText},
				},
			},
			shouldHaveSQL: `DROP COLUMN "order"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, warning := gen.generateReverseOperation(tt.op)
			assert.Contains(t, sql, tt.shouldHaveSQL)
			assert.Empty(t, warning)
		})
	}
}

func TestGenerateReverse_DropColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.DropColumn{Table: "users", Column: "email"}
	sql, warning := gen.generateReverseOperation(op)

	// DropColumn is irreversible
	assert.Empty(t, sql)
	assert.Contains(t, warning, "Cannot reverse DROP COLUMN")
}

func TestGenerateReverse_RenameColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name          string
		op            diff.RenameColumn
		shouldHaveSQL string
	}{
		{
			name: "reverse RENAME COLUMN",
			op: diff.RenameColumn{
				Table: "users",
				From:  "old_name",
				To:    "new_name",
			},
			shouldHaveSQL: "new_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, warning := gen.generateReverseOperation(tt.op)
			// Reverse rename should swap From and To
			assert.Contains(t, sql, "RENAME COLUMN")
			assert.Contains(t, sql, tt.shouldHaveSQL)
			assert.Empty(t, warning)
		})
	}
}

func TestGenerateReverse_AlterColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.AlterColumn{
		Table:       "users",
		Column:      "age",
		OldType:     schema.DataType{Kind: schema.TypeInt32},
		NewType:     schema.DataType{Kind: schema.TypeInt64},
		OldNullable: false,
		NewNullable: false,
	}

	sql, warning := gen.generateReverseOperation(op)

	// AlterColumn changes are hard to reverse safely
	assert.Empty(t, sql)
	assert.Contains(t, warning, "may not be safely reversible")
}

func TestGenerateReverse_CreateIndex(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateIndex{
		Table: "users",
		Index: schema.Index{
			Name:    "idx_users_email",
			Columns: []schema.IndexColumn{{Name: "email"}},
		},
	}

	sql, warning := gen.generateReverseOperation(op)

	// CreateIndex reverse is DROP INDEX
	assert.Contains(t, sql, "DROP INDEX")
	assert.Contains(t, sql, "idx_users_email")
	assert.Empty(t, warning)
}

func TestGenerateReverse_DropIndex(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.DropIndex{Table: "users", Name: "idx_users_email"}
	sql, warning := gen.generateReverseOperation(op)

	// DropIndex is irreversible without schema info
	assert.Empty(t, sql)
	assert.Contains(t, warning, "Cannot reverse DROP INDEX")
}

func TestGenerateReverse_CreateForeignKey(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateForeignKey{
		Table: "posts",
		Constraint: schema.ForeignKey{
			Name:       "fk_posts_user_id",
			Columns:    []string{"user_id"},
			RefTable:   "users",
			RefColumns: []string{"id"},
		},
	}

	sql, warning := gen.generateReverseOperation(op)

	// Reverse CREATE FK is DROP CONSTRAINT
	assert.Contains(t, sql, "DROP CONSTRAINT")
	assert.Empty(t, warning)
}

func TestGenerateReverse_DropForeignKey(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.DropForeignKey{Table: "posts", Name: "fk_posts_user_id"}
	sql, warning := gen.generateReverseOperation(op)

	// DropForeignKey is irreversible without constraint definition
	assert.Empty(t, sql)
	assert.Contains(t, warning, "Cannot reverse DROP CONSTRAINT")
}

func TestGenerateReverse_CreateEnum(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateEnum{
		Enum: schema.Enum{
			Name: "status",
			Values: []schema.EnumValue{
				{Label: "active"},
				{Label: "inactive"},
			},
		},
	}

	sql, warning := gen.generateReverseOperation(op)

	// Reverse CREATE ENUM is DROP TYPE
	assert.Contains(t, sql, "DROP TYPE")
	assert.Empty(t, warning)
}

func TestGenerateReverse_AlterEnum(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.AlterEnum{
		Name:      "status",
		AddValues: []string{"pending"},
	}

	sql, warning := gen.generateReverseOperation(op)

	// AlterEnum is hard to reverse (can't remove enum values)
	assert.Empty(t, sql)
	assert.Contains(t, warning, "Cannot reverse ALTER TYPE")
}

func TestGenerateReverse_RenameTable(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name          string
		op            diff.RenameTable
		shouldHaveSQL string
	}{
		{
			name: "reverse RENAME TABLE",
			op: diff.RenameTable{
				From: "old_table",
				To:   "new_table",
			},
			shouldHaveSQL: "old_table",
		},
		{
			name: "reverse RENAME TABLE with reserved",
			op: diff.RenameTable{
				From: "user",
				To:   "order",
			},
			shouldHaveSQL: `"user"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, warning := gen.generateReverseOperation(tt.op)
			// Reverse should swap From and To
			assert.Contains(t, sql, "RENAME TO")
			assert.Contains(t, sql, tt.shouldHaveSQL)
			assert.Empty(t, warning)
		})
	}
}

func TestGenerateReverse_MultipleOperations(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	ops := []diff.Operation{
		diff.CreateEnum{
			Enum: schema.Enum{
				Name: "role",
				Values: []schema.EnumValue{
					{Label: "USER"},
					{Label: "ADMIN"},
				},
			},
		},
		diff.CreateTable{
			Table: schema.Table{Name: "users"},
		},
		diff.CreateIndex{
			Table: "users",
			Index: schema.Index{
				Name:    "idx_email",
				Columns: []schema.IndexColumn{{Name: "email"}},
			},
		},
	}

	statements, warnings, err := gen.GenerateReverse(ops)
	require.NoError(t, err)

	// Should reverse in opposite order: DROP INDEX, DROP TABLE, DROP TYPE
	require.Len(t, statements, 3)
	assert.Contains(t, statements[0], "DROP INDEX")
	assert.Contains(t, statements[1], "DROP TABLE")
	assert.Contains(t, statements[2], "DROP TYPE")
	assert.Empty(t, warnings)
}

func TestGenerateReverse_MixedSafeAndUnsafe(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	ops := []diff.Operation{
		diff.CreateTable{
			Table: schema.Table{Name: "users"},
		},
		diff.DropColumn{Table: "users", Column: "temp"},
	}

	statements, warnings, err := gen.GenerateReverse(ops)
	require.NoError(t, err)

	// Should have 1 statement (DROP TABLE) and 1 warning (DropColumn irreversible)
	require.Len(t, statements, 1)
	assert.Contains(t, statements[0], "DROP TABLE")
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "Cannot reverse DROP COLUMN")
}

func TestGenerateReverse_EmptyOperations(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	statements, warnings, err := gen.GenerateReverse([]diff.Operation{})
	require.NoError(t, err)

	assert.Empty(t, statements)
	assert.Empty(t, warnings)
}
