package prisma

import (
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// Converter converts Prisma AST to Internal Schema.
type Converter struct {
	warnings  []string
	enumNames map[string]bool
}

// NewConverter creates a new AST to Internal Schema converter.
func NewConverter() *Converter {
	return &Converter{
		warnings:  []string{},
		enumNames: make(map[string]bool),
	}
}

// Convert converts a Prisma AST schema to Internal Schema.
func (c *Converter) Convert(ast *Schema) (*schema.Schema, error) {
	s := &schema.Schema{
		Tables: []schema.Table{},
		Enums:  []schema.Enum{},
	}

	// First pass: collect enum names for relation detection
	enumNames := make(map[string]bool)
	for _, decl := range ast.Declarations {
		if enum, ok := decl.(*EnumDeclaration); ok {
			enumNames[enum.Name] = true
		}
	}
	c.enumNames = enumNames

	// Second pass: convert declarations
	for _, decl := range ast.Declarations {
		switch d := decl.(type) {
		case *ModelDeclaration:
			table, err := c.convertModel(d)
			if err != nil {
				return nil, fmt.Errorf("failed to convert model %q: %w", d.Name, err)
			}
			s.Tables = append(s.Tables, *table)
		case *EnumDeclaration:
			enum := c.convertEnum(d)
			s.Enums = append(s.Enums, *enum)
		case *DatasourceDeclaration:
			// MVP: datasource is parsed but not used in internal schema
			continue
		case *GeneratorDeclaration:
			// MVP: generator is parsed but not used in internal schema
			continue
		}
	}

	return s, nil
}

// convertModel converts a ModelDeclaration to schema.Table.
func (c *Converter) convertModel(model *ModelDeclaration) (*schema.Table, error) {
	table := &schema.Table{
		Name:        model.Name,
		Columns:     []schema.Column{},
		Constraints: []schema.Constraint{},
		Indexes:     []schema.Index{},
	}

	// Check for @@map attribute to get actual table name
	for _, attr := range model.Attributes {
		if attr.Name == "map" && len(attr.Args) > 0 {
			if tableName, ok := attr.Args[0].Value.(string); ok {
				table.Name = tableName
			}
		}
	}

	// Convert fields to columns
	var primaryKeyFields []string
	var uniqueFields []string

	for _, field := range model.Fields {
		// Skip relation fields (MVP: not supported)
		if c.isRelationField(field) {
			c.addWarning(fmt.Sprintf("model %s: relation field %q skipped (relations not supported in MVP)",
				model.Name, field.Name))
			continue
		}

		col, err := c.convertField(field, model.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %q: %w", field.Name, err)
		}

		// Check for @id attribute
		if c.hasAttribute(field.Attributes, "id") {
			primaryKeyFields = append(primaryKeyFields, col.Name)
		}

		// Check for @unique attribute
		if c.hasAttribute(field.Attributes, "unique") {
			uniqueFields = append(uniqueFields, col.Name)
		}

		table.Columns = append(table.Columns, *col)
	}

	// Process model-level attributes
	for _, attr := range model.Attributes {
		switch attr.Name {
		case "id":
			// Composite primary key: @@id([field1, field2])
			if len(attr.Args) > 0 {
				if fields, ok := attr.Args[0].Value.([]string); ok {
					table.PrimaryKey = &schema.PrimaryKey{
						Columns: fields,
					}
				}
			}
		case "unique":
			// Composite unique constraint: @@unique([field1, field2])
			if len(attr.Args) > 0 {
				if fields, ok := attr.Args[0].Value.([]string); ok {
					table.Constraints = append(table.Constraints, schema.Constraint{
						Type:    schema.ConstraintUnique,
						Columns: fields,
					})
				}
			}
		case "index":
			// Index: @@index([field1, field2])
			if len(attr.Args) > 0 {
				if fields, ok := attr.Args[0].Value.([]string); ok {
					indexCols := make([]schema.IndexColumn, len(fields))
					for i, f := range fields {
						indexCols[i] = schema.IndexColumn{Name: f}
					}
					table.Indexes = append(table.Indexes, schema.Index{
						Columns: indexCols,
						Unique:  false,
					})
				}
			}
		case "map":
			// Already handled above
		default:
			c.addWarning(fmt.Sprintf("model %s: unknown model attribute @@%s", model.Name, attr.Name))
		}
	}

	// Set primary key from field-level @id if not set by @@id
	if table.PrimaryKey == nil && len(primaryKeyFields) > 0 {
		table.PrimaryKey = &schema.PrimaryKey{
			Columns: primaryKeyFields,
		}
	}

	// Add single-column unique constraints
	for _, field := range uniqueFields {
		table.Constraints = append(table.Constraints, schema.Constraint{
			Type:    schema.ConstraintUnique,
			Columns: []string{field},
		})
	}

	return table, nil
}

// convertField converts a Field to schema.Column.
func (c *Converter) convertField(field Field, modelName string) (*schema.Column, error) {
	col := &schema.Column{
		Name:     field.Name,
		Nullable: field.Type.Optional,
	}

	// Check for @map attribute to get actual column name
	for _, attr := range field.Attributes {
		if attr.Name == "map" && len(attr.Args) > 0 {
			if colName, ok := attr.Args[0].Value.(string); ok {
				col.Name = colName
			}
		}
	}

	// Convert Prisma type to Internal DataType
	dataType, err := c.convertType(field.Type, field.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert type for field %q: %w", field.Name, err)
	}
	col.Type = dataType

	// Extract @default attribute
	for _, attr := range field.Attributes {
		if attr.Name == "default" {
			defaultVal := c.convertDefault(attr)
			if defaultVal != nil {
				col.Default = defaultVal
			}
		}
	}

	return col, nil
}

// convertType converts Prisma FieldType to Internal DataType.
func (c *Converter) convertType(ft FieldType, attrs []FieldAttribute) (schema.DataType, error) {
	dt := schema.DataType{}

	// Check for array type
	if ft.List {
		dt.ArrayDepth = 1
	}

	// Check for @db.* annotation for specific database type
	dbType := c.getDBAnnotation(attrs)

	// Map Prisma type to Internal type
	switch ft.Name {
	case "String":
		if dbType == "Uuid" {
			dt.Kind = schema.TypeUUID
		} else if dbType == "Text" {
			dt.Kind = schema.TypeText
		} else {
			dt.Kind = schema.TypeText // Default String → Text
		}
	case "Int":
		dt.Kind = schema.TypeInt32
	case "BigInt":
		dt.Kind = schema.TypeInt64
	case "Boolean":
		dt.Kind = schema.TypeBool
	case "DateTime":
		if dbType == "Timestamptz" {
			dt.Kind = schema.TypeTimestampTZ
		} else {
			dt.Kind = schema.TypeTimestamp
		}
	case "Json":
		if dbType == "JsonB" {
			dt.Kind = schema.TypeJSONB
		} else {
			dt.Kind = schema.TypeJSON
		}
	case "Float":
		dt.Kind = schema.TypeFloat64
	case "Decimal":
		dt.Kind = schema.TypeNumeric
	case "Bytes":
		dt.Kind = schema.TypeBytes
	default:
		// Assume it's an enum type
		dt.Kind = schema.TypeEnum
		dt.EnumName = ft.Name
	}

	return dt, nil
}

// getDBAnnotation extracts @db.* annotation value.
func (c *Converter) getDBAnnotation(attrs []FieldAttribute) string {
	for _, attr := range attrs {
		if strings.HasPrefix(attr.Name, "db.") {
			return strings.TrimPrefix(attr.Name, "db.")
		}
	}
	return ""
}

// convertDefault converts a @default attribute to schema.DefaultValue.
func (c *Converter) convertDefault(attr FieldAttribute) *schema.DefaultValue {
	if len(attr.Args) == 0 {
		return nil
	}

	arg := attr.Args[0]

	// Check if it's a function call
	if fc, ok := arg.Value.(*FunctionCall); ok {
		switch strings.ToLower(fc.Name) {
		case "autoincrement":
			return &schema.DefaultValue{
				Kind: schema.DefaultSequence,
			}
		case "now":
			return &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "now()",
			}
		case "uuid":
			return &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "gen_random_uuid()",
			}
		case "dbgenerated":
			// Extract expression from dbgenerated("...")
			if len(fc.Args) > 0 {
				if expr, ok := fc.Args[0].Value.(string); ok {
					return &schema.DefaultValue{
						Kind:       schema.DefaultExpression,
						Expression: expr,
					}
				}
			}
			return nil
		default:
			c.addWarning(fmt.Sprintf("unknown default function: %s()", fc.Name))
			return nil
		}
	}

	// Literal value
	switch v := arg.Value.(type) {
	case string:
		return &schema.DefaultValue{
			Kind:    schema.DefaultLiteral,
			Literal: fmt.Sprintf("'%s'", v),
		}
	case int:
		return &schema.DefaultValue{
			Kind:    schema.DefaultLiteral,
			Literal: fmt.Sprintf("%d", v),
		}
	case float64:
		return &schema.DefaultValue{
			Kind:    schema.DefaultLiteral,
			Literal: fmt.Sprintf("%f", v),
		}
	case bool:
		return &schema.DefaultValue{
			Kind:    schema.DefaultLiteral,
			Literal: fmt.Sprintf("%t", v),
		}
	default:
		return nil
	}
}

// convertEnum converts an EnumDeclaration to schema.Enum.
func (c *Converter) convertEnum(enum *EnumDeclaration) *schema.Enum {
	e := &schema.Enum{
		Name:   enum.Name,
		Values: []schema.EnumValue{},
	}

	for _, val := range enum.Values {
		e.Values = append(e.Values, schema.EnumValue{
			Label: val.Name,
		})
	}

	return e
}

// isRelationField checks if a field is a relation (model reference).
func (c *Converter) isRelationField(field Field) bool {
	// In Prisma, relation fields are identified by:
	// 1. Type is not a built-in scalar type
	// 2. Type is not an enum
	// 3. Type refers to another model (starts with uppercase)
	typeName := field.Type.Name

	// Built-in types
	builtins := map[string]bool{
		"String":   true,
		"Int":      true,
		"BigInt":   true,
		"Float":    true,
		"Decimal":  true,
		"Boolean":  true,
		"DateTime": true,
		"Json":     true,
		"Bytes":    true,
	}

	// If it's a builtin, it's not a relation
	if builtins[typeName] {
		return false
	}

	// If it's an enum, it's not a relation
	if c.enumNames[typeName] {
		return false
	}

	// If it starts with uppercase, it's likely a relation
	if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
		return true
	}

	return false
}

// hasAttribute checks if a field has a specific attribute.
func (c *Converter) hasAttribute(attrs []FieldAttribute, name string) bool {
	for _, attr := range attrs {
		if attr.Name == name {
			return true
		}
	}
	return false
}

// addWarning adds a warning message.
func (c *Converter) addWarning(msg string) {
	c.warnings = append(c.warnings, msg)
}

// Warnings returns all collected warnings.
func (c *Converter) Warnings() []string {
	return c.warnings
}
