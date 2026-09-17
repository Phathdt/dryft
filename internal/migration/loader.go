package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// Migration represents a loaded migration file.
type Migration struct {
	Filename  string
	Timestamp string
	Name      string
	UpSQL     []string // Statements in Up section
	DownSQL   []string // Statements in Down section
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

// ExtractStatements extracts SQL statements from migration content.
// section can be "up" or "down".
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

	// Remove comment-only lines before splitting
	var cleanedLines []string
	for _, line := range strings.Split(sectionContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}
	sectionContent = strings.Join(cleanedLines, "\n")

	// Split by semicolon
	statements := splitStatements(sectionContent)

	// Clean up - trim whitespace and filter empty statements
	var cleaned []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
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

// LoadSchemaFromMigrations loads schema from migration files in directory.
// This is the main entry point for building schema from migration history (100% offline).
//
// Returns:
//   - *schema.Schema if migrations exist and are parseable
//   - nil if directory is empty or doesn't exist (first migration scenario)
//   - error if parsing fails
func LoadSchemaFromMigrations(directory string) (*schema.Schema, error) {
	loader := NewLoader(directory)
	migrations, err := loader.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Empty migrations directory - return nil (first migration scenario)
	if len(migrations) == 0 {
		return nil, nil
	}

	// Build schema from migrations
	builder := NewSchemaBuilder()
	parser := NewParser()

	for _, migration := range migrations {
		// Only process Up statements to build current state
		for _, sqlStmt := range migration.UpSQL {
			stmt, err := parser.Parse(sqlStmt)
			if err != nil {
				return nil, fmt.Errorf("failed to parse statement in %s: %w\nSQL: %s", migration.Filename, err, sqlStmt)
			}

			if err := builder.Apply(stmt); err != nil {
				return nil, fmt.Errorf("failed to apply statement in %s: %w\nSQL: %s", migration.Filename, err, sqlStmt)
			}
		}
	}

	return builder.Build(), nil
}
