package config

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Database: DatabaseConfig{
					Provider: "postgresql",
					URL:      "postgresql://localhost/db",
				},
				Schema: SchemaConfig{
					File: "schema.prisma",
				},
				Migration: MigrationConfig{
					Directory: "migrations",
					Format:    "goose",
				},
			},
			wantErr: false,
		},
		{
			name: "missing database URL",
			config: &Config{
				Database: DatabaseConfig{
					Provider: "postgresql",
					URL:      "",
				},
				Schema: SchemaConfig{
					File: "schema.prisma",
				},
				Migration: MigrationConfig{
					Directory: "migrations",
					Format:    "goose",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid database provider",
			config: &Config{
				Database: DatabaseConfig{
					Provider: "mysql",
					URL:      "mysql://localhost/db",
				},
				Schema: SchemaConfig{
					File: "schema.prisma",
				},
				Migration: MigrationConfig{
					Directory: "migrations",
					Format:    "goose",
				},
			},
			wantErr: true,
		},
		{
			name: "missing schema file",
			config: &Config{
				Database: DatabaseConfig{
					Provider: "postgresql",
					URL:      "postgresql://localhost/db",
				},
				Schema: SchemaConfig{
					File: "",
				},
				Migration: MigrationConfig{
					Directory: "migrations",
					Format:    "goose",
				},
			},
			wantErr: true,
		},
		{
			name: "missing migration directory",
			config: &Config{
				Database: DatabaseConfig{
					Provider: "postgresql",
					URL:      "postgresql://localhost/db",
				},
				Schema: SchemaConfig{
					File: "schema.prisma",
				},
				Migration: MigrationConfig{
					Directory: "",
					Format:    "goose",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid migration format",
			config: &Config{
				Database: DatabaseConfig{
					Provider: "postgresql",
					URL:      "postgresql://localhost/db",
				},
				Schema: SchemaConfig{
					File: "schema.prisma",
				},
				Migration: MigrationConfig{
					Directory: "migrations",
					Format:    "migrate",
				},
			},
			wantErr: true,
		},
		{
			name: "unexpanded env var in URL",
			config: &Config{
				Database: DatabaseConfig{
					Provider: "postgresql",
					URL:      "${DATABASE_URL}",
				},
				Schema: SchemaConfig{
					File: "schema.prisma",
				},
				Migration: MigrationConfig{
					Directory: "migrations",
					Format:    "goose",
				},
			},
			wantErr: false, // Currently passes - design decision needed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
