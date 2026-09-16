package migration

import (
	"fmt"

	"github.com/phathdt/dryft/internal/schema"
)

// SchemaBuilder builds a schema from SQL statements by applying them chronologically.
type SchemaBuilder struct {
	tables map[string]*schema.Table // Keyed by table name
	enums  map[string]*schema.Enum  // Keyed by type name
}

// NewSchemaBuilder creates a new schema builder.
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		tables: make(map[string]*schema.Table),
		enums:  make(map[string]*schema.Enum),
	}
}

// Apply applies a parsed statement to the schema state.
func (b *SchemaBuilder) Apply(stmt Statement) error {
	switch s := stmt.(type) {
	case *CreateTable:
		return b.applyCreateTable(s)
	case *DropTable:
		return b.applyDropTable(s)
	case *AlterTable:
		return b.applyAlterTable(s)
	case *CreateType:
		return b.applyCreateType(s)
	case *DropType:
		return b.applyDropType(s)
	case *CreateIndex:
		return b.applyCreateIndex(s)
	case *DropIndex:
		return b.applyDropIndex(s)
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// Build returns the final schema.
func (b *SchemaBuilder) Build() *schema.Schema {
	// Convert maps to slices
	tables := make([]schema.Table, 0, len(b.tables))
	for _, t := range b.tables {
		tables = append(tables, *t)
	}

	enums := make([]schema.Enum, 0, len(b.enums))
	for _, e := range b.enums {
		enums = append(enums, *e)
	}

	return &schema.Schema{
		Tables: tables,
		Enums:  enums,
	}
}

// applyCreateTable applies a CREATE TABLE statement.
func (b *SchemaBuilder) applyCreateTable(stmt *CreateTable) error {
	if _, exists := b.tables[stmt.Name]; exists {
		if stmt.IfNotExists {
			return nil // Skip silently
		}
		return fmt.Errorf("table %s already exists", stmt.Name)
	}

	table := &schema.Table{
		Name:        stmt.Name,
		Columns:     []schema.Column{},
		Indexes:     []schema.Index{},
		ForeignKeys: []schema.ForeignKey{},
		Constraints: []schema.Constraint{},
	}

	// Convert ColumnDef → schema.Column
	for _, colDef := range stmt.Columns {
		col, err := b.convertColumn(colDef)
		if err != nil {
			return fmt.Errorf("failed to convert column %s: %w", colDef.Name, err)
		}
		table.Columns = append(table.Columns, col)

		// Handle column-level constraints
		for _, constraint := range colDef.Constraints {
			if err := b.applyColumnConstraint(table, colDef.Name, constraint); err != nil {
				return err
			}
		}
	}

	// Convert TableConstraint → schema constraints
	for _, constraint := range stmt.Constraints {
		if err := b.applyTableConstraint(table, constraint); err != nil {
			return err
		}
	}

	b.tables[stmt.Name] = table
	return nil
}

// applyDropTable applies a DROP TABLE statement.
func (b *SchemaBuilder) applyDropTable(stmt *DropTable) error {
	if _, exists := b.tables[stmt.Name]; !exists {
		if stmt.IfExists {
			return nil
		}
		return fmt.Errorf("table %s does not exist", stmt.Name)
	}

	delete(b.tables, stmt.Name)
	return nil
}

// applyAlterTable applies an ALTER TABLE statement.
func (b *SchemaBuilder) applyAlterTable(stmt *AlterTable) error {
	table, exists := b.tables[stmt.Table]
	if !exists {
		return fmt.Errorf("table %s does not exist", stmt.Table)
	}

	switch action := stmt.Action.(type) {
	case *AddColumn:
		return b.applyAddColumn(table, action)
	case *DropColumn:
		return b.applyDropColumn(table, action)
	case *RenameColumn:
		return b.applyRenameColumn(table, action)
	case *AlterColumn:
		return b.applyAlterColumn(table, action)
	default:
		return fmt.Errorf("unsupported alter action: %T", action)
	}
}

// applyCreateType applies a CREATE TYPE statement.
func (b *SchemaBuilder) applyCreateType(stmt *CreateType) error {
	if _, exists := b.enums[stmt.Name]; exists {
		return fmt.Errorf("type %s already exists", stmt.Name)
	}

	enum := &schema.Enum{
		Name:   stmt.Name,
		Values: []schema.EnumValue{},
	}

	for i, val := range stmt.Values {
		enum.Values = append(enum.Values, schema.EnumValue{
			Label: val,
			Order: i,
		})
	}

	b.enums[stmt.Name] = enum
	return nil
}

// applyDropType applies a DROP TYPE statement.
func (b *SchemaBuilder) applyDropType(stmt *DropType) error {
	if _, exists := b.enums[stmt.Name]; !exists {
		if stmt.IfExists {
			return nil
		}
		return fmt.Errorf("type %s does not exist", stmt.Name)
	}

	delete(b.enums, stmt.Name)
	return nil
}

// applyCreateIndex applies a CREATE INDEX statement.
func (b *SchemaBuilder) applyCreateIndex(stmt *CreateIndex) error {
	table, exists := b.tables[stmt.Table]
	if !exists {
		return fmt.Errorf("table %s does not exist", stmt.Table)
	}

	// Check if index already exists
	for _, idx := range table.Indexes {
		if idx.Name == stmt.Name {
			return fmt.Errorf("index %s already exists on table %s", stmt.Name, stmt.Table)
		}
	}

	indexColumns := make([]schema.IndexColumn, 0, len(stmt.Columns))
	for _, col := range stmt.Columns {
		indexColumns = append(indexColumns, schema.IndexColumn{
			Name:  col.Name,
			Order: parseOrder(col.Order),
		})
	}

	index := schema.Index{
		Name:    stmt.Name,
		Columns: indexColumns,
		Unique:  stmt.Unique,
		Type:    parseIndexMethod(stmt.Method),
		Where:   stmt.Where,
	}

	table.Indexes = append(table.Indexes, index)
	return nil
}

// applyDropIndex applies a DROP INDEX statement.
func (b *SchemaBuilder) applyDropIndex(stmt *DropIndex) error {
	// Index can be on any table - search all
	for _, table := range b.tables {
		for i, idx := range table.Indexes {
			if idx.Name == stmt.Name {
				table.Indexes = append(table.Indexes[:i], table.Indexes[i+1:]...)
				return nil
			}
		}
	}

	if stmt.IfExists {
		return nil
	}
	return fmt.Errorf("index %s does not exist", stmt.Name)
}

// applyAddColumn applies an ADD COLUMN action.
func (b *SchemaBuilder) applyAddColumn(table *schema.Table, action *AddColumn) error {
	// Check if column already exists
	for _, col := range table.Columns {
		if col.Name == action.Column.Name {
			return fmt.Errorf("column %s already exists in table %s", action.Column.Name, table.Name)
		}
	}

	col, err := b.convertColumn(action.Column)
	if err != nil {
		return err
	}

	table.Columns = append(table.Columns, col)
	return nil
}

// applyDropColumn applies a DROP COLUMN action.
func (b *SchemaBuilder) applyDropColumn(table *schema.Table, action *DropColumn) error {
	for i, col := range table.Columns {
		if col.Name == action.Name {
			// Remove column
			table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
			return nil
		}
	}

	if action.IfExists {
		return nil
	}
	return fmt.Errorf("column %s does not exist in table %s", action.Name, table.Name)
}

// applyRenameColumn applies a RENAME COLUMN action.
func (b *SchemaBuilder) applyRenameColumn(table *schema.Table, action *RenameColumn) error {
	for i := range table.Columns {
		if table.Columns[i].Name == action.OldName {
			table.Columns[i].Name = action.NewName
			return nil
		}
	}
	return fmt.Errorf("column %s does not exist in table %s", action.OldName, table.Name)
}

// applyAlterColumn applies an ALTER COLUMN action.
func (b *SchemaBuilder) applyAlterColumn(table *schema.Table, action *AlterColumn) error {
	for i := range table.Columns {
		if table.Columns[i].Name == action.Name {
			col := &table.Columns[i]

			if action.SetNotNull {
				col.Nullable = false
			}
			if action.DropNotNull {
				col.Nullable = true
			}
			if action.SetDefault != nil {
				col.Default = &schema.DefaultValue{
					Kind:       schema.DefaultExpression,
					Expression: *action.SetDefault,
				}
			}
			if action.DropDefault {
				col.Default = nil
			}
			if action.SetType != "" {
				dataType, err := mapSQLType(action.SetType, b.enums)
				if err != nil {
					return err
				}
				col.Type = dataType
			}

			return nil
		}
	}
	return fmt.Errorf("column %s does not exist in table %s", action.Name, table.Name)
}

// convertColumn converts a ColumnDef to schema.Column.
func (b *SchemaBuilder) convertColumn(def ColumnDef) (schema.Column, error) {
	dataType, err := mapSQLType(def.Type, b.enums)
	if err != nil {
		return schema.Column{}, err
	}

	col := schema.Column{
		Name:     def.Name,
		Type:     dataType,
		Nullable: def.Nullable,
	}

	if def.Default != nil {
		col.Default = &schema.DefaultValue{
			Kind:       schema.DefaultExpression,
			Expression: *def.Default,
		}
	}

	return col, nil
}

// applyTableConstraint applies a table constraint to the table.
func (b *SchemaBuilder) applyTableConstraint(table *schema.Table, constraint TableConstraint) error {
	switch constraint.Type {
	case PrimaryKeyConstraint:
		table.PrimaryKey = &schema.PrimaryKey{
			Columns: constraint.Columns,
			Name:    constraint.Name,
		}
	case ForeignKeyConstraint:
		fk := schema.ForeignKey{
			Name:       constraint.Name,
			Columns:    constraint.Columns,
			RefTable:   constraint.RefTable,
			RefColumns: constraint.RefColumns,
			OnDelete:   convertReferentialAction(constraint.OnDelete),
			OnUpdate:   convertReferentialAction(constraint.OnUpdate),
		}
		table.ForeignKeys = append(table.ForeignKeys, fk)
	case UniqueConstraint:
		c := schema.Constraint{
			Name:    constraint.Name,
			Type:    schema.ConstraintUnique,
			Columns: constraint.Columns,
		}
		table.Constraints = append(table.Constraints, c)
	case CheckConstraint:
		c := schema.Constraint{
			Name:       constraint.Name,
			Type:       schema.ConstraintCheck,
			Expression: constraint.CheckExpr,
		}
		table.Constraints = append(table.Constraints, c)
	default:
		return fmt.Errorf("unsupported constraint type: %s", constraint.Type)
	}

	return nil
}

// applyColumnConstraint applies a column-level constraint to the table.
func (b *SchemaBuilder) applyColumnConstraint(table *schema.Table, columnName string, constraint ColumnConstraint) error {
	switch constraint.Type {
	case PrimaryKeyConstraint:
		// Convert column-level PRIMARY KEY to table-level
		if table.PrimaryKey == nil {
			table.PrimaryKey = &schema.PrimaryKey{
				Columns: []string{columnName},
				Name:    constraint.Name,
			}
		} else {
			// Append to existing primary key (composite key)
			table.PrimaryKey.Columns = append(table.PrimaryKey.Columns, columnName)
		}
	case ForeignKeyConstraint:
		fk := schema.ForeignKey{
			Name:       constraint.Name,
			Columns:    []string{columnName},
			RefTable:   constraint.RefTable,
			RefColumns: constraint.RefColumns,
			OnDelete:   convertReferentialAction(constraint.OnDelete),
			OnUpdate:   convertReferentialAction(constraint.OnUpdate),
		}
		table.ForeignKeys = append(table.ForeignKeys, fk)
	case UniqueConstraint:
		c := schema.Constraint{
			Name:    constraint.Name,
			Type:    schema.ConstraintUnique,
			Columns: []string{columnName},
		}
		table.Constraints = append(table.Constraints, c)
	case CheckConstraint:
		c := schema.Constraint{
			Name:       constraint.Name,
			Type:       schema.ConstraintCheck,
			Expression: constraint.CheckExpr,
		}
		table.Constraints = append(table.Constraints, c)
	}

	return nil
}

// convertReferentialAction converts string to ReferentialAction.
func convertReferentialAction(action string) schema.ReferentialAction {
	switch action {
	case "NO ACTION":
		return schema.ActionNoAction
	case "RESTRICT":
		return schema.ActionRestrict
	case "CASCADE":
		return schema.ActionCascade
	case "SET NULL":
		return schema.ActionSetNull
	case "SET DEFAULT":
		return schema.ActionSetDefault
	default:
		return schema.ActionNone
	}
}

// parseIndexMethod converts string to IndexType.
func parseIndexMethod(method string) schema.IndexType {
	switch method {
	case "btree", "":
		return schema.IndexBTree
	case "hash":
		return schema.IndexHash
	case "gin":
		return schema.IndexGIN
	case "gist":
		return schema.IndexGiST
	case "spgist":
		return schema.IndexSPGiST
	case "brin":
		return schema.IndexBRIN
	default:
		return schema.IndexBTree
	}
}

// parseOrder converts string to SortOrder.
func parseOrder(order string) schema.SortOrder {
	switch order {
	case "DESC":
		return schema.SortDesc
	default:
		return schema.SortAsc
	}
}
