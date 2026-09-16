package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Integration_RealMigrationSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create a realistic migration sequence
	migrations := []struct {
		filename string
		content  string
	}{
		{
			"20240101000000_create_users.sql",
			`-- +goose Up
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	username VARCHAR(50) NOT NULL,
	created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);

-- +goose Down
DROP INDEX idx_users_email;
DROP TABLE users;`,
		},
		{
			"20240102120000_add_user_profile_columns.sql",
			`-- +goose Up
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN avatar_url TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN avatar_url;
ALTER TABLE users DROP COLUMN bio;`,
		},
		{
			"20240103153000_create_posts.sql",
			`-- +goose Up
CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');

CREATE TABLE posts (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	content TEXT,
	status post_status NOT NULL DEFAULT 'draft',
	published_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_published ON posts(published_at) WHERE status = 'published';

-- +goose Down
DROP INDEX idx_posts_published;
DROP INDEX idx_posts_status;
DROP INDEX idx_posts_user_id;
DROP TABLE posts;
DROP TYPE post_status;`,
		},
	}

	for _, mig := range migrations {
		err := os.WriteFile(filepath.Join(tmpDir, mig.filename), []byte(mig.content), 0644)
		require.NoError(t, err)
	}

	// Load migrations
	loader := NewLoader(tmpDir)
	loaded, err := loader.LoadAll()
	require.NoError(t, err)
	require.Len(t, loaded, 3)

	// Verify first migration (create_users)
	assert.Equal(t, "20240101000000", loaded[0].Timestamp)
	assert.Equal(t, "create_users", loaded[0].Name)
	assert.Len(t, loaded[0].UpSQL, 2) // CREATE TABLE + CREATE INDEX
	assert.Contains(t, loaded[0].UpSQL[0], "CREATE TABLE users")
	assert.Contains(t, loaded[0].UpSQL[1], "CREATE INDEX idx_users_email")
	assert.Len(t, loaded[0].DownSQL, 2)

	// Verify second migration (add_user_profile_columns)
	assert.Equal(t, "20240102120000", loaded[1].Timestamp)
	assert.Equal(t, "add_user_profile_columns", loaded[1].Name)
	assert.Len(t, loaded[1].UpSQL, 2) // Two ALTER statements
	assert.Contains(t, loaded[1].UpSQL[0], "ADD COLUMN bio")
	assert.Contains(t, loaded[1].UpSQL[1], "ADD COLUMN avatar_url")

	// Verify third migration (create_posts with enum and indexes)
	assert.Equal(t, "20240103153000", loaded[2].Timestamp)
	assert.Equal(t, "create_posts", loaded[2].Name)
	assert.Len(t, loaded[2].UpSQL, 5) // CREATE TYPE + CREATE TABLE + 3 CREATE INDEX
	assert.Contains(t, loaded[2].UpSQL[0], "CREATE TYPE post_status")
	assert.Contains(t, loaded[2].UpSQL[1], "CREATE TABLE posts")
	assert.Contains(t, loaded[2].UpSQL[2], "CREATE INDEX idx_posts_user_id")
	assert.Contains(t, loaded[2].UpSQL[3], "CREATE INDEX idx_posts_status")
	assert.Contains(t, loaded[2].UpSQL[4], "CREATE INDEX idx_posts_published")
}

func TestLoader_Integration_EmptyMigrationsDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	loader := NewLoader(tmpDir)
	migrations, err := loader.LoadAll()

	require.NoError(t, err)
	assert.Empty(t, migrations)
}

func TestLoader_Integration_WithMalformedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	files := map[string]string{
		"20240101000000_valid.sql": `-- +goose Up
CREATE TABLE valid (id INT);

-- +goose Down
DROP TABLE valid;`,
		"invalid.sql": "not a valid migration filename",
		"README.md":   "documentation file",
		"backup.bak":  "backup file",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	loader := NewLoader(tmpDir)
	migrations, err := loader.LoadAll()

	require.NoError(t, err)
	require.Len(t, migrations, 1) // Only valid.sql loaded
	assert.Equal(t, "valid", migrations[0].Name)
}
