package prisma

import (
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// relationField is a Prisma relation field emitted inside a model block.
type relationField struct {
	Name       string
	Type       string
	Attributes []string
}

// relationPlan holds the relation fields for every table, keyed by table name.
type relationPlan map[string][]relationField

// buildRelationPlan derives Prisma relation fields from the foreign keys in a schema.
//
// Prisma models a relation as a pair of fields: the owning side carries
// @relation(fields:..., references:...) and the referenced side carries a
// back-reference. A foreign key alone only produces the scalar column, so both
// sides have to be synthesized here.
func (w *Writer) buildRelationPlan(s *schema.Schema) (relationPlan, error) {
	tablesByName := make(map[string]*schema.Table, len(s.Tables))
	for i := range s.Tables {
		tablesByName[s.Tables[i].Name] = &s.Tables[i]
	}

	// Prisma requires an explicit relation name once two relations connect the same
	// pair of models, and for any self-relation. Count the pairs to detect that.
	pairCount := make(map[string]int)
	for _, table := range s.Tables {
		for _, fk := range table.ForeignKeys {
			pairCount[table.Name+"\x00"+fk.RefTable]++
		}
	}

	plan := make(relationPlan, len(s.Tables))

	for i := range s.Tables {
		table := &s.Tables[i]
		for _, fk := range table.ForeignKeys {
			if _, found := tablesByName[fk.RefTable]; !found {
				// The target lives outside the introspected schema, so no model exists to
				// point at. The scalar column is still emitted, which stays valid Prisma.
				continue
			}

			ownerModel := w.naming.TransformModelName(table.Name)
			refModel := w.naming.TransformModelName(fk.RefTable)

			forwardName := w.forwardRelationFieldName(fk)
			backName := w.naming.TransformFieldName(table.Name)

			// Disambiguate both sides when the pair is ambiguous or self-referencing.
			relationName := ""
			if pairCount[table.Name+"\x00"+fk.RefTable] > 1 || table.Name == fk.RefTable {
				relationName = ownerModel + capitalize(forwardName)
				backName = w.naming.TransformFieldName(table.Name) + capitalize(forwardName)
			}

			forward, err := w.forwardRelationField(forwardName, refModel, relationName, table, fk)
			if err != nil {
				return nil, err
			}
			plan[table.Name] = append(plan[table.Name], forward)

			plan[fk.RefTable] = append(plan[fk.RefTable], w.backRelationField(
				backName, ownerModel, relationName, table, fk,
			))
		}
	}

	return plan, nil
}

// forwardRelationField builds the owning side of a relation.
func (w *Writer) forwardRelationField(
	fieldName, refModel, relationName string,
	table *schema.Table,
	fk schema.ForeignKey,
) (relationField, error) {
	scalarFields := make([]string, len(fk.Columns))
	for i, col := range fk.Columns {
		scalarFields[i] = w.naming.TransformFieldName(col)
	}

	refFields := make([]string, len(fk.RefColumns))
	for i, col := range fk.RefColumns {
		refFields[i] = w.naming.TransformFieldName(col)
	}

	if len(refFields) == 0 {
		return relationField{}, fmt.Errorf(
			"foreign key %q on table %q has no referenced columns", fk.Name, table.Name,
		)
	}

	var args []string
	if relationName != "" {
		args = append(args, fmt.Sprintf("%q", relationName))
	}
	args = append(args,
		fmt.Sprintf("fields: [%s]", strings.Join(scalarFields, ", ")),
		fmt.Sprintf("references: [%s]", strings.Join(refFields, ", ")),
	)
	if action := prismaReferentialAction(fk.OnDelete); action != "" {
		args = append(args, "onDelete: "+action)
	}
	if action := prismaReferentialAction(fk.OnUpdate); action != "" {
		args = append(args, "onUpdate: "+action)
	}

	// A relation is optional exactly when the underlying scalar columns are nullable.
	fieldType := refModel
	if anyColumnNullable(table, fk.Columns) {
		fieldType += "?"
	}

	return relationField{
		Name:       fieldName,
		Type:       fieldType,
		Attributes: []string{fmt.Sprintf("@relation(%s)", strings.Join(args, ", "))},
	}, nil
}

// backRelationField builds the referenced side of a relation. It is a list unless the
// foreign key columns are unique, which makes the relation one-to-one.
func (w *Writer) backRelationField(
	fieldName, ownerModel, relationName string,
	table *schema.Table,
	fk schema.ForeignKey,
) relationField {
	fieldType := ownerModel + "[]"
	if hasUniqueOn(table, fk.Columns) {
		fieldType = ownerModel + "?"
	}

	var attrs []string
	if relationName != "" {
		attrs = append(attrs, fmt.Sprintf("@relation(%q)", relationName))
	}

	return relationField{
		Name:       fieldName,
		Type:       fieldType,
		Attributes: attrs,
	}
}

// forwardRelationFieldName names the owning side of a relation.
//
// A single "user_id" column yields "user", which reads naturally and keeps two keys
// into the same table ("author_id", "editor_id") distinct. Anything else falls back
// to the referenced table name.
func (w *Writer) forwardRelationFieldName(fk schema.ForeignKey) string {
	if len(fk.Columns) == 1 {
		col := fk.Columns[0]
		base := strings.TrimSuffix(col, "_id")
		if base == col {
			base = strings.TrimSuffix(col, "Id")
		}
		if base != col && base != "" {
			return w.naming.TransformFieldName(base)
		}
	}
	return w.naming.TransformFieldName(fk.RefTable)
}

// anyColumnNullable reports whether any of the named columns is nullable.
func anyColumnNullable(table *schema.Table, columns []string) bool {
	for _, name := range columns {
		for _, col := range table.Columns {
			if col.Name == name && col.Nullable {
				return true
			}
		}
	}
	return false
}

// hasUniqueOn reports whether the exact column set is backed by a primary key,
// unique constraint, or unique index.
func hasUniqueOn(table *schema.Table, columns []string) bool {
	if table.PrimaryKey != nil && sameColumnSet(table.PrimaryKey.Columns, columns) {
		return true
	}

	for _, constraint := range table.Constraints {
		if constraint.Type == schema.ConstraintUnique && sameColumnSet(constraint.Columns, columns) {
			return true
		}
	}

	for _, index := range table.Indexes {
		if !index.Unique {
			continue
		}
		names := make([]string, len(index.Columns))
		for i, idxCol := range index.Columns {
			names[i] = idxCol.Name
		}
		if sameColumnSet(names, columns) {
			return true
		}
	}

	return false
}

// sameColumnSet compares two column lists ignoring order.
func sameColumnSet(a, b []string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}

	seen := make(map[string]int, len(a))
	for _, name := range a {
		seen[name]++
	}
	for _, name := range b {
		seen[name]--
		if seen[name] < 0 {
			return false
		}
	}

	return true
}

// prismaReferentialAction maps an internal referential action to Prisma syntax.
// An empty result means the action is left to the Prisma default.
func prismaReferentialAction(action schema.ReferentialAction) string {
	switch action {
	case schema.ActionCascade:
		return "Cascade"
	case schema.ActionRestrict:
		return "Restrict"
	case schema.ActionSetNull:
		return "SetNull"
	case schema.ActionSetDefault:
		return "SetDefault"
	case schema.ActionNoAction:
		return "NoAction"
	default:
		return ""
	}
}
