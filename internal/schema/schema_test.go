package schema

import (
	"strings"
	"testing"
)

func TestSchema_Validate_ValidSchema(t *testing.T) {
	schema := Schema{
		Tables: []Table{
			{
				Name: "users",
				Columns: []Column{
					{Name: "id", Type: DataType{Kind: TypeUUID}},
					{Name: "name", Type: DataType{Kind: TypeText}},
				},
			},
			{
				Name: "posts",
				Columns: []Column{
					{Name: "id", Type: DataType{Kind: TypeUUID}},
					{Name: "user_id", Type: DataType{Kind: TypeUUID}},
					{Name: "title", Type: DataType{Kind: TypeText}},
				},
				ForeignKeys: []ForeignKey{
					{
						Name:       "fk_posts_user",
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
		},
		Enums: []Enum{
			{
				Name: "status",
				Values: []EnumValue{
					{Label: "active", Order: 1},
					{Label: "inactive", Order: 2},
				},
			},
		},
	}

	if err := schema.Validate(); err != nil {
		t.Errorf("expected valid schema, got error: %v", err)
	}
}

func TestSchema_Validate_EmptySchema(t *testing.T) {
	schema := Schema{}

	if err := schema.Validate(); err != nil {
		t.Errorf("empty schema should be valid, got error: %v", err)
	}
}

func TestSchema_Validate_DuplicateTableNames(t *testing.T) {
	schema := Schema{
		Tables: []Table{
			{
				Name: "users",
				Columns: []Column{
					{Name: "id", Type: DataType{Kind: TypeUUID}},
				},
			},
			{
				Name: "users",
				Columns: []Column{
					{Name: "id", Type: DataType{Kind: TypeInt32}},
				},
			},
		},
	}

	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate table names, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate table name") {
		t.Errorf("expected 'duplicate table name' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "users") {
		t.Errorf("expected table name 'users' in error, got: %v", err)
	}
}

func TestSchema_Validate_DuplicateEnumNames(t *testing.T) {
	schema := Schema{
		Enums: []Enum{
			{Name: "status", Values: []EnumValue{{Label: "active", Order: 1}}},
			{Name: "status", Values: []EnumValue{{Label: "pending", Order: 1}}},
		},
	}

	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate enum names, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate enum name") {
		t.Errorf("expected 'duplicate enum name' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("expected enum name 'status' in error, got: %v", err)
	}
}

func TestSchema_Validate_ForeignKeyToNonExistentTable(t *testing.T) {
	schema := Schema{
		Tables: []Table{
			{
				Name: "posts",
				Columns: []Column{
					{Name: "id", Type: DataType{Kind: TypeUUID}},
					{Name: "user_id", Type: DataType{Kind: TypeUUID}},
				},
				ForeignKeys: []ForeignKey{
					{
						Name:       "fk_posts_user",
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
		},
	}

	err := schema.Validate()
	if err == nil {
		t.Fatal("expected error for FK to non-existent table, got nil")
	}

	if !strings.Contains(err.Error(), "references non-existent table") {
		t.Errorf("expected 'references non-existent table' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "users") {
		t.Errorf("expected reference to 'users' table in error, got: %v", err)
	}
}

func TestTable_Validate_ValidTable(t *testing.T) {
	table := Table{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
			{Name: "email", Type: DataType{Kind: TypeText}},
			{Name: "age", Type: DataType{Kind: TypeInt32}},
		},
		PrimaryKey: &PrimaryKey{
			Name:    "pk_users",
			Columns: []string{"id"},
		},
		Indexes: []Index{
			{
				Name: "idx_users_email",
				Columns: []IndexColumn{
					{Name: "email", Order: SortAsc},
				},
			},
		},
	}

	if err := table.Validate(); err != nil {
		t.Errorf("expected valid table, got error: %v", err)
	}
}

func TestTable_Validate_NoColumns(t *testing.T) {
	table := Table{
		Name:    "empty_table",
		Columns: []Column{},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected error for table with no columns, got nil")
	}

	if !strings.Contains(err.Error(), "has no columns") {
		t.Errorf("expected 'has no columns' in error, got: %v", err)
	}
}

func TestTable_Validate_DuplicateColumnNames(t *testing.T) {
	table := Table{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
			{Name: "id", Type: DataType{Kind: TypeInt32}},
		},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate column names, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate column name") {
		t.Errorf("expected 'duplicate column name' in error, got: %v", err)
	}
}

func TestTable_Validate_PrimaryKeyNonExistentColumn(t *testing.T) {
	table := Table{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
		},
		PrimaryKey: &PrimaryKey{
			Name:    "pk_users",
			Columns: []string{"user_id"},
		},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected error for PK on non-existent column, got nil")
	}

	if !strings.Contains(err.Error(), "primary key references non-existent column") {
		t.Errorf("expected 'primary key references non-existent column' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "user_id") {
		t.Errorf("expected column name 'user_id' in error, got: %v", err)
	}
}

func TestTable_Validate_ForeignKeyNonExistentColumn(t *testing.T) {
	table := Table{
		Name: "posts",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
		},
		ForeignKeys: []ForeignKey{
			{
				Name:       "fk_posts_user",
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
		},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected error for FK on non-existent column, got nil")
	}

	if !strings.Contains(err.Error(), "foreign key") && !strings.Contains(err.Error(), "non-existent column") {
		t.Errorf("expected FK error about non-existent column, got: %v", err)
	}
}

func TestTable_Validate_ForeignKeyColumnCountMismatch(t *testing.T) {
	table := Table{
		Name: "posts",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
			{Name: "user_id", Type: DataType{Kind: TypeUUID}},
		},
		ForeignKeys: []ForeignKey{
			{
				Name:       "fk_posts_user",
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{"id", "email"},
			},
		},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected error for FK column count mismatch, got nil")
	}

	if !strings.Contains(err.Error(), "1 columns but 2 reference columns") {
		t.Errorf("expected column count mismatch error, got: %v", err)
	}
}

func TestTable_Validate_IndexNonExistentColumn(t *testing.T) {
	table := Table{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
		},
		Indexes: []Index{
			{
				Name: "idx_users_email",
				Columns: []IndexColumn{
					{Name: "email", Order: SortAsc},
				},
			},
		},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected error for index on non-existent column, got nil")
	}

	if !strings.Contains(err.Error(), "index") && !strings.Contains(err.Error(), "non-existent column") {
		t.Errorf("expected index error about non-existent column, got: %v", err)
	}
}

func TestTable_Validate_ConstraintNonExistentColumn(t *testing.T) {
	table := Table{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
		},
		Constraints: []Constraint{
			{
				Name:    "unique_email",
				Type:    ConstraintUnique,
				Columns: []string{"email"},
			},
		},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected error for constraint on non-existent column, got nil")
	}

	if !strings.Contains(err.Error(), "constraint") && !strings.Contains(err.Error(), "non-existent column") {
		t.Errorf("expected constraint error about non-existent column, got: %v", err)
	}
}

func TestEnum_Construction(t *testing.T) {
	enum := Enum{
		Name: "user_status",
		Values: []EnumValue{
			{Label: "active", Order: 1},
			{Label: "suspended", Order: 2},
			{Label: "deleted", Order: 3},
		},
	}

	if enum.Name != "user_status" {
		t.Errorf("expected name 'user_status', got: %s", enum.Name)
	}
	if len(enum.Values) != 3 {
		t.Errorf("expected 3 values, got: %d", len(enum.Values))
	}
}
