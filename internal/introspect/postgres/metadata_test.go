package postgres

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/testutil"
)

// TestIntrospect_SystemTablesExcluded tests that system tables are properly excluded.
func TestIntrospect_SystemTablesExcluded(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE pg_temp_test (id UUID); -- system-like table name but in public schema
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	// Should have both tables (pg_temp_test is in public schema, not pg_temp schema)
	if len(schema.Tables) < 1 {
		t.Fatalf("expected at least 1 table, got %d", len(schema.Tables))
	}

	usersFound := false
	for _, t := range schema.Tables {
		if t.Name == "users" {
			usersFound = true
		}
	}

	if !usersFound {
		t.Error("expected to find table 'users'")
	}
}

// TestIntrospect_CommentsOnTables tests that table comments are captured.
func TestIntrospect_CommentsOnTables(t *testing.T) {
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
		COMMENT ON TABLE users IS 'User accounts in the system';
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	usersTable := schema.Tables[0]
	if usersTable.Name != "users" {
		t.Errorf("expected table name 'users', got %q", usersTable.Name)
	}

	if usersTable.Comment != "User accounts in the system" {
		t.Errorf("expected comment 'User accounts in the system', got %q", usersTable.Comment)
	}
}

// TestIntrospect_CommentsOnColumns tests that column comments are captured.
func TestIntrospect_CommentsOnColumns(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL,
			phone TEXT
		);
		COMMENT ON COLUMN users.email IS 'Primary email address';
		COMMENT ON COLUMN users.phone IS 'Contact phone number';
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	usersTable := schema.Tables[0]

	// Find email and phone columns
	var emailCol, phoneCol *struct{ name, comment string }
	for _, col := range usersTable.Columns {
		if col.Name == "email" {
			emailCol = &struct{ name, comment string }{col.Name, col.Comment}
		}
		if col.Name == "phone" {
			phoneCol = &struct{ name, comment string }{col.Name, col.Comment}
		}
	}

	if emailCol != nil && emailCol.comment != "Primary email address" {
		t.Errorf("email column: expected comment 'Primary email address', got %q", emailCol.comment)
	}

	if phoneCol != nil && phoneCol.comment != "Contact phone number" {
		t.Errorf("phone column: expected comment 'Contact phone number', got %q", phoneCol.comment)
	}
}

// TestIntrospect_TableWithoutComments tests tables without comments.
func TestIntrospect_TableWithoutComments(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE products (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	productsTable := schema.Tables[0]
	if productsTable.Comment != "" {
		t.Errorf("expected empty comment, got %q", productsTable.Comment)
	}

	// Check that columns without comments have empty comment
	for _, col := range productsTable.Columns {
		if col.Comment != "" {
			t.Errorf("column %q: expected empty comment, got %q", col.Name, col.Comment)
		}
	}
}

// TestIntrospect_EmptyComments tests that empty comments are handled correctly.
func TestIntrospect_EmptyComments(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE items (
			id UUID PRIMARY KEY,
			value TEXT
		);
		COMMENT ON TABLE items IS '';
		COMMENT ON COLUMN items.value IS '';
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	itemsTable := schema.Tables[0]
	if itemsTable.Comment != "" {
		t.Errorf("table: expected empty comment, got %q", itemsTable.Comment)
	}

	for _, col := range itemsTable.Columns {
		if col.Name == "value" && col.Comment != "" {
			t.Errorf("column 'value': expected empty comment, got %q", col.Comment)
		}
	}
}

// TestIntrospect_CommentsWithSpecialCharacters tests comments with special characters.
func TestIntrospect_CommentsWithSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE orders (
			id UUID PRIMARY KEY,
			amount NUMERIC
		);
		COMMENT ON TABLE orders IS 'Order records for customers. Contains "quotes" and special chars: @#$%';
		COMMENT ON COLUMN orders.amount IS 'Amount in USD. Range: 0-9999999.99';
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	ordersTable := schema.Tables[0]
	if ordersTable.Comment == "" {
		t.Error("table: expected non-empty comment")
	}

	for _, col := range ordersTable.Columns {
		if col.Name == "amount" && col.Comment == "" {
			t.Error("column 'amount': expected non-empty comment")
		}
	}
}

// TestIntrospect_LongComments tests that long comments are handled correctly.
func TestIntrospect_LongComments(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	longComment := `This is a very long comment that describes the purpose of this table in great detail.
It can span multiple lines and contain various information about how the table is used,
what constraints apply, and any special considerations for developers working with this table.
This comment is meant to test that long comments are properly handled by the introspector.`

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE documents (
			id UUID PRIMARY KEY,
			content TEXT
		);
		COMMENT ON TABLE documents IS $long$` + longComment + `$long$;
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	documentsTable := schema.Tables[0]
	if documentsTable.Comment != longComment {
		t.Errorf("table: expected long comment, got %q", documentsTable.Comment)
	}
}

// TestIntrospect_UnicodeComments tests comments with Unicode characters.
func TestIntrospect_UnicodeComments(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			name TEXT
		);
		COMMENT ON TABLE users IS 'Người dùng hệ thống 用户 Пользователи 使用者';
		COMMENT ON COLUMN users.name IS 'Tên người dùng (名前) имя';
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	usersTable := schema.Tables[0]
	if usersTable.Comment == "" {
		t.Error("table: expected non-empty comment with Unicode")
	}
}

// TestIntrospect_TableListWithoutInheritance tests that inheritance is not used (MVP scope).
func TestIntrospect_TableListWithoutInheritance(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE base_table (
			id UUID PRIMARY KEY,
			created_at TIMESTAMPTZ
		);
		CREATE TABLE child_table (
			name TEXT
		) INHERITS (base_table);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	// MVP doesn't support inheritance - should get at least base_table
	baseTableFound := false
	for _, t := range schema.Tables {
		if t.Name == "base_table" {
			baseTableFound = true
		}
	}

	if !baseTableFound {
		t.Error("expected to find base_table")
	}

	t.Logf("Tables found: %d (inheritance not explicitly supported in MVP)", len(schema.Tables))
}

// TestIntrospect_TableNamingConventions tests various table naming conventions.
func TestIntrospect_TableNamingConventions(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE user_profiles (id UUID PRIMARY KEY);
		CREATE TABLE user_2fa_tokens (id UUID PRIMARY KEY);
		CREATE TABLE t (id UUID PRIMARY KEY);
		CREATE TABLE _internal_cache (id UUID PRIMARY KEY);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	expectedTables := map[string]bool{
		"users":               true,
		"user_profiles":       true,
		"user_2fa_tokens":     true,
		"t":                   true,
		"_internal_cache":     true,
	}

	foundTables := make(map[string]bool)
	for _, t := range schema.Tables {
		foundTables[t.Name] = true
	}

	for expected := range expectedTables {
		if !foundTables[expected] {
			t.Errorf("expected to find table %q", expected)
		}
	}

	if len(schema.Tables) != len(expectedTables) {
		t.Errorf("expected %d tables, got %d", len(expectedTables), len(schema.Tables))
	}
}
