package schema

import (
	"strings"
	"testing"
)

func TestValidate_Integration_ValidComplexSchema(t *testing.T) {
	schema := Schema{
		Tables: []Table{
			{
				Name: "users",
				Columns: []Column{
					{Name: "id", Type: DataType{Kind: TypeUUID}, Nullable: false},
					{Name: "email", Type: DataType{Kind: TypeVarChar, Precision: 255}, Nullable: false},
					{Name: "name", Type: DataType{Kind: TypeText}},
					{Name: "age", Type: DataType{Kind: TypeInt32}},
					{Name: "created_at", Type: DataType{Kind: TypeTimestampTZ}, Nullable: false},
				},
				PrimaryKey: &PrimaryKey{
					Name:    "pk_users",
					Columns: []string{"id"},
				},
				Indexes: []Index{
					{
						Name:    "idx_users_email",
						Columns: []IndexColumn{{Name: "email", Order: SortAsc}},
						Unique:  true,
						Type:    IndexBTree,
					},
				},
				Constraints: []Constraint{
					{
						Name:       "check_age_positive",
						Type:       ConstraintCheck,
						Expression: "age >= 0",
					},
				},
			},
			{
				Name: "posts",
				Columns: []Column{
					{Name: "id", Type: DataType{Kind: TypeUUID}},
					{Name: "user_id", Type: DataType{Kind: TypeUUID}},
					{Name: "title", Type: DataType{Kind: TypeText}},
					{Name: "content", Type: DataType{Kind: TypeText}},
					{Name: "published_at", Type: DataType{Kind: TypeTimestampTZ}},
				},
				PrimaryKey: &PrimaryKey{
					Name:    "pk_posts",
					Columns: []string{"id"},
				},
				ForeignKeys: []ForeignKey{
					{
						Name:       "fk_posts_user",
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
						OnDelete:   ActionCascade,
						OnUpdate:   ActionNoAction,
					},
				},
			},
		},
		Enums: []Enum{
			{
				Name: "post_status",
				Values: []EnumValue{
					{Label: "draft", Order: 1},
					{Label: "published", Order: 2},
					{Label: "archived", Order: 3},
				},
			},
		},
	}

	// Validate schema
	if err := schema.Validate(); err != nil {
		t.Fatalf("expected valid complex schema, got error: %v", err)
	}

	// Validate each table
	for _, table := range schema.Tables {
		if err := table.Validate(); err != nil {
			t.Errorf("table %q validation failed: %v", table.Name, err)
		}
	}

	// Validate each column
	for _, table := range schema.Tables {
		for _, col := range table.Columns {
			if err := col.Validate(); err != nil {
				t.Errorf("column %q.%q validation failed: %v", table.Name, col.Name, err)
			}
		}
	}

	// Validate each foreign key
	for _, table := range schema.Tables {
		for _, fk := range table.ForeignKeys {
			if err := fk.Validate(); err != nil {
				t.Errorf("foreign key %q validation failed: %v", fk.Name, err)
			}
		}
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	schema := Schema{
		Tables: []Table{
			{Name: "users", Columns: []Column{{Name: "id", Type: DataType{Kind: TypeUUID}}}},
			{Name: "users", Columns: []Column{{Name: "id", Type: DataType{Kind: TypeInt32}}}}, // duplicate
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
						RefTable:   "accounts", // non-existent table
						RefColumns: []string{"id"},
					},
				},
			},
		},
		Enums: []Enum{
			{Name: "status", Values: []EnumValue{{Label: "active", Order: 1}}},
			{Name: "status", Values: []EnumValue{{Label: "inactive", Order: 1}}}, // duplicate
		},
	}

	err := schema.Validate()
	if err == nil {
		t.Fatal("expected validation errors, got nil")
	}

	errMsg := err.Error()

	// Should report duplicate table name
	if !strings.Contains(errMsg, "duplicate table name") {
		t.Error("expected error about duplicate table name")
	}

	// Should report duplicate enum name
	if !strings.Contains(errMsg, "duplicate enum name") {
		t.Error("expected error about duplicate enum name")
	}

	// Should report FK to non-existent table
	if !strings.Contains(errMsg, "references non-existent table") {
		t.Error("expected error about FK to non-existent table")
	}
}

func TestTable_Validate_MultipleErrors(t *testing.T) {
	table := Table{
		Name: "broken_table",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
			{Name: "email", Type: DataType{Kind: TypeText}},
			{Name: "email", Type: DataType{Kind: TypeVarChar, Precision: 255}}, // duplicate
		},
		PrimaryKey: &PrimaryKey{
			Name:    "pk_test",
			Columns: []string{"user_id"}, // non-existent column
		},
		ForeignKeys: []ForeignKey{
			{
				Name:       "fk_test",
				Columns:    []string{"org_id"}, // non-existent column
				RefTable:   "organizations",
				RefColumns: []string{"id"},
			},
		},
		Indexes: []Index{
			{
				Name: "idx_test",
				Columns: []IndexColumn{
					{Name: "status"}, // non-existent column
				},
			},
		},
	}

	err := table.Validate()
	if err == nil {
		t.Fatal("expected validation errors, got nil")
	}

	errMsg := err.Error()

	// Should report duplicate column
	if !strings.Contains(errMsg, "duplicate column name") {
		t.Error("expected error about duplicate column name")
	}

	// Should report PK on non-existent column
	if !strings.Contains(errMsg, "primary key references non-existent column") {
		t.Error("expected error about PK on non-existent column")
	}

	// Should report FK on non-existent column
	if !strings.Contains(errMsg, "foreign key") && !strings.Contains(errMsg, "non-existent column") {
		t.Error("expected error about FK on non-existent column")
	}

	// Should report index on non-existent column
	if !strings.Contains(errMsg, "index") && !strings.Contains(errMsg, "non-existent column") {
		t.Error("expected error about index on non-existent column")
	}
}

func TestValidate_ErrorMessages_AreClear(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() error
		wantInError []string
	}{
		{
			name: "duplicate table with specific name",
			setup: func() error {
				s := Schema{
					Tables: []Table{
						{Name: "products", Columns: []Column{{Name: "id", Type: DataType{Kind: TypeUUID}}}},
						{Name: "products", Columns: []Column{{Name: "id", Type: DataType{Kind: TypeInt32}}}},
					},
				}
				return s.Validate()
			},
			wantInError: []string{"duplicate table name", "products"},
		},
		{
			name: "FK to specific missing table",
			setup: func() error {
				s := Schema{
					Tables: []Table{
						{
							Name: "orders",
							Columns: []Column{
								{Name: "id", Type: DataType{Kind: TypeUUID}},
								{Name: "customer_id", Type: DataType{Kind: TypeUUID}},
							},
							ForeignKeys: []ForeignKey{
								{
									Name:       "fk_orders_customer",
									Columns:    []string{"customer_id"},
									RefTable:   "customers",
									RefColumns: []string{"id"},
								},
							},
						},
					},
				}
				return s.Validate()
			},
			wantInError: []string{"references non-existent table", "customers"},
		},
		{
			name: "PK on specific missing column",
			setup: func() error {
				t := Table{
					Name: "items",
					Columns: []Column{
						{Name: "id", Type: DataType{Kind: TypeUUID}},
					},
					PrimaryKey: &PrimaryKey{
						Name:    "pk_items",
						Columns: []string{"item_id"},
					},
				}
				return t.Validate()
			},
			wantInError: []string{"primary key references non-existent column", "item_id"},
		},
		{
			name: "empty column name",
			setup: func() error {
				c := Column{
					Name: "",
					Type: DataType{Kind: TypeText},
				}
				return c.Validate()
			},
			wantInError: []string{"name cannot be empty"},
		},
		{
			name: "unknown type with column name",
			setup: func() error {
				c := Column{
					Name: "mystery_field",
					Type: DataType{Kind: TypeUnknown},
				}
				return c.Validate()
			},
			wantInError: []string{"unknown type", "mystery_field"},
		},
		{
			name: "FK column count mismatch with numbers",
			setup: func() error {
				fk := ForeignKey{
					Name:       "fk_test",
					Columns:    []string{"col1"},
					RefTable:   "other",
					RefColumns: []string{"id1", "id2"},
				}
				return fk.Validate()
			},
			wantInError: []string{"1 columns but 2 reference columns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantInError {
				if !strings.Contains(errMsg, want) {
					t.Errorf("expected %q in error message, got: %s", want, errMsg)
				}
			}
		})
	}
}

func TestValidate_CompositePrimaryKey(t *testing.T) {
	table := Table{
		Name: "user_roles",
		Columns: []Column{
			{Name: "user_id", Type: DataType{Kind: TypeUUID}},
			{Name: "role_id", Type: DataType{Kind: TypeUUID}},
			{Name: "assigned_at", Type: DataType{Kind: TypeTimestampTZ}},
		},
		PrimaryKey: &PrimaryKey{
			Name:    "pk_user_roles",
			Columns: []string{"user_id", "role_id"},
		},
	}

	if err := table.Validate(); err != nil {
		t.Errorf("expected valid composite PK, got error: %v", err)
	}
}

func TestValidate_CompositeForeignKey(t *testing.T) {
	table := Table{
		Name: "order_items",
		Columns: []Column{
			{Name: "order_id", Type: DataType{Kind: TypeUUID}},
			{Name: "product_id", Type: DataType{Kind: TypeUUID}},
			{Name: "variant_id", Type: DataType{Kind: TypeUUID}},
			{Name: "quantity", Type: DataType{Kind: TypeInt32}},
		},
		ForeignKeys: []ForeignKey{
			{
				Name:       "fk_order_items_variant",
				Columns:    []string{"product_id", "variant_id"},
				RefTable:   "product_variants",
				RefColumns: []string{"product_id", "id"},
			},
		},
	}

	if err := table.Validate(); err != nil {
		t.Errorf("expected valid composite FK, got error: %v", err)
	}
}

func TestValidate_NoPrimaryKey(t *testing.T) {
	table := Table{
		Name: "logs",
		Columns: []Column{
			{Name: "timestamp", Type: DataType{Kind: TypeTimestampTZ}},
			{Name: "message", Type: DataType{Kind: TypeText}},
		},
		PrimaryKey: nil, // No PK is valid
	}

	if err := table.Validate(); err != nil {
		t.Errorf("table without PK should be valid, got error: %v", err)
	}
}

func TestValidate_EmptyForeignKeysList(t *testing.T) {
	table := Table{
		Name: "standalone",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
		},
		ForeignKeys: []ForeignKey{}, // Empty FK list is valid
	}

	if err := table.Validate(); err != nil {
		t.Errorf("table with empty FK list should be valid, got error: %v", err)
	}
}

func TestValidate_PartialIndexWithWhereClause(t *testing.T) {
	table := Table{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
			{Name: "email", Type: DataType{Kind: TypeText}},
			{Name: "deleted_at", Type: DataType{Kind: TypeTimestampTZ}},
		},
		Indexes: []Index{
			{
				Name: "idx_active_users_email",
				Columns: []IndexColumn{
					{Name: "email", Order: SortAsc},
				},
				Where: "deleted_at IS NULL",
				Type:  IndexBTree,
			},
		},
	}

	if err := table.Validate(); err != nil {
		t.Errorf("partial index should be valid, got error: %v", err)
	}
}

func TestValidate_CheckConstraintWithExpression(t *testing.T) {
	table := Table{
		Name: "products",
		Columns: []Column{
			{Name: "id", Type: DataType{Kind: TypeUUID}},
			{Name: "price", Type: DataType{Kind: TypeNumeric, Precision: 10, Scale: 2}},
			{Name: "discount", Type: DataType{Kind: TypeNumeric, Precision: 5, Scale: 2}},
		},
		Constraints: []Constraint{
			{
				Name:       "check_price_positive",
				Type:       ConstraintCheck,
				Expression: "price > 0",
			},
			{
				Name:       "check_discount_range",
				Type:       ConstraintCheck,
				Expression: "discount >= 0 AND discount <= 100",
			},
		},
	}

	if err := table.Validate(); err != nil {
		t.Errorf("CHECK constraints should be valid, got error: %v", err)
	}
}
