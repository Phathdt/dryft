package prisma

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

// Composite primary keys must be emitted as a single model-level @@id([...]),
// because Prisma rejects a model carrying more than one field-level @id.
func TestWriteTable_CompositePrimaryKeyUsesModelLevelID(t *testing.T) {
	out := writeSingleTable(t, &schema.Table{
		Name: "user_roles",
		Columns: []schema.Column{
			{Name: "user_id", Type: schema.DataType{Kind: schema.TypeInt32}},
			{Name: "role_id", Type: schema.DataType{Kind: schema.TypeInt32}},
		},
		PrimaryKey: &schema.PrimaryKey{
			Name:    "user_roles_pkey",
			Columns: []string{"user_id", "role_id"},
		},
	})

	if !strings.Contains(out, "@@id([userId, roleId])") {
		t.Errorf("composite PK should produce @@id([userId, roleId]), got:\n%s", out)
	}
	if strings.Contains(out, "@id") && !strings.Contains(out, "@@id") {
		t.Errorf("composite PK must not emit field-level @id, got:\n%s", out)
	}
	if count := strings.Count(out, " @id"); count != 0 {
		t.Errorf("expected zero field-level @id for composite PK, found %d:\n%s", count, out)
	}
}

// A single-column primary key keeps the idiomatic field-level @id.
func TestWriteTable_SinglePrimaryKeyUsesFieldLevelID(t *testing.T) {
	out := writeSingleTable(t, &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}},
		},
		PrimaryKey: &schema.PrimaryKey{Name: "users_pkey", Columns: []string{"id"}},
	})

	if !strings.Contains(out, "@id") {
		t.Errorf("single-column PK should emit field-level @id, got:\n%s", out)
	}
	if strings.Contains(out, "@@id") {
		t.Errorf("single-column PK should not emit @@id, got:\n%s", out)
	}
}

// Enum-typed columns must reference the generated enum model name, not String,
// otherwise the emitted schema loses the enum type on round-trip.
func TestWriteTable_EnumColumnReferencesEnumType(t *testing.T) {
	out := writeSingleTable(t, &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "status", Type: schema.DataType{Kind: schema.TypeEnum, EnumName: "user_status"}},
		},
	})

	if !strings.Contains(out, "status UserStatus") {
		t.Errorf("enum column should use enum type UserStatus, got:\n%s", out)
	}
}

// Nullable and array enum columns keep their modifiers alongside the enum name.
func TestWriteTable_EnumColumnModifiers(t *testing.T) {
	out := writeSingleTable(t, &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{
				Name:     "status",
				Type:     schema.DataType{Kind: schema.TypeEnum, EnumName: "user_status"},
				Nullable: true,
			},
			{
				Name: "tags",
				Type: schema.DataType{Kind: schema.TypeEnum, EnumName: "tag_kind", ArrayDepth: 1},
			},
		},
	})

	if !strings.Contains(out, "UserStatus?") {
		t.Errorf("nullable enum column should render UserStatus?, got:\n%s", out)
	}
	if !strings.Contains(out, "TagKind[]") {
		t.Errorf("enum array column should render TagKind[], got:\n%s", out)
	}
}

// Model-level attributes must be separated from the field list by a blank line;
// the formatter previously dropped empty separator lines.
func TestWriteTable_BlankLineBeforeModelAttributes(t *testing.T) {
	out := writeSingleTable(t, &schema.Table{
		Name: "user_roles",
		Columns: []schema.Column{
			{Name: "user_id", Type: schema.DataType{Kind: schema.TypeInt32}},
		},
	})

	if !strings.Contains(out, "\n\n  @@map(\"user_roles\")") {
		t.Errorf("expected blank line before @@map, got:\n%s", out)
	}
}

// TestWriteTable_FieldNameCollisionIsRejected covers columns whose distinct
// database names transform to the same Prisma field name. Emitting both would
// produce a model with duplicate fields, which Prisma rejects.
func TestWriteTable_FieldNameCollisionIsRejected(t *testing.T) {
	cases := []struct {
		name    string
		columns []string
	}{
		{"consecutive underscores", []string{"user_id", "user__id"}},
		{"already camelCase", []string{"user_id", "userId"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			table := &schema.Table{Name: "probe"}
			for _, col := range tc.columns {
				table.Columns = append(table.Columns, schema.Column{
					Name: col,
					Type: schema.DataType{Kind: schema.TypeInt32},
				})
			}

			w := NewWriter(DefaultNamingConvention())
			out, err := w.writeTable(table, nil)
			if err == nil {
				t.Fatalf("expected collision error for %v, got model:\n%s", tc.columns, out)
			}
			if !strings.Contains(err.Error(), "both map to Prisma field") {
				t.Errorf("expected error to name the colliding field, got: %v", err)
			}
			for _, col := range tc.columns {
				if !strings.Contains(err.Error(), col) {
					t.Errorf("expected error to name column %q, got: %v", col, err)
				}
			}
		})
	}
}

// TestWrite_ModelNameCollisionIsRejected covers two tables whose names
// transform to the same Prisma model name.
func TestWrite_ModelNameCollisionIsRejected(t *testing.T) {
	w := NewWriter(DefaultNamingConvention())
	_, err := w.Write(&schema.Schema{
		Tables: []schema.Table{
			{Name: "user_roles", Columns: []schema.Column{{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}}}},
			{Name: "user__roles", Columns: []schema.Column{{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}}}},
		},
	})

	if err == nil {
		t.Fatal("expected model name collision error, got nil")
	}
	if !strings.Contains(err.Error(), "both map to Prisma model") {
		t.Errorf("expected error to name the colliding model, got: %v", err)
	}
}

// writeSingleTable renders one table and returns the emitted model block.
func writeSingleTable(t *testing.T, table *schema.Table) string {
	t.Helper()

	w := NewWriter(DefaultNamingConvention())
	out, err := w.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable(%q) returned error: %v", table.Name, err)
	}
	return out
}
