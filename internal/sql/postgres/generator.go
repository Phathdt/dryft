package postgres

import (
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
)

// Generator generates PostgreSQL DDL statements.
type Generator struct {
	opts sql.GeneratorOptions
}

// NewGenerator creates a new PostgreSQL generator.
func NewGenerator(opts sql.GeneratorOptions) *Generator {
	return &Generator{opts: opts}
}

// Generate produces forward SQL statements.
func (g *Generator) Generate(ops []diff.Operation) ([]string, error) {
	var statements []string

	for _, op := range ops {
		sql, err := g.generateOperation(op)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SQL for %v: %w", op.Kind(), err)
		}
		if sql != "" {
			statements = append(statements, sql)
		}
	}

	return statements, nil
}

// GenerateReverse produces reverse SQL statements.
func (g *Generator) GenerateReverse(ops []diff.Operation) ([]string, []string, error) {
	var statements []string
	var warnings []string

	// Reverse operations in opposite order
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		sql, warning := g.generateReverseOperation(op)
		if warning != "" {
			warnings = append(warnings, warning)
		}
		if sql != "" {
			statements = append(statements, sql)
		}
	}

	return statements, warnings, nil
}

// generateOperation generates SQL for a single operation.
func (g *Generator) generateOperation(op diff.Operation) (string, error) {
	switch o := op.(type) {
	case diff.CreateTable:
		return g.generateCreateTable(o), nil
	case diff.DropTable:
		return g.generateDropTable(o), nil
	case diff.AddColumn:
		return g.generateAddColumn(o), nil
	case diff.DropColumn:
		return g.generateDropColumn(o), nil
	case diff.RenameColumn:
		return g.generateRenameColumn(o), nil
	case diff.AlterColumn:
		return g.generateAlterColumn(o), nil
	case diff.CreateIndex:
		return g.generateCreateIndex(o), nil
	case diff.DropIndex:
		return g.generateDropIndex(o), nil
	case diff.CreateForeignKey:
		return g.generateCreateForeignKey(o), nil
	case diff.DropForeignKey:
		return g.generateDropForeignKey(o), nil
	case diff.CreateEnum:
		return g.generateCreateEnum(o), nil
	case diff.AlterEnum:
		return g.generateAlterEnum(o), nil
	case diff.RenameTable:
		return g.generateRenameTable(o), nil
	default:
		return "", fmt.Errorf("unknown operation type: %T", op)
	}
}

// generateCreateTable generates CREATE TABLE statement.
func (g *Generator) generateCreateTable(op diff.CreateTable) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", quoteIdentifier(op.Table.Name)))

	// Columns
	for i, col := range op.Table.Columns {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("  ")
		b.WriteString(quoteIdentifier(col.Name))
		b.WriteString(" ")
		b.WriteString(mapDataType(col.Type))

		if !col.Nullable {
			b.WriteString(" NOT NULL")
		}

		if col.Default != nil {
			defaultVal := formatDefault(col.Default)
			if defaultVal != "" {
				b.WriteString(fmt.Sprintf(" DEFAULT %s", defaultVal))
			}
		}
	}

	// Primary key
	if op.Table.PrimaryKey != nil && len(op.Table.PrimaryKey.Columns) > 0 {
		b.WriteString(",\n  PRIMARY KEY (")
		b.WriteString(formatColumnList(op.Table.PrimaryKey.Columns))
		b.WriteString(")")
	}

	// Unique constraints
	for _, constraint := range op.Table.Constraints {
		if constraint.Type == schema.ConstraintUnique {
			b.WriteString(",\n  UNIQUE (")
			b.WriteString(formatColumnList(constraint.Columns))
			b.WriteString(")")
		}
	}

	b.WriteString("\n);")
	return b.String()
}

// generateDropTable generates DROP TABLE statement.
func (g *Generator) generateDropTable(op diff.DropTable) string {
	return fmt.Sprintf("DROP TABLE %s;", quoteIdentifier(op.Name))
}

// generateAddColumn generates ALTER TABLE ADD COLUMN statement.
func (g *Generator) generateAddColumn(op diff.AddColumn) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ALTER TABLE %s\nADD COLUMN %s %s",
		quoteIdentifier(op.Table),
		quoteIdentifier(op.Column.Name),
		mapDataType(op.Column.Type)))

	if !op.Column.Nullable {
		b.WriteString(" NOT NULL")
	}

	if op.Column.Default != nil {
		defaultVal := formatDefault(op.Column.Default)
		if defaultVal != "" {
			b.WriteString(fmt.Sprintf(" DEFAULT %s", defaultVal))
		}
	}

	b.WriteString(";")
	return b.String()
}

// generateDropColumn generates ALTER TABLE DROP COLUMN statement.
func (g *Generator) generateDropColumn(op diff.DropColumn) string {
	return fmt.Sprintf("ALTER TABLE %s\nDROP COLUMN %s;",
		quoteIdentifier(op.Table),
		quoteIdentifier(op.Column))
}

// generateRenameColumn generates ALTER TABLE RENAME COLUMN statement.
func (g *Generator) generateRenameColumn(op diff.RenameColumn) string {
	return fmt.Sprintf("ALTER TABLE %s\nRENAME COLUMN %s TO %s;",
		quoteIdentifier(op.Table),
		quoteIdentifier(op.From),
		quoteIdentifier(op.To))
}

// generateAlterColumn generates ALTER TABLE ALTER COLUMN statements.
func (g *Generator) generateAlterColumn(op diff.AlterColumn) string {
	var statements []string

	// Type change
	if op.OldType.Kind != op.NewType.Kind {
		stmt := fmt.Sprintf("ALTER TABLE %s\nALTER COLUMN %s TYPE %s;",
			quoteIdentifier(op.Table),
			quoteIdentifier(op.Column),
			mapDataType(op.NewType))
		statements = append(statements, stmt)
	}

	// Nullable change
	if op.OldNullable != op.NewNullable {
		if op.NewNullable {
			stmt := fmt.Sprintf("ALTER TABLE %s\nALTER COLUMN %s DROP NOT NULL;",
				quoteIdentifier(op.Table),
				quoteIdentifier(op.Column))
			statements = append(statements, stmt)
		} else {
			stmt := fmt.Sprintf("ALTER TABLE %s\nALTER COLUMN %s SET NOT NULL;",
				quoteIdentifier(op.Table),
				quoteIdentifier(op.Column))
			statements = append(statements, stmt)
		}
	}

	// Default change
	if !defaultsEqual(op.OldDefault, op.NewDefault) {
		if op.NewDefault == nil {
			stmt := fmt.Sprintf("ALTER TABLE %s\nALTER COLUMN %s DROP DEFAULT;",
				quoteIdentifier(op.Table),
				quoteIdentifier(op.Column))
			statements = append(statements, stmt)
		} else {
			defaultVal := formatDefault(op.NewDefault)
			if defaultVal != "" {
				stmt := fmt.Sprintf("ALTER TABLE %s\nALTER COLUMN %s SET DEFAULT %s;",
					quoteIdentifier(op.Table),
					quoteIdentifier(op.Column),
					defaultVal)
				statements = append(statements, stmt)
			}
		}
	}

	return strings.Join(statements, "\n\n")
}

// generateCreateIndex generates CREATE INDEX statement.
func (g *Generator) generateCreateIndex(op diff.CreateIndex) string {
	indexName := op.Index.Name
	if indexName == "" {
		// Generate index name
		colNames := make([]string, len(op.Index.Columns))
		for i, col := range op.Index.Columns {
			colNames[i] = col.Name
		}
		indexName = fmt.Sprintf("idx_%s_%s", op.Table, strings.Join(colNames, "_"))
	}

	unique := ""
	if op.Index.Unique {
		unique = "UNIQUE "
	}

	columns := make([]string, len(op.Index.Columns))
	for i, col := range op.Index.Columns {
		columns[i] = quoteIdentifier(col.Name)
	}

	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);",
		unique,
		quoteIdentifier(indexName),
		quoteIdentifier(op.Table),
		strings.Join(columns, ", "))
}

// generateDropIndex generates DROP INDEX statement.
func (g *Generator) generateDropIndex(op diff.DropIndex) string {
	return fmt.Sprintf("DROP INDEX %s;", quoteIdentifier(op.Name))
}

// generateCreateForeignKey generates ALTER TABLE ADD CONSTRAINT statement.
func (g *Generator) generateCreateForeignKey(op diff.CreateForeignKey) string {
	fkName := op.Constraint.Name
	if fkName == "" {
		fkName = fmt.Sprintf("fk_%s_%s", op.Table, strings.Join(op.Constraint.Columns, "_"))
	}

	onDelete := ""
	if op.Constraint.OnDelete != schema.ActionNone {
		onDelete = fmt.Sprintf(" ON DELETE %s", op.Constraint.OnDelete.String())
	}

	onUpdate := ""
	if op.Constraint.OnUpdate != schema.ActionNone {
		onUpdate = fmt.Sprintf(" ON UPDATE %s", op.Constraint.OnUpdate.String())
	}

	return fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s\nFOREIGN KEY (%s)\nREFERENCES %s (%s)%s%s;",
		quoteIdentifier(op.Table),
		quoteIdentifier(fkName),
		formatColumnList(op.Constraint.Columns),
		quoteIdentifier(op.Constraint.RefTable),
		formatColumnList(op.Constraint.RefColumns),
		onDelete,
		onUpdate)
}

// generateDropForeignKey generates ALTER TABLE DROP CONSTRAINT statement.
func (g *Generator) generateDropForeignKey(op diff.DropForeignKey) string {
	return fmt.Sprintf("ALTER TABLE %s\nDROP CONSTRAINT %s;",
		quoteIdentifier(op.Table),
		quoteIdentifier(op.Name))
}

// generateCreateEnum generates CREATE TYPE statement.
func (g *Generator) generateCreateEnum(op diff.CreateEnum) string {
	values := make([]string, len(op.Enum.Values))
	for i, v := range op.Enum.Values {
		values[i] = fmt.Sprintf("'%s'", v.Label)
	}

	return fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);",
		quoteIdentifier(op.Enum.Name),
		strings.Join(values, ", "))
}

// generateAlterEnum generates ALTER TYPE ADD VALUE statements.
func (g *Generator) generateAlterEnum(op diff.AlterEnum) string {
	var statements []string

	for _, value := range op.AddValues {
		stmt := fmt.Sprintf("ALTER TYPE %s ADD VALUE '%s';",
			quoteIdentifier(op.Name),
			value)
		statements = append(statements, stmt)
	}

	// Note: PostgreSQL cannot drop enum values
	// Drop values would require more complex migration

	return strings.Join(statements, "\n")
}

// generateRenameTable generates ALTER TABLE RENAME statement.
func (g *Generator) generateRenameTable(op diff.RenameTable) string {
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s;",
		quoteIdentifier(op.From),
		quoteIdentifier(op.To))
}

// generateReverseOperation generates reverse SQL for an operation.
func (g *Generator) generateReverseOperation(op diff.Operation) (string, string) {
	switch o := op.(type) {
	case diff.CreateTable:
		return fmt.Sprintf("DROP TABLE %s;", quoteIdentifier(o.Table.Name)), ""
	case diff.DropTable:
		return "", fmt.Sprintf("-- WARNING: Cannot reverse DROP TABLE %s (data lost)", o.Name)
	case diff.AddColumn:
		return fmt.Sprintf("ALTER TABLE %s\nDROP COLUMN %s;",
			quoteIdentifier(o.Table),
			quoteIdentifier(o.Column.Name)), ""
	case diff.DropColumn:
		return "", fmt.Sprintf("-- WARNING: Cannot reverse DROP COLUMN %s.%s (data lost)",
			o.Table, o.Column)
	case diff.RenameColumn:
		return fmt.Sprintf("ALTER TABLE %s\nRENAME COLUMN %s TO %s;",
			quoteIdentifier(o.Table),
			quoteIdentifier(o.To),
			quoteIdentifier(o.From)), ""
	case diff.AlterColumn:
		return "", fmt.Sprintf("-- WARNING: ALTER COLUMN %s.%s may not be safely reversible",
			o.Table, o.Column)
	case diff.CreateIndex:
		return fmt.Sprintf("DROP INDEX %s;", quoteIdentifier(o.Index.Name)), ""
	case diff.DropIndex:
		return "", fmt.Sprintf("-- WARNING: Cannot reverse DROP INDEX %s without schema info", o.Name)
	case diff.CreateForeignKey:
		fkName := o.Constraint.Name
		if fkName == "" {
			fkName = fmt.Sprintf("fk_%s_%s", o.Table, strings.Join(o.Constraint.Columns, "_"))
		}
		return fmt.Sprintf("ALTER TABLE %s\nDROP CONSTRAINT %s;",
			quoteIdentifier(o.Table),
			quoteIdentifier(fkName)), ""
	case diff.DropForeignKey:
		return "", fmt.Sprintf("-- WARNING: Cannot reverse DROP CONSTRAINT %s without schema info", o.Name)
	case diff.CreateEnum:
		return fmt.Sprintf("DROP TYPE %s;", quoteIdentifier(o.Enum.Name)), ""
	case diff.AlterEnum:
		return "", fmt.Sprintf("-- WARNING: Cannot reverse ALTER TYPE %s (enum values cannot be dropped)", o.Name)
	case diff.RenameTable:
		return fmt.Sprintf("ALTER TABLE %s RENAME TO %s;",
			quoteIdentifier(o.To),
			quoteIdentifier(o.From)), ""
	default:
		return "", fmt.Sprintf("-- WARNING: Unknown operation type %T", op)
	}
}

// defaultsEqual checks if two defaults are equal.
func defaultsEqual(a, b *schema.DefaultValue) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Kind != b.Kind {
		return false
	}
	if a.Literal != b.Literal {
		return false
	}
	if a.Expression != b.Expression {
		return false
	}
	return true
}
