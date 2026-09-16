package schema

import (
	"fmt"
	"strings"
)

// PostgreSQLToInternal maps PostgreSQL type names to internal TypeKind.
var PostgreSQLToInternal = map[string]TypeKind{
	"uuid":                        TypeUUID,
	"text":                        TypeText,
	"varchar":                     TypeVarChar,
	"character varying":           TypeVarChar,
	"char":                        TypeChar,
	"bpchar":                      TypeChar,
	"character":                   TypeChar,
	"integer":                     TypeInt32,
	"int":                         TypeInt32,
	"int4":                        TypeInt32,
	"smallint":                    TypeInt32,
	"int2":                        TypeInt32,
	"bigint":                      TypeInt64,
	"int8":                        TypeInt64,
	"boolean":                     TypeBool,
	"bool":                        TypeBool,
	"real":                        TypeFloat32,
	"float4":                      TypeFloat32,
	"double precision":            TypeFloat64,
	"float8":                      TypeFloat64,
	"numeric":                     TypeNumeric,
	"decimal":                     TypeNumeric,
	"timestamp":                   TypeTimestamp,
	"timestamp without time zone": TypeTimestamp,
	"timestamptz":                 TypeTimestampTZ,
	"timestamp with time zone":    TypeTimestampTZ,
	"date":                        TypeDate,
	"json":                        TypeJSON,
	"jsonb":                       TypeJSONB,
	"bytea":                       TypeBytes,
}

// ParsePostgreSQLType parses a PostgreSQL type string into a DataType.
// Examples:
//   - "uuid" → DataType{Kind: TypeUUID}
//   - "varchar(100)" → DataType{Kind: TypeVarChar, Precision: &100}
//   - "numeric(10,2)" → DataType{Kind: TypeNumeric, Precision: &10, Scale: &2}
//   - "text[]" → DataType{Kind: TypeText, ArrayDepth: 1}
func ParsePostgreSQLType(pgType string) (DataType, error) {
	if pgType == "" {
		return DataType{}, fmt.Errorf("empty type string")
	}

	// Count and strip array brackets
	arrayDepth := 0
	typeName := pgType
	for strings.HasSuffix(typeName, "[]") {
		arrayDepth++
		typeName = strings.TrimSuffix(typeName, "[]")
	}

	// Parse precision/scale if present
	var precision, scale int
	var baseType string

	if idx := strings.Index(typeName, "("); idx != -1 {
		baseType = strings.TrimSpace(typeName[:idx])
		paramsStr := strings.TrimSuffix(typeName[idx+1:], ")")

		if strings.Contains(paramsStr, ",") {
			// numeric(10,2)
			var err error
			if _, err = fmt.Sscanf(paramsStr, "%d,%d", &precision, &scale); err != nil {
				return DataType{}, fmt.Errorf("invalid precision/scale format in type %q", pgType)
			}
		} else {
			// varchar(100)
			var err error
			if _, err = fmt.Sscanf(paramsStr, "%d", &precision); err != nil {
				return DataType{}, fmt.Errorf("invalid precision format in type %q", pgType)
			}
		}
	} else {
		baseType = typeName
	}

	baseType = strings.ToLower(strings.TrimSpace(baseType))

	kind, ok := PostgreSQLToInternal[baseType]
	if !ok {
		return DataType{}, fmt.Errorf("unsupported PostgreSQL type: %q", baseType)
	}

	return DataType{
		Kind:       kind,
		Precision:  precision,
		Scale:      scale,
		ArrayDepth: arrayDepth,
	}, nil
}

// ToPrisma returns the Prisma schema notation for this DataType.
// Examples:
//   - TypeUUID → "String @db.Uuid"
//   - TypeText → "String"
//   - TypeInt32 → "Int"
//   - TypeInt64 → "BigInt"
//   - TypeTimestampTZ → "DateTime @db.Timestamptz"
//   - TypeJSONB → "Json @db.JsonB"
//   - TypeVarChar(100) → "String @db.VarChar(100)"
//   - TypeNumeric(10,2) → "Decimal @db.Numeric(10,2)"
//   - TypeText[] → "String[]"
func (dt DataType) ToPrisma() string {
	var baseType string
	var dbAnnotation string

	switch dt.Kind {
	case TypeUUID:
		baseType = "String"
		dbAnnotation = "@db.Uuid"
	case TypeText:
		baseType = "String"
	case TypeVarChar:
		baseType = "String"
		if dt.Precision > 0 {
			dbAnnotation = fmt.Sprintf("@db.VarChar(%d)", dt.Precision)
		} else {
			dbAnnotation = "@db.VarChar"
		}
	case TypeChar:
		baseType = "String"
		if dt.Precision > 0 {
			dbAnnotation = fmt.Sprintf("@db.Char(%d)", dt.Precision)
		} else {
			dbAnnotation = "@db.Char"
		}
	case TypeInt32:
		baseType = "Int"
	case TypeInt64:
		baseType = "BigInt"
	case TypeBool:
		baseType = "Boolean"
	case TypeFloat32:
		baseType = "Float"
		dbAnnotation = "@db.Real"
	case TypeFloat64:
		baseType = "Float"
	case TypeNumeric:
		baseType = "Decimal"
		if dt.Precision > 0 && dt.Scale > 0 {
			dbAnnotation = fmt.Sprintf("@db.Numeric(%d,%d)", dt.Precision, dt.Scale)
		} else if dt.Precision > 0 {
			dbAnnotation = fmt.Sprintf("@db.Numeric(%d)", dt.Precision)
		}
	case TypeTimestamp:
		baseType = "DateTime"
	case TypeTimestampTZ:
		baseType = "DateTime"
		dbAnnotation = "@db.Timestamptz"
	case TypeDate:
		baseType = "DateTime"
		dbAnnotation = "@db.Date"
	case TypeJSON:
		baseType = "Json"
	case TypeJSONB:
		baseType = "Json"
		dbAnnotation = "@db.JsonB"
	case TypeBytes:
		baseType = "Bytes"
	case TypeEnum:
		baseType = "String"
	default:
		baseType = "String"
	}

	// Add array notation
	for i := 0; i < dt.ArrayDepth; i++ {
		baseType += "[]"
	}

	// Combine base type and annotation
	if dbAnnotation != "" {
		return baseType + " " + dbAnnotation
	}
	return baseType
}

// ToPostgreSQLDDL returns the PostgreSQL DDL type notation for this DataType.
// Examples:
//   - TypeUUID → "UUID"
//   - TypeText → "TEXT"
//   - TypeVarChar(100) → "VARCHAR(100)"
//   - TypeNumeric(10,2) → "NUMERIC(10,2)"
//   - TypeTimestampTZ → "TIMESTAMPTZ"
//   - TypeText[] → "TEXT[]"
func (dt DataType) ToPostgreSQLDDL() string {
	var baseType string

	switch dt.Kind {
	case TypeUUID:
		baseType = "UUID"
	case TypeText:
		baseType = "TEXT"
	case TypeVarChar:
		if dt.Precision > 0 {
			baseType = fmt.Sprintf("VARCHAR(%d)", dt.Precision)
		} else {
			baseType = "VARCHAR"
		}
	case TypeChar:
		if dt.Precision > 0 {
			baseType = fmt.Sprintf("CHAR(%d)", dt.Precision)
		} else {
			baseType = "CHAR"
		}
	case TypeInt32:
		baseType = "INTEGER"
	case TypeInt64:
		baseType = "BIGINT"
	case TypeBool:
		baseType = "BOOLEAN"
	case TypeFloat32:
		baseType = "REAL"
	case TypeFloat64:
		baseType = "DOUBLE PRECISION"
	case TypeNumeric:
		if dt.Precision > 0 && dt.Scale > 0 {
			baseType = fmt.Sprintf("NUMERIC(%d,%d)", dt.Precision, dt.Scale)
		} else if dt.Precision > 0 {
			baseType = fmt.Sprintf("NUMERIC(%d)", dt.Precision)
		} else {
			baseType = "NUMERIC"
		}
	case TypeTimestamp:
		baseType = "TIMESTAMP"
	case TypeTimestampTZ:
		baseType = "TIMESTAMPTZ"
	case TypeDate:
		baseType = "DATE"
	case TypeJSON:
		baseType = "JSON"
	case TypeJSONB:
		baseType = "JSONB"
	case TypeBytes:
		baseType = "BYTEA"
	case TypeEnum:
		baseType = "TEXT"
	default:
		baseType = "TEXT"
	}

	// Add array notation
	for i := 0; i < dt.ArrayDepth; i++ {
		baseType += "[]"
	}

	return baseType
}
