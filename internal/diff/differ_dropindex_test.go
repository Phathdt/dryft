package diff

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestDiffer_DropIndexWithoutName(t *testing.T) {
	tests := []struct {
		name     string
		before   *schema.Schema
		after    *schema.Schema
		wantOps  int
		checkOps func(t *testing.T, ops []Operation)
	}{
		{
			name: "drop unnamed index - Index object should be populated",
			before: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "users",
						Columns: []schema.Column{
							{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}},
							{Name: "email", Type: schema.DataType{Kind: schema.TypeText}},
						},
						Indexes: []schema.Index{
							{
								Name: "", // Unnamed index (will be auto-generated)
								Columns: []schema.IndexColumn{
									{Name: "email"},
								},
							},
						},
					},
				},
			},
			after: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "users",
						Columns: []schema.Column{
							{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}},
							{Name: "email", Type: schema.DataType{Kind: schema.TypeText}},
						},
						Indexes: []schema.Index{}, // Index removed
					},
				},
			},
			wantOps: 1, // DropIndex
			checkOps: func(t *testing.T, ops []Operation) {
				dropIndex, ok := ops[0].(DropIndex)
				if !ok {
					t.Fatal("operation is not DropIndex type")
				}

				if dropIndex.Name != "" {
					t.Errorf("DropIndex.Name = %q, want empty", dropIndex.Name)
				}

				if len(dropIndex.Index.Columns) != 1 {
					t.Fatalf("DropIndex.Index should have 1 column, got %d", len(dropIndex.Index.Columns))
				}

				if dropIndex.Index.Columns[0].Name != "email" {
					t.Errorf("DropIndex.Index.Columns[0].Name = %q, want %q", dropIndex.Index.Columns[0].Name, "email")
				}

				if dropIndex.Table != "users" {
					t.Errorf("DropIndex.Table = %q, want %q", dropIndex.Table, "users")
				}
			},
		},
		{
			name: "drop named index - Name should be preserved",
			before: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "orders",
						Columns: []schema.Column{
							{Name: "id", Type: schema.DataType{Kind: schema.TypeText}},
							{Name: "status", Type: schema.DataType{Kind: schema.TypeText}},
						},
						Indexes: []schema.Index{
							{
								Name: "idx_status_custom",
								Columns: []schema.IndexColumn{
									{Name: "status"},
								},
							},
						},
					},
				},
			},
			after: &schema.Schema{
				Tables: []schema.Table{
					{
						Name: "orders",
						Columns: []schema.Column{
							{Name: "id", Type: schema.DataType{Kind: schema.TypeText}},
							{Name: "status", Type: schema.DataType{Kind: schema.TypeText}},
						},
						Indexes: []schema.Index{}, // Index removed
					},
				},
			},
			wantOps: 1, // DropIndex
			checkOps: func(t *testing.T, ops []Operation) {
				dropIndex, ok := ops[0].(DropIndex)
				if !ok {
					t.Fatal("operation is not DropIndex type")
				}

				if dropIndex.Name != "idx_status_custom" {
					t.Errorf("DropIndex.Name = %q, want %q", dropIndex.Name, "idx_status_custom")
				}

				if len(dropIndex.Index.Columns) != 1 {
					t.Fatalf("DropIndex.Index should have 1 column, got %d", len(dropIndex.Index.Columns))
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
