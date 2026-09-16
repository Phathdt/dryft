// Package config provides configuration management for dryft.
package config

// Config represents the main configuration structure for dryft.
type Config struct {
	Database  DatabaseConfig  `yaml:"database"`
	Schema    SchemaConfig    `yaml:"schema"`
	Migration MigrationConfig `yaml:"migration"`
	Goose     GooseConfig     `yaml:"goose"`
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Provider string `yaml:"provider"`
	URL      string `yaml:"url"`
}

// SchemaConfig defines schema file location and naming conventions.
type SchemaConfig struct {
	File   string       `yaml:"file"`
	Naming NamingConfig `yaml:"naming"`
}

// NamingConfig specifies naming conventions for fields and tables.
type NamingConfig struct {
	Fields string `yaml:"fields"`
	Tables string `yaml:"tables"`
}

// MigrationConfig configures migration generation settings.
type MigrationConfig struct {
	Directory string `yaml:"directory"`
	Format    string `yaml:"format"`
	Naming    string `yaml:"naming"`
}

// GooseConfig holds Goose-specific configuration options.
type GooseConfig struct {
	VersionTable string `yaml:"version_table"`
}
