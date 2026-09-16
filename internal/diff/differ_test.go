package diff

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestDiffer_DiffEnums(t *testing.T) {
	tests := []struct {
		name     string
		before   *schema.Schema
		after    *schema.Schema
		wantOps  int
		wantKind OperationKind
	}{
		{
			name: "create enum",
			before: &schema.Schema{
				Enums: []schema.Enum{},
			},
			after: &schema.Schema{
				Enums: []schema.Enum{
					{
						Name: "status",
						Values: []schema.EnumValue{
							{Label: "active"},
							{Label: "inactive"},
						},
					},
				},
			},
			wantOps:  1,
			wantKind: OpCreateEnum,
		},
		{
			name: "alter enum - add value",
			before: &schema.Schema{
				Enums: []schema.Enum{
					{
						Name: "status",
						Values: []schema.EnumValue{
							{Label: "active"},
						},
					},
				},
			},
			after: &schema.Schema{
				Enums: []schema.Enum{
					{
						Name: "status",
						Values: []schema.EnumValue{
							{Label: "active"},
							{Label: "pending"},
						},
					},
				},
			},
			wantOps:  1,
			wantKind: OpAlterEnum,
		},
		{
			name: "no change",
			before: &schema.Schema{
				Enums: []schema.Enum{
					{
						Name: "status",
						Values: []schema.EnumValue{
							{Label: "active"},
						},
					},
				},
			},
			after: &schema.Schema{
				Enums: []schema.Enum{
					{
						Name: "status",
						Values: []schema.EnumValue{
							{Label: "active"},
						},
					},
				},
			},
			wantOps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiffer(nil)
			ops, err := d.Diff(tt.before, tt.after)
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			if len(ops) != tt.wantOps {
				t.Errorf("got %d operations, want %d", len(ops), tt.wantOps)
			}

			if tt.wantOps > 0 && len(ops) > 0 {
				if ops[0].Kind() != tt.wantKind {
					t.Errorf("got operation kind %v, want %v", ops[0].Kind(), tt.wantKind)
				}
			}
		})
	}
}

func TestDiffer_DiffTables(t *testing.T) {
	tests := []struct {
		name     string
		before   *schema.Schema
		after    *schema.Schema
		wantOps  int
		wantKind OperationKind
	}{
		{
			name: "create table",
			before: &schema.Schema{
				Tables: []schema.Table{},
			},
			after: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "users",
						Columns: []schema.Column{
							{
								Name: "id",
								Type: schema.DataType{Kind: schema.TypeUUID},
							},
						},
					},
				},
			},
			wantOps:  1,
			wantKind: OpCreateTable,
		},
		{
			name: "drop table",
			before: &schema.Schema{
				Tables: []schema.Table{
					{Name: "users"},
				},
			},
			after: &schema.Schema{
				Tables: []schema.Table{},
			},
			wantOps:  1,
			wantKind: OpDropTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiffer(nil)
			ops, err := d.Diff(tt.before, tt.after)
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			if len(ops) != tt.wantOps {
				t.Errorf("got %d operations, want %d", len(ops), tt.wantOps)
			}

			if tt.wantOps > 0 && len(ops) > 0 {
				if ops[0].Kind() != tt.wantKind {
					t.Errorf("got operation kind %v, want %v", ops[0].Kind(), tt.wantKind)
				}
			}
		})
	}
}

func TestDiffer_DiffColumns(t *testing.T) {
	tests := []struct {
		name     string
		before   schema.Table
		after    schema.Table
		wantOps  int
		wantKind OperationKind
	}{
		{
			name: "add column",
			before: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}},
				},
			},
			after: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}},
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}},
				},
			},
			wantOps:  1,
			wantKind: OpAddColumn,
		},
		{
			name: "drop column",
			before: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}},
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}},
				},
			},
			after: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}},
				},
			},
			wantOps:  1,
			wantKind: OpDropColumn,
		},
		{
			name: "alter column type",
			before: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}},
				},
			},
			after: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeInt64}},
				},
			},
			wantOps:  1,
			wantKind: OpAlterColumn,
		},
		{
			name: "alter column nullable",
			before: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}, Nullable: true},
				},
			},
			after: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
				},
			},
			wantOps:  1,
			wantKind: OpAlterColumn,
		},
		{
			name: "rename column with hint",
			before: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "name", Type: schema.DataType{Kind: schema.TypeText}},
				},
			},
			after: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "username", Type: schema.DataType{Kind: schema.TypeText}},
				},
			},
			wantOps:  1,
			wantKind: OpRenameColumn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var renameHints map[string]string
			if tt.name == "rename column with hint" {
				renameHints = map[string]string{"name": "username"}
			}

			d := NewDiffer(renameHints)
			ops, err := d.diffTable(tt.before, tt.after)
			if err != nil {
				t.Fatalf("diffTable() error = %v", err)
			}

			if len(ops) != tt.wantOps {
				t.Errorf("got %d operations, want %d", len(ops), tt.wantOps)
			}

			if tt.wantOps > 0 && len(ops) > 0 {
				if ops[0].Kind() != tt.wantKind {
					t.Errorf("got operation kind %v, want %v", ops[0].Kind(), tt.wantKind)
				}
			}
		})
	}
}

func TestDiffer_DiffIndexes(t *testing.T) {
	tests := []struct {
		name     string
		before   schema.Table
		after    schema.Table
		wantOps  int
		wantKind OperationKind
	}{
		{
			name: "create index",
			before: schema.Table{
				Name:    "users",
				Indexes: []schema.Index{},
			},
			after: schema.Table{
				Name: "users",
				Indexes: []schema.Index{
					{
						Columns: []schema.IndexColumn{{Name: "email"}},
					},
				},
			},
			wantOps:  1,
			wantKind: OpCreateIndex,
		},
		{
			name: "drop index",
			before: schema.Table{
				Name: "users",
				Indexes: []schema.Index{
					{
						Name:    "idx_email",
						Columns: []schema.IndexColumn{{Name: "email"}},
					},
				},
			},
			after: schema.Table{
				Name:    "users",
				Indexes: []schema.Index{},
			},
			wantOps:  1,
			wantKind: OpDropIndex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiffer(nil)
			ops, err := d.diffTable(tt.before, tt.after)
			if err != nil {
				t.Fatalf("diffTable() error = %v", err)
			}

			if len(ops) != tt.wantOps {
				t.Errorf("got %d operations, want %d", len(ops), tt.wantOps)
			}

			if tt.wantOps > 0 && len(ops) > 0 {
				if ops[0].Kind() != tt.wantKind {
					t.Errorf("got operation kind %v, want %v", ops[0].Kind(), tt.wantKind)
				}
			}
		})
	}
}

func TestDiffer_DiffForeignKeys(t *testing.T) {
	tests := []struct {
		name     string
		before   schema.Table
		after    schema.Table
		wantOps  int
		wantKind OperationKind
	}{
		{
			name: "create foreign key",
			before: schema.Table{
				Name:        "posts",
				ForeignKeys: []schema.ForeignKey{},
			},
			after: schema.Table{
				Name: "posts",
				ForeignKeys: []schema.ForeignKey{
					{
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
			wantOps:  1,
			wantKind: OpCreateForeignKey,
		},
		{
			name: "drop foreign key",
			before: schema.Table{
				Name: "posts",
				ForeignKeys: []schema.ForeignKey{
					{
						Name:       "fk_user",
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
			after: schema.Table{
				Name:        "posts",
				ForeignKeys: []schema.ForeignKey{},
			},
			wantOps:  1,
			wantKind: OpDropForeignKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiffer(nil)
			ops, err := d.diffTable(tt.before, tt.after)
			if err != nil {
				t.Fatalf("diffTable() error = %v", err)
			}

			if len(ops) != tt.wantOps {
				t.Errorf("got %d operations, want %d", len(ops), tt.wantOps)
			}

			if tt.wantOps > 0 && len(ops) > 0 {
				if ops[0].Kind() != tt.wantKind {
					t.Errorf("got operation kind %v, want %v", ops[0].Kind(), tt.wantKind)
				}
			}
		})
	}
}

func TestDiffer_ColumnsEqual(t *testing.T) {
	d := NewDiffer(nil)

	tests := []struct {
		name string
		a    schema.Column
		b    schema.Column
		want bool
	}{
		{
			name: "identical columns",
			a: schema.Column{
				Name: "id",
				Type: schema.DataType{Kind: schema.TypeUUID},
			},
			b: schema.Column{
				Name: "id",
				Type: schema.DataType{Kind: schema.TypeUUID},
			},
			want: true,
		},
		{
			name: "different types",
			a: schema.Column{
				Name: "id",
				Type: schema.DataType{Kind: schema.TypeInt32},
			},
			b: schema.Column{
				Name: "id",
				Type: schema.DataType{Kind: schema.TypeInt64},
			},
			want: false,
		},
		{
			name: "different nullable",
			a: schema.Column{
				Name:     "email",
				Type:     schema.DataType{Kind: schema.TypeText},
				Nullable: true,
			},
			b: schema.Column{
				Name:     "email",
				Type:     schema.DataType{Kind: schema.TypeText},
				Nullable: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.columnsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("columnsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeExpression(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "CURRENT_TIMESTAMP",
			want:  "now()",
		},
		{
			input: "  gen_random_uuid()  ",
			want:  "uuid_generate_v4()",
		},
		{
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeExpression(tt.input); got != tt.want {
				t.Errorf("normalizeExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
