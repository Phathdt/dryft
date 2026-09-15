package config

import (
	"os"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "single env var",
			input:    "postgresql://${DB_USER}@localhost/db",
			envVars:  map[string]string{"DB_USER": "testuser"},
			expected: "postgresql://testuser@localhost/db",
		},
		{
			name:     "multiple env vars",
			input:    "${DB_HOST}:${DB_PORT}",
			envVars:  map[string]string{"DB_HOST": "localhost", "DB_PORT": "5432"},
			expected: "localhost:5432",
		},
		{
			name:     "no env vars",
			input:    "plain string",
			envVars:  map[string]string{},
			expected: "plain string",
		},
		{
			name:     "env var not set",
			input:    "${MISSING_VAR}",
			envVars:  map[string]string{},
			expected: "${MISSING_VAR}",
		},
		{
			name:     "empty env var value",
			input:    "${EMPTY}",
			envVars:  map[string]string{"EMPTY": ""},
			expected: "${EMPTY}", // Empty values are not expanded (left as-is)
		},
		{
			name:     "mixed text and env vars",
			input:    "prefix_${VAR}_suffix",
			envVars:  map[string]string{"VAR": "value"},
			expected: "prefix_value_suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "dryft-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	configContent := `database:
  provider: postgresql
  url: ${DATABASE_URL}

schema:
  file: prisma/schema.prisma
  naming:
    fields: camelCase
    tables: PascalCase

migration:
  directory: migrations
  format: goose
  naming: timestamp

goose:
  version_table: goose_db_version
`
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Set required env var
	os.Setenv("DATABASE_URL", "postgresql://user:pass@localhost:5432/testdb")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify config structure
	if cfg.Database.Provider != "postgresql" {
		t.Errorf("Database.Provider = %q, want %q", cfg.Database.Provider, "postgresql")
	}
	if cfg.Database.URL != "postgresql://user:pass@localhost:5432/testdb" {
		t.Errorf("Database.URL = %q, want expanded URL", cfg.Database.URL)
	}
	if cfg.Schema.File != "prisma/schema.prisma" {
		t.Errorf("Schema.File = %q, want %q", cfg.Schema.File, "prisma/schema.prisma")
	}
	if cfg.Migration.Directory != "migrations" {
		t.Errorf("Migration.Directory = %q, want %q", cfg.Migration.Directory, "migrations")
	}
}

func TestLoadInvalidFile(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file, got nil")
	}
}
