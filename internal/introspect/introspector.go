package introspect

import (
	"context"

	"github.com/phathdt/dryft/internal/schema"
)

// Introspector defines the interface for database schema introspection.
type Introspector interface {
	// Introspect retrieves the complete database schema.
	Introspect(ctx context.Context) (*schema.Schema, error)

	// IntrospectTable retrieves the schema for a specific table.
	IntrospectTable(ctx context.Context, tableName string) (*schema.Table, error)

	// Close releases any resources held by the introspector.
	Close() error
}
