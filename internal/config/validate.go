package config

import (
	"fmt"
	"strings"
)

// Validate checks the configuration for required fields and valid values.
func (c *Config) Validate() error {
	var errors []string

	if c.Database.URL == "" {
		errors = append(errors, "database.url is required")
	}

	if c.Database.Provider != "postgresql" {
		errors = append(errors, "database.provider must be 'postgresql'")
	}

	if c.Schema.File == "" {
		errors = append(errors, "schema.file is required")
	}

	if c.Migration.Directory == "" {
		errors = append(errors, "migration.directory is required")
	}

	if c.Migration.Format != "goose" {
		errors = append(errors, "migration.format must be 'goose'")
	}

	if len(errors) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
