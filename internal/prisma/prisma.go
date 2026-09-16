package prisma

import (
	"fmt"

	"github.com/phathdt/dryft/internal/schema"
)

// ParseResult contains the parsed schema and any warnings.
type ParseResult struct {
	Schema   *schema.Schema
	Warnings []string
}

// Parse parses a Prisma schema string and returns the Internal Schema representation.
// It returns both the schema and any warnings (e.g., unsupported features, unknown attributes).
func Parse(input string) (*ParseResult, error) {
	// Lexer → Parser → AST
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// AST → Internal Schema
	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		return nil, fmt.Errorf("conversion error: %w", err)
	}

	return &ParseResult{
		Schema:   s,
		Warnings: converter.Warnings(),
	}, nil
}

// ParseFile parses a Prisma schema from a file path.
func ParseFile(path string) (*ParseResult, error) {
	// This would read the file, but for now we'll keep it simple
	// and let the caller read the file and pass the content to Parse
	return nil, fmt.Errorf("ParseFile not implemented - use Parse() with file content")
}
