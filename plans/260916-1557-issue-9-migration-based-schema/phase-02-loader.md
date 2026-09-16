# Phase 2: Migration Loader

**Status:** Not Started  
**Dependencies:** Phase 1 (Parser)  
**Estimated Effort:** 1 day  
**Risk Level:** Low

## Objective

Load migration files từ disk, sort chronologically, extract statements, và orchestrate parsing pipeline.

## Scope

### In Scope
- Read .sql files từ migrations directory
- Parse Goose filename format: `YYYYMMDDHHMMSS_name.sql`
- Sort migrations by timestamp
- Extract SQL statements (split by `;`)
- Handle Goose Up/Down sections
- Return ordered statements ready for builder

### Out of Scope
- Goose state tracking (goose_db_version table)
- Migration execution (Goose's responsibility)
- Migration validation beyond parsing

## Data Structures

```go
// internal/migration/loader.go

package migration

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a loaded migration file.
type Migration struct {
	Filename  string
	Timestamp string
	Name      string
	UpSQL     []string   // Statements in Up section
	DownSQL   []string   // Statements in Down section
}

// Loader loads migrations from disk.
type Loader struct {
	directory string
	parser    *Parser
}

// NewLoader creates a new migration loader.
func NewLoader(directory string) *Loader {
	return &Loader{
		directory: directory,
		parser:    NewParser(),
	}
}

// LoadAll loads all migrations from directory, sorted by timestamp.
func (l *Loader) LoadAll() ([]Migration, error) {
	// Implementation
}

// ExtractStatements extracts SQL statements from migration content.
func (l *Loader) ExtractStatements(content string, section string) ([]string, error) {
	// section: "up" or "down"
	// Implementation
}
```

## Implementation Steps

### 1. File Discovery
```go
func (l *Loader) LoadAll() ([]Migration, error) {
	entries, err := os.ReadDir(l.directory)
	if err != nil {
		if os.IsNotExist(err) {
			return []Migration{}, nil // Empty dir OK
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		timestamp, name, ok := ParseFilename(entry.Name())
		if !ok {
			// Skip malformed filenames
			continue
		}

		content, err := os.ReadFile(filepath.Join(l.directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		migration := Migration{
			Filename:  entry.Name(),
			Timestamp: timestamp,
			Name:      name,
		}

		// Extract Up/Down sections
		migration.UpSQL, err = l.ExtractStatements(string(content), "up")
		if err != nil {
			return nil, fmt.Errorf("failed to extract Up statements from %s: %w", entry.Name(), err)
		}

		migration.DownSQL, err = l.ExtractStatements(string(content), "down")
		if err != nil {
			return nil, fmt.Errorf("failed to extract Down statements from %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration)
	}

	// Sort by timestamp
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Timestamp < migrations[j].Timestamp
	})

	return migrations, nil
}
```

### 2. Statement Extraction

Parse Goose format:
```
-- +goose Up
CREATE TABLE users (...);
ALTER TABLE posts ...;

-- +goose Down
DROP TABLE users;
```

```go
func (l *Loader) ExtractStatements(content string, section string) ([]string, error) {
	var marker string
	var endMarker string

	if section == "up" {
		marker = "-- +goose Up"
		endMarker = "-- +goose Down"
	} else {
		marker = "-- +goose Down"
		endMarker = "" // Down section goes to EOF
	}

	// Find section start
	startIdx := strings.Index(content, marker)
	if startIdx == -1 {
		return []string{}, nil // Section not found
	}

	// Extract section content
	sectionContent := content[startIdx+len(marker):]
	if endMarker != "" {
		if endIdx := strings.Index(sectionContent, endMarker); endIdx != -1 {
			sectionContent = sectionContent[:endIdx]
		}
	}

	// Split by semicolon, filter empty/comments
	statements := splitStatements(sectionContent)
	
	// Clean up
	var cleaned []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		cleaned = append(cleaned, stmt)
	}

	return cleaned, nil
}

// splitStatements splits SQL content by semicolons, respecting quotes.
func splitStatements(content string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(content); i++ {
		ch := content[i]

		switch ch {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
			current.WriteByte(ch)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			current.WriteByte(ch)
		case ';':
			if !inSingleQuote && !inDoubleQuote {
				statements = append(statements, current.String())
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}

	// Add remaining content
	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}
```

### 3. Public API

```go
// LoadSchemaFromMigrations builds schema from migration files.
// Returns nil if migrations directory is empty or doesn't exist.
func LoadSchemaFromMigrations(directory string) (*schema.Schema, error) {
	loader := NewLoader(directory)
	migrations, err := loader.LoadAll()
	if err != nil {
		return nil, err
	}

	if len(migrations) == 0 {
		return nil, nil // No migrations found
	}

	// Build schema from migrations (Phase 3 - builder)
	builder := NewSchemaBuilder()
	for _, mig := range migrations {
		for _, sql := range mig.UpSQL {
			stmt, err := loader.parser.Parse(sql)
			if err != nil {
				return nil, fmt.Errorf("failed to parse statement in %s: %w\nSQL: %s", mig.Filename, err, sql)
			}
			if err := builder.Apply(stmt); err != nil {
				return nil, fmt.Errorf("failed to apply statement in %s: %w", mig.Filename, err)
			}
		}
	}

	return builder.Build(), nil
}
```

## Testing

### Unit Tests (`loader_test.go`)

```go
func TestLoader_LoadAll(t *testing.T) {
	// Create temp directory with test migrations
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
		"invalid_name.sql": "should be skipped",
	}

	for name, content := range files {
		os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
	}

	loader := NewLoader(tmpDir)
	migrations, err := loader.LoadAll()
	
	require.NoError(t, err)
	require.Len(t, migrations, 2) // invalid_name.sql skipped
	
	// Check order
	assert.Equal(t, "20240101000000", migrations[0].Timestamp)
	assert.Equal(t, "create_users", migrations[0].Name)
	assert.Equal(t, "20240102000000", migrations[1].Timestamp)
	
	// Check statements
	assert.Len(t, migrations[0].UpSQL, 1)
	assert.Contains(t, migrations[0].UpSQL[0], "CREATE TABLE users")
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
			name:    "semicolon in string",
			content: "INSERT INTO users (name) VALUES ('John; Doe');",
			want:    []string{"INSERT INTO users (name) VALUES ('John; Doe')"},
		},
		{
			name:    "quoted identifier with semicolon",
			content: `CREATE TABLE "weird;table" (id INT);`,
			want:    []string{`CREATE TABLE "weird;table" (id INT)`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitStatements(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

### Integration Test

```go
func TestLoadSchemaFromMigrations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	
	// Create migration sequence
	migrations := []struct {
		filename string
		content  string
	}{
		{
			"20240101000000_create_users.sql",
			`-- +goose Up
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL
);

-- +goose Down
DROP TABLE users;`,
		},
		{
			"20240102000000_add_bio.sql",
			`-- +goose Up
ALTER TABLE users ADD COLUMN bio TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN bio;`,
		},
	}

	for _, mig := range migrations {
		os.WriteFile(filepath.Join(tmpDir, mig.filename), []byte(mig.content), 0644)
	}

	// Load schema
	schema, err := LoadSchemaFromMigrations(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Verify schema state
	require.Len(t, schema.Tables, 1)
	assert.Equal(t, "users", schema.Tables[0].Name)
	
	// Should have 3 columns: id, email, bio (from both migrations)
	require.Len(t, schema.Tables[0].Columns, 3)
	
	columnNames := []string{}
	for _, col := range schema.Tables[0].Columns {
		columnNames = append(columnNames, col.Name)
	}
	assert.Contains(t, columnNames, "id")
	assert.Contains(t, columnNames, "email")
	assert.Contains(t, columnNames, "bio")
}
```

## Error Handling

```go
type LoadError struct {
	Filename string
	Cause    error
}

func (e *LoadError) Error() string {
	return fmt.Sprintf("failed to load migration %s: %v", e.Filename, e.Cause)
}
```

## Files to Create

```
internal/migration/
├── loader.go
├── loader_test.go
└── testdata/
    └── migrations/
        ├── 20240101000000_initial.sql
        └── 20240102000000_add_column.sql
```

## Validation

**Done When:**
- [ ] Loader reads and sorts migrations correctly
- [ ] Statement extraction handles Goose format
- [ ] Semicolon splitting respects quotes
- [ ] Empty/missing directories handled gracefully
- [ ] Malformed filenames skipped with warning
- [ ] Unit tests pass (>85% coverage)
- [ ] Integration test with real migration sequence passes

## Notes

- Chỉ parse Up section - Down section không cần cho schema building
- Skip malformed filenames thay vì fail hard (resilience)
- Performance: O(n) files, O(m) statements per file - acceptable cho typical migration counts (<1000)
