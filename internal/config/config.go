package config

type Config struct {
	Database  DatabaseConfig  `yaml:"database"`
	Schema    SchemaConfig    `yaml:"schema"`
	Migration MigrationConfig `yaml:"migration"`
	Goose     GooseConfig     `yaml:"goose"`
}

type DatabaseConfig struct {
	Provider string `yaml:"provider"`
	URL      string `yaml:"url"`
}

type SchemaConfig struct {
	File   string       `yaml:"file"`
	Naming NamingConfig `yaml:"naming"`
}

type NamingConfig struct {
	Fields string `yaml:"fields"`
	Tables string `yaml:"tables"`
}

type MigrationConfig struct {
	Directory string `yaml:"directory"`
	Format    string `yaml:"format"`
	Naming    string `yaml:"naming"`
}

type GooseConfig struct {
	VersionTable string `yaml:"version_table"`
}
