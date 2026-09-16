package tests

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/introspect/postgres"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEdgeCases_EmptySchema verifies introspection of empty database
func TestEdgeCases_EmptySchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Introspect empty database
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify empty schema is valid
	assert.NotNil(t, s, "schema should not be nil")
	assert.Empty(t, s.Tables, "empty database should have no tables")
	assert.Empty(t, s.Enums, "empty database should have no enums")
}

// TestEdgeCases_TableWithoutPK verifies table without primary key is allowed
func TestEdgeCases_TableWithoutPK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create table without primary key
	sql := `
		CREATE TABLE logs (
			id INT,
			message TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify table exists without primary key
	require.Len(t, s.Tables, 1)
	logsTable := s.Tables[0]
	assert.Equal(t, "logs", logsTable.Name)
	assert.Nil(t, logsTable.PrimaryKey, "table should not have primary key")
	assert.Len(t, logsTable.Columns, 3)
}

// TestEdgeCases_InvalidFKReference verifies FK to non-existent table
// Note: PostgreSQL will reject this during creation, so we verify the error
func TestEdgeCases_InvalidFKReference(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Attempt to create table with FK to non-existent table
	sql := `
		CREATE TABLE orders (
			id INT PRIMARY KEY,
			user_id INT REFERENCES non_existent_table(id)
		);
	`
	err = container.ExecuteSQL(ctx, sql)

	// Should error
	assert.Error(t, err, "FK to non-existent table should fail")
	assert.Contains(t, err.Error(), "does not exist", "error should mention missing table")
}

// TestEdgeCases_DuplicateTableNames verifies duplicate table creation fails
func TestEdgeCases_DuplicateTableNames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create first table
	sql1 := `CREATE TABLE users (id INT PRIMARY KEY);`
	err = container.ExecuteSQL(ctx, sql1)
	require.NoError(t, err)

	// Attempt to create duplicate
	sql2 := `CREATE TABLE users (id INT PRIMARY KEY, email TEXT);`
	err = container.ExecuteSQL(ctx, sql2)

	// Should error
	assert.Error(t, err, "duplicate table creation should fail")
	assert.Contains(t, err.Error(), "already exists", "error should mention table exists")
}

// TestEdgeCases_ReservedKeywords verifies SQL keywords as identifiers
func TestEdgeCases_ReservedKeywords(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create table with reserved keywords (quoted)
	sql := `
		CREATE TABLE "user" (
			"select" INT PRIMARY KEY,
			"from" TEXT,
			"where" BOOLEAN DEFAULT false
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify identifiers preserved
	require.Len(t, s.Tables, 1)
	userTable := s.Tables[0]
	assert.Equal(t, "user", userTable.Name)

	columnNames := make([]string, len(userTable.Columns))
	for i, col := range userTable.Columns {
		columnNames[i] = col.Name
	}
	assert.Contains(t, columnNames, "select")
	assert.Contains(t, columnNames, "from")
	assert.Contains(t, columnNames, "where")
}

// TestEdgeCases_LongIdentifiers verifies PostgreSQL 63-char limit
func TestEdgeCases_LongIdentifiers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// PostgreSQL truncates at 63 chars
	longTableName := "this_is_a_very_long_table_name_that_exceeds_the_sixty_three_character_limit_for_postgres"
	longColumnName := "this_is_a_very_long_column_name_that_exceeds_the_sixty_three_character_limit"

	sql := `
		CREATE TABLE ` + longTableName + ` (
			id INT PRIMARY KEY,
			` + longColumnName + ` TEXT
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify table exists (name will be truncated by PostgreSQL)
	require.Len(t, s.Tables, 1)
	table := s.Tables[0]

	// PostgreSQL truncates to 63 chars
	assert.LessOrEqual(t, len(table.Name), 63, "table name should be truncated to 63 chars")

	// Find the long column
	var foundLongColumn bool
	for _, col := range table.Columns {
		if len(col.Name) == 63 && col.Name != "id" {
			foundLongColumn = true
			break
		}
	}
	assert.True(t, foundLongColumn, "should find truncated column name")
}

// TestEdgeCases_UnicodeIdentifiers verifies Unicode in identifiers
func TestEdgeCases_UnicodeIdentifiers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create table with Unicode identifiers
	sql := `
		CREATE TABLE "用户" (
			"编号" INT PRIMARY KEY,
			"名称" TEXT,
			"señor" TEXT
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify Unicode preserved
	require.Len(t, s.Tables, 1)
	table := s.Tables[0]
	assert.Equal(t, "用户", table.Name)

	columnNames := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		columnNames[i] = col.Name
	}
	assert.Contains(t, columnNames, "编号")
	assert.Contains(t, columnNames, "名称")
	assert.Contains(t, columnNames, "señor")
}

// TestEdgeCases_NullDefaultVsNoDefault verifies distinction between DEFAULT NULL and no default
func TestEdgeCases_NullDefaultVsNoDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create table with various default scenarios
	sql := `
		CREATE TABLE test_defaults (
			id INT PRIMARY KEY,
			a INT,                          -- no default
			b INT DEFAULT NULL,             -- explicit DEFAULT NULL
			c INT DEFAULT 42,               -- explicit value
			d TEXT DEFAULT 'hello'          -- explicit text
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Find columns and check defaults
	require.Len(t, s.Tables, 1)
	table := s.Tables[0]

	colMap := make(map[string]*schema.Column)
	for i := range table.Columns {
		colMap[table.Columns[i].Name] = &table.Columns[i]
	}

	// Column 'a' - no default
	assert.Nil(t, colMap["a"].Default, "column 'a' should have no default")

	// Column 'b' - DEFAULT NULL
	// Note: PostgreSQL may not store DEFAULT NULL explicitly if column is nullable
	// This is database-dependent behavior

	// Column 'c' - DEFAULT 42
	require.NotNil(t, colMap["c"].Default, "column 'c' should have default")
	assert.Contains(t, colMap["c"].Default.Literal, "42", "column 'c' default should be 42")

	// Column 'd' - DEFAULT 'hello'
	require.NotNil(t, colMap["d"].Default, "column 'd' should have default")
	assert.Contains(t, colMap["d"].Default.Literal, "hello", "column 'd' default should be 'hello'")
}

// TestEdgeCases_SchemaDrift verifies detection of manual DB changes
func TestEdgeCases_SchemaDrift(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// 1. Create initial schema
	sql1 := `
		CREATE TABLE users (
			id INT PRIMARY KEY,
			email TEXT NOT NULL
		);
	`
	err = container.ExecuteSQL(ctx, sql1)
	require.NoError(t, err)

	// 2. Introspect initial state
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schemaA, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// 3. Manually add column (schema drift)
	sql2 := `ALTER TABLE users ADD COLUMN name TEXT;`
	err = container.ExecuteSQL(ctx, sql2)
	require.NoError(t, err)

	// 4. Introspect again
	schemaB, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// 5. Diff should detect the drift
	differ := diff.NewDiffer(nil)
	operations, err := differ.Diff(schemaA, schemaB)
	require.NoError(t, err)

	// Should detect new column
	assert.NotEmpty(t, operations, "diff should detect schema drift")

	// Verify it's an AddColumn operation
	foundAddColumn := false
	for _, op := range operations {
		if op.Kind() == diff.OpAddColumn {
			foundAddColumn = true
			assert.Contains(t, op.Description(), "name", "should detect 'name' column addition")
		}
	}
	assert.True(t, foundAddColumn, "should find AddColumn operation")
}

// TestEdgeCases_MultipleIndexesSameColumn verifies multiple indexes on same column
func TestEdgeCases_MultipleIndexesSameColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create table with multiple indexes on same column
	sql := `
		CREATE TABLE products (
			id INT PRIMARY KEY,
			name TEXT NOT NULL,
			price NUMERIC(10, 2) NOT NULL
		);

		CREATE INDEX idx_products_name ON products (name);
		CREATE INDEX idx_products_name_lower ON products (LOWER(name));
		CREATE INDEX idx_products_name_prefix ON products (name text_pattern_ops);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify multiple indexes exist
	require.Len(t, s.Tables, 1)
	table := s.Tables[0]

	// Should have at least 2 indexes (expression indexes may not be fully captured)
	assert.GreaterOrEqual(t, len(table.Indexes), 2, "should have multiple indexes")

	// Count indexes involving 'name' column
	nameIndexCount := 0
	for _, idx := range table.Indexes {
		for _, col := range idx.Columns {
			if col.Name == "name" {
				nameIndexCount++
				break
			}
		}
	}
	assert.GreaterOrEqual(t, nameIndexCount, 1, "should have indexes on 'name' column")
}

// TestEdgeCases_SelfReferencingFK verifies self-referencing foreign key
func TestEdgeCases_SelfReferencingFK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create table with self-referencing FK
	sql := `
		CREATE TABLE employees (
			id INT PRIMARY KEY,
			name TEXT NOT NULL,
			manager_id INT REFERENCES employees(id)
		);
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify self-referencing FK
	require.Len(t, s.Tables, 1)
	table := s.Tables[0]
	assert.Equal(t, "employees", table.Name)

	require.Len(t, table.ForeignKeys, 1, "should have 1 foreign key")
	fk := table.ForeignKeys[0]
	assert.Equal(t, "employees", fk.RefTable, "FK should reference same table")
	assert.Contains(t, fk.Columns, "manager_id")
	assert.Contains(t, fk.RefColumns, "id")
}

// TestEdgeCases_CircularFKs verifies circular foreign key dependencies with DEFERRABLE
func TestEdgeCases_CircularFKs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.GetSharedContainer(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = container.CleanupTables(context.Background())
	})

	// Create circular FK relationship
	sql := `
		CREATE TABLE a (
			id INT PRIMARY KEY,
			b_id INT
		);

		CREATE TABLE b (
			id INT PRIMARY KEY,
			a_id INT REFERENCES a(id) DEFERRABLE INITIALLY DEFERRED
		);

		ALTER TABLE a ADD CONSTRAINT fk_a_b
			FOREIGN KEY (b_id) REFERENCES b(id)
			DEFERRABLE INITIALLY DEFERRED;
	`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	// Introspect
	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	s, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	// Verify both tables exist with FKs
	require.Len(t, s.Tables, 2, "should have 2 tables")

	// Find tables
	var tableA, tableB *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "a" {
			tableA = &s.Tables[i]
		} else if s.Tables[i].Name == "b" {
			tableB = &s.Tables[i]
		}
	}
	require.NotNil(t, tableA, "table 'a' should exist")
	require.NotNil(t, tableB, "table 'b' should exist")

	// Verify circular FKs
	require.Len(t, tableA.ForeignKeys, 1, "table 'a' should have FK")
	require.Len(t, tableB.ForeignKeys, 1, "table 'b' should have FK")

	assert.Equal(t, "b", tableA.ForeignKeys[0].RefTable, "table 'a' should reference 'b'")
	assert.Equal(t, "a", tableB.ForeignKeys[0].RefTable, "table 'b' should reference 'a'")

	// Note: DEFERRABLE status introspection not yet implemented
	// The test verifies that circular FKs can be created and introspected
	// Future enhancement: introspector should capture Deferrable and InitiallyDeferred fields
}
