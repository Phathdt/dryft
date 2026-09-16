package prisma

import (
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// Writer converts Internal Schema to Prisma Schema Language.
type Writer struct {
	naming    NamingConvention
	formatter *Formatter
}

// NewWriter creates a new Prisma schema writer with the specified naming convention.
func NewWriter(naming NamingConvention) *Writer {
	return &Writer{
		naming:    naming,
		formatter: NewFormatter(),
	}
}

// Write converts a complete schema to Prisma schema format.
func (w *Writer) Write(s *schema.Schema) (string, error) {
	if s == nil {
		return "", fmt.Errorf("schema cannot be nil")
	}

	var blocks []string

	// Add generator block
	blocks = append(blocks, w.writeGenerator())

	// Add datasource block
	blocks = append(blocks, w.writeDatasource())

	// Add enums
	for _, enum := range s.Enums {
		enumName := w.naming.TransformModelName(enum.Name)
		if IsReservedModelName(enumName) {
			return "", fmt.Errorf(
				"enum %q maps to reserved Prisma name %q; rename the enum type",
				enum.Name, enumName,
			)
		}
		blocks = append(blocks, w.writeEnum(&enum))
	}

	// Add models (tables). Track model names for the same collision reason as fields:
	// "user_role" and "user__role" both transform to "UserRole".
	seenModels := make(map[string]string, len(s.Tables))

	// Build relation plan for the entire schema before processing models. This lets
	// us emit both forward and back-reference fields.
	relationPlan, err := w.buildRelationPlan(s)
	if err != nil {
		return "", fmt.Errorf("failed to build relation plan: %w", err)
	}

	for _, table := range s.Tables {
		modelName := w.naming.TransformModelName(table.Name)
		if IsReservedModelName(modelName) {
			return "", fmt.Errorf(
				"table %q maps to reserved Prisma name %q; rename the table or use a custom naming convention",
				table.Name, modelName,
			)
		}
		if previous, exists := seenModels[modelName]; exists {
			return "", fmt.Errorf(
				"tables %q and %q both map to Prisma model %q; rename one table or use snake_case model naming",
				previous, table.Name, modelName,
			)
		}
		seenModels[modelName] = table.Name

		modelBlock, err := w.writeTable(&table, relationPlan[table.Name])
		if err != nil {
			return "", fmt.Errorf("failed to write table %q: %w", table.Name, err)
		}
		blocks = append(blocks, modelBlock)
	}

	return w.formatter.JoinBlocks(blocks), nil
}

// writeGenerator generates the standard Prisma client generator block.
func (w *Writer) writeGenerator() string {
	content := []string{
		`provider = "prisma-client-js"`,
	}
	return w.formatter.FormatBlock("generator", "client", content)
}

// writeDatasource generates the PostgreSQL datasource block.
func (w *Writer) writeDatasource() string {
	content := []string{
		`provider = "postgresql"`,
		`url      = env("DATABASE_URL")`,
	}
	return w.formatter.FormatBlock("datasource", "db", content)
}

// writeEnum converts an internal enum to Prisma enum syntax.
func (w *Writer) writeEnum(enum *schema.Enum) string {
	var content []string
	for _, value := range enum.Values {
		content = append(content, w.formatter.FormatEnumValue(value.Label))
	}
	return w.formatter.FormatBlock("enum", w.naming.TransformModelName(enum.Name), content)
}

// writeTable converts an internal table to Prisma model syntax.
// relations carries the relation fields derived for this table, which may be empty.
func (w *Writer) writeTable(table *schema.Table, relations []relationField) (string, error) {
	modelName := w.naming.TransformModelName(table.Name)

	// Collect field lines for alignment
	var fieldLines []FieldLine

	// Track transformed field names to catch collisions. Distinct columns such as
	// "user_id" and "user__id" both transform to "userId", which would emit a model
	// with duplicate fields that Prisma rejects. Fail loudly instead.
	seenFields := make(map[string]string, len(table.Columns))

	for _, col := range table.Columns {
		fieldLine, err := w.writeColumn(&col, table)
		if err != nil {
			return "", fmt.Errorf("failed to write column %q: %w", col.Name, err)
		}

		if IsReservedFieldName(fieldLine.Name) {
			return "", fmt.Errorf(
				"column %q maps to reserved Prisma field name %q; rename the column",
				col.Name, fieldLine.Name,
			)
		}

		if previous, exists := seenFields[fieldLine.Name]; exists {
			return "", fmt.Errorf(
				"columns %q and %q both map to Prisma field %q; rename one column or use snake_case field naming",
				previous, col.Name, fieldLine.Name,
			)
		}
		seenFields[fieldLine.Name] = col.Name

		fieldLines = append(fieldLines, fieldLine)
	}

	// Relation fields share the model's field namespace, so a relation named after an
	// existing scalar column has to be reported rather than silently duplicated.
	for _, rel := range relations {
		if previous, exists := seenFields[rel.Name]; exists {
			return "", fmt.Errorf(
				"relation field %q on table %q collides with column %q; rename the column or the foreign key",
				rel.Name, table.Name, previous,
			)
		}
		seenFields[rel.Name] = rel.Name

		fieldLines = append(fieldLines, FieldLine(rel))
	}

	// Align fields
	alignedFields := w.formatter.AlignFields(fieldLines)

	// Add model-level attributes
	var modelAttrs []string

	// Composite primary keys are declared at model level; single-column keys use @id.
	if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) > 1 {
		prismaFields := make([]string, len(table.PrimaryKey.Columns))
		for i, colName := range table.PrimaryKey.Columns {
			prismaFields[i] = w.naming.TransformFieldName(colName)
		}
		modelAttrs = append(modelAttrs, fmt.Sprintf("@@id([%s])", strings.Join(prismaFields, ", ")))
	}

	// Add @@map if table name differs from model name
	if table.Name != modelName {
		modelAttrs = append(modelAttrs, fmt.Sprintf(`@@map("%s")`, table.Name))
	}

	// Add @@unique for composite unique constraints
	for _, constraint := range table.Constraints {
		if constraint.Type == schema.ConstraintUnique && len(constraint.Columns) > 1 {
			prismaFields := make([]string, len(constraint.Columns))
			for i, colName := range constraint.Columns {
				prismaFields[i] = w.naming.TransformFieldName(colName)
			}
			modelAttrs = append(modelAttrs, fmt.Sprintf("@@unique([%s])", strings.Join(prismaFields, ", ")))
		}
	}

	// Add @@index for indexes
	for _, index := range table.Indexes {
		if !index.Unique {
			prismaFields := make([]string, len(index.Columns))
			for i, idxCol := range index.Columns {
				prismaFields[i] = w.naming.TransformFieldName(idxCol.Name)
			}
			modelAttrs = append(modelAttrs, fmt.Sprintf("@@index([%s])", strings.Join(prismaFields, ", ")))
		}
	}

	// Combine fields and attributes
	content := alignedFields
	if len(modelAttrs) > 0 {
		content = append(content, "")
		content = append(content, modelAttrs...)
	}

	return w.formatter.FormatBlock("model", modelName, content), nil
}

// writeColumn converts an internal column to a Prisma field line.
func (w *Writer) writeColumn(col *schema.Column, table *schema.Table) (FieldLine, error) {
	fieldName := w.naming.TransformFieldName(col.Name)

	// Enum columns reference the generated enum block, not a scalar type.
	var prismaType string
	if col.Type.Kind == schema.TypeEnum && col.Type.EnumName != "" {
		prismaType = w.naming.TransformModelName(col.Type.EnumName)
		for i := 0; i < col.Type.ArrayDepth; i++ {
			prismaType += "[]"
		}
	} else {
		prismaType = col.Type.ToPrisma()
	}

	var attributes []string

	// Single-column primary keys use the field-level @id attribute. Composite
	// primary keys are invalid as repeated @id and are emitted as @@id instead.
	if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) == 1 &&
		table.PrimaryKey.Columns[0] == col.Name {
		attributes = append(attributes, "@id")
	}

	// Check if this column has a unique constraint
	if w.isUniqueColumn(col.Name, table) {
		attributes = append(attributes, "@unique")
	}

	// Add @default attribute
	if col.Default != nil {
		defaultAttr := w.formatDefault(col.Default)
		if defaultAttr != "" {
			attributes = append(attributes, defaultAttr)
		}
	}

	// Add @updatedAt for timestamp columns that track updates
	if w.isUpdatedAtField(col.Name) {
		attributes = append(attributes, "@updatedAt")
	}

	// Extract base type and @db annotation
	baseType, dbAnnotation := w.parseTypeAnnotation(prismaType)

	// Add optional marker if nullable
	if col.Nullable {
		baseType += "?"
	}

	// Add @db annotation if present
	if dbAnnotation != "" {
		attributes = append(attributes, dbAnnotation)
	}

	// Add @map if field name differs from column name
	if col.Name != fieldName {
		attributes = append(attributes, fmt.Sprintf(`@map("%s")`, col.Name))
	}

	return FieldLine{
		Name:       fieldName,
		Type:       baseType,
		Attributes: attributes,
	}, nil
}

// parseTypeAnnotation splits "String @db.Uuid" into ("String", "@db.Uuid").
func (w *Writer) parseTypeAnnotation(prismaType string) (baseType, annotation string) {
	parts := strings.SplitN(prismaType, " ", 2)
	baseType = parts[0]
	if len(parts) > 1 {
		annotation = parts[1]
	}
	return
}

// formatDefault formats a default value for Prisma syntax.
func (w *Writer) formatDefault(def *schema.DefaultValue) string {
	switch def.Kind {
	case schema.DefaultLiteral:
		// Handle boolean literals
		if def.Literal == "true" || def.Literal == "false" {
			return fmt.Sprintf("@default(%s)", def.Literal)
		}
		// Handle numeric literals
		if !strings.Contains(def.Literal, "'") {
			return fmt.Sprintf("@default(%s)", def.Literal)
		}
		// Handle string literals - remove quotes
		literal := strings.Trim(def.Literal, "'\"")
		return fmt.Sprintf(`@default("%s")`, literal)

	case schema.DefaultExpression:
		// Map common PostgreSQL expressions to Prisma functions
		expr := strings.ToLower(strings.TrimSpace(def.Expression))
		switch {
		case strings.HasPrefix(expr, "now()"):
			return "@default(now())"
		case strings.HasPrefix(expr, "uuid_generate_v4()"), strings.HasPrefix(expr, "gen_random_uuid()"):
			return "@default(uuid())"
		case strings.HasPrefix(expr, "current_timestamp"):
			return "@default(now())"
		default:
			// For other expressions, use dbgenerated
			return fmt.Sprintf(`@default(dbgenerated("%s"))`, def.Expression)
		}

	case schema.DefaultSequence:
		// Sequences are typically used for auto-increment, map to autoincrement()
		return "@default(autoincrement())"
	}

	return ""
}

// isUniqueColumn checks if a column has a unique constraint.
func (w *Writer) isUniqueColumn(colName string, table *schema.Table) bool {
	// Check single-column unique constraints
	for _, constraint := range table.Constraints {
		if constraint.Type == schema.ConstraintUnique && len(constraint.Columns) == 1 {
			if constraint.Columns[0] == colName {
				return true
			}
		}
	}

	// Check unique indexes
	for _, index := range table.Indexes {
		if index.Unique && len(index.Columns) == 1 {
			if index.Columns[0].Name == colName {
				return true
			}
		}
	}

	return false
}

// isUpdatedAtField checks if a field name suggests it tracks update timestamps.
func (w *Writer) isUpdatedAtField(colName string) bool {
	normalized := strings.ToLower(colName)
	return normalized == "updated_at" || normalized == "updatedat" || normalized == "modified_at"
}
