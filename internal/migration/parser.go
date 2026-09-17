package migration

import (
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// Parser parses SQL DDL statements into AST representations.
type Parser struct{}

// NewParser creates a new SQL parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a SQL statement and returns its AST representation.
func (p *Parser) Parse(sql string) (Statement, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return nil, &ParseError{SQL: sql, Message: "empty SQL statement"}
	}

	// Remove trailing semicolon
	sql = strings.TrimSuffix(sql, ";")
	sql = strings.TrimSpace(sql)

	tokens := tokenize(sql)
	if len(tokens) == 0 {
		return nil, &ParseError{SQL: sql, Message: "no tokens found"}
	}

	// Determine statement type from first keyword
	switch strings.ToUpper(tokens[0]) {
	case "CREATE":
		return p.parseCreate(tokens, sql)
	case "DROP":
		return p.parseDrop(tokens, sql)
	case "ALTER":
		return p.parseAlter(tokens, sql)
	default:
		return nil, &ParseError{SQL: sql, Message: fmt.Sprintf("unsupported statement type: %s", tokens[0])}
	}
}

// parseCreate handles CREATE TABLE, CREATE TYPE, CREATE INDEX statements.
func (p *Parser) parseCreate(tokens []string, sql string) (Statement, error) {
	if len(tokens) < 2 {
		return nil, &ParseError{SQL: sql, Message: "incomplete CREATE statement"}
	}

	switch strings.ToUpper(tokens[1]) {
	case "TABLE":
		return p.parseCreateTable(tokens[2:], sql)
	case "TYPE":
		return p.parseCreateType(tokens[2:], sql)
	case "INDEX", "UNIQUE":
		return p.parseCreateIndex(tokens[1:], sql)
	default:
		return nil, &ParseError{SQL: sql, Message: fmt.Sprintf("unsupported CREATE statement: CREATE %s", tokens[1])}
	}
}

// parseDrop handles DROP TABLE, DROP TYPE, DROP INDEX statements.
func (p *Parser) parseDrop(tokens []string, sql string) (Statement, error) {
	if len(tokens) < 2 {
		return nil, &ParseError{SQL: sql, Message: "incomplete DROP statement"}
	}

	switch strings.ToUpper(tokens[1]) {
	case "TABLE":
		return p.parseDropTable(tokens[2:], sql)
	case "TYPE":
		return p.parseDropType(tokens[2:], sql)
	case "INDEX":
		return p.parseDropIndex(tokens[2:], sql)
	default:
		return nil, &ParseError{SQL: sql, Message: fmt.Sprintf("unsupported DROP statement: DROP %s", tokens[1])}
	}
}

// parseAlter handles ALTER TABLE and ALTER TYPE statements.
func (p *Parser) parseAlter(tokens []string, sql string) (Statement, error) {
	if len(tokens) < 2 {
		return nil, &ParseError{SQL: sql, Message: "incomplete ALTER statement"}
	}

	switch strings.ToUpper(tokens[1]) {
	case "TABLE":
		return p.parseAlterTable(tokens[2:], sql)
	case "TYPE":
		return p.parseAlterType(tokens[2:], sql)
	default:
		return nil, &ParseError{SQL: sql, Message: fmt.Sprintf("unsupported ALTER statement: ALTER %s", tokens[1])}
	}
}

// parseCreateTable parses CREATE TABLE statement.
func (p *Parser) parseCreateTable(tokens []string, sql string) (*CreateTable, error) {
	stmt := &CreateTable{}
	idx := 0

	// Check for IF NOT EXISTS
	if idx+2 < len(tokens) && strings.ToUpper(tokens[idx]) == "IF" &&
		strings.ToUpper(tokens[idx+1]) == "NOT" && strings.ToUpper(tokens[idx+2]) == "EXISTS" {
		stmt.IfNotExists = true
		idx += 3
	}

	// Get table name
	if idx >= len(tokens) {
		return nil, &ParseError{SQL: sql, Message: "missing table name"}
	}
	stmt.Name = unquoteIdentifier(tokens[idx])

	// Find column definitions within parentheses
	parenStart := strings.Index(sql[strings.Index(sql, stmt.Name)+len(stmt.Name):], "(")
	if parenStart == -1 {
		return nil, &ParseError{SQL: sql, Message: "missing opening parenthesis"}
	}

	// Extract content between parentheses
	parenContent := extractParentheses(sql[strings.Index(sql, stmt.Name)+len(stmt.Name)+parenStart:])
	if parenContent == "" {
		return nil, &ParseError{SQL: sql, Message: "empty table definition"}
	}

	// Parse column and constraint definitions
	definitions := splitByComma(parenContent)
	for _, def := range definitions {
		def = strings.TrimSpace(def)
		if def == "" {
			continue
		}

		// Check if it's a table-level constraint
		defUpper := strings.ToUpper(def)
		if strings.HasPrefix(defUpper, "PRIMARY KEY") || strings.HasPrefix(defUpper, "FOREIGN KEY") ||
			strings.HasPrefix(defUpper, "UNIQUE") || strings.HasPrefix(defUpper, "CHECK") ||
			strings.HasPrefix(defUpper, "CONSTRAINT") {
			constraint, err := p.parseTableConstraint(def, sql)
			if err != nil {
				return nil, err
			}
			stmt.Constraints = append(stmt.Constraints, constraint)
		} else {
			// It's a column definition
			col, err := p.parseColumnDef(def, sql)
			if err != nil {
				return nil, err
			}
			stmt.Columns = append(stmt.Columns, col)
		}
	}

	return stmt, nil
}

// parseColumnDef parses a column definition.
func (p *Parser) parseColumnDef(def string, sql string) (ColumnDef, error) {
	tokens := tokenize(def)
	if len(tokens) < 2 {
		return ColumnDef{}, &ParseError{SQL: sql, Message: "incomplete column definition"}
	}

	col := ColumnDef{
		Name:     unquoteIdentifier(tokens[0]),
		Nullable: true, // Default is nullable
	}

	// Parse data type (may include precision/scale)
	typeIdx := 1
	col.Type = tokens[typeIdx]

	// Handle types with parameters like VARCHAR(255) or DECIMAL(10,2)
	if typeIdx+1 < len(tokens) && strings.HasPrefix(tokens[typeIdx+1], "(") {
		// Type has precision/scale
		parenContent := extractParentheses(def[strings.Index(def, tokens[typeIdx])+len(tokens[typeIdx]):])
		col.Type = tokens[typeIdx] + "(" + parenContent + ")"
		typeIdx++
	}

	// Parse column constraints and modifiers
	idx := typeIdx + 1
	for idx < len(tokens) {
		switch strings.ToUpper(tokens[idx]) {
		case "NOT":
			if idx+1 < len(tokens) && strings.ToUpper(tokens[idx+1]) == "NULL" {
				col.Nullable = false
				idx += 2
			} else {
				idx++
			}
		case "NULL":
			col.Nullable = true
			idx++
		case "DEFAULT":
			if idx+1 >= len(tokens) {
				return col, &ParseError{SQL: sql, Message: "missing DEFAULT value"}
			}
			idx++
			// Extract default value (may be expression with parentheses)
			defaultVal := tokens[idx]
			// Check if next token is a function call (e.g., NOW())
			if idx+1 < len(tokens) && strings.HasPrefix(tokens[idx+1], "(") {
				// Function call like NOW()
				defaultVal = defaultVal + tokens[idx+1]
				idx++
			} else if strings.HasPrefix(defaultVal, "(") {
				// Default is an expression wrapped in parentheses
				defaultVal = extractParentheses(def[strings.Index(def[strings.Index(def, "DEFAULT")+7:], "(")+strings.Index(def, "DEFAULT")+7:])
			}
			// Check for type cast (::typename)
			if idx+1 < len(tokens) && tokens[idx+1] == "::" {
				// Include type cast in default value
				defaultVal = defaultVal + "::" + tokens[idx+2]
				idx += 2
			}
			col.Default = &defaultVal
			idx++
		case "PRIMARY":
			if idx+1 < len(tokens) && strings.ToUpper(tokens[idx+1]) == "KEY" {
				col.Constraints = append(col.Constraints, ColumnConstraint{Type: PrimaryKeyConstraint})
				col.Nullable = false
				idx += 2
			} else {
				idx++
			}
		case "UNIQUE":
			col.Constraints = append(col.Constraints, ColumnConstraint{Type: UniqueConstraint})
			idx++
		case "REFERENCES":
			// Parse foreign key reference
			if idx+1 >= len(tokens) {
				return col, &ParseError{SQL: sql, Message: "incomplete REFERENCES clause"}
			}
			idx++
			fk := ColumnConstraint{Type: ForeignKeyConstraint}
			fk.RefTable = unquoteIdentifier(tokens[idx])
			idx++

			// Parse referenced columns
			if idx < len(tokens) && strings.HasPrefix(tokens[idx], "(") {
				refCols := extractParentheses(def[strings.Index(def[strings.Index(def, fk.RefTable)+len(fk.RefTable):], "(")+strings.Index(def, fk.RefTable)+len(fk.RefTable):])
				fk.RefColumns = splitByComma(refCols)
				for i := range fk.RefColumns {
					fk.RefColumns[i] = unquoteIdentifier(strings.TrimSpace(fk.RefColumns[i]))
				}
				idx++
			}

			// Parse ON DELETE/UPDATE
			for idx < len(tokens) {
				if strings.ToUpper(tokens[idx]) == "ON" {
					if idx+1 >= len(tokens) {
						break
					}
					action := strings.ToUpper(tokens[idx+1])
					switch action {
					case "DELETE":
						if idx+2 < len(tokens) {
							fk.OnDelete = parseReferentialAction(tokens[idx+2:])
							idx += 2 + countActionTokens(tokens[idx+2:])
						}
					case "UPDATE":
						if idx+2 < len(tokens) {
							fk.OnUpdate = parseReferentialAction(tokens[idx+2:])
							idx += 2 + countActionTokens(tokens[idx+2:])
						}
					}
				} else {
					break
				}
			}

			col.Constraints = append(col.Constraints, fk)
		case "CHECK":
			if idx+1 < len(tokens) && strings.HasPrefix(tokens[idx+1], "(") {
				checkExpr := extractParentheses(def[strings.Index(def[strings.Index(def, "CHECK")+5:], "(")+strings.Index(def, "CHECK")+5:])
				col.Constraints = append(col.Constraints, ColumnConstraint{
					Type:      CheckConstraint,
					CheckExpr: checkExpr,
				})
				idx += 2
			} else {
				idx++
			}
		default:
			idx++
		}
	}

	return col, nil
}

// parseTableConstraint parses a table-level constraint.
func (p *Parser) parseTableConstraint(def string, sql string) (TableConstraint, error) {
	tokens := tokenize(def)
	constraint := TableConstraint{}
	idx := 0

	// Check for CONSTRAINT name
	if strings.ToUpper(tokens[idx]) == "CONSTRAINT" {
		if idx+1 >= len(tokens) {
			return constraint, &ParseError{SQL: sql, Message: "missing constraint name"}
		}
		idx++
		constraint.Name = unquoteIdentifier(tokens[idx])
		idx++
	}

	if idx >= len(tokens) {
		return constraint, &ParseError{SQL: sql, Message: "incomplete constraint definition"}
	}

	switch strings.ToUpper(tokens[idx]) {
	case "PRIMARY":
		if idx+1 < len(tokens) && strings.ToUpper(tokens[idx+1]) == "KEY" {
			constraint.Type = PrimaryKeyConstraint
			idx += 2
			// Extract columns
			if idx < len(tokens) && strings.HasPrefix(tokens[idx], "(") {
				cols := extractParentheses(def[strings.Index(def[strings.Index(def, "KEY")+3:], "(")+strings.Index(def, "KEY")+3:])
				constraint.Columns = splitByComma(cols)
				for i := range constraint.Columns {
					constraint.Columns[i] = unquoteIdentifier(strings.TrimSpace(constraint.Columns[i]))
				}
			}
		}
	case "FOREIGN":
		if idx+1 < len(tokens) && strings.ToUpper(tokens[idx+1]) == "KEY" {
			constraint.Type = ForeignKeyConstraint
			idx += 2
			// Extract source columns
			if idx < len(tokens) && strings.HasPrefix(tokens[idx], "(") {
				cols := extractParentheses(def[strings.Index(def[strings.Index(def, "KEY")+3:], "(")+strings.Index(def, "KEY")+3:])
				constraint.Columns = splitByComma(cols)
				for i := range constraint.Columns {
					constraint.Columns[i] = unquoteIdentifier(strings.TrimSpace(constraint.Columns[i]))
				}
				idx++
			}

			// Parse REFERENCES
			for idx < len(tokens) && strings.ToUpper(tokens[idx]) != "REFERENCES" {
				idx++
			}
			if idx < len(tokens) {
				idx++ // Skip REFERENCES
				if idx < len(tokens) {
					constraint.RefTable = unquoteIdentifier(tokens[idx])
					idx++

					// Extract referenced columns
					if idx < len(tokens) && strings.HasPrefix(tokens[idx], "(") {
						refCols := extractParentheses(def[strings.Index(def[strings.Index(def, constraint.RefTable)+len(constraint.RefTable):], "(")+strings.Index(def, constraint.RefTable)+len(constraint.RefTable):])
						constraint.RefColumns = splitByComma(refCols)
						for i := range constraint.RefColumns {
							constraint.RefColumns[i] = unquoteIdentifier(strings.TrimSpace(constraint.RefColumns[i]))
						}
						idx++
					}

					// Parse ON DELETE/UPDATE
					for idx < len(tokens) {
						if strings.ToUpper(tokens[idx]) == "ON" {
							if idx+1 >= len(tokens) {
								break
							}
							action := strings.ToUpper(tokens[idx+1])
							switch action {
							case "DELETE":
								if idx+2 < len(tokens) {
									constraint.OnDelete = parseReferentialAction(tokens[idx+2:])
									idx += 2 + countActionTokens(tokens[idx+2:])
								}
							case "UPDATE":
								if idx+2 < len(tokens) {
									constraint.OnUpdate = parseReferentialAction(tokens[idx+2:])
									idx += 2 + countActionTokens(tokens[idx+2:])
								}
							}
						} else {
							break
						}
					}
				}
			}
		}
	case "UNIQUE":
		constraint.Type = UniqueConstraint
		idx++
		// Extract columns
		if idx < len(tokens) && strings.HasPrefix(tokens[idx], "(") {
			cols := extractParentheses(def[strings.Index(def[strings.Index(def, "UNIQUE")+6:], "(")+strings.Index(def, "UNIQUE")+6:])
			constraint.Columns = splitByComma(cols)
			for i := range constraint.Columns {
				constraint.Columns[i] = unquoteIdentifier(strings.TrimSpace(constraint.Columns[i]))
			}
		}
	case "CHECK":
		constraint.Type = CheckConstraint
		idx++
		// Extract check expression
		if idx < len(tokens) && strings.HasPrefix(tokens[idx], "(") {
			constraint.CheckExpr = extractParentheses(def[strings.Index(def[strings.Index(def, "CHECK")+5:], "(")+strings.Index(def, "CHECK")+5:])
		}
	}

	return constraint, nil
}

// parseDropTable parses DROP TABLE statement.
func (p *Parser) parseDropTable(tokens []string, sql string) (*DropTable, error) {
	stmt := &DropTable{}
	idx := 0

	// Check for IF EXISTS
	if idx+1 < len(tokens) && strings.ToUpper(tokens[idx]) == "IF" &&
		strings.ToUpper(tokens[idx+1]) == "EXISTS" {
		stmt.IfExists = true
		idx += 2
	}

	// Get table name
	if idx >= len(tokens) {
		return nil, &ParseError{SQL: sql, Message: "missing table name"}
	}
	stmt.Name = unquoteIdentifier(tokens[idx])
	idx++

	// Check for CASCADE
	if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "CASCADE" {
		stmt.Cascade = true
	}

	return stmt, nil
}

// parseAlterTable parses ALTER TABLE statement.
func (p *Parser) parseAlterTable(tokens []string, sql string) (*AlterTable, error) {
	if len(tokens) < 2 {
		return nil, &ParseError{SQL: sql, Message: "incomplete ALTER TABLE statement"}
	}

	stmt := &AlterTable{
		Table: unquoteIdentifier(tokens[0]),
	}

	// Parse the action
	actionTokens := tokens[1:]
	if len(actionTokens) == 0 {
		return nil, &ParseError{SQL: sql, Message: "missing ALTER TABLE action"}
	}

	switch strings.ToUpper(actionTokens[0]) {
	case "ADD":
		if len(actionTokens) < 2 {
			return nil, &ParseError{SQL: sql, Message: "incomplete ADD action"}
		}
		if strings.ToUpper(actionTokens[1]) == "COLUMN" {
			// ADD COLUMN
			colDef := strings.Join(actionTokens[2:], " ")
			col, err := p.parseColumnDef(colDef, sql)
			if err != nil {
				return nil, err
			}
			stmt.Action = &AddColumn{Column: col}
		}
	case "DROP":
		if len(actionTokens) < 2 {
			return nil, &ParseError{SQL: sql, Message: "incomplete DROP action"}
		}
		if strings.ToUpper(actionTokens[1]) == "COLUMN" {
			drop := &DropColumn{}
			idx := 2

			// Check for IF EXISTS
			if idx+1 < len(actionTokens) && strings.ToUpper(actionTokens[idx]) == "IF" &&
				strings.ToUpper(actionTokens[idx+1]) == "EXISTS" {
				drop.IfExists = true
				idx += 2
			}

			if idx >= len(actionTokens) {
				return nil, &ParseError{SQL: sql, Message: "missing column name"}
			}
			drop.Name = unquoteIdentifier(actionTokens[idx])
			idx++

			// Check for CASCADE
			if idx < len(actionTokens) && strings.ToUpper(actionTokens[idx]) == "CASCADE" {
				drop.Cascade = true
			}

			stmt.Action = drop
		}
	case "RENAME":
		if len(actionTokens) < 4 || strings.ToUpper(actionTokens[1]) != "COLUMN" {
			return nil, &ParseError{SQL: sql, Message: "incomplete RENAME COLUMN action"}
		}
		if strings.ToUpper(actionTokens[3]) != "TO" {
			return nil, &ParseError{SQL: sql, Message: "missing TO in RENAME COLUMN"}
		}
		if len(actionTokens) < 5 {
			return nil, &ParseError{SQL: sql, Message: "missing new column name"}
		}
		stmt.Action = &RenameColumn{
			OldName: unquoteIdentifier(actionTokens[2]),
			NewName: unquoteIdentifier(actionTokens[4]),
		}
	case "ALTER":
		if len(actionTokens) < 3 || strings.ToUpper(actionTokens[1]) != "COLUMN" {
			return nil, &ParseError{SQL: sql, Message: "incomplete ALTER COLUMN action"}
		}
		alter := &AlterColumn{
			Name: unquoteIdentifier(actionTokens[2]),
		}
		idx := 3

		for idx < len(actionTokens) {
			switch strings.ToUpper(actionTokens[idx]) {
			case "SET":
				if idx+1 >= len(actionTokens) {
					return nil, &ParseError{SQL: sql, Message: "incomplete SET clause"}
				}
				idx++
				switch strings.ToUpper(actionTokens[idx]) {
				case "NOT":
					if idx+1 < len(actionTokens) && strings.ToUpper(actionTokens[idx+1]) == "NULL" {
						alter.SetNotNull = true
						idx += 2
					} else {
						idx++
					}
				case "DEFAULT":
					if idx+1 >= len(actionTokens) {
						return nil, &ParseError{SQL: sql, Message: "missing DEFAULT value"}
					}
					idx++
					defaultVal := actionTokens[idx]
					// Check for type cast (::typename)
					if idx+1 < len(actionTokens) && actionTokens[idx+1] == "::" {
						// Include type cast in default value
						if idx+2 < len(actionTokens) {
							defaultVal = defaultVal + "::" + actionTokens[idx+2]
							idx += 2
						}
					}
					alter.SetDefault = &defaultVal
					idx++
				default:
					idx++
				}
			case "DROP":
				if idx+1 >= len(actionTokens) {
					return nil, &ParseError{SQL: sql, Message: "incomplete DROP clause"}
				}
				idx++
				switch strings.ToUpper(actionTokens[idx]) {
				case "NOT":
					if idx+1 < len(actionTokens) && strings.ToUpper(actionTokens[idx+1]) == "NULL" {
						alter.DropNotNull = true
						idx += 2
					} else {
						idx++
					}
				case "DEFAULT":
					alter.DropDefault = true
					idx++
				default:
					idx++
				}
			case "TYPE":
				if idx+1 >= len(actionTokens) {
					return nil, &ParseError{SQL: sql, Message: "missing TYPE value"}
				}
				idx++
				alter.SetType = actionTokens[idx]
				idx++
			default:
				idx++
			}
		}

		stmt.Action = alter
	default:
		return nil, &ParseError{SQL: sql, Message: fmt.Sprintf("unsupported ALTER TABLE action: %s", actionTokens[0])}
	}

	return stmt, nil
}

// parseCreateType parses CREATE TYPE (enum) statement.
func (p *Parser) parseCreateType(tokens []string, sql string) (*CreateType, error) {
	if len(tokens) < 1 {
		return nil, &ParseError{SQL: sql, Message: "missing type name"}
	}

	stmt := &CreateType{
		Name: unquoteIdentifier(tokens[0]),
	}

	// Find AS ENUM
	asIdx := -1
	for i, token := range tokens {
		if strings.ToUpper(token) == "AS" {
			asIdx = i
			break
		}
	}

	if asIdx == -1 || asIdx+1 >= len(tokens) || strings.ToUpper(tokens[asIdx+1]) != "ENUM" {
		return nil, &ParseError{SQL: sql, Message: "expected AS ENUM"}
	}

	// Extract enum values from parentheses
	parenStart := strings.Index(sql[strings.Index(strings.ToUpper(sql), "ENUM")+4:], "(")
	if parenStart == -1 {
		return nil, &ParseError{SQL: sql, Message: "missing enum values"}
	}

	valuesStr := extractParentheses(sql[strings.Index(strings.ToUpper(sql), "ENUM")+4+parenStart:])
	if valuesStr == "" {
		return nil, &ParseError{SQL: sql, Message: "empty enum values"}
	}

	// Split by comma and trim quotes
	values := splitByComma(valuesStr)
	for _, v := range values {
		v = strings.TrimSpace(v)
		// Remove surrounding quotes
		if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') {
			v = v[1 : len(v)-1]
		}
		stmt.Values = append(stmt.Values, v)
	}

	return stmt, nil
}

// parseDropType parses DROP TYPE statement.
func (p *Parser) parseDropType(tokens []string, sql string) (*DropType, error) {
	stmt := &DropType{}
	idx := 0

	// Check for IF EXISTS
	if idx+1 < len(tokens) && strings.ToUpper(tokens[idx]) == "IF" &&
		strings.ToUpper(tokens[idx+1]) == "EXISTS" {
		stmt.IfExists = true
		idx += 2
	}

	// Get type name
	if idx >= len(tokens) {
		return nil, &ParseError{SQL: sql, Message: "missing type name"}
	}
	stmt.Name = unquoteIdentifier(tokens[idx])
	idx++

	// Check for CASCADE
	if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "CASCADE" {
		stmt.Cascade = true
	}

	return stmt, nil
}

// parseAlterType parses ALTER TYPE statement.
func (p *Parser) parseAlterType(tokens []string, sql string) (*AlterType, error) {
	if len(tokens) < 2 {
		return nil, &ParseError{SQL: sql, Message: "incomplete ALTER TYPE statement"}
	}

	stmt := &AlterType{
		Name: unquoteIdentifier(tokens[0]),
	}

	// Parse action
	if strings.ToUpper(tokens[1]) == "ADD" {
		if len(tokens) < 3 || strings.ToUpper(tokens[2]) != "VALUE" {
			return nil, &ParseError{SQL: sql, Message: "expected ADD VALUE"}
		}

		if len(tokens) < 4 {
			return nil, &ParseError{SQL: sql, Message: "missing enum value"}
		}

		value := tokens[3]
		// Remove quotes
		if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
			value = value[1 : len(value)-1]
		}

		addValue := &AddEnumValue{Value: value}

		// Check for BEFORE/AFTER
		if len(tokens) > 4 {
			if strings.ToUpper(tokens[4]) == "BEFORE" && len(tokens) > 5 {
				before := tokens[5]
				if len(before) >= 2 && (before[0] == '\'' || before[0] == '"') {
					before = before[1 : len(before)-1]
				}
				addValue.Before = before
			} else if strings.ToUpper(tokens[4]) == "AFTER" && len(tokens) > 5 {
				after := tokens[5]
				if len(after) >= 2 && (after[0] == '\'' || after[0] == '"') {
					after = after[1 : len(after)-1]
				}
				addValue.After = after
			}
		}

		stmt.Action = addValue
	}

	return stmt, nil
}

// parseCreateIndex parses CREATE INDEX statement.
func (p *Parser) parseCreateIndex(tokens []string, sql string) (*CreateIndex, error) {
	stmt := &CreateIndex{}
	idx := 0

	// Check for UNIQUE
	if strings.ToUpper(tokens[idx]) == "UNIQUE" {
		stmt.Unique = true
		idx++
	}

	// Skip INDEX keyword
	if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "INDEX" {
		idx++
	}

	// Check for CONCURRENTLY
	if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "CONCURRENTLY" {
		stmt.Concurrently = true
		idx++
	}

	// Get index name
	if idx >= len(tokens) {
		return nil, &ParseError{SQL: sql, Message: "missing index name"}
	}
	stmt.Name = unquoteIdentifier(tokens[idx])
	idx++

	// Expect ON
	if idx >= len(tokens) || strings.ToUpper(tokens[idx]) != "ON" {
		return nil, &ParseError{SQL: sql, Message: "expected ON"}
	}
	idx++

	// Get table name
	if idx >= len(tokens) {
		return nil, &ParseError{SQL: sql, Message: "missing table name"}
	}
	stmt.Table = unquoteIdentifier(tokens[idx])
	idx++

	// Check for USING method
	if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "USING" {
		idx++
		if idx >= len(tokens) {
			return nil, &ParseError{SQL: sql, Message: "missing index method"}
		}
		stmt.Method = strings.ToLower(tokens[idx])
		idx++
	} else {
		stmt.Method = "btree" // Default method
	}

	// Extract columns from parentheses
	if idx >= len(tokens) {
		return nil, &ParseError{SQL: sql, Message: "missing column list"}
	}

	// Find the column list in the original SQL
	colStart := strings.Index(sql[strings.LastIndex(sql, stmt.Table)+len(stmt.Table):], "(")
	if colStart == -1 {
		return nil, &ParseError{SQL: sql, Message: "missing column list"}
	}

	colsStr := extractParentheses(sql[strings.LastIndex(sql, stmt.Table)+len(stmt.Table)+colStart:])
	if colsStr == "" {
		return nil, &ParseError{SQL: sql, Message: "empty column list"}
	}

	// Parse columns
	cols := splitByComma(colsStr)
	for _, col := range cols {
		col = strings.TrimSpace(col)
		colTokens := tokenize(col)
		if len(colTokens) == 0 {
			continue
		}

		indexCol := IndexColumnDef{
			Name: unquoteIdentifier(colTokens[0]),
		}

		// Check for ASC/DESC
		if len(colTokens) > 1 {
			order := strings.ToUpper(colTokens[1])
			if order == "ASC" || order == "DESC" {
				indexCol.Order = order
			}
		}

		stmt.Columns = append(stmt.Columns, indexCol)
	}

	// Check for WHERE clause (partial index)
	whereIdx := -1
	for i := idx; i < len(tokens); i++ {
		if strings.ToUpper(tokens[i]) == "WHERE" {
			whereIdx = i
			break
		}
	}

	if whereIdx != -1 {
		// Extract WHERE clause from SQL
		wherePos := strings.LastIndex(strings.ToUpper(sql), "WHERE")
		if wherePos != -1 {
			stmt.Where = strings.TrimSpace(sql[wherePos+5:])
		}
	}

	return stmt, nil
}

// parseDropIndex parses DROP INDEX statement.
func (p *Parser) parseDropIndex(tokens []string, sql string) (*DropIndex, error) {
	stmt := &DropIndex{}
	idx := 0

	// Check for CONCURRENTLY
	if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "CONCURRENTLY" {
		stmt.Concurrently = true
		idx++
	}

	// Check for IF EXISTS
	if idx+1 < len(tokens) && strings.ToUpper(tokens[idx]) == "IF" &&
		strings.ToUpper(tokens[idx+1]) == "EXISTS" {
		stmt.IfExists = true
		idx += 2
	}

	// Get index name
	if idx >= len(tokens) {
		return nil, &ParseError{SQL: sql, Message: "missing index name"}
	}
	stmt.Name = unquoteIdentifier(tokens[idx])
	idx++

	// Check for CASCADE
	if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "CASCADE" {
		stmt.Cascade = true
	}

	return stmt, nil
}

// SQLTypeToSchemaType maps SQL type strings to schema.TypeKind.
func SQLTypeToSchemaType(sqlType string) schema.TypeKind {
	// Extract base type (before parentheses or brackets)
	baseType := sqlType
	if idx := strings.Index(sqlType, "("); idx != -1 {
		baseType = sqlType[:idx]
	}
	if idx := strings.Index(sqlType, "["); idx != -1 {
		baseType = sqlType[:idx]
	}

	baseType = strings.ToUpper(strings.TrimSpace(baseType))

	switch baseType {
	case "INTEGER", "INT", "INT4":
		return schema.TypeInt32
	case "BIGINT", "INT8":
		return schema.TypeInt64
	case "SERIAL", "SERIAL4":
		return schema.TypeInt32
	case "BIGSERIAL", "SERIAL8":
		return schema.TypeInt64
	case "SMALLINT", "INT2":
		return schema.TypeInt32
	case "TEXT":
		return schema.TypeText
	case "VARCHAR", "CHARACTER VARYING":
		return schema.TypeVarChar
	case "CHAR", "CHARACTER":
		return schema.TypeChar
	case "UUID":
		return schema.TypeUUID
	case "BOOLEAN", "BOOL":
		return schema.TypeBool
	case "REAL", "FLOAT4":
		return schema.TypeFloat32
	case "DOUBLE PRECISION", "FLOAT8":
		return schema.TypeFloat64
	case "NUMERIC", "DECIMAL":
		return schema.TypeNumeric
	case "TIMESTAMP", "TIMESTAMP WITHOUT TIME ZONE":
		return schema.TypeTimestamp
	case "TIMESTAMPTZ", "TIMESTAMP WITH TIME ZONE":
		return schema.TypeTimestampTZ
	case "DATE":
		return schema.TypeDate
	case "JSON":
		return schema.TypeJSON
	case "JSONB":
		return schema.TypeJSONB
	case "BYTEA":
		return schema.TypeBytes
	default:
		return schema.TypeUnknown
	}
}

// ParseError represents a SQL parsing error.
type ParseError struct {
	SQL     string
	Message string
	Pos     int
}

func (e *ParseError) Error() string {
	if e.Pos > 0 {
		return fmt.Sprintf("parse error at position %d: %s\nSQL: %s", e.Pos, e.Message, e.SQL)
	}
	return fmt.Sprintf("parse error: %s\nSQL: %s", e.Message, e.SQL)
}

// tokenize splits SQL into tokens, preserving quoted strings.
func tokenize(sql string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)
	inParen := 0

	for i, ch := range sql {
		switch {
		case (ch == '\'' || ch == '"') && !inQuote:
			inQuote = true
			quoteChar = ch
			current.WriteRune(ch)
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
			current.WriteRune(ch)
		case ch == '(' && !inQuote:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			inParen++
			current.WriteRune(ch)
		case ch == ')' && !inQuote:
			inParen--
			current.WriteRune(ch)
			if inParen == 0 && current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		case (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ',') && !inQuote && inParen == 0:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			if ch == ',' {
				tokens = append(tokens, ",")
			}
		default:
			current.WriteRune(ch)
		}

		// Handle end of string
		if i == len(sql)-1 && current.Len() > 0 {
			tokens = append(tokens, current.String())
		}
	}

	return tokens
}

// unquoteIdentifier removes surrounding quotes from an identifier.
func unquoteIdentifier(ident string) string {
	ident = strings.TrimSpace(ident)
	if len(ident) >= 2 && ident[0] == '"' && ident[len(ident)-1] == '"' {
		return ident[1 : len(ident)-1]
	}
	return ident
}

// extractParentheses extracts content between matching parentheses.
func extractParentheses(s string) string {
	start := strings.Index(s, "(")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[start+1 : i]
			}
		}
	}

	return ""
}

// splitByComma splits a string by commas, respecting nested parentheses.
func splitByComma(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range s {
		switch {
		case (ch == '\'' || ch == '"') && !inQuote:
			inQuote = true
			quoteChar = ch
			current.WriteRune(ch)
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
			current.WriteRune(ch)
		case ch == '(' && !inQuote:
			depth++
			current.WriteRune(ch)
		case ch == ')' && !inQuote:
			depth--
			current.WriteRune(ch)
		case ch == ',' && depth == 0 && !inQuote:
			if current.Len() > 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

// parseReferentialAction parses ON DELETE/UPDATE actions.
func parseReferentialAction(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}

	action := strings.ToUpper(tokens[0])
	switch action {
	case "CASCADE":
		return "CASCADE"
	case "RESTRICT":
		return "RESTRICT"
	case "NO":
		if len(tokens) > 1 && strings.ToUpper(tokens[1]) == "ACTION" {
			return "NO ACTION"
		}
		return "NO ACTION"
	case "SET":
		if len(tokens) > 1 {
			if strings.ToUpper(tokens[1]) == "NULL" {
				return "SET NULL"
			} else if strings.ToUpper(tokens[1]) == "DEFAULT" {
				return "SET DEFAULT"
			}
		}
		return action
	default:
		return action
	}
}

// countActionTokens counts how many tokens are part of the referential action.
func countActionTokens(tokens []string) int {
	if len(tokens) == 0 {
		return 0
	}

	action := strings.ToUpper(tokens[0])
	switch action {
	case "CASCADE", "RESTRICT":
		return 1
	case "NO", "SET":
		if len(tokens) > 1 {
			return 2
		}
		return 1
	default:
		return 1
	}
}
