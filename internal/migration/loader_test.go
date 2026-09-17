package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "simple statements",
			content: "CREATE TABLE a; DROP TABLE b;",
			want:    []string{"CREATE TABLE a", " DROP TABLE b"},
		},
		{
			name:    "semicolon in single quote",
			content: "INSERT INTO users (name) VALUES ('John; Doe');",
			want:    []string{"INSERT INTO users (name) VALUES ('John; Doe')"},
		},
		{
			name:    "semicolon in double quote",
			content: `CREATE TABLE "weird;table" (id INT);`,
			want:    []string{`CREATE TABLE "weird;table" (id INT)`},
		},
		{
			name:    "mixed quotes",
			content: `INSERT INTO t (a, b) VALUES ('x;y', "z;w");`,
			want:    []string{`INSERT INTO t (a, b) VALUES ('x;y', "z;w")`},
		},
		{
			name:    "no semicolon",
			content: "CREATE TABLE users (id INT)",
			want:    []string{"CREATE TABLE users (id INT)"},
		},
		{
			name:    "empty content",
			content: "",
			want:    []string{},
		},
		{
			name:    "multiple semicolons",
			content: "CREATE TABLE a;;;DROP TABLE b;",
			want:    []string{"CREATE TABLE a", "", "", "DROP TABLE b"},
		},
		{
			name:    "nested quotes",
			content: `INSERT INTO t VALUES ('He said "hello; world"');`,
			want:    []string{`INSERT INTO t VALUES ('He said "hello; world"')`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitStatements(tt.content)
			assert.Equal(t, len(tt.want), len(got), "length mismatch")
			for i := range tt.want {
				assert.Equal(t, tt.want[i], got[i])
			}
		})
	}
}

func TestLoader_ExtractStatements(t *testing.T) {
	tests := []struct {
		name    string
		content string
		section string
		want    []string
	}{
		{
			name: "up section with multiple statements",
			content: `
-- +goose Up
CREATE TABLE users (id INT);
CREATE TABLE posts (id INT);

-- +goose Down
DROP TABLE posts;
DROP TABLE users;
`,
			section: "up",
			want: []string{
				"CREATE TABLE users (id INT)",
				"CREATE TABLE posts (id INT)",
			},
		},
		{
			name: "down section",
			content: `
-- +goose Up
CREATE TABLE users (id INT);

-- +goose Down
DROP TABLE users;
`,
			section: "down",
			want:    []string{"DROP TABLE users"},
		},
		{
			name:    "section not found",
			content: "some random sql",
			section: "up",
			want:    []string{},
		},
		{
			name: "up section only (no down)",
			content: `
-- +goose Up
CREATE TABLE users (id INT);
`,
			section: "up",
			want:    []string{"CREATE TABLE users (id INT)"},
		},
		{
			name: "down section at end of file",
			content: `
-- +goose Up
CREATE TABLE users (id INT);

-- +goose Down
DROP TABLE users;
-- final comment`,
			section: "down",
			want:    []string{"DROP TABLE users"},
		},
		{
			name: "empty statements filtered",
			content: `
-- +goose Up
CREATE TABLE a (id INT);
;
;
CREATE TABLE b (id INT);

-- +goose Down
DROP TABLE b;
`,
			section: "up",
			want: []string{
				"CREATE TABLE a (id INT)",
				"CREATE TABLE b (id INT)",
			},
		},
		{
			name: "comments filtered",
			content: `
-- +goose Up
-- This is a comment
CREATE TABLE users (id INT);
-- Another comment
CREATE TABLE posts (id INT);

-- +goose Down
DROP TABLE posts;
`,
			section: "up",
			want: []string{
				"CREATE TABLE users (id INT)",
				"CREATE TABLE posts (id INT)",
			},
		},
		{
			name: "semicolon in string preserved",
			content: `
-- +goose Up
INSERT INTO config (key, value) VALUES ('url', 'http://example.com;port=5432');

-- +goose Down
DELETE FROM config WHERE key = 'url';
`,
			section: "up",
			want:    []string{"INSERT INTO config (key, value) VALUES ('url', 'http://example.com;port=5432')"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader("")
			got, err := loader.ExtractStatements(tt.content, tt.section)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoader_LoadAll(t *testing.T) {
	t.Run("load and sort migrations", func(t *testing.T) {
		tmpDir := t.TempDir()

		files := map[string]string{
			"20240101000000_create_users.sql": `
-- +goose Up
CREATE TABLE users (id SERIAL PRIMARY KEY);

-- +goose Down
DROP TABLE users;
`,
			"20240102000000_add_email.sql": `
-- +goose Up
ALTER TABLE users ADD COLUMN email TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN email;
`,
			"20240103150000_create_posts.sql": `
-- +goose Up
CREATE TABLE posts (id SERIAL PRIMARY KEY, user_id INT);

-- +goose Down
DROP TABLE posts;
`,
		}

		for name, content := range files {
			err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
			require.NoError(t, err)
		}

		loader := NewLoader(tmpDir)
		migrations, err := loader.LoadAll()

		require.NoError(t, err)
		require.Len(t, migrations, 3)

		// Check chronological order
		assert.Equal(t, "20240101000000", migrations[0].Timestamp)
		assert.Equal(t, "create_users", migrations[0].Name)
		assert.Equal(t, "20240102000000", migrations[1].Timestamp)
		assert.Equal(t, "add_email", migrations[1].Name)
		assert.Equal(t, "20240103150000", migrations[2].Timestamp)
		assert.Equal(t, "create_posts", migrations[2].Name)

		// Check statements extracted
		assert.Len(t, migrations[0].UpSQL, 1)
		assert.Contains(t, migrations[0].UpSQL[0], "CREATE TABLE users")
		assert.Len(t, migrations[0].DownSQL, 1)
		assert.Contains(t, migrations[0].DownSQL[0], "DROP TABLE users")

		assert.Len(t, migrations[1].UpSQL, 1)
		assert.Contains(t, migrations[1].UpSQL[0], "ALTER TABLE users ADD COLUMN email")

		assert.Len(t, migrations[2].UpSQL, 1)
		assert.Contains(t, migrations[2].UpSQL[0], "CREATE TABLE posts")
	})

	t.Run("skip malformed filenames", func(t *testing.T) {
		tmpDir := t.TempDir()

		files := map[string]string{
			"20240101000000_valid.sql": `
-- +goose Up
CREATE TABLE valid (id INT);

-- +goose Down
DROP TABLE valid;
`,
			"invalid_name.sql": "should be skipped",
			"README.md":        "not a migration",
			"2024010100_short_timestamp.sql": `
-- +goose Up
CREATE TABLE short (id INT);

-- +goose Down
DROP TABLE short;
`,
		}

		for name, content := range files {
			err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
			require.NoError(t, err)
		}

		loader := NewLoader(tmpDir)
		migrations, err := loader.LoadAll()

		require.NoError(t, err)
		require.Len(t, migrations, 1) // Only valid.sql loaded
		assert.Equal(t, "20240101000000", migrations[0].Timestamp)
		assert.Equal(t, "valid", migrations[0].Name)
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		loader := NewLoader(tmpDir)
		migrations, err := loader.LoadAll()

		require.NoError(t, err)
		assert.Empty(t, migrations)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		loader := NewLoader("/path/that/does/not/exist")
		migrations, err := loader.LoadAll()

		require.NoError(t, err)
		assert.Empty(t, migrations)
	})

	t.Run("directory with subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a subdirectory (should be skipped)
		subdir := filepath.Join(tmpDir, "subdir")
		err := os.Mkdir(subdir, 0755)
		require.NoError(t, err)

		// Create a valid migration
		err = os.WriteFile(filepath.Join(tmpDir, "20240101000000_test.sql"), []byte(`
-- +goose Up
CREATE TABLE test (id INT);

-- +goose Down
DROP TABLE test;
`), 0644)
		require.NoError(t, err)

		loader := NewLoader(tmpDir)
		migrations, err := loader.LoadAll()

		require.NoError(t, err)
		require.Len(t, migrations, 1)
		assert.Equal(t, "test", migrations[0].Name)
	})

	t.Run("migration without goose markers", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "20240101000000_no_markers.sql"), []byte(`
CREATE TABLE users (id INT);
DROP TABLE users;
`), 0644)
		require.NoError(t, err)

		loader := NewLoader(tmpDir)
		migrations, err := loader.LoadAll()

		require.NoError(t, err)
		require.Len(t, migrations, 1)
		// No statements extracted (no goose markers)
		assert.Empty(t, migrations[0].UpSQL)
		assert.Empty(t, migrations[0].DownSQL)
	})
}
