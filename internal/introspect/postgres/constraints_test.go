package postgres

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/testutil"
)

// TestIntrospect_DeferrableConstraints tests deferrable foreign key constraints.
func TestIntrospect_DeferrableConstraints(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE posts (
			id UUID PRIMARY KEY,
			user_id UUID
		);
		ALTER TABLE posts ADD CONSTRAINT fk_posts_user
			FOREIGN KEY (user_id) REFERENCES users(id)
			DEFERRABLE INITIALLY DEFERRED;
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	fks, err := inspector.IntrospectForeignKeys(ctx, "public", "posts")
	if err != nil {
		t.Fatalf("introspect foreign keys: %v", err)
	}

	if len(fks) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(fks))
	}

	fk := fks[0]
	if fk.Name != "fk_posts_user" {
		t.Errorf("expected constraint name 'fk_posts_user', got %q", fk.Name)
	}
	if len(fk.Columns) != 1 || fk.Columns[0] != "user_id" {
		t.Errorf("expected columns [user_id], got %v", fk.Columns)
	}
	if fk.RefTable != "users" {
		t.Errorf("expected ref table 'users', got %q", fk.RefTable)
	}
}

// TestIntrospect_CircularForeignKeys tests circular foreign key relationships.
func TestIntrospect_CircularForeignKeys(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			profile_id UUID
		);

		CREATE TABLE profiles (
			id UUID PRIMARY KEY,
			user_id UUID
		);

		ALTER TABLE users ADD CONSTRAINT fk_users_profile
			FOREIGN KEY (profile_id) REFERENCES profiles(id)
			DEFERRABLE INITIALLY DEFERRED;

		ALTER TABLE profiles ADD CONSTRAINT fk_profiles_user
			FOREIGN KEY (user_id) REFERENCES users(id)
			DEFERRABLE INITIALLY DEFERRED;
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	userFKs, err := inspector.IntrospectForeignKeys(ctx, "public", "users")
	if err != nil {
		t.Fatalf("introspect users foreign keys: %v", err)
	}

	profileFKs, err := inspector.IntrospectForeignKeys(ctx, "public", "profiles")
	if err != nil {
		t.Fatalf("introspect profiles foreign keys: %v", err)
	}

	if len(userFKs) != 1 {
		t.Errorf("expected 1 FK on users, got %d", len(userFKs))
	}
	if len(profileFKs) != 1 {
		t.Errorf("expected 1 FK on profiles, got %d", len(profileFKs))
	}

	// Verify circular relationships are detected
	if userFKs[0].RefTable != "profiles" {
		t.Errorf("expected users.profile_id -> profiles, got %q", userFKs[0].RefTable)
	}
	if profileFKs[0].RefTable != "users" {
		t.Errorf("expected profiles.user_id -> users, got %q", profileFKs[0].RefTable)
	}
}

// TestIntrospect_ComplexCheckConstraints tests multiple complex CHECK constraints.
func TestIntrospect_ComplexCheckConstraints(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE orders (
			id UUID PRIMARY KEY,
			amount NUMERIC(10,2),
			discount NUMERIC(10,2),
			quantity INTEGER,
			CHECK (amount > 0),
			CHECK (discount >= 0 AND discount < amount),
			CHECK (amount - discount > 0),
			CHECK (quantity > 0)
		);
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	constraints, err := inspector.IntrospectCheckConstraints(ctx, "public", "orders")
	if err != nil {
		t.Fatalf("introspect check constraints: %v", err)
	}

	if len(constraints) < 3 {
		t.Errorf("expected at least 3 CHECK constraints, got %d", len(constraints))
	}

	for _, constraint := range constraints {
		if constraint.Type != schema.ConstraintCheck {
			t.Errorf("expected ConstraintCheck, got %v", constraint.Type)
		}
		if constraint.Name == "" {
			t.Error("expected constraint name, got empty")
		}
		if constraint.Expression == "" {
			t.Error("expected constraint expression, got empty")
		}
	}
}

// TestIntrospect_MultiColumnForeignKey tests composite foreign keys.
func TestIntrospect_MultiColumnForeignKey(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE order_lines (
			order_id UUID,
			line_num INTEGER,
			product_id UUID,
			PRIMARY KEY (order_id, line_num)
		);

		CREATE TABLE order_line_audit (
			order_id UUID,
			line_num INTEGER,
			audit_info TEXT,
			PRIMARY KEY (order_id, line_num),
			FOREIGN KEY (order_id, line_num) REFERENCES order_lines(order_id, line_num)
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

	fks, err := inspector.IntrospectForeignKeys(ctx, "public", "order_line_audit")
	if err != nil {
		t.Fatalf("introspect foreign keys: %v", err)
	}

	if len(fks) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(fks))
	}

	fk := fks[0]
	if len(fk.Columns) != 2 {
		t.Errorf("expected 2 columns in FK, got %d", len(fk.Columns))
	}
	if len(fk.RefColumns) != 2 {
		t.Errorf("expected 2 ref columns in FK, got %d", len(fk.RefColumns))
	}

	// Verify column order is preserved
	if fk.Columns[0] != "order_id" || fk.Columns[1] != "line_num" {
		t.Errorf("expected columns [order_id, line_num], got %v", fk.Columns)
	}
	if fk.RefColumns[0] != "order_id" || fk.RefColumns[1] != "line_num" {
		t.Errorf("expected ref columns [order_id, line_num], got %v", fk.RefColumns)
	}
}

// TestIntrospect_CascadeActions tests various referential actions on foreign keys.
func TestIntrospect_CascadeActions(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TABLE categories (id UUID PRIMARY KEY);

		CREATE TABLE products (
			id UUID PRIMARY KEY,
			category_id UUID,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE ON UPDATE CASCADE
		);

		CREATE TABLE cart_items (
			id UUID PRIMARY KEY,
			product_id UUID,
			FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL ON UPDATE RESTRICT
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

	productFKs, err := inspector.IntrospectForeignKeys(ctx, "public", "products")
	if err != nil {
		t.Fatalf("introspect products foreign keys: %v", err)
	}

	cartFKs, err := inspector.IntrospectForeignKeys(ctx, "public", "cart_items")
	if err != nil {
		t.Fatalf("introspect cart_items foreign keys: %v", err)
	}

	if len(productFKs) != 1 {
		t.Fatalf("expected 1 FK on products, got %d", len(productFKs))
	}
	if len(cartFKs) != 1 {
		t.Fatalf("expected 1 FK on cart_items, got %d", len(cartFKs))
	}

	// Check CASCADE actions
	if productFKs[0].OnDelete != schema.ActionCascade {
		t.Errorf("products: expected OnDelete CASCADE, got %v", productFKs[0].OnDelete)
	}
	if productFKs[0].OnUpdate != schema.ActionCascade {
		t.Errorf("products: expected OnUpdate CASCADE, got %v", productFKs[0].OnUpdate)
	}

	// Check SET NULL and RESTRICT actions
	if cartFKs[0].OnDelete != schema.ActionSetNull {
		t.Errorf("cart_items: expected OnDelete SET NULL, got %v", cartFKs[0].OnDelete)
	}
	if cartFKs[0].OnUpdate != schema.ActionRestrict {
		t.Errorf("cart_items: expected OnUpdate RESTRICT, got %v", cartFKs[0].OnUpdate)
	}
}

// TestMapReferentialAction tests all referential action mappings.
func TestMapReferentialAction(t *testing.T) {
	tests := []struct {
		pgCode string
		want   schema.ReferentialAction
	}{
		{"a", schema.ActionNoAction},
		{"r", schema.ActionRestrict},
		{"c", schema.ActionCascade},
		{"n", schema.ActionSetNull},
		{"d", schema.ActionSetDefault},
		{"x", schema.ActionNone},
		{"", schema.ActionNone},
	}

	for _, tt := range tests {
		result := mapReferentialAction(tt.pgCode)
		if result != tt.want {
			t.Errorf("mapReferentialAction(%q) = %v, want %v", tt.pgCode, result, tt.want)
		}
	}
}
