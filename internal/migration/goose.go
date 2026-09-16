package migration

import (
	"fmt"
	"strings"
	"time"
)

// GooseMigration represents a Goose-formatted migration.
type GooseMigration struct {
	Name     string
	Up       []string
	Down     []string
	Warnings []string
}

// FormatGoose formats migration statements into Goose format.
func FormatGoose(name string, up, down []string, warnings []string) string {
	var b strings.Builder

	// Up migration
	b.WriteString("-- +goose Up\n")
	if len(up) > 0 {
		b.WriteString("\n")
		for _, stmt := range up {
			b.WriteString(stmt)
			b.WriteString("\n\n")
		}
	}

	// Down migration
	b.WriteString("-- +goose Down\n")
	if len(down) > 0 {
		b.WriteString("\n")
		for _, stmt := range down {
			b.WriteString(stmt)
			b.WriteString("\n\n")
		}
	}

	// Warnings
	if len(warnings) > 0 {
		b.WriteString("-- Warnings:\n")
		for _, warning := range warnings {
			b.WriteString("-- ")
			b.WriteString(warning)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// GenerateFilename generates a timestamp-based Goose migration filename.
// Format: YYYYMMDDHHMMSS_name.sql
func GenerateFilename(name string) string {
	timestamp := time.Now().Format("20060102150405")
	sanitized := sanitizeName(name)
	return fmt.Sprintf("%s_%s.sql", timestamp, sanitized)
}

// sanitizeName sanitizes migration name for filename.
func sanitizeName(name string) string {
	// Replace spaces and dashes with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Remove non-alphanumeric characters except underscore
	var result strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		   (ch >= '0' && ch <= '9') || ch == '_' {
			result.WriteRune(ch)
		}
	}

	// Convert to lowercase
	return strings.ToLower(result.String())
}

// ParseFilename parses a Goose migration filename.
func ParseFilename(filename string) (timestamp, name string, ok bool) {
	// Format: YYYYMMDDHHMMSS_name.sql
	if !strings.HasSuffix(filename, ".sql") {
		return "", "", false
	}

	filename = strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	// Validate timestamp is 14 digits
	if len(parts[0]) != 14 {
		return "", "", false
	}

	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			return "", "", false
		}
	}

	return parts[0], parts[1], true
}
