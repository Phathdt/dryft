package postgres

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/testutil"
)

func TestIntrospectColumns_BasicTypes(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL,
			age INTEGER,
			balance NUMERIC(10,2),
			bio VARCHAR(500),
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	columns, err := inspector.IntrospectColumns(ctx, "public", "users")
	if err != nil {
		t.Fatalf("introspect columns: %v", err)
	}

	if len(columns) != 7 {
		t.Fatalf("expected 7 columns, got %d", len(columns))
	}

	tests := []struct {
		name       string
		wantType   schema.TypeKind
		wantNull   bool
		hasDefault bool
	}{
		{"id", schema.TypeUUID, false, false},
		{"email", schema.TypeText, false, false},
		{"age", schema.TypeInt32, true, false},
		{"balance", schema.TypeNumeric, true, false},
		{"bio", schema.TypeVarChar, true, false},
		{"active", schema.TypeBool, true, true},
		{"created_at", schema.TypeTimestampTZ, true, true},
	}

	for i, tt := range tests {
		col := columns[i]
		if col.Name != tt.name {
			t.Errorf("column[%d]: expected name %q, got %q", i, tt.name, col.Name)
		}
		if col.Type.Kind != tt.wantType {
			t.Errorf("column %q: expected type %v, got %v", tt.name, tt.wantType, col.Type.Kind)
		}
		if col.Nullable != tt.wantNull {
			t.Errorf("column %q: expected nullable=%v, got %v", tt.name, tt.wantNull, col.Nullable)
		}
		if (col.Default != nil) != tt.hasDefault {
			t.Errorf("column %q: expected hasDefault=%v, got %v", tt.name, tt.hasDefault, col.Default != nil)
		}
	}
}

func TestIntrospectColumns_TypeModifiers(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE products (
			code VARCHAR(100),
			price NUMERIC(10,2),
			name CHAR(50)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	columns, err := inspector.IntrospectColumns(ctx, "public", "products")
	if err != nil {
		t.Fatalf("introspect columns: %v", err)
	}

	tests := []struct {
		name      string
		precision int
		scale     int
	}{
		{"code", 100, 0},
		{"price", 10, 2},
		{"name", 50, 0},
	}

	for i, tt := range tests {
		col := columns[i]
		if col.Type.Precision != tt.precision {
			t.Errorf("column %q: expected precision=%d, got %d", tt.name, tt.precision, col.Type.Precision)
		}
		if col.Type.Scale != tt.scale {
			t.Errorf("column %q: expected scale=%d, got %d", tt.name, tt.scale, col.Type.Scale)
		}
	}
}

func TestIntrospectColumns_ArrayTypes(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			tags TEXT[],
			ratings INTEGER[]
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	columns, err := inspector.IntrospectColumns(ctx, "public", "posts")
	if err != nil {
		t.Fatalf("introspect columns: %v", err)
	}

	if columns[1].Name != "tags" {
		t.Errorf("expected column[1] to be tags, got %q", columns[1].Name)
	}
	if columns[1].Type.Kind != schema.TypeText {
		t.Errorf("tags: expected TypeText, got %v", columns[1].Type.Kind)
	}
	if columns[1].Type.ArrayDepth != 1 {
		t.Errorf("tags: expected array depth 1, got %d", columns[1].Type.ArrayDepth)
	}

	if columns[2].Name != "ratings" {
		t.Errorf("expected column[2] to be ratings, got %q", columns[2].Name)
	}
	if columns[2].Type.Kind != schema.TypeInt32 {
		t.Errorf("ratings: expected TypeInt32, got %v", columns[2].Type.Kind)
	}
	if columns[2].Type.ArrayDepth != 1 {
		t.Errorf("ratings: expected array depth 1, got %d", columns[2].Type.ArrayDepth)
	}
}

func TestIntrospectColumns_SerialDetection(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			order_num BIGSERIAL
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	columns, err := inspector.IntrospectColumns(ctx, "public", "orders")
	if err != nil {
		t.Fatalf("introspect columns: %v", err)
	}

	if columns[0].Name != "id" {
		t.Errorf("expected column[0] to be id, got %q", columns[0].Name)
	}
	if columns[0].Default == nil {
		t.Fatal("id: expected default value")
	}
	if columns[0].Default.Kind != schema.DefaultSequence {
		t.Errorf("id: expected DefaultSequence, got %v", columns[0].Default.Kind)
	}
	if columns[0].Default.Sequence == nil {
		t.Fatal("id: expected sequence reference")
	}
	if !columns[0].Default.Sequence.Owned {
		t.Error("id: expected owned sequence (SERIAL)")
	}
}

func TestIntrospectColumns_IdentityDetection(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE invoices (
			id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			invoice_num BIGINT GENERATED BY DEFAULT AS IDENTITY
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	columns, err := inspector.IntrospectColumns(ctx, "public", "invoices")
	if err != nil {
		t.Fatalf("introspect columns: %v", err)
	}

	if columns[0].Name != "id" {
		t.Errorf("expected column[0] to be id, got %q", columns[0].Name)
	}
	if columns[0].Default == nil {
		t.Fatal("id: expected default value")
	}
	if columns[0].Default.Kind != schema.DefaultSequence {
		t.Errorf("id: expected DefaultSequence, got %v", columns[0].Default.Kind)
	}
	if columns[0].Default.Sequence == nil {
		t.Fatal("id: expected sequence reference")
	}
	if !columns[0].Default.Sequence.Owned {
		t.Error("id: expected owned sequence (IDENTITY)")
	}
}

func TestIntrospectColumns_DefaultExpressions(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE TABLE events (
			id UUID DEFAULT uuid_generate_v4(),
			name TEXT DEFAULT 'unnamed',
			counter INTEGER DEFAULT 0,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	columns, err := inspector.IntrospectColumns(ctx, "public", "events")
	if err != nil {
		t.Fatalf("introspect columns: %v", err)
	}

	tests := []struct {
		name        string
		defaultKind schema.DefaultKind
	}{
		{"id", schema.DefaultExpression},
		{"name", schema.DefaultLiteral},
		{"counter", schema.DefaultLiteral},
		{"active", schema.DefaultLiteral},
		{"created_at", schema.DefaultExpression},
	}

	for i, tt := range tests {
		col := columns[i]
		if col.Name != tt.name {
			t.Errorf("column[%d]: expected name %q, got %q", i, tt.name, col.Name)
		}
		if col.Default == nil {
			t.Errorf("column %q: expected default value", tt.name)
			continue
		}
		if col.Default.Kind != tt.defaultKind {
			t.Errorf("column %q: expected default kind %v, got %v", tt.name, tt.defaultKind, col.Default.Kind)
		}
	}
}

func TestIntrospectColumns_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE locations (
			id INTEGER,
			data XML
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	_, err = inspector.IntrospectColumns(ctx, "public", "locations")
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}

	if !containsAny(err.Error(), "unsupported PostgreSQL type") {
		t.Errorf("expected error message to contain 'unsupported PostgreSQL type', got: %v", err)
	}
}

func TestIntrospectColumns_Comments(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT
		);
		COMMENT ON COLUMN users.email IS 'User email address';
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	columns, err := inspector.IntrospectColumns(ctx, "public", "users")
	if err != nil {
		t.Fatalf("introspect columns: %v", err)
	}

	if columns[0].Comment != "" {
		t.Errorf("id: expected empty comment, got %q", columns[0].Comment)
	}

	if columns[1].Comment != "User email address" {
		t.Errorf("email: expected comment 'User email address', got %q", columns[1].Comment)
	}
}

func TestIntrospectTable(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			price NUMERIC(10,2)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	table, err := inspector.IntrospectTable(ctx, "products")
	if err != nil {
		t.Fatalf("introspect table: %v", err)
	}

	if table.Name != "products" {
		t.Errorf("expected table name 'products', got %q", table.Name)
	}

	if len(table.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(table.Columns))
	}

	if table.Columns[0].Name != "id" {
		t.Errorf("expected first column 'id', got %q", table.Columns[0].Name)
	}
	if table.Columns[1].Name != "name" {
		t.Errorf("expected second column 'name', got %q", table.Columns[1].Name)
	}
	if table.Columns[2].Name != "price" {
		t.Errorf("expected third column 'price', got %q", table.Columns[2].Name)
	}
}
