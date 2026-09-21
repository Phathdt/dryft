package prisma

import (
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// Converter converts Prisma AST to Internal Schema.
type Converter struct {
	warnings           []string
	enumNames          map[string]bool
	modelToTableMap    map[string]string            // Maps Prisma model names to DB table names
	modelFieldToDbCol  map[string]map[string]string // Maps (modelName → (prismaField → dbColumn))
}

// NewConverter creates a new AST to Internal Schema converter.
func NewConverter() *Converter {
	return &Converter{
		warnings:          []string{},
		enumNames:         make(map[string]bool),
		modelToTableMap:   make(map[string]string),
		modelFieldToDbCol: make(map[string]map[string]string),
	}
}

// Convert converts a Prisma AST schema to Internal Schema.
func (c *Converter) Convert(ast *Schema) (*schema.Schema, error) {
	s := &schema.Schema{
		Tables: []schema.Table{},
		Enums:  []schema.Enum{},
	}

	// First pass: collect enum names, model→table mappings, and field mappings
	enumNames := make(map[string]bool)
	modelToTableMap := make(map[string]string)
	modelFieldToDbCol := make(map[string]map[string]string)

	for _, decl := range ast.Declarations {
		if enum, ok := decl.(*EnumDeclaration); ok {
			enumNames[enum.Name] = true
		}

		if model, ok := decl.(*ModelDeclaration); ok {
			// Default: model name = table name
			tableName := model.Name

			// Check for @@map attribute
			for _, attr := range model.Attributes {
				if attr.Name == "map" && len(attr.Args) > 0 {
					if mapped, ok := attr.Args[0].Value.(string); ok {
						tableName = mapped
						break
					}
				}
			}

			modelToTableMap[model.Name] = tableName

			// Build field→column mapping for this model
			fieldMap := make(map[string]string)
			for _, field := range model.Fields {
				// Skip relation fields (type is another model, capitalized)
				// Scalar types: String, Int, BigInt, Float, Decimal, Boolean, DateTime, Json, Bytes
				// Enum types: custom enum names (also capitalized, but we'll include all scalar fields)
				// Relation types: another Model (we skip these)

				// Simple heuristic: if field type is not in our scalar list and is capitalized,
				// it's likely a relation. But to be safe, we include everything except known relations.
				// Actually, simpler: just map all fields here. Relation fields won't hurt.

				prismaFieldName := field.Name
				dbColumnName := prismaFieldName // default

				// Check for @map attribute
				for _, attr := range field.Attributes {
					if attr.Name == "map" && len(attr.Args) > 0 {
						if mapped, ok := attr.Args[0].Value.(string); ok {
							dbColumnName = mapped
							break
						}
					}
				}

				fieldMap[prismaFieldName] = dbColumnName
			}
			modelFieldToDbCol[model.Name] = fieldMap
		}
	}

	c.enumNames = enumNames
	c.modelToTableMap = modelToTableMap
	c.modelFieldToDbCol = modelFieldToDbCol

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

	// Third pass: validate FK reference columns exist in target tables
	if err := c.validateForeignKeyReferences(s); err != nil {
		return nil, err
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
		// Skip relation fields - they will be processed separately for FK generation
		if c.isRelationField(field) {
			// Relation fields don't create columns, only FKs
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

	// Build Prisma field name → DB column name mapping (scalar fields only)
	prismaToDbFieldMap := make(map[string]string)
	scalarFields := make(map[string]bool)
	relationFields := make(map[string]bool)

	for _, field := range model.Fields {
		// Track relation fields separately
		if c.isRelationField(field) {
			relationFields[field.Name] = true
			continue
		}

		prismaFieldName := field.Name
		dbColumnName := prismaFieldName // default: same as Prisma name

		// Check for @map attribute
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Args) > 0 {
				if mapped, ok := attr.Args[0].Value.(string); ok {
					dbColumnName = mapped
					break
				}
			}
		}
		prismaToDbFieldMap[prismaFieldName] = dbColumnName
		scalarFields[prismaFieldName] = true
	}

	// Store field mappings for this model in global map (used for FK reference validation)
	c.modelFieldToDbCol[model.Name] = prismaToDbFieldMap

	// Process model-level attributes
	for _, attr := range model.Attributes {
		switch attr.Name {
		case "id":
			// Composite primary key: @@id([field1, field2])
			if len(attr.Args) > 0 {
				if fields, ok := attr.Args[0].Value.([]string); ok {
					dbColumns, err := c.mapFieldNames(fields, prismaToDbFieldMap, scalarFields, relationFields, model.Name)
					if err != nil {
						return nil, err
					}
					table.PrimaryKey = &schema.PrimaryKey{
						Columns: dbColumns,
					}
				}
			}
		case "unique":
			// Composite unique constraint: @@unique([field1, field2])
			if len(attr.Args) > 0 {
				if fields, ok := attr.Args[0].Value.([]string); ok {
					dbColumns, err := c.mapFieldNames(fields, prismaToDbFieldMap, scalarFields, relationFields, model.Name)
					if err != nil {
						return nil, err
					}
					table.Constraints = append(table.Constraints, schema.Constraint{
						Type:    schema.ConstraintUnique,
						Columns: dbColumns,
					})
				}
			}
		case "index":
			// Index: @@index([field1, field2], name: "idx_name")
			if len(attr.Args) > 0 {
				if fields, ok := attr.Args[0].Value.([]string); ok {
					dbColumns, err := c.mapFieldNames(fields, prismaToDbFieldMap, scalarFields, relationFields, model.Name)
					if err != nil {
						return nil, err
					}
					indexCols := make([]schema.IndexColumn, len(dbColumns))
					for i, col := range dbColumns {
						indexCols[i] = schema.IndexColumn{Name: col}
					}

					// Extract index name from named argument
					indexName := ""
					for _, arg := range attr.Args {
						if arg.Name == "name" {
							if name, ok := arg.Value.(string); ok {
								indexName = name
								break
							}
						}
					}

					table.Indexes = append(table.Indexes, schema.Index{
						Name:    indexName,
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

	// Process relation fields to generate foreign keys
	for _, field := range model.Fields {
		if !c.isRelationField(field) {
			continue
		}

		// Process relation field to generate FK constraint
		fk, err := c.buildForeignKeyFromRelation(field, model, table.Name, prismaToDbFieldMap)
		if err != nil {
			return nil, fmt.Errorf("failed to build foreign key from relation field %q: %w", field.Name, err)
		}

		// Skip if no FK (passive side of relation)
		if fk != nil {
			table.ForeignKeys = append(table.ForeignKeys, *fk)
		}
	}

	return table, nil
}

// mapFieldNames converts Prisma field names to DB column names
func (c *Converter) mapFieldNames(prismaFields []string, fieldMap map[string]string, scalarFields map[string]bool, relationFields map[string]bool, modelName string) ([]string, error) {
	dbColumns := make([]string, len(prismaFields))
	for i, prismaField := range prismaFields {
		// Check if field is a relation
		if relationFields[prismaField] {
			return nil, fmt.Errorf("model %s: field %q is a relation field and cannot be used in model attributes (@@id, @@unique, @@index)", modelName, prismaField)
		}
		// Check if field exists in scalar fields
		if !scalarFields[prismaField] {
			return nil, fmt.Errorf("model %s: field %q referenced in model attribute does not exist", modelName, prismaField)
		}
		dbColumns[i] = fieldMap[prismaField]
	}
	return dbColumns, nil
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
		switch dbType {
		case "Uuid":
			dt.Kind = schema.TypeUUID
		case "Text":
			dt.Kind = schema.TypeText
		default:
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

// validateForeignKeyReferences validates that all FK reference columns exist in target tables.
func (c *Converter) validateForeignKeyReferences(s *schema.Schema) error {
	// Build table lookup map
	tablesByName := make(map[string]*schema.Table)
	for i := range s.Tables {
		tablesByName[s.Tables[i].Name] = &s.Tables[i]
	}

	// Validate each FK
	for _, table := range s.Tables {
		for _, fk := range table.ForeignKeys {
			// Check if referenced table exists
			refTable, exists := tablesByName[fk.RefTable]
			if !exists {
				return fmt.Errorf(
					"table %q: foreign key references non-existent table %q",
					table.Name, fk.RefTable,
				)
			}

			// Build column lookup for referenced table
			refColumns := make(map[string]bool)
			for _, col := range refTable.Columns {
				refColumns[col.Name] = true
			}

			// Validate all reference columns exist
			for _, refCol := range fk.RefColumns {
				if !refColumns[refCol] {
					return fmt.Errorf(
						"table %q: foreign key references non-existent column %q in table %q",
						table.Name, refCol, fk.RefTable,
					)
				}
			}
		}
	}

	return nil
}

// relationSpec holds parsed @relation attribute data.
type relationSpec struct {
	name       string   // Optional relation name
	fields     []string // Local fields (e.g., [userId])
	references []string // Referenced fields (e.g., [id])
	onDelete   string   // OnDelete action (e.g., "Cascade")
	onUpdate   string   // OnUpdate action (e.g., "Restrict")
}

// buildForeignKeyFromRelation creates a ForeignKey from a Prisma relation field.
// Returns nil for passive side (back-reference fields like User.posts Post[]).
func (c *Converter) buildForeignKeyFromRelation(
	field Field,
	model *ModelDeclaration,
	tableName string,
	prismaToDbFieldMap map[string]string,
) (*schema.ForeignKey, error) {
	// List fields (e.g., Post[]) are passive side - no FK generation
	if field.Type.List {
		// Parse @relation to check for invalid fields/references on list type
		relSpec, err := c.parseRelationAttribute(field.Attributes)
		if err != nil {
			return nil, err
		}
		if relSpec != nil && (len(relSpec.fields) > 0 || len(relSpec.references) > 0) {
			return nil, fmt.Errorf("list field %q cannot have fields or references in @relation", field.Name)
		}
		// Valid passive side - skip FK generation
		return nil, nil
	}

	// Scalar fields are active side - parse @relation and generate FK
	relSpec, err := c.parseRelationAttribute(field.Attributes)
	if err != nil {
		return nil, err
	}

	// If no @relation attribute or no fields specified, skip FK generation
	if relSpec == nil || len(relSpec.fields) == 0 {
		return nil, nil
	}

	// Validate fields and references match in count
	if len(relSpec.fields) != len(relSpec.references) {
		return nil, fmt.Errorf(
			"@relation fields and references count mismatch: %d fields vs %d references",
			len(relSpec.fields), len(relSpec.references),
		)
	}

	// Map Prisma field names to DB column names
	fkColumns := make([]string, len(relSpec.fields))
	for i, prismaField := range relSpec.fields {
		dbColumn, ok := prismaToDbFieldMap[prismaField]
		if !ok {
			return nil, fmt.Errorf(
				"field %q in @relation references non-existent scalar field %q",
				field.Name, prismaField,
			)
		}
		fkColumns[i] = dbColumn
	}

	// Referenced table: use mapped table name from model→table map
	refModelName := field.Type.Name
	refTable, ok := c.modelToTableMap[refModelName]
	if !ok {
		// Model not found in map - this shouldn't happen if Convert() ran properly
		// Fall back to model name for safety
		refTable = refModelName
		c.addWarning(fmt.Sprintf(
			"model %q: referenced model %q not found in model→table map, using model name as table name",
			model.Name, refModelName,
		))
	}

	// Map referenced Prisma field names to DB column names
	refFieldMap, hasMap := c.modelFieldToDbCol[refModelName]
	refColumns := make([]string, len(relSpec.references))
	for i, prismaField := range relSpec.references {
		if hasMap {
			if dbCol, found := refFieldMap[prismaField]; found {
				refColumns[i] = dbCol
			} else {
				// Field not in map - use as-is (might be unmapped field)
				refColumns[i] = prismaField
			}
		} else {
			// No map for this model - use Prisma field name as-is
			refColumns[i] = prismaField
		}
	}

	// Parse referential actions
	onDelete, err := c.parseReferentialAction(relSpec.onDelete)
	if err != nil {
		return nil, fmt.Errorf("invalid onDelete action: %w", err)
	}

	onUpdate, err := c.parseReferentialAction(relSpec.onUpdate)
	if err != nil {
		return nil, fmt.Errorf("invalid onUpdate action: %w", err)
	}

	// Build FK constraint
	fk := &schema.ForeignKey{
		Name:       c.generateForeignKeyName(tableName, fkColumns),
		Columns:    fkColumns,
		RefTable:   refTable,
		RefColumns: refColumns,
		OnDelete:   onDelete,
		OnUpdate:   onUpdate,
	}

	return fk, nil
}

// generateForeignKeyName creates a consistent FK constraint name.
func (c *Converter) generateForeignKeyName(tableName string, columns []string) string {
	return fmt.Sprintf("fk_%s_%s", tableName, strings.Join(columns, "_"))
}

// parseRelationAttribute extracts @relation attribute data.
// Returns nil if no @relation attribute found.
func (c *Converter) parseRelationAttribute(attrs []FieldAttribute) (*relationSpec, error) {
	var relationAttr *FieldAttribute
	for i := range attrs {
		if attrs[i].Name == "relation" {
			relationAttr = &attrs[i]
			break
		}
	}

	if relationAttr == nil {
		return nil, nil
	}

	spec := &relationSpec{}

	// Parse arguments
	for _, arg := range relationAttr.Args {
		switch arg.Name {
		case "": // Positional argument - relation name
			if name, ok := arg.Value.(string); ok {
				spec.name = name
			}
		case "fields":
			if fields, ok := arg.Value.([]string); ok {
				spec.fields = fields
			} else {
				return nil, fmt.Errorf("@relation fields argument must be an array")
			}
		case "references":
			if refs, ok := arg.Value.([]string); ok {
				spec.references = refs
			} else {
				return nil, fmt.Errorf("@relation references argument must be an array")
			}
		case "onDelete":
			if action, ok := arg.Value.(string); ok {
				spec.onDelete = action
			} else {
				return nil, fmt.Errorf("@relation onDelete must be a string")
			}
		case "onUpdate":
			if action, ok := arg.Value.(string); ok {
				spec.onUpdate = action
			} else {
				return nil, fmt.Errorf("@relation onUpdate must be a string")
			}
		default:
			// Unknown argument - warn but don't error
			c.addWarning(fmt.Sprintf("unknown @relation argument: %s", arg.Name))
		}
	}

	return spec, nil
}

// parseReferentialAction maps Prisma action names to schema.ReferentialAction.
func (c *Converter) parseReferentialAction(action string) (schema.ReferentialAction, error) {
	switch action {
	case "", "NoAction":
		return schema.ActionNoAction, nil
	case "Cascade":
		return schema.ActionCascade, nil
	case "Restrict":
		return schema.ActionRestrict, nil
	case "SetNull":
		return schema.ActionSetNull, nil
	case "SetDefault":
		return schema.ActionSetDefault, nil
	default:
		return schema.ActionNone, fmt.Errorf("unknown referential action: %s", action)
	}
}
