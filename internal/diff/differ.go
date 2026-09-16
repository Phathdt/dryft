// Package diff provides schema comparison and operation generation.
package diff

import (
	"fmt"

	"github.com/phathdt/dryft/internal/schema"
)

// Differ performs semantic diff between two schemas.
type Differ struct {
	renameHints map[string]string // "table.column" -> "table.newcolumn"
}

// NewDiffer creates a new differ with optional rename hints.
func NewDiffer(renameHints map[string]string) *Differ {
	if renameHints == nil {
		renameHints = make(map[string]string)
	}
	return &Differ{
		renameHints: renameHints,
	}
}

// Diff compares two schemas and returns a list of operations.
func (d *Differ) Diff(before, after *schema.Schema) ([]Operation, error) {
	var ops []Operation

	// 1. Diff enums first (they must be created before tables that use them)
	enumOps, err := d.diffEnums(before, after)
	if err != nil {
		return nil, fmt.Errorf("failed to diff enums: %w", err)
	}
	ops = append(ops, enumOps...)

	// 2. Diff tables
	tableOps, err := d.diffTables(before, after)
	if err != nil {
		return nil, fmt.Errorf("failed to diff tables: %w", err)
	}
	ops = append(ops, tableOps...)

	return ops, nil
}

// diffEnums compares enums between schemas.
func (d *Differ) diffEnums(before, after *schema.Schema) ([]Operation, error) {
	var ops []Operation

	beforeEnums := make(map[string]schema.Enum)
	for _, e := range before.Enums {
		beforeEnums[e.Name] = e
	}

	afterEnums := make(map[string]schema.Enum)
	for _, e := range after.Enums {
		afterEnums[e.Name] = e
	}

	// Find created enums
	for name, enum := range afterEnums {
		if _, exists := beforeEnums[name]; !exists {
			ops = append(ops, CreateEnum{Enum: enum})
		}
	}

	// Find altered enums
	for name, afterEnum := range afterEnums {
		if beforeEnum, exists := beforeEnums[name]; exists {
			alterOp := d.diffEnum(beforeEnum, afterEnum)
			if alterOp != nil {
				ops = append(ops, *alterOp)
			}
		}
	}

	// TODO: Find dropped enums (should happen after tables are migrated)
	// Note: In practice, we should check if enum is still used by tables
	// For MVP, we skip dropping enums to avoid breaking references
	// Uncomment when DropEnum operation is implemented:
	// for name := range beforeEnums {
	// 	if _, ok := afterEnums[name]; !ok {
	// 		ops = append(ops, diff.DropEnum{Name: name})
	// 	}
	// }

	return ops, nil
}

// diffEnum compares two enum definitions.
func (d *Differ) diffEnum(before, after schema.Enum) *AlterEnum {
	beforeValues := make(map[string]bool)
	for _, v := range before.Values {
		beforeValues[v.Label] = true
	}

	afterValues := make(map[string]bool)
	for _, v := range after.Values {
		afterValues[v.Label] = true
	}

	var addValues, dropValues []string

	// Find added values
	for _, v := range after.Values {
		if !beforeValues[v.Label] {
			addValues = append(addValues, v.Label)
		}
	}

	// Find dropped values
	for _, v := range before.Values {
		if !afterValues[v.Label] {
			dropValues = append(dropValues, v.Label)
		}
	}

	if len(addValues) == 0 && len(dropValues) == 0 {
		return nil
	}

	return &AlterEnum{
		Name:       after.Name,
		AddValues:  addValues,
		DropValues: dropValues,
	}
}

// diffTables compares tables between schemas.
func (d *Differ) diffTables(before, after *schema.Schema) ([]Operation, error) {
	var ops []Operation

	beforeTables := make(map[string]schema.Table)
	for _, t := range before.Tables {
		beforeTables[t.Name] = t
	}

	afterTables := make(map[string]schema.Table)
	for _, t := range after.Tables {
		afterTables[t.Name] = t
	}

	// Find created tables
	for name, table := range afterTables {
		if _, exists := beforeTables[name]; !exists {
			ops = append(ops, CreateTable{Table: table})
		}
	}

	// Find altered tables
	for name, afterTable := range afterTables {
		if beforeTable, exists := beforeTables[name]; exists {
			tableOps, err := d.diffTable(beforeTable, afterTable)
			if err != nil {
				return nil, fmt.Errorf("failed to diff table %q: %w", name, err)
			}
			ops = append(ops, tableOps...)
		}
	}

	// Find dropped tables
	for name := range beforeTables {
		if _, exists := afterTables[name]; !exists {
			ops = append(ops, DropTable{Name: name})
		}
	}

	return ops, nil
}

// diffTable compares two table definitions.
func (d *Differ) diffTable(before, after schema.Table) ([]Operation, error) {
	var ops []Operation

	// 1. Diff columns
	columnOps, err := d.diffColumns(before, after)
	if err != nil {
		return nil, err
	}
	ops = append(ops, columnOps...)

	// 2. Diff indexes
	indexOps := d.diffIndexes(before, after)
	ops = append(ops, indexOps...)

	// 3. Diff foreign keys
	fkOps := d.diffForeignKeys(before, after)
	ops = append(ops, fkOps...)

	return ops, nil
}

// diffColumns compares columns between tables.
func (d *Differ) diffColumns(before, after schema.Table) ([]Operation, error) {
	var ops []Operation

	beforeCols := make(map[string]schema.Column)
	for _, c := range before.Columns {
		beforeCols[c.Name] = c
	}

	afterCols := make(map[string]schema.Column)
	for _, c := range after.Columns {
		afterCols[c.Name] = c
	}

	// Check for rename hints
	processedRenames := make(map[string]bool)

	for oldName, newName := range d.renameHints {
		// Format: "table.column" or just "column"
		// For MVP, we'll support simple column renames
		if beforeCol, exists := beforeCols[oldName]; exists {
			if afterCol, exists := afterCols[newName]; exists {
				ops = append(ops, RenameColumn{
					Table: after.Name,
					From:  oldName,
					To:    newName,
				})
				processedRenames[oldName] = true
				processedRenames[newName] = true

				// Check if properties changed after rename
				if !d.columnsEqual(beforeCol, afterCol) {
					ops = append(ops, d.createAlterColumn(after.Name, newName, beforeCol, afterCol))
				}
			}
		}
	}

	// Find added columns
	for name, col := range afterCols {
		if processedRenames[name] {
			continue
		}
		if _, exists := beforeCols[name]; !exists {
			ops = append(ops, AddColumn{
				Table:  after.Name,
				Column: col,
			})
		}
	}

	// Find altered columns
	for name, afterCol := range afterCols {
		if processedRenames[name] {
			continue
		}
		if beforeCol, exists := beforeCols[name]; exists {
			if !d.columnsEqual(beforeCol, afterCol) {
				ops = append(ops, d.createAlterColumn(after.Name, name, beforeCol, afterCol))
			}
		}
	}

	// Find dropped columns
	for name := range beforeCols {
		if processedRenames[name] {
			continue
		}
		if _, exists := afterCols[name]; !exists {
			ops = append(ops, DropColumn{
				Table:  after.Name,
				Column: name,
			})
		}
	}

	return ops, nil
}

// columnsEqual checks if two columns are equal.
func (d *Differ) columnsEqual(a, b schema.Column) bool {
	// Compare type
	if !d.dataTypesEqual(a.Type, b.Type) {
		return false
	}

	// Compare nullable
	if a.Nullable != b.Nullable {
		return false
	}

	// Compare default
	if !d.defaultsEqual(a.Default, b.Default) {
		return false
	}

	return true
}

// dataTypesEqual checks if two data types are equal.
func (d *Differ) dataTypesEqual(a, b schema.DataType) bool {
	if a.Kind != b.Kind {
		return false
	}
	if a.Precision != b.Precision {
		return false
	}
	if a.Scale != b.Scale {
		return false
	}
	if a.ArrayDepth != b.ArrayDepth {
		return false
	}
	if a.EnumName != b.EnumName {
		return false
	}
	return true
}

// defaultsEqual checks if two defaults are equal.
func (d *Differ) defaultsEqual(a, b *schema.DefaultValue) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Kind != b.Kind {
		return false
	}
	// Normalize literals for comparison (handles enum casts, quotes, etc.)
	if NormalizeLiteral(a.Literal) != NormalizeLiteral(b.Literal) {
		return false
	}
	// Normalize expressions for comparison
	if normalizeExpression(a.Expression) != normalizeExpression(b.Expression) {
		return false
	}
	// Compare sequence references
	if !d.sequenceRefsEqual(a.Sequence, b.Sequence) {
		return false
	}
	return true
}

// sequenceRefsEqual checks if two sequence references are equal.
func (d *Differ) sequenceRefsEqual(a, b *schema.SequenceRef) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name && a.Owned == b.Owned
}

// createAlterColumn creates an AlterColumn operation.
func (d *Differ) createAlterColumn(table, column string, before, after schema.Column) AlterColumn {
	return AlterColumn{
		Table:       table,
		Column:      column,
		OldType:     before.Type,
		NewType:     after.Type,
		OldNullable: before.Nullable,
		NewNullable: after.Nullable,
		OldDefault:  before.Default,
		NewDefault:  after.Default,
	}
}

// diffIndexes compares indexes between tables.
func (d *Differ) diffIndexes(before, after schema.Table) []Operation {
	var ops []Operation

	beforeIndexes := make(map[string]schema.Index)
	for _, idx := range before.Indexes {
		key := d.indexKey(idx)
		beforeIndexes[key] = idx
	}

	afterIndexes := make(map[string]schema.Index)
	for _, idx := range after.Indexes {
		key := d.indexKey(idx)
		afterIndexes[key] = idx
	}

	// Find created indexes
	for key, idx := range afterIndexes {
		if _, exists := beforeIndexes[key]; !exists {
			ops = append(ops, CreateIndex{
				Table: after.Name,
				Index: idx,
			})
		}
	}

	// Find dropped indexes
	for key, idx := range beforeIndexes {
		if _, exists := afterIndexes[key]; !exists {
			ops = append(ops, DropIndex{
				Table: before.Name,
				Name:  idx.Name,
			})
		}
	}

	return ops
}

// indexKey generates a unique key for an index based on its columns.
func (d *Differ) indexKey(idx schema.Index) string {
	key := ""
	for i, col := range idx.Columns {
		if i > 0 {
			key += ","
		}
		key += col.Name
	}
	return key
}

// diffForeignKeys compares foreign keys between tables.
func (d *Differ) diffForeignKeys(before, after schema.Table) []Operation {
	var ops []Operation

	beforeFKs := make(map[string]schema.ForeignKey)
	for _, fk := range before.ForeignKeys {
		key := d.foreignKeyKey(fk)
		beforeFKs[key] = fk
	}

	afterFKs := make(map[string]schema.ForeignKey)
	for _, fk := range after.ForeignKeys {
		key := d.foreignKeyKey(fk)
		afterFKs[key] = fk
	}

	// Find created foreign keys
	for key, fk := range afterFKs {
		if _, exists := beforeFKs[key]; !exists {
			ops = append(ops, CreateForeignKey{
				Table:      after.Name,
				Constraint: fk,
			})
		}
	}

	// Find dropped foreign keys
	for key, fk := range beforeFKs {
		if _, exists := afterFKs[key]; !exists {
			ops = append(ops, DropForeignKey{
				Table: before.Name,
				Name:  fk.Name,
			})
		}
	}

	return ops
}

// foreignKeyKey generates a unique key for a foreign key.
func (d *Differ) foreignKeyKey(fk schema.ForeignKey) string {
	key := fk.RefTable + ":"
	for i, col := range fk.Columns {
		if i > 0 {
			key += ","
		}
		key += col
	}
	return key
}
