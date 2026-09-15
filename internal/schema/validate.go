package schema

import (
	"fmt"
	"strings"
)

// Validate checks schema-level invariants.
func (s Schema) Validate() error {
	var errs []string

	// Check for duplicate table names
	tableNames := make(map[string]bool)
	for _, table := range s.Tables {
		if tableNames[table.Name] {
			errs = append(errs, fmt.Sprintf("duplicate table name: %q", table.Name))
		}
		tableNames[table.Name] = true
	}

	// Check for duplicate enum names
	enumNames := make(map[string]bool)
	for _, enum := range s.Enums {
		if enumNames[enum.Name] {
			errs = append(errs, fmt.Sprintf("duplicate enum name: %q", enum.Name))
		}
		enumNames[enum.Name] = true
	}

	// Check that foreign keys reference existing tables
	for _, table := range s.Tables {
		for _, fk := range table.ForeignKeys {
			if !tableNames[fk.RefTable] {
				errs = append(errs, fmt.Sprintf("table %q foreign key %q references non-existent table %q",
					table.Name, fk.Name, fk.RefTable))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("schema validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// Validate checks table-level invariants.
func (t Table) Validate() error {
	var errs []string

	// Table must have at least one column
	if len(t.Columns) == 0 {
		errs = append(errs, fmt.Sprintf("table %q has no columns", t.Name))
	}

	// Check for duplicate column names
	columnNames := make(map[string]bool)
	for _, col := range t.Columns {
		if columnNames[col.Name] {
			errs = append(errs, fmt.Sprintf("table %q has duplicate column name: %q", t.Name, col.Name))
		}
		columnNames[col.Name] = true
	}

	// Validate primary key columns exist
	if t.PrimaryKey != nil {
		for _, pkCol := range t.PrimaryKey.Columns {
			if !columnNames[pkCol] {
				errs = append(errs, fmt.Sprintf("table %q primary key references non-existent column: %q",
					t.Name, pkCol))
			}
		}
	}

	// Validate foreign keys
	for _, fk := range t.ForeignKeys {
		// Check FK columns exist
		for _, fkCol := range fk.Columns {
			if !columnNames[fkCol] {
				errs = append(errs, fmt.Sprintf("table %q foreign key %q references non-existent column: %q",
					t.Name, fk.Name, fkCol))
			}
		}

		// Check column count matches ref column count
		if len(fk.Columns) != len(fk.RefColumns) {
			errs = append(errs, fmt.Sprintf("table %q foreign key %q has %d columns but %d reference columns",
				t.Name, fk.Name, len(fk.Columns), len(fk.RefColumns)))
		}
	}

	// Validate index columns exist
	for _, idx := range t.Indexes {
		for _, idxCol := range idx.Columns {
			if !columnNames[idxCol.Name] {
				errs = append(errs, fmt.Sprintf("table %q index %q references non-existent column: %q",
					t.Name, idx.Name, idxCol.Name))
			}
		}
	}

	// Validate constraint columns exist
	for _, constraint := range t.Constraints {
		for _, col := range constraint.Columns {
			if !columnNames[col] {
				errs = append(errs, fmt.Sprintf("table %q constraint %q references non-existent column: %q",
					t.Name, constraint.Name, col))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("table validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// Validate checks column-level invariants.
func (c Column) Validate() error {
	var errs []string

	// Name must not be empty
	if strings.TrimSpace(c.Name) == "" {
		errs = append(errs, "column name cannot be empty")
	}

	// Type must be valid (not TypeUnknown)
	if c.Type.Kind == TypeUnknown {
		errs = append(errs, fmt.Sprintf("column %q has unknown type", c.Name))
	}

	if len(errs) > 0 {
		return fmt.Errorf("column validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// Validate checks foreign key invariants.
func (fk ForeignKey) Validate() error {
	var errs []string

	// Name must not be empty
	if strings.TrimSpace(fk.Name) == "" {
		errs = append(errs, "foreign key name cannot be empty")
	}

	// Must have at least one column
	if len(fk.Columns) == 0 {
		errs = append(errs, fmt.Sprintf("foreign key %q has no columns", fk.Name))
	}

	// RefTable must not be empty
	if strings.TrimSpace(fk.RefTable) == "" {
		errs = append(errs, fmt.Sprintf("foreign key %q has empty reference table", fk.Name))
	}

	// Column count must match reference column count
	if len(fk.Columns) != len(fk.RefColumns) {
		errs = append(errs, fmt.Sprintf("foreign key %q has %d columns but %d reference columns",
			fk.Name, len(fk.Columns), len(fk.RefColumns)))
	}

	if len(errs) > 0 {
		return fmt.Errorf("foreign key validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}
