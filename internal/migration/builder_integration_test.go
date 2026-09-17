package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderIntegration_FullMigrationSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create temporary directory for migration files
	tmpDir := t.TempDir()

	// Create migration files
	migrationFiles := []struct {
		filename string
		content  string
	}{
		{
			filename: "20240101120000_create_users.sql",
			content: `-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_email ON users(email);

-- +goose Down
DROP TABLE users;
`,
		},
		{
			filename: "20240102120000_add_user_fields.sql",
			content: `-- +goose Up
ALTER TABLE users ADD COLUMN name TEXT;
ALTER TABLE users ADD COLUMN age INTEGER;

-- +goose Down
ALTER TABLE users DROP COLUMN age;
ALTER TABLE users DROP COLUMN name;
`,
		},
		{
			filename: "20240103120000_create_user_role_enum.sql",
			content: `-- +goose Up
CREATE TYPE user_role AS ENUM ('admin', 'user', 'guest');

ALTER TABLE users ADD COLUMN role user_role;

-- +goose Down
ALTER TABLE users DROP COLUMN role;
DROP TYPE user_role;
`,
		},
		{
			filename: "20240104120000_create_posts.sql",
			content: `-- +goose Up
CREATE TABLE posts (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_posts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published ON posts(published) WHERE published = true;

-- +goose Down
DROP TABLE posts;
`,
		},
	}

	// Write migration files
	for _, m := range migrationFiles {
		path := filepath.Join(tmpDir, m.filename)
		err := os.WriteFile(path, []byte(m.content), 0644)
		require.NoError(t, err)
	}

	// Load migrations
	loader := NewLoader(tmpDir)
	migrations, err := loader.LoadAll()
	require.NoError(t, err)
	require.NotEmpty(t, migrations)

	// Build schema by parsing and applying each statement
	builder := NewSchemaBuilder()
	parser := NewParser()
	for _, migration := range migrations {
		for _, sqlStmt := range migration.UpSQL {
			stmt, err := parser.Parse(sqlStmt)
			require.NoError(t, err, "failed to parse statement from %s: %s", migration.Filename, sqlStmt)

			err = builder.Apply(stmt)
			require.NoError(t, err, "failed to apply statement: %T", stmt)
		}
	}

	result := builder.Build()

	// Verify schema
	require.Len(t, result.Tables, 2, "expected 2 tables (users, posts)")
	require.Len(t, result.Enums, 1, "expected 1 enum (user_role)")

	// Verify users table
	var usersTable *schema.Table
	for i := range result.Tables {
		if result.Tables[i].Name == "users" {
			usersTable = &result.Tables[i]
			break
		}
	}
	require.NotNil(t, usersTable, "users table not found")

	assert.Len(t, usersTable.Columns, 6, "users should have 6 columns: id, email, created_at, name, age, role")
	assert.NotNil(t, usersTable.PrimaryKey, "users should have primary key")
	assert.Equal(t, []string{"id"}, usersTable.PrimaryKey.Columns)
	assert.Len(t, usersTable.Indexes, 1, "users should have 1 index (idx_users_email)")

	// Verify posts table
	var postsTable *schema.Table
	for i := range result.Tables {
		if result.Tables[i].Name == "posts" {
			postsTable = &result.Tables[i]
			break
		}
	}
	require.NotNil(t, postsTable, "posts table not found")

	assert.Len(t, postsTable.Columns, 6, "posts should have 6 columns")
	assert.NotNil(t, postsTable.PrimaryKey)
	assert.Len(t, postsTable.ForeignKeys, 1, "posts should have 1 foreign key")
	assert.Len(t, postsTable.Indexes, 2, "posts should have 2 indexes")

	// Verify foreign key
	fk := postsTable.ForeignKeys[0]
	assert.Equal(t, "fk_posts_user", fk.Name)
	assert.Equal(t, []string{"user_id"}, fk.Columns)
	assert.Equal(t, "users", fk.RefTable)
	assert.Equal(t, []string{"id"}, fk.RefColumns)
	assert.Equal(t, schema.ActionCascade, fk.OnDelete)

	// Verify user_role enum
	enum := result.Enums[0]
	assert.Equal(t, "user_role", enum.Name)
	assert.Len(t, enum.Values, 3)
	assert.Equal(t, "admin", enum.Values[0].Label)
	assert.Equal(t, "user", enum.Values[1].Label)
	assert.Equal(t, "guest", enum.Values[2].Label)
}

func TestBuilderIntegration_ComplexAlterations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	migrationFiles := []struct {
		filename string
		content  string
	}{
		{
			filename: "20240101120000_initial.sql",
			content: `-- +goose Up
CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    price NUMERIC(10, 2),
    old_field TEXT
);
`,
		},
		{
			filename: "20240102120000_rename_column.sql",
			content: `-- +goose Up
ALTER TABLE products RENAME COLUMN old_field TO description;
`,
		},
		{
			filename: "20240103120000_modify_columns.sql",
			content: `-- +goose Up
ALTER TABLE products ALTER COLUMN price SET NOT NULL;
ALTER TABLE products ADD COLUMN stock INTEGER DEFAULT 0;
`,
		},
		{
			filename: "20240104120000_drop_and_add.sql",
			content: `-- +goose Up
ALTER TABLE products DROP COLUMN description;
ALTER TABLE products ADD COLUMN tags TEXT[];
`,
		},
	}

	for _, m := range migrationFiles {
		path := filepath.Join(tmpDir, m.filename)
		err := os.WriteFile(path, []byte(m.content), 0644)
		require.NoError(t, err)
	}

	loader := NewLoader(tmpDir)
	migrations, err := loader.LoadAll()
	require.NoError(t, err)

	builder := NewSchemaBuilder()
	parser := NewParser()
	for _, migration := range migrations {
		for _, sqlStmt := range migration.UpSQL {
			stmt, err := parser.Parse(sqlStmt)
			require.NoError(t, err, "failed to parse: %s", sqlStmt)

			err = builder.Apply(stmt)
			require.NoError(t, err)
		}
	}

	result := builder.Build()
	require.Len(t, result.Tables, 1)

	table := result.Tables[0]
	assert.Equal(t, "products", table.Name)

	// Verify final columns: id, name, price, stock, tags (old_field renamed to description then dropped)
	assert.Len(t, table.Columns, 5, "should have 5 columns")

	// Check specific columns
	columnNames := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		columnNames[i] = col.Name
	}
	assert.Contains(t, columnNames, "id")
	assert.Contains(t, columnNames, "name")
	assert.Contains(t, columnNames, "price")
	assert.Contains(t, columnNames, "stock")
	assert.Contains(t, columnNames, "tags")
	assert.NotContains(t, columnNames, "old_field")
	assert.NotContains(t, columnNames, "description")

	// Verify price is NOT NULL
	var priceCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "price" {
			priceCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, priceCol)
	assert.False(t, priceCol.Nullable, "price should be NOT NULL after alteration")

	// Verify tags is array
	var tagsCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "tags" {
			tagsCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, tagsCol)
	assert.Equal(t, schema.TypeText, tagsCol.Type.Kind)
	assert.Equal(t, 1, tagsCol.Type.ArrayDepth, "tags should be an array")
}

func TestBuilderIntegration_IndexOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	migrationFiles := []struct {
		filename string
		content  string
	}{
		{
			filename: "20240101120000_create_table.sql",
			content: `-- +goose Up
CREATE TABLE documents (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    metadata JSONB
);
`,
		},
		{
			filename: "20240102120000_add_indexes.sql",
			content: `-- +goose Up
CREATE INDEX idx_documents_title ON documents(title);
CREATE INDEX idx_documents_metadata ON documents USING gin(metadata);
CREATE UNIQUE INDEX idx_documents_content_hash ON documents((md5(content)));
`,
		},
		{
			filename: "20240103120000_drop_index.sql",
			content: `-- +goose Up
DROP INDEX idx_documents_content_hash;
`,
		},
	}

	for _, m := range migrationFiles {
		path := filepath.Join(tmpDir, m.filename)
		err := os.WriteFile(path, []byte(m.content), 0644)
		require.NoError(t, err)
	}

	loader := NewLoader(tmpDir)
	migrations, err := loader.LoadAll()
	require.NoError(t, err)

	builder := NewSchemaBuilder()
	parser := NewParser()
	for _, migration := range migrations {
		for _, sqlStmt := range migration.UpSQL {
			stmt, err := parser.Parse(sqlStmt)
			require.NoError(t, err, "failed to parse: %s", sqlStmt)

			err = builder.Apply(stmt)
			require.NoError(t, err)
		}
	}

	result := builder.Build()
	require.Len(t, result.Tables, 1)

	table := result.Tables[0]
	assert.Equal(t, "documents", table.Name)

	// Should have 2 indexes remaining (content_hash was dropped)
	assert.Len(t, table.Indexes, 2)

	indexNames := make([]string, len(table.Indexes))
	for i, idx := range table.Indexes {
		indexNames[i] = idx.Name
	}
	assert.Contains(t, indexNames, "idx_documents_title")
	assert.Contains(t, indexNames, "idx_documents_metadata")
	assert.NotContains(t, indexNames, "idx_documents_content_hash")

	// Verify GIN index type
	var metadataIdx *schema.Index
	for i := range table.Indexes {
		if table.Indexes[i].Name == "idx_documents_metadata" {
			metadataIdx = &table.Indexes[i]
			break
		}
	}
	require.NotNil(t, metadataIdx)
	assert.Equal(t, schema.IndexGIN, metadataIdx.Type)
}

func TestBuilderIntegration_EmptyMigrationsDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	loader := NewLoader(tmpDir)
	migrations, err := loader.LoadAll()
	require.NoError(t, err)
	assert.Empty(t, migrations, "should return empty migrations for empty directory")

	builder := NewSchemaBuilder()
	parser := NewParser()
	for _, migration := range migrations {
		for _, sqlStmt := range migration.UpSQL {
			stmt, err := parser.Parse(sqlStmt)
			require.NoError(t, err)

			err = builder.Apply(stmt)
			require.NoError(t, err)
		}
	}

	result := builder.Build()
	assert.Empty(t, result.Tables, "should have no tables")
	assert.Empty(t, result.Enums, "should have no enums")
}
