package postgres

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/testutil"
)

// TestIntrospect_ExpressionIndex tests indexes on expressions.
func TestIntrospect_ExpressionIndex(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL
		);

		CREATE INDEX idx_users_email_lower ON users (lower(email));
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	indexes, err := inspector.IntrospectIndexes(ctx, "public", "users")
	if err != nil {
		t.Fatalf("introspect indexes: %v", err)
	}

	// Should detect the expression index
	found := false
	for _, idx := range indexes {
		if idx.Name == "idx_users_email_lower" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find expression index idx_users_email_lower")
	}
}

// TestIntrospect_PartialIndexComplexPredicate tests partial indexes with complex WHERE clauses.
func TestIntrospect_PartialIndexComplexPredicate(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE orders (
			id UUID PRIMARY KEY,
			status TEXT,
			amount NUMERIC,
			created_at TIMESTAMPTZ
		);

		CREATE INDEX idx_active_orders ON orders (created_at)
		WHERE status = 'active' AND amount > 100;
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	indexes, err := inspector.IntrospectIndexes(ctx, "public", "orders")
	if err != nil {
		t.Fatalf("introspect indexes: %v", err)
	}

	if len(indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(indexes))
	}

	idx := indexes[0]
	if idx.Name != "idx_active_orders" {
		t.Errorf("expected index name 'idx_active_orders', got %q", idx.Name)
	}

	if idx.Where == "" {
		t.Error("expected WHERE clause, got empty")
	}

	// Verify predicate contains both conditions
	if idx.Where != "" {
		if !containsAny(idx.Where, "status") || !containsAny(idx.Where, "amount") {
			t.Errorf("expected WHERE clause to contain 'status' and 'amount', got: %q", idx.Where)
		}
	}
}

// TestIntrospect_UniqueIndex tests unique indexes.
func TestIntrospect_UniqueIndex(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL,
			username TEXT NOT NULL
		);

		CREATE UNIQUE INDEX idx_users_email ON users (email);
		CREATE INDEX idx_users_username ON users (username);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	indexes, err := inspector.IntrospectIndexes(ctx, "public", "users")
	if err != nil {
		t.Fatalf("introspect indexes: %v", err)
	}

	if len(indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(indexes))
	}

	uniqueIdx := indexes[0]
	if uniqueIdx.Name != "idx_users_email" {
		t.Errorf("expected first index 'idx_users_email', got %q", uniqueIdx.Name)
	}
	if !uniqueIdx.Unique {
		t.Error("expected unique index to have Unique=true")
	}

	nonUniqueIdx := indexes[1]
	if nonUniqueIdx.Name != "idx_users_username" {
		t.Errorf("expected second index 'idx_users_username', got %q", nonUniqueIdx.Name)
	}
	if nonUniqueIdx.Unique {
		t.Error("expected non-unique index to have Unique=false")
	}
}

// TestIntrospect_MultiColumnIndex tests indexes on multiple columns.
func TestIntrospect_MultiColumnIndex(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE order_items (
			order_id UUID,
			line_num INTEGER,
			product_id UUID,
			quantity INTEGER,
			PRIMARY KEY (order_id, line_num)
		);

		CREATE INDEX idx_product_qty ON order_items (product_id, quantity);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	indexes, err := inspector.IntrospectIndexes(ctx, "public", "order_items")
	if err != nil {
		t.Fatalf("introspect indexes: %v", err)
	}

	if len(indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(indexes))
	}

	idx := indexes[0]
	if idx.Name != "idx_product_qty" {
		t.Errorf("expected index name 'idx_product_qty', got %q", idx.Name)
	}

	if len(idx.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(idx.Columns))
	}

	if idx.Columns[0].Name != "product_id" {
		t.Errorf("expected first column 'product_id', got %q", idx.Columns[0].Name)
	}
	if idx.Columns[1].Name != "quantity" {
		t.Errorf("expected second column 'quantity', got %q", idx.Columns[1].Name)
	}
}

// TestIntrospect_DifferentIndexTypes tests various index types (BTREE, HASH, GIN, GIST).
func TestIntrospect_DifferentIndexTypes(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE documents (
			id UUID PRIMARY KEY,
			title TEXT NOT NULL,
			tags TEXT[] NOT NULL,
			content TSVECTOR
		);

		CREATE INDEX idx_docs_title ON documents USING btree (title);
		CREATE INDEX idx_docs_tags ON documents USING gin (tags);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	indexes, err := inspector.IntrospectIndexes(ctx, "public", "documents")
	if err != nil {
		t.Fatalf("introspect indexes: %v", err)
	}

	if len(indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(indexes))
	}

	// Find indexes by name (order may vary)
	indexMap := make(map[string]schema.Index)
	for _, idx := range indexes {
		indexMap[idx.Name] = idx
	}

	btreeIdx, ok := indexMap["idx_docs_title"]
	if !ok {
		t.Fatalf("expected index 'idx_docs_title', not found")
	}
	if btreeIdx.Type != schema.IndexBTree {
		t.Errorf("expected BTREE index type, got %v", btreeIdx.Type)
	}

	ginIdx, ok := indexMap["idx_docs_tags"]
	if !ok {
		t.Fatalf("expected index 'idx_docs_tags', not found")
	}
	if ginIdx.Type != schema.IndexGIN {
		t.Errorf("expected GIN index type, got %v", ginIdx.Type)
	}
}

// TestMapIndexType tests index type mapping.
func TestMapIndexType(t *testing.T) {
	tests := []struct {
		pgType string
		want   schema.IndexType
	}{
		{"btree", schema.IndexBTree},
		{"hash", schema.IndexHash},
		{"gin", schema.IndexGIN},
		{"gist", schema.IndexGiST},
		{"spgist", schema.IndexSPGiST},
		{"brin", schema.IndexBRIN},
		{"unknown", schema.IndexBTree}, // Default to BTREE
		{"", schema.IndexBTree},
	}

	for _, tt := range tests {
		result := mapIndexType(tt.pgType)
		if result != tt.want {
			t.Errorf("mapIndexType(%q) = %v, want %v", tt.pgType, result, tt.want)
		}
	}
}

// TestIntrospect_IndexOnMultipleTablesExcludesPK tests that primary keys are excluded.
func TestIntrospect_IndexOnMultipleTablesExcludesPK(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255),
			sku VARCHAR(50)
		);

		CREATE TABLE suppliers (
			id UUID PRIMARY KEY,
			name VARCHAR(255)
		);

		CREATE INDEX idx_products_sku ON products (sku);
		CREATE INDEX idx_suppliers_name ON suppliers (name);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	prodIndexes, err := inspector.IntrospectIndexes(ctx, "public", "products")
	if err != nil {
		t.Fatalf("introspect products indexes: %v", err)
	}

	supplyIndexes, err := inspector.IntrospectIndexes(ctx, "public", "suppliers")
	if err != nil {
		t.Fatalf("introspect suppliers indexes: %v", err)
	}

	// Should only get the secondary indexes, not the PK index
	if len(prodIndexes) != 1 {
		t.Errorf("expected 1 index on products (sku), got %d", len(prodIndexes))
	}

	if len(supplyIndexes) != 1 {
		t.Errorf("expected 1 index on suppliers (name), got %d", len(supplyIndexes))
	}

	if prodIndexes[0].Name != "idx_products_sku" {
		t.Errorf("expected 'idx_products_sku', got %q", prodIndexes[0].Name)
	}

	if supplyIndexes[0].Name != "idx_suppliers_name" {
		t.Errorf("expected 'idx_suppliers_name', got %q", supplyIndexes[0].Name)
	}
}
