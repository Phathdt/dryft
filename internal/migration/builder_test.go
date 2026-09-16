package migration

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaBuilder_CreateTable(t *testing.T) {
	builder := NewSchemaBuilder()

	stmt := &CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "SERIAL", Nullable: false},
			{Name: "email", Type: "TEXT", Nullable: false},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: true},
		},
	}

	err := builder.Apply(stmt)
	require.NoError(t, err)

	result := builder.Build()
	require.Len(t, result.Tables, 1)

	table := result.Tables[0]
	assert.Equal(t, "users", table.Name)
	assert.Len(t, table.Columns, 3)
	assert.Equal(t, "id", table.Columns[0].Name)
	assert.Equal(t, schema.TypeInt32, table.Columns[0].Type.Kind)
	assert.Equal(t, "email", table.Columns[1].Name)
	assert.Equal(t, schema.TypeText, table.Columns[1].Type.Kind)
}

func TestSchemaBuilder_CreateTableIfNotExists(t *testing.T) {
	builder := NewSchemaBuilder()

	// First create
	builder.Apply(&CreateTable{Name: "users"})

	// Second create with IF NOT EXISTS should succeed
	err := builder.Apply(&CreateTable{
		Name:        "users",
		IfNotExists: true,
	})
	assert.NoError(t, err)

	// Without IF NOT EXISTS should fail
	err = builder.Apply(&CreateTable{Name: "users"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestSchemaBuilder_DropTable(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{Name: "users"})
	err := builder.Apply(&DropTable{Name: "users"})
	require.NoError(t, err)

	result := builder.Build()
	assert.Len(t, result.Tables, 0)
}

func TestSchemaBuilder_DropTableIfExists(t *testing.T) {
	builder := NewSchemaBuilder()

	// Drop non-existent table with IF EXISTS should succeed
	err := builder.Apply(&DropTable{
		Name:     "nonexistent",
		IfExists: true,
	})
	assert.NoError(t, err)

	// Without IF EXISTS should fail
	err = builder.Apply(&DropTable{Name: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestSchemaBuilder_AddColumn(t *testing.T) {
	builder := NewSchemaBuilder()

	// Create table
	builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "INTEGER", Nullable: false},
		},
	})

	// Add column
	err := builder.Apply(&AlterTable{
		Table: "users",
		Action: &AddColumn{
			Column: ColumnDef{Name: "email", Type: "TEXT", Nullable: true},
		},
	})
	require.NoError(t, err)

	result := builder.Build()
	table := result.Tables[0]
	assert.Len(t, table.Columns, 2)
	assert.Equal(t, "email", table.Columns[1].Name)
	assert.Equal(t, schema.TypeText, table.Columns[1].Type.Kind)
	assert.True(t, table.Columns[1].Nullable)
}

func TestSchemaBuilder_DropColumn(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "INTEGER"},
			{Name: "legacy", Type: "TEXT"},
		},
	})

	err := builder.Apply(&AlterTable{
		Table: "users",
		Action: &DropColumn{
			Name: "legacy",
		},
	})
	require.NoError(t, err)

	result := builder.Build()
	table := result.Tables[0]
	assert.Len(t, table.Columns, 1)
	assert.Equal(t, "id", table.Columns[0].Name)
}

func TestSchemaBuilder_DropColumnIfExists(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "INTEGER"},
		},
	})

	// Drop non-existent column with IF EXISTS should succeed
	err := builder.Apply(&AlterTable{
		Table: "users",
		Action: &DropColumn{
			Name:     "nonexistent",
			IfExists: true,
		},
	})
	assert.NoError(t, err)

	// Without IF EXISTS should fail
	err = builder.Apply(&AlterTable{
		Table: "users",
		Action: &DropColumn{
			Name: "nonexistent",
		},
	})
	assert.Error(t, err)
}

func TestSchemaBuilder_RenameColumn(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "old_name", Type: "TEXT"},
		},
	})

	err := builder.Apply(&AlterTable{
		Table: "users",
		Action: &RenameColumn{
			OldName: "old_name",
			NewName: "new_name",
		},
	})
	require.NoError(t, err)

	result := builder.Build()
	table := result.Tables[0]
	assert.Equal(t, "new_name", table.Columns[0].Name)
}

func TestSchemaBuilder_AlterColumn(t *testing.T) {
	t.Run("set not null", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "email", Type: "TEXT", Nullable: true},
			},
		})

		err := builder.Apply(&AlterTable{
			Table: "users",
			Action: &AlterColumn{
				Name:       "email",
				SetNotNull: true,
			},
		})
		require.NoError(t, err)

		result := builder.Build()
		assert.False(t, result.Tables[0].Columns[0].Nullable)
	})

	t.Run("drop not null", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "email", Type: "TEXT", Nullable: false},
			},
		})

		err := builder.Apply(&AlterTable{
			Table: "users",
			Action: &AlterColumn{
				Name:        "email",
				DropNotNull: true,
			},
		})
		require.NoError(t, err)

		result := builder.Build()
		assert.True(t, result.Tables[0].Columns[0].Nullable)
	})

	t.Run("set default", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "active", Type: "BOOLEAN", Nullable: false},
			},
		})

		defaultVal := "true"
		err := builder.Apply(&AlterTable{
			Table: "users",
			Action: &AlterColumn{
				Name:       "active",
				SetDefault: &defaultVal,
			},
		})
		require.NoError(t, err)

		result := builder.Build()
		col := result.Tables[0].Columns[0]
		require.NotNil(t, col.Default)
		assert.Equal(t, schema.DefaultLiteral, col.Default.Kind)
		assert.Equal(t, "true", col.Default.Literal)
	})

	t.Run("drop default", func(t *testing.T) {
		builder := NewSchemaBuilder()
		defaultVal := "true"
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "active", Type: "BOOLEAN", Default: &defaultVal},
			},
		})

		err := builder.Apply(&AlterTable{
			Table: "users",
			Action: &AlterColumn{
				Name:        "active",
				DropDefault: true,
			},
		})
		require.NoError(t, err)

		result := builder.Build()
		assert.Nil(t, result.Tables[0].Columns[0].Default)
	})

	t.Run("change type", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "age", Type: "INTEGER"},
			},
		})

		err := builder.Apply(&AlterTable{
			Table: "users",
			Action: &AlterColumn{
				Name:    "age",
				SetType: "BIGINT",
			},
		})
		require.NoError(t, err)

		result := builder.Build()
		assert.Equal(t, schema.TypeInt64, result.Tables[0].Columns[0].Type.Kind)
	})
}

func TestSchemaBuilder_CreateEnum(t *testing.T) {
	builder := NewSchemaBuilder()

	stmt := &CreateType{
		Name:   "user_role",
		Values: []string{"admin", "user", "guest"},
	}

	err := builder.Apply(stmt)
	require.NoError(t, err)

	result := builder.Build()
	require.Len(t, result.Enums, 1)

	enum := result.Enums[0]
	assert.Equal(t, "user_role", enum.Name)
	assert.Len(t, enum.Values, 3)
	assert.Equal(t, "admin", enum.Values[0].Label)
	assert.Equal(t, 0, enum.Values[0].Order)
	assert.Equal(t, "user", enum.Values[1].Label)
	assert.Equal(t, 1, enum.Values[1].Order)
}

func TestSchemaBuilder_DropEnum(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateType{Name: "user_role", Values: []string{"admin"}})
	err := builder.Apply(&DropType{Name: "user_role"})
	require.NoError(t, err)

	result := builder.Build()
	assert.Len(t, result.Enums, 0)
}

func TestSchemaBuilder_CreateIndex(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "email", Type: "TEXT"},
		},
	})

	err := builder.Apply(&CreateIndex{
		Name:   "idx_users_email",
		Table:  "users",
		Columns: []IndexColumnDef{{Name: "email"}},
		Unique: true,
		Method: "btree",
	})
	require.NoError(t, err)

	result := builder.Build()
	table := result.Tables[0]
	require.Len(t, table.Indexes, 1)

	idx := table.Indexes[0]
	assert.Equal(t, "idx_users_email", idx.Name)
	assert.True(t, idx.Unique)
	assert.Equal(t, schema.IndexBTree, idx.Type)
	assert.Len(t, idx.Columns, 1)
	assert.Equal(t, "email", idx.Columns[0].Name)
}

func TestSchemaBuilder_CreateIndexWithMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected schema.IndexType
	}{
		{"btree", schema.IndexBTree},
		{"hash", schema.IndexHash},
		{"gin", schema.IndexGIN},
		{"gist", schema.IndexGiST},
		{"", schema.IndexBTree}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			builder := NewSchemaBuilder()
			builder.Apply(&CreateTable{
				Name:    "test",
				Columns: []ColumnDef{{Name: "data", Type: "TEXT"}},
			})

			builder.Apply(&CreateIndex{
				Name:    "idx_test",
				Table:   "test",
				Columns: []IndexColumnDef{{Name: "data"}},
				Method:  tt.method,
			})

			result := builder.Build()
			assert.Equal(t, tt.expected, result.Tables[0].Indexes[0].Type)
		})
	}
}

func TestSchemaBuilder_DropIndex(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{
		Name:    "users",
		Columns: []ColumnDef{{Name: "email", Type: "TEXT"}},
	})
	builder.Apply(&CreateIndex{
		Name:    "idx_users_email",
		Table:   "users",
		Columns: []IndexColumnDef{{Name: "email"}},
	})

	err := builder.Apply(&DropIndex{Name: "idx_users_email"})
	require.NoError(t, err)

	result := builder.Build()
	assert.Len(t, result.Tables[0].Indexes, 0)
}

func TestSchemaBuilder_TableConstraints(t *testing.T) {
	t.Run("primary key", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "id", Type: "INTEGER"},
			},
			Constraints: []TableConstraint{
				{
					Type:    PrimaryKeyConstraint,
					Name:    "users_pkey",
					Columns: []string{"id"},
				},
			},
		})

		result := builder.Build()
		table := result.Tables[0]
		require.NotNil(t, table.PrimaryKey)
		assert.Equal(t, "users_pkey", table.PrimaryKey.Name)
		assert.Equal(t, []string{"id"}, table.PrimaryKey.Columns)
	})

	t.Run("foreign key", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{Name: "users", Columns: []ColumnDef{{Name: "id", Type: "INTEGER"}}})
		builder.Apply(&CreateTable{
			Name: "posts",
			Columns: []ColumnDef{
				{Name: "id", Type: "INTEGER"},
				{Name: "user_id", Type: "INTEGER"},
			},
			Constraints: []TableConstraint{
				{
					Type:       ForeignKeyConstraint,
					Name:       "posts_user_id_fkey",
					Columns:    []string{"user_id"},
					RefTable:   "users",
					RefColumns: []string{"id"},
					OnDelete:   "CASCADE",
				},
			},
		})

		result := builder.Build()
		var posts schema.Table
		for _, t := range result.Tables {
			if t.Name == "posts" {
				posts = t
			}
		}

		require.Len(t, posts.ForeignKeys, 1)
		fk := posts.ForeignKeys[0]
		assert.Equal(t, "posts_user_id_fkey", fk.Name)
		assert.Equal(t, []string{"user_id"}, fk.Columns)
		assert.Equal(t, "users", fk.RefTable)
		assert.Equal(t, []string{"id"}, fk.RefColumns)
		assert.Equal(t, schema.ActionCascade, fk.OnDelete)
	})

	t.Run("unique constraint", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "email", Type: "TEXT"},
			},
			Constraints: []TableConstraint{
				{
					Type:    UniqueConstraint,
					Name:    "users_email_key",
					Columns: []string{"email"},
				},
			},
		})

		result := builder.Build()
		table := result.Tables[0]
		require.Len(t, table.Constraints, 1)
		c := table.Constraints[0]
		assert.Equal(t, "users_email_key", c.Name)
		assert.Equal(t, schema.ConstraintUnique, c.Type)
		assert.Equal(t, []string{"email"}, c.Columns)
	})

	t.Run("check constraint", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name: "users",
			Columns: []ColumnDef{
				{Name: "age", Type: "INTEGER"},
			},
			Constraints: []TableConstraint{
				{
					Type:      CheckConstraint,
					Name:      "users_age_check",
					CheckExpr: "age >= 0",
				},
			},
		})

		result := builder.Build()
		table := result.Tables[0]
		require.Len(t, table.Constraints, 1)
		c := table.Constraints[0]
		assert.Equal(t, "users_age_check", c.Name)
		assert.Equal(t, schema.ConstraintCheck, c.Type)
		assert.Equal(t, "age >= 0", c.Expression)
	})
}

func TestSchemaBuilder_ErrorHandling(t *testing.T) {
	t.Run("alter non-existent table", func(t *testing.T) {
		builder := NewSchemaBuilder()
		err := builder.Apply(&AlterTable{
			Table:  "nonexistent",
			Action: &AddColumn{Column: ColumnDef{Name: "col", Type: "TEXT"}},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("add duplicate column", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name:    "users",
			Columns: []ColumnDef{{Name: "email", Type: "TEXT"}},
		})

		err := builder.Apply(&AlterTable{
			Table:  "users",
			Action: &AddColumn{Column: ColumnDef{Name: "email", Type: "TEXT"}},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("rename non-existent column", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name:    "users",
			Columns: []ColumnDef{{Name: "email", Type: "TEXT"}},
		})

		err := builder.Apply(&AlterTable{
			Table: "users",
			Action: &RenameColumn{
				OldName: "nonexistent",
				NewName: "new_name",
			},
		})
		assert.Error(t, err)
	})

	t.Run("create duplicate enum", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateType{Name: "status", Values: []string{"active"}})

		err := builder.Apply(&CreateType{Name: "status", Values: []string{"inactive"}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("create index on non-existent table", func(t *testing.T) {
		builder := NewSchemaBuilder()
		err := builder.Apply(&CreateIndex{
			Name:    "idx_test",
			Table:   "nonexistent",
			Columns: []IndexColumnDef{{Name: "col"}},
		})
		assert.Error(t, err)
	})

	t.Run("create duplicate index", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{
			Name:    "users",
			Columns: []ColumnDef{{Name: "email", Type: "TEXT"}},
		})
		builder.Apply(&CreateIndex{
			Name:    "idx_test",
			Table:   "users",
			Columns: []IndexColumnDef{{Name: "email"}},
		})

		err := builder.Apply(&CreateIndex{
			Name:    "idx_test",
			Table:   "users",
			Columns: []IndexColumnDef{{Name: "email"}},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestSchemaBuilder_IncrementalMigrations(t *testing.T) {
	builder := NewSchemaBuilder()

	// Migration 1: Create users table
	err := builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "SERIAL", Nullable: false},
			{Name: "email", Type: "TEXT", Nullable: false},
		},
		Constraints: []TableConstraint{
			{Type: PrimaryKeyConstraint, Columns: []string{"id"}},
		},
	})
	require.NoError(t, err)

	// Migration 2: Add name column
	err = builder.Apply(&AlterTable{
		Table:  "users",
		Action: &AddColumn{Column: ColumnDef{Name: "name", Type: "TEXT", Nullable: true}},
	})
	require.NoError(t, err)

	// Migration 3: Create enum
	err = builder.Apply(&CreateType{
		Name:   "user_role",
		Values: []string{"admin", "user"},
	})
	require.NoError(t, err)

	// Migration 4: Add role column with enum type
	err = builder.Apply(&AlterTable{
		Table:  "users",
		Action: &AddColumn{Column: ColumnDef{Name: "role", Type: "user_role", Nullable: true}},
	})
	require.NoError(t, err)

	// Migration 5: Create index
	err = builder.Apply(&CreateIndex{
		Name:    "idx_users_email",
		Table:   "users",
		Columns: []IndexColumnDef{{Name: "email"}},
		Unique:  true,
	})
	require.NoError(t, err)

	// Verify final state
	result := builder.Build()

	require.Len(t, result.Tables, 1)
	table := result.Tables[0]
	assert.Equal(t, "users", table.Name)
	assert.Len(t, table.Columns, 4) // id, email, name, role
	assert.Len(t, table.Indexes, 1)
	assert.NotNil(t, table.PrimaryKey)

	require.Len(t, result.Enums, 1)
	assert.Equal(t, "user_role", result.Enums[0].Name)
}
