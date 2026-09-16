package tests

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/introspect/postgres"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConstraints_SingleColumnPK verifies single column primary key detection
func TestConstraints_SingleColumnPK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, introspectedSchema.Tables, 1)
	table := introspectedSchema.Tables[0]
	assert.Equal(t, "users", table.Name)

	require.NotNil(t, table.PrimaryKey, "should have primary key")
	assert.Equal(t, []string{"id"}, table.PrimaryKey.Columns)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "@id")
}

// TestConstraints_MultiColumnPK verifies composite primary key detection
func TestConstraints_MultiColumnPK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE user_roles (
			user_id UUID NOT NULL,
			role_id INT NOT NULL,
			assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, role_id)
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, introspectedSchema.Tables, 1)
	table := introspectedSchema.Tables[0]

	require.NotNil(t, table.PrimaryKey, "should have composite primary key")
	assert.Equal(t, []string{"user_id", "role_id"}, table.PrimaryKey.Columns)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "@@id([userId, roleId])")
}

// TestConstraints_FKCascade verifies ON DELETE CASCADE foreign key
func TestConstraints_FKCascade(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE authors (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE books (
			id UUID PRIMARY KEY,
			title TEXT NOT NULL,
			author_id UUID NOT NULL REFERENCES authors(id) ON DELETE CASCADE
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, introspectedSchema.Tables, 2)

	var booksTable *schema.Table
	for i := range introspectedSchema.Tables {
		if introspectedSchema.Tables[i].Name == "books" {
			booksTable = &introspectedSchema.Tables[i]
			break
		}
	}
	require.NotNil(t, booksTable)

	assert.Len(t, booksTable.ForeignKeys, 1)
	fk := booksTable.ForeignKeys[0]
	assert.Equal(t, "authors", fk.RefTable)
	assert.Equal(t, []string{"author_id"}, fk.Columns)
	assert.Equal(t, []string{"id"}, fk.RefColumns)
	assert.Equal(t, schema.ActionCascade, fk.OnDelete)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "@relation")
	assert.Contains(t, prismaText, "onDelete: Cascade")
}

// TestConstraints_FKSetNull verifies ON DELETE SET NULL foreign key
func TestConstraints_FKSetNull(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE departments (
			id INT PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE employees (
			id INT PRIMARY KEY,
			name TEXT NOT NULL,
			department_id INT REFERENCES departments(id) ON DELETE SET NULL
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	var employeesTable *schema.Table
	for i := range introspectedSchema.Tables {
		if introspectedSchema.Tables[i].Name == "employees" {
			employeesTable = &introspectedSchema.Tables[i]
			break
		}
	}
	require.NotNil(t, employeesTable)

	assert.Len(t, employeesTable.ForeignKeys, 1)
	fk := employeesTable.ForeignKeys[0]
	assert.Equal(t, "departments", fk.RefTable)
	assert.Equal(t, schema.ActionSetNull, fk.OnDelete)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "onDelete: SetNull")
}

// TestConstraints_FKRestrict verifies ON DELETE RESTRICT foreign key
func TestConstraints_FKRestrict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE categories (
			id INT PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE products (
			id INT PRIMARY KEY,
			name TEXT NOT NULL,
			category_id INT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	var productsTable *schema.Table
	for i := range introspectedSchema.Tables {
		if introspectedSchema.Tables[i].Name == "products" {
			productsTable = &introspectedSchema.Tables[i]
			break
		}
	}
	require.NotNil(t, productsTable)

	assert.Len(t, productsTable.ForeignKeys, 1)
	fk := productsTable.ForeignKeys[0]
	assert.Equal(t, "categories", fk.RefTable)
	assert.Equal(t, schema.ActionRestrict, fk.OnDelete)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "onDelete: Restrict")
}

// TestConstraints_FKNoAction verifies ON DELETE NO ACTION foreign key
func TestConstraints_FKNoAction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE teams (
			id INT PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE members (
			id INT PRIMARY KEY,
			name TEXT NOT NULL,
			team_id INT NOT NULL REFERENCES teams(id) ON DELETE NO ACTION
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	var membersTable *schema.Table
	for i := range introspectedSchema.Tables {
		if introspectedSchema.Tables[i].Name == "members" {
			membersTable = &introspectedSchema.Tables[i]
			break
		}
	}
	require.NotNil(t, membersTable)

	assert.Len(t, membersTable.ForeignKeys, 1)
	fk := membersTable.ForeignKeys[0]
	assert.Equal(t, "teams", fk.RefTable)
	assert.Equal(t, schema.ActionNoAction, fk.OnDelete)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "onDelete: NoAction")
}

// TestConstraints_CompositeForeignKey verifies multi-column foreign key
func TestConstraints_CompositeForeignKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE orders (
			order_id INT NOT NULL,
			customer_id INT NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			PRIMARY KEY (order_id, customer_id)
		);

		CREATE TABLE order_items (
			id INT PRIMARY KEY,
			order_id INT NOT NULL,
			customer_id INT NOT NULL,
			product_name TEXT NOT NULL,
			FOREIGN KEY (order_id, customer_id) REFERENCES orders(order_id, customer_id)
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	var orderItemsTable *schema.Table
	for i := range introspectedSchema.Tables {
		if introspectedSchema.Tables[i].Name == "order_items" {
			orderItemsTable = &introspectedSchema.Tables[i]
			break
		}
	}
	require.NotNil(t, orderItemsTable)

	assert.Len(t, orderItemsTable.ForeignKeys, 1)
	fk := orderItemsTable.ForeignKeys[0]
	assert.Equal(t, "orders", fk.RefTable)
	assert.Equal(t, []string{"order_id", "customer_id"}, fk.Columns)
	assert.Equal(t, []string{"order_id", "customer_id"}, fk.RefColumns)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "@relation")
}

// TestConstraints_DeferrableFK verifies DEFERRABLE INITIALLY DEFERRED foreign key
// NOTE: Currently the introspector doesn't capture deferrable flags (condeferrable, condeferred)
// This test verifies the FK is detected; deferrable support can be added later
func TestConstraints_DeferrableFK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE nodes (
			id INT PRIMARY KEY,
			parent_id INT,
			name TEXT NOT NULL
		);

		ALTER TABLE nodes
		ADD CONSTRAINT fk_nodes_parent
		FOREIGN KEY (parent_id) REFERENCES nodes(id)
		DEFERRABLE INITIALLY DEFERRED;
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, introspectedSchema.Tables, 1)
	table := introspectedSchema.Tables[0]

	assert.Len(t, table.ForeignKeys, 1)
	fk := table.ForeignKeys[0]
	assert.Equal(t, "nodes", fk.RefTable)
	assert.Equal(t, []string{"parent_id"}, fk.Columns)
	assert.Equal(t, "fk_nodes_parent", fk.Name)

	// TODO: Add assertions for fk.Deferrable and fk.InitiallyDeferred
	// once the introspector is updated to query condeferrable and condeferred

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	_, err = writer.Write(introspectedSchema)
	require.NoError(t, err)
}

// TestConstraints_CheckConstraint verifies CHECK constraint detection
func TestConstraints_CheckConstraint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE users (
			id INT PRIMARY KEY,
			username TEXT NOT NULL,
			age INT NOT NULL CHECK (age >= 18),
			email TEXT NOT NULL CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z]{2,}$')
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, introspectedSchema.Tables, 1)
	table := introspectedSchema.Tables[0]

	var checkConstraints []schema.Constraint
	for _, c := range table.Constraints {
		if c.Type == schema.ConstraintCheck {
			checkConstraints = append(checkConstraints, c)
		}
	}

	assert.GreaterOrEqual(t, len(checkConstraints), 2, "should have at least 2 CHECK constraints")

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	_, err = writer.Write(introspectedSchema)
	require.NoError(t, err)
}

// TestConstraints_UniqueConstraint verifies UNIQUE constraint detection
func TestConstraints_UniqueConstraint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	sql := `
		CREATE TABLE accounts (
			id INT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			phone TEXT,
			country_code TEXT,
			UNIQUE (phone, country_code)
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	introspectedSchema, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, introspectedSchema.Tables, 1)
	table := introspectedSchema.Tables[0]

	var uniqueConstraints []schema.Constraint
	for _, c := range table.Constraints {
		if c.Type == schema.ConstraintUnique {
			uniqueConstraints = append(uniqueConstraints, c)
		}
	}

	assert.GreaterOrEqual(t, len(uniqueConstraints), 1, "should have at least 1 UNIQUE constraint")

	var foundMultiColumn bool
	for _, uc := range uniqueConstraints {
		if len(uc.Columns) == 2 {
			foundMultiColumn = true
			assert.Contains(t, uc.Columns, "phone")
			assert.Contains(t, uc.Columns, "country_code")
		}
	}
	assert.True(t, foundMultiColumn, "should have multi-column unique constraint")

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(introspectedSchema)
	require.NoError(t, err)

	assert.Contains(t, prismaText, "@unique")
	assert.Contains(t, prismaText, "@@unique([")
}
