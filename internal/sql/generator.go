package sql

import "github.com/phathdt/dryft/internal/diff"

// Generator generates SQL statements from operations.
type Generator interface {
	// Generate produces forward SQL statements from operations.
	Generate(ops []diff.Operation) ([]string, error)

	// GenerateReverse produces reverse SQL statements (Down migration).
	// Returns SQL statements and warnings for irreversible operations.
	GenerateReverse(ops []diff.Operation) ([]string, []string, error)
}

// GeneratorOptions configures SQL generation behavior.
type GeneratorOptions struct {
	// UseIfNotExists adds IF NOT EXISTS clauses where supported.
	UseIfNotExists bool

	// QuoteIdentifiers forces quoting of all identifiers.
	QuoteIdentifiers bool

	// Dialect specifies the SQL dialect.
	Dialect string
}
