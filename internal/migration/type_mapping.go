package migration

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// mapSQLType maps a SQL type string to schema.DataType.
func mapSQLType(sqlType string, enums map[string]*schema.Enum) (schema.DataType, error) {
	// Normalize type string
	normalized := strings.TrimSpace(sqlType)

	// Check for array types (e.g., "text[]", "integer[][]")
	arrayDepth := 0
	for strings.HasSuffix(normalized, "[]") {
		arrayDepth++
		normalized = strings.TrimSuffix(normalized, "[]")
		normalized = strings.TrimSpace(normalized)
	}

	// Extract precision and scale for numeric types (e.g., "NUMERIC(10,2)", "VARCHAR(255)")
	precision := 0
	scale := 0
	typeBase := normalized

	// Match patterns like: TYPE(precision) or TYPE(precision,scale)
	re := regexp.MustCompile(`^(\w+(?:\s+\w+)*)\((\d+)(?:,\s*(\d+))?\)`)
	if matches := re.FindStringSubmatch(normalized); matches != nil {
		typeBase = matches[1]
		if p, err := strconv.Atoi(matches[2]); err == nil {
			precision = p
		}
		if matches[3] != "" {
			if s, err := strconv.Atoi(matches[3]); err == nil {
				scale = s
			}
		}
	}

	// Remove quotes from type name (e.g., "UserRole" → UserRole)
	typeBase = strings.Trim(typeBase, `"`)

	// Normalize to uppercase for comparison
	typeUpper := strings.ToUpper(typeBase)

	// Handle "WITH TIME ZONE" and "WITHOUT TIME ZONE"
	typeUpper = strings.ReplaceAll(typeUpper, " WITH TIME ZONE", "TZ")
	typeUpper = strings.ReplaceAll(typeUpper, " WITHOUT TIME ZONE", "")
	typeUpper = strings.TrimSpace(typeUpper)

	// Map to TypeKind
	var kind schema.TypeKind
	var enumName string

	switch typeUpper {
	// Integer types
	case "INTEGER", "INT", "INT4":
		kind = schema.TypeInt32
	case "BIGINT", "INT8":
		kind = schema.TypeInt64
	case "SMALLINT", "INT2":
		kind = schema.TypeInt32 // Map smallint to int32 for simplicity
	case "SERIAL", "SERIAL4":
		kind = schema.TypeInt32
	case "BIGSERIAL", "SERIAL8":
		kind = schema.TypeInt64

	// Text types
	case "TEXT":
		kind = schema.TypeText
	case "VARCHAR", "CHARACTER VARYING":
		kind = schema.TypeVarChar
	case "CHAR", "CHARACTER", "BPCHAR":
		kind = schema.TypeChar

	// UUID
	case "UUID":
		kind = schema.TypeUUID

	// Boolean
	case "BOOLEAN", "BOOL":
		kind = schema.TypeBool

	// Floating point
	case "REAL", "FLOAT4":
		kind = schema.TypeFloat32
	case "DOUBLE PRECISION", "FLOAT8":
		kind = schema.TypeFloat64

	// Numeric/Decimal
	case "NUMERIC", "DECIMAL":
		kind = schema.TypeNumeric

	// Timestamp types
	case "TIMESTAMP", "TIMESTAMPTZ":
		if strings.Contains(typeUpper, "TZ") {
			kind = schema.TypeTimestampTZ
		} else {
			kind = schema.TypeTimestamp
		}
	case "DATE":
		kind = schema.TypeDate

	// JSON types
	case "JSON":
		kind = schema.TypeJSON
	case "JSONB":
		kind = schema.TypeJSONB

	// Binary data
	case "BYTEA":
		kind = schema.TypeBytes

	default:
		// Check if it's a registered enum type
		if _, exists := enums[typeBase]; exists {
			kind = schema.TypeEnum
			enumName = typeBase
		} else {
			return schema.DataType{}, fmt.Errorf("unsupported SQL type: %s", sqlType)
		}
	}

	return schema.DataType{
		Kind:       kind,
		Precision:  precision,
		Scale:      scale,
		ArrayDepth: arrayDepth,
		EnumName:   enumName,
	}, nil
}
