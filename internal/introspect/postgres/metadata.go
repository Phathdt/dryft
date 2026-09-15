package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// detectSequenceType determines if a column uses SERIAL or IDENTITY.
// Returns "serial" for SERIAL columns, "identity" for IDENTITY columns, or "" for neither.
func (p *PostgresIntrospector) detectSequenceType(ctx context.Context, schemaName, tableName, columnName string) (string, error) {
	query := `
		SELECT d.deptype AS dependency_type
		FROM pg_depend d
		JOIN pg_attribute a ON a.attrelid = d.refobjid AND a.attnum = d.refobjsubid
		WHERE d.classid = 'pg_class'::regclass
		  AND a.attrelid = ($1 || '.' || $2)::regclass
		  AND a.attname = $3
		  AND d.deptype IN ('a', 'i')`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var depType string
	err := p.pool.QueryRow(queryCtx, query, schemaName, tableName, columnName).Scan(&depType)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", nil
		}
		return "", fmt.Errorf("query sequence dependency: %w", err)
	}

	switch depType {
	case "a":
		return "serial", nil
	case "i":
		return "identity", nil
	default:
		return "", nil
	}
}

// parseDefault parses a PostgreSQL default expression into a DefaultValue.
func (p *PostgresIntrospector) parseDefault(ctx context.Context, schemaName, tableName, columnName, expr string) (*schema.DefaultValue, error) {
	if containsNextval(expr) {
		seqType, err := p.detectSequenceType(ctx, schemaName, tableName, columnName)
		if err != nil {
			return nil, fmt.Errorf("detect sequence type: %w", err)
		}

		seqName := extractSequenceName(expr)
		if seqName == "" {
			return &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: expr,
			}, nil
		}

		return &schema.DefaultValue{
			Kind: schema.DefaultSequence,
			Sequence: &schema.SequenceRef{
				Name:  seqName,
				Owned: seqType == "serial" || seqType == "identity",
			},
		}, nil
	}

	if isExpression(expr) {
		return &schema.DefaultValue{
			Kind:       schema.DefaultExpression,
			Expression: expr,
		}, nil
	}

	return &schema.DefaultValue{
		Kind:    schema.DefaultLiteral,
		Literal: expr,
	}, nil
}

// extractSequenceName extracts the sequence name from a nextval() expression.
func extractSequenceName(expr string) string {
	start := -1
	end := -1
	inQuote := false

	for i, ch := range expr {
		if ch == '\'' {
			if !inQuote {
				start = i + 1
				inQuote = true
			} else {
				end = i
				break
			}
		}
	}

	if start > 0 && end > start {
		return expr[start:end]
	}

	return ""
}

// containsNextval checks if expression contains nextval() call.
func containsNextval(expr string) bool {
	return len(expr) >= 7 && (expr[:7] == "nextval" || strings.Contains(strings.ToLower(expr), "nextval("))
}

// isExpression determines if a default is an expression vs literal.
func isExpression(expr string) bool {
	if strings.Contains(expr, "(") && strings.Contains(expr, ")") {
		return true
	}

	keywords := []string{
		"CURRENT_TIMESTAMP",
		"CURRENT_DATE",
		"CURRENT_TIME",
		"now()",
		"gen_random_uuid()",
		"uuid_generate_v4()",
	}

	lowerExpr := strings.ToLower(expr)
	for _, kw := range keywords {
		if strings.Contains(lowerExpr, strings.ToLower(kw)) {
			return true
		}
	}

	return false
}
