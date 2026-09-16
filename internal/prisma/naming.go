package prisma

import (
	"strings"
	"unicode"
)

// NamingConvention defines how database names are transformed to Prisma names.
type NamingConvention struct {
	// Fields specifies how to transform column names (e.g., "camelCase", "snake_case").
	Fields string
	// Models specifies how to transform table names (e.g., "PascalCase", "snake_case").
	Models string
}

// DefaultNamingConvention returns the standard Prisma naming convention:
// - snake_case database columns → camelCase Prisma fields
// - snake_case database tables → PascalCase Prisma models
func DefaultNamingConvention() NamingConvention {
	return NamingConvention{
		Fields: "camelCase",
		Models: "PascalCase",
	}
}

// prismaReservedNames lists identifiers Prisma refuses as model or enum names
// because they collide with block keywords or generated client types.
var prismaReservedNames = map[string]bool{
	"model":       true,
	"enum":        true,
	"datasource":  true,
	"generator":   true,
	"type":        true,
	"view":        true,
	"String":      true,
	"Int":         true,
	"BigInt":      true,
	"Float":       true,
	"Decimal":     true,
	"Boolean":     true,
	"DateTime":    true,
	"Json":        true,
	"Bytes":       true,
	"Unsupported": true,
}

// IsReservedModelName reports whether name cannot be used as a Prisma model or
// enum name. Comparison is case-insensitive for the block keywords, which Prisma
// treats as reserved regardless of casing.
func IsReservedModelName(name string) bool {
	if prismaReservedNames[name] {
		return true
	}
	return prismaReservedNames[strings.ToLower(name)]
}

// IsReservedFieldName reports whether name cannot be used as a Prisma field name.
// Currently Prisma does not reserve field names beyond avoiding collisions within
// a model, so this returns false. The function exists for symmetry and future-proofing.
func IsReservedFieldName(name string) bool {
	return false
}

// TransformFieldName converts a database column name to a Prisma field name.
func (nc NamingConvention) TransformFieldName(dbName string) string {
	switch nc.Fields {
	case "camelCase":
		return toCamelCase(dbName)
	case "snake_case":
		return dbName
	default:
		return toCamelCase(dbName)
	}
}

// TransformModelName converts a database table name to a Prisma model name.
func (nc NamingConvention) TransformModelName(dbName string) string {
	switch nc.Models {
	case "PascalCase":
		return toPascalCase(dbName)
	case "snake_case":
		return dbName
	default:
		return toPascalCase(dbName)
	}
}

// toCamelCase converts snake_case to camelCase.
// Examples: "user_id" → "userId", "created_at" → "createdAt"
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}

	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}

	// First part stays lowercase, rest are capitalized
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			result += capitalize(parts[i])
		}
	}

	return result
}

// toPascalCase converts snake_case to PascalCase.
// Examples: "users" → "Users", "user_profiles" → "UserProfiles"
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	parts := strings.Split(s, "_")
	var result string

	for _, part := range parts {
		if part != "" {
			result += capitalize(part)
		}
	}

	return result
}

// capitalize capitalizes the first letter of a string.
func capitalize(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
