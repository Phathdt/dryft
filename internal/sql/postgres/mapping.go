package postgres

import (
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// quoteIdentifier wraps an identifier in double quotes if needed.
func quoteIdentifier(name string) string {
	// Check if identifier needs quoting
	if needsQuoting(name) {
		return fmt.Sprintf(`"%s"`, name)
	}
	return name
}

// needsQuoting checks if an identifier needs quoting.
func needsQuoting(name string) bool {
	// Always quote if contains uppercase, special chars, or is reserved word
	if isReservedWord(name) {
		return true
	}

	// Check for uppercase letters
	for _, ch := range name {
		if ch >= 'A' && ch <= 'Z' {
			return true
		}
	}

	return false
}

// isReservedWord checks if a word is a PostgreSQL reserved keyword.
func isReservedWord(word string) bool {
	reserved := map[string]bool{
		"user":     true,
		"table":    true,
		"column":   true,
		"index":    true,
		"primary":  true,
		"foreign":  true,
		"key":      true,
		"default":  true,
		"order":    true,
		"group":    true,
		"select":   true,
		"insert":   true,
		"update":   true,
		"delete":   true,
		"create":   true,
		"drop":     true,
		"alter":    true,
		"grant":    true,
		"revoke":   true,
		"sequence": true,
	}
	return reserved[strings.ToLower(word)]
}

// formatColumnList formats a list of column names.
func formatColumnList(columns []string) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdentifier(col)
	}
	return strings.Join(quoted, ", ")
}

// mapDataType converts internal DataType to PostgreSQL SQL type string.
func mapDataType(dt schema.DataType) string {
	var baseType string

	switch dt.Kind {
	case schema.TypeUUID:
		baseType = "UUID"
	case schema.TypeText:
		baseType = "TEXT"
	case schema.TypeVarChar:
		if dt.Precision > 0 {
			baseType = fmt.Sprintf("VARCHAR(%d)", dt.Precision)
		} else {
			baseType = "VARCHAR"
		}
	case schema.TypeChar:
		if dt.Precision > 0 {
			baseType = fmt.Sprintf("CHAR(%d)", dt.Precision)
		} else {
			baseType = "CHAR"
		}
	case schema.TypeInt32:
		baseType = "INTEGER"
	case schema.TypeInt64:
		baseType = "BIGINT"
	case schema.TypeBool:
		baseType = "BOOLEAN"
	case schema.TypeFloat32:
		baseType = "REAL"
	case schema.TypeFloat64:
		baseType = "DOUBLE PRECISION"
	case schema.TypeNumeric:
		if dt.Precision > 0 && dt.Scale > 0 {
			baseType = fmt.Sprintf("NUMERIC(%d,%d)", dt.Precision, dt.Scale)
		} else if dt.Precision > 0 {
			baseType = fmt.Sprintf("NUMERIC(%d)", dt.Precision)
		} else {
			baseType = "NUMERIC"
		}
	case schema.TypeTimestamp:
		baseType = "TIMESTAMP"
	case schema.TypeTimestampTZ:
		baseType = "TIMESTAMPTZ"
	case schema.TypeDate:
		baseType = "DATE"
	case schema.TypeJSON:
		baseType = "JSON"
	case schema.TypeJSONB:
		baseType = "JSONB"
	case schema.TypeBytes:
		baseType = "BYTEA"
	case schema.TypeEnum:
		baseType = quoteIdentifier(dt.EnumName)
	default:
		baseType = "TEXT" // Fallback
	}

	// Handle arrays
	if dt.ArrayDepth > 0 {
		for i := 0; i < dt.ArrayDepth; i++ {
			baseType += "[]"
		}
	}

	return baseType
}

// formatDefault formats a default value for SQL.
func formatDefault(dv *schema.DefaultValue) string {
	if dv == nil {
		return ""
	}

	switch dv.Kind {
	case schema.DefaultLiteral:
		return dv.Literal
	case schema.DefaultExpression:
		return dv.Expression
	case schema.DefaultSequence:
		if dv.Sequence != nil {
			return fmt.Sprintf("nextval('%s'::regclass)", dv.Sequence.Name)
		}
		return ""
	default:
		return ""
	}
}
