package migration

import (
	"strings"
	"testing"
)

func TestFormatGoose(t *testing.T) {
	up := []string{
		"CREATE TABLE users (\n  id UUID PRIMARY KEY\n);",
		"CREATE INDEX idx_users_email ON users (email);",
	}

	down := []string{
		"DROP INDEX idx_users_email;",
		"DROP TABLE users;",
	}

	warnings := []string{
		"Cannot reverse DROP COLUMN operation",
	}

	result := FormatGoose("test", up, down, warnings)

	// Check structure
	if !strings.Contains(result, "-- +goose Up") {
		t.Error("expected '-- +goose Up' marker")
	}

	if !strings.Contains(result, "-- +goose Down") {
		t.Error("expected '-- +goose Down' marker")
	}

	// Check up statements
	if !strings.Contains(result, "CREATE TABLE users") {
		t.Error("expected CREATE TABLE in up section")
	}

	if !strings.Contains(result, "CREATE INDEX") {
		t.Error("expected CREATE INDEX in up section")
	}

	// Check down statements
	if !strings.Contains(result, "DROP INDEX") {
		t.Error("expected DROP INDEX in down section")
	}

	if !strings.Contains(result, "DROP TABLE") {
		t.Error("expected DROP TABLE in down section")
	}

	// Check warnings
	if !strings.Contains(result, "-- Warnings:") {
		t.Error("expected warnings section")
	}

	if !strings.Contains(result, "Cannot reverse DROP COLUMN") {
		t.Error("expected warning message")
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name string
		want string // Pattern to match
	}{
		{"create_users_table", "_create_users_table.sql"},
		{"Add User Email", "_add_user_email.sql"},
		{"Drop Old Index", "_drop_old_index.sql"},
		{"Update-Schema", "_update_schema.sql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := GenerateFilename(tt.name)

			// Check suffix
			if !strings.HasSuffix(filename, tt.want) {
				t.Errorf("GenerateFilename(%q) suffix = %v, want suffix %v",
					tt.name, filename, tt.want)
			}

			// Check format: should start with timestamp
			if len(filename) < 14 {
				t.Errorf("GenerateFilename(%q) = %v, too short", tt.name, filename)
			}

			// Check timestamp part is numeric
			timestamp := filename[:14]
			for _, ch := range timestamp {
				if ch < '0' || ch > '9' {
					t.Errorf("GenerateFilename(%q) timestamp %q contains non-digit",
						tt.name, timestamp)
				}
			}
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"create_users_table", "create_users_table"},
		{"Add User Email", "add_user_email"},
		{"Drop-Old-Index", "drop_old_index"},
		{"Update Schema!", "update_schema"},
		{"Fix@Bug#123", "fixbug123"},
		{"CamelCase", "camelcase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		filename string
		wantTS   string
		wantName string
		wantOK   bool
	}{
		{
			filename: "20260915180000_create_users.sql",
			wantTS:   "20260915180000",
			wantName: "create_users",
			wantOK:   true,
		},
		{
			filename: "20260915180000_add_email_column.sql",
			wantTS:   "20260915180000",
			wantName: "add_email_column",
			wantOK:   true,
		},
		{
			filename: "invalid_filename.sql",
			wantTS:   "",
			wantName: "",
			wantOK:   false,
		},
		{
			filename: "20260915180000.sql",
			wantTS:   "",
			wantName: "",
			wantOK:   false,
		},
		{
			filename: "notasqlfile.txt",
			wantTS:   "",
			wantName: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			gotTS, gotName, gotOK := ParseFilename(tt.filename)

			if gotOK != tt.wantOK {
				t.Errorf("ParseFilename(%q) ok = %v, want %v",
					tt.filename, gotOK, tt.wantOK)
			}

			if gotTS != tt.wantTS {
				t.Errorf("ParseFilename(%q) timestamp = %v, want %v",
					tt.filename, gotTS, tt.wantTS)
			}

			if gotName != tt.wantName {
				t.Errorf("ParseFilename(%q) name = %v, want %v",
					tt.filename, gotName, tt.wantName)
			}
		})
	}
}

func TestFormatGoose_EmptyMigration(t *testing.T) {
	result := FormatGoose("empty", nil, nil, nil)

	if !strings.Contains(result, "-- +goose Up") {
		t.Error("expected '-- +goose Up' marker")
	}

	if !strings.Contains(result, "-- +goose Down") {
		t.Error("expected '-- +goose Down' marker")
	}

	if strings.Contains(result, "-- Warnings:") {
		t.Error("unexpected warnings section")
	}
}

func TestFormatGoose_NoWarnings(t *testing.T) {
	up := []string{"CREATE TABLE test (id INTEGER);"}
	down := []string{"DROP TABLE test;"}

	result := FormatGoose("test", up, down, nil)

	if strings.Contains(result, "-- Warnings:") {
		t.Error("unexpected warnings section")
	}
}
