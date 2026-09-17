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

func TestDiffer_SequenceRefsEqual(t *testing.T) {
	d := NewDiffer(nil)

	tests := []struct {
		name string
		a    *schema.SequenceRef
		b    *schema.SequenceRef
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "first nil second not nil",
			a:    nil,
			b:    &schema.SequenceRef{Name: "seq1", Owned: false},
			want: false,
		},
		{
			name: "first not nil second nil",
			a:    &schema.SequenceRef{Name: "seq1", Owned: false},
			b:    nil,
			want: false,
		},
		{
			name: "same sequence unowned",
			a:    &schema.SequenceRef{Name: "seq1", Owned: false},
			b:    &schema.SequenceRef{Name: "seq1", Owned: false},
			want: true,
		},
		{
			name: "same sequence owned",
			a:    &schema.SequenceRef{Name: "seq1", Owned: true},
			b:    &schema.SequenceRef{Name: "seq1", Owned: true},
			want: true,
		},
		{
			name: "different sequence names",
			a:    &schema.SequenceRef{Name: "seq1", Owned: false},
			b:    &schema.SequenceRef{Name: "seq2", Owned: false},
			want: false,
		},
		{
			name: "same name different owned",
			a:    &schema.SequenceRef{Name: "seq1", Owned: true},
			b:    &schema.SequenceRef{Name: "seq1", Owned: false},
			want: false,
		},
		{
			name: "empty sequence names",
			a:    &schema.SequenceRef{Name: "", Owned: false},
			b:    &schema.SequenceRef{Name: "", Owned: false},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.sequenceRefsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("sequenceRefsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDiffer_NewTableWithIndexes is a regression test for Issue #21.
// Ensures that when a new table is created with indexes, separate CreateIndex
// operations are generated (not just bundled in CreateTable).
func TestDiffer_NewTableWithIndexes(t *testing.T) {
	tests := []struct {
		name      string
		before    *schema.Schema
		after     *schema.Schema
		wantOps   int
		checkOps  func(*testing.T, []Operation)
	}{
		{
			name: "new table with single index",
			before: &schema.Schema{
				Tables: []schema.Table{},
			},
			after: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "users",
						Columns: []schema.Column{
							{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}},
							{Name: "email", Type: schema.DataType{Kind: schema.TypeText}},
						},
						Indexes: []schema.Index{
							{
								Name:    "idx_users_email",
								Columns: []schema.IndexColumn{{Name: "email"}},
							},
						},
					},
				},
			},
			wantOps: 2, // CreateTable + CreateIndex
			checkOps: func(t *testing.T, ops []Operation) {
				if len(ops) != 2 {
					t.Fatalf("expected 2 operations, got %d", len(ops))
				}

				// First operation: CreateTable
				if ops[0].Kind() != OpCreateTable {
					t.Errorf("first operation should be CreateTable, got %v", ops[0].Kind())
				}

				// Second operation: CreateIndex
				if ops[1].Kind() != OpCreateIndex {
					t.Errorf("second operation should be CreateIndex, got %v", ops[1].Kind())
				}

				createIndex, ok := ops[1].(CreateIndex)
				if !ok {
					t.Fatal("second operation is not CreateIndex type")
				}

				if createIndex.Table != "users" {
					t.Errorf("CreateIndex.Table = %q, want %q", createIndex.Table, "users")
				}

				if createIndex.Index.Name != "idx_users_email" {
					t.Errorf("CreateIndex.Index.Name = %q, want %q", createIndex.Index.Name, "idx_users_email")
				}
			},
		},
		{
			name: "new table with multiple indexes",
			before: &schema.Schema{
				Tables: []schema.Table{},
			},
			after: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "orders",
						Columns: []schema.Column{
							{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}},
							{Name: "user_id", Type: schema.DataType{Kind: schema.TypeUUID}},
							{Name: "status", Type: schema.DataType{Kind: schema.TypeText}},
							{Name: "created_at", Type: schema.DataType{Kind: schema.TypeTimestamp}},
						},
						Indexes: []schema.Index{
							{
								Name:    "idx_orders_user_id",
								Columns: []schema.IndexColumn{{Name: "user_id"}},
							},
							{
								Name: "idx_orders_status_created",
								Columns: []schema.IndexColumn{
									{Name: "status"},
									{Name: "created_at"},
								},
							},
						},
					},
				},
			},
			wantOps: 3, // CreateTable + 2x CreateIndex
			checkOps: func(t *testing.T, ops []Operation) {
				if len(ops) != 3 {
					t.Fatalf("expected 3 operations, got %d", len(ops))
				}

				// First operation: CreateTable
				if ops[0].Kind() != OpCreateTable {
					t.Errorf("first operation should be CreateTable, got %v", ops[0].Kind())
				}

				// Second and third: CreateIndex
				for i := 1; i <= 2; i++ {
					if ops[i].Kind() != OpCreateIndex {
						t.Errorf("operation %d should be CreateIndex, got %v", i, ops[i].Kind())
					}
				}
			},
		},
		{
			name: "new table with composite index",
			before: &schema.Schema{
				Tables: []schema.Table{},
			},
			after: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "user_roles",
						Columns: []schema.Column{
							{Name: "user_id", Type: schema.DataType{Kind: schema.TypeText}},
							{Name: "role_id", Type: schema.DataType{Kind: schema.TypeText}},
							{Name: "granted_at", Type: schema.DataType{Kind: schema.TypeTimestamp}},
						},
						PrimaryKey: &schema.PrimaryKey{
							Columns: []string{"user_id", "role_id"},
						},
						Indexes: []schema.Index{
							{
								Name: "idx_role_granted",
								Columns: []schema.IndexColumn{
									{Name: "role_id"},
									{Name: "granted_at"},
								},
							},
						},
					},
				},
			},
			wantOps: 2, // CreateTable + CreateIndex
			checkOps: func(t *testing.T, ops []Operation) {
				createIndex, ok := ops[1].(CreateIndex)
				if !ok {
					t.Fatal("second operation is not CreateIndex type")
				}

				if len(createIndex.Index.Columns) != 2 {
					t.Errorf("index should have 2 columns, got %d", len(createIndex.Index.Columns))
				}

				if createIndex.Index.Columns[0].Name != "role_id" {
					t.Errorf("first column = %q, want %q", createIndex.Index.Columns[0].Name, "role_id")
				}

				if createIndex.Index.Columns[1].Name != "granted_at" {
					t.Errorf("second column = %q, want %q", createIndex.Index.Columns[1].Name, "granted_at")
				}
			},
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
				for i, op := range ops {
					t.Logf("  op[%d]: %v", i, op.Kind())
				}
			}

			if tt.checkOps != nil {
				tt.checkOps(t, ops)
			}
		})
	}
}
