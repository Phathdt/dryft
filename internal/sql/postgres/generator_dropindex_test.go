package postgres

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
)

func TestGenerator_DropIndexWithoutName(t *testing.T) {
	tests := []struct {
		name    string
		op      diff.DropIndex
		wantSQL string
	}{
		{
			name: "drop unnamed index - should auto-generate name",
			op: diff.DropIndex{
				Table: "users",
				Name:  "", // Empty name
				Index: schema.Index{
					Columns: []schema.IndexColumn{
						{Name: "email"},
					},
				},
			},
			wantSQL: "DROP INDEX idx_users_email;",
		},
		{
			name: "drop unnamed composite index",
			op: diff.DropIndex{
				Table: "orders",
				Name:  "", // Empty name
				Index: schema.Index{
					Columns: []schema.IndexColumn{
						{Name: "user_id"},
						{Name: "created_at"},
					},
				},
			},
			wantSQL: "DROP INDEX idx_orders_user_id_created_at;",
		},
		{
			name: "drop named index - should use provided name",
			op: diff.DropIndex{
				Table: "products",
				Name:  "idx_custom_name",
				Index: schema.Index{
					Columns: []schema.IndexColumn{
						{Name: "sku"},
					},
				},
			},
			wantSQL: "DROP INDEX idx_custom_name;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator(sql.GeneratorOptions{})
			gotSQL := g.generateDropIndex(tt.op)

			if gotSQL != tt.wantSQL {
				t.Errorf("generateDropIndex() =\n%s\nwant:\n%s", gotSQL, tt.wantSQL)
			}
		})
	}
}

func TestGenerator_ReverseDropIndex(t *testing.T) {
	tests := []struct {
		name       string
		op         diff.DropIndex
		wantSQL    string
		wantUnique bool
	}{
		{
			name: "reverse drop unnamed index - should generate CREATE INDEX",
			op: diff.DropIndex{
				Table: "users",
				Name:  "",
				Index: schema.Index{
					Columns: []schema.IndexColumn{
						{Name: "email"},
					},
					Unique: false,
				},
			},
			wantSQL:    "CREATE INDEX idx_users_email ON users (email);",
			wantUnique: false,
		},
		{
			name: "reverse drop unnamed unique index",
			op: diff.DropIndex{
				Table: "products",
				Name:  "",
				Index: schema.Index{
					Columns: []schema.IndexColumn{
						{Name: "sku"},
					},
					Unique: true,
				},
			},
			wantSQL:    "CREATE UNIQUE INDEX idx_products_sku ON products (sku);",
			wantUnique: true,
		},
		{
			name: "reverse drop named composite index",
			op: diff.DropIndex{
				Table: "orders",
				Name:  "idx_status_date",
				Index: schema.Index{
					Columns: []schema.IndexColumn{
						{Name: "status"},
						{Name: "created_at"},
					},
					Unique: false,
				},
			},
			wantSQL:    "CREATE INDEX idx_status_date ON orders (status, created_at);",
			wantUnique: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator(sql.GeneratorOptions{})
			gotSQL, warning := g.generateReverseOperation(tt.op)

			if warning != "" {
				t.Errorf("unexpected warning: %s", warning)
			}

			if gotSQL != tt.wantSQL {
				t.Errorf("generateReverseOperation() =\n%s\nwant:\n%s", gotSQL, tt.wantSQL)
			}

			// Verify UNIQUE keyword presence
			hasUnique := strings.Contains(gotSQL, "UNIQUE")
			if hasUnique != tt.wantUnique {
				t.Errorf("UNIQUE keyword presence = %v, want %v", hasUnique, tt.wantUnique)
			}
		})
	}
}
