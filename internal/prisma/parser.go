package prisma

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser parses Prisma schema tokens into an AST.
type Parser struct {
	lexer   *Lexer
	current Token
	peek    Token
	errors  []string
}

// NewParser creates a new parser for the given input.
func NewParser(input string) *Parser {
	l := NewLexer(input)
	p := &Parser{
		lexer:  l,
		errors: []string{},
	}
	// Read two tokens to initialize current and peek
	p.nextToken()
	p.nextToken()
	return p
}

// nextToken advances to the next token, skipping comments.
func (p *Parser) nextToken() {
	p.current = p.peek
	for {
		p.peek = p.lexer.NextToken()
		if p.peek.Type != TokenComment {
			break
		}
	}
}

// ParseSchema parses the entire schema and returns the AST root.
func (p *Parser) ParseSchema() (*Schema, error) {
	schema := &Schema{
		Declarations: []Declaration{},
	}

	for p.current.Type != TokenEOF {
		decl := p.parseDeclaration()
		if decl != nil {
			schema.Declarations = append(schema.Declarations, decl)
		}
		// If we hit an error and can't continue, break
		if p.current.Type == TokenIllegal {
			break
		}
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors:\n%s", strings.Join(p.errors, "\n"))
	}

	return schema, nil
}

// parseDeclaration parses a top-level declaration.
func (p *Parser) parseDeclaration() Declaration {
	switch p.current.Type {
	case TokenModel:
		return p.parseModel()
	case TokenEnum:
		return p.parseEnum()
	case TokenDatasource:
		return p.parseDatasource()
	case TokenGenerator:
		return p.parseGenerator()
	default:
		p.addError(fmt.Sprintf("unexpected token %s at line %d:%d, expected declaration",
			p.current.Type, p.current.Line, p.current.Column))
		p.nextToken() // skip invalid token
		return nil
	}
}

// parseModel parses a model declaration.
func (p *Parser) parseModel() *ModelDeclaration {
	pos := Position{Line: p.current.Line, Column: p.current.Column}
	p.nextToken() // consume 'model'

	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected model name at line %d:%d, got %s",
			p.current.Line, p.current.Column, p.current.Type))
		return nil
	}

	model := &ModelDeclaration{
		Position:   pos,
		Name:       p.current.Literal,
		Fields:     []Field{},
		Attributes: []ModelAttribute{},
	}
	p.nextToken() // consume model name

	if !p.expect(TokenLBrace) {
		return nil
	}
	p.nextToken() // consume '{'

	// Parse fields and model attributes
	for p.current.Type != TokenRBrace && p.current.Type != TokenEOF {
		if p.current.Type == TokenAtAt {
			// Model-level attribute
			attr := p.parseModelAttribute()
			if attr != nil {
				model.Attributes = append(model.Attributes, *attr)
			}
		} else if p.current.Type == TokenIdent {
			// Field declaration
			field := p.parseField()
			if field != nil {
				model.Fields = append(model.Fields, *field)
			}
		} else {
			p.addError(fmt.Sprintf("unexpected token %s at line %d:%d in model body",
				p.current.Type, p.current.Line, p.current.Column))
			p.nextToken()
		}
	}

	if !p.expect(TokenRBrace) {
		return nil
	}
	p.nextToken() // consume '}'

	return model
}

// parseEnum parses an enum declaration.
func (p *Parser) parseEnum() *EnumDeclaration {
	pos := Position{Line: p.current.Line, Column: p.current.Column}
	p.nextToken() // consume 'enum'

	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected enum name at line %d:%d, got %s",
			p.current.Line, p.current.Column, p.current.Type))
		return nil
	}

	enum := &EnumDeclaration{
		Position: pos,
		Name:     p.current.Literal,
		Values:   []EnumValue{},
	}
	p.nextToken() // consume enum name

	if !p.expect(TokenLBrace) {
		return nil
	}
	p.nextToken() // consume '{'

	// Parse enum values
	for p.current.Type != TokenRBrace && p.current.Type != TokenEOF {
		if p.current.Type == TokenIdent {
			enum.Values = append(enum.Values, EnumValue{
				Position: Position{Line: p.current.Line, Column: p.current.Column},
				Name:     p.current.Literal,
			})
			p.nextToken()
		} else {
			p.addError(fmt.Sprintf("unexpected token %s at line %d:%d in enum body",
				p.current.Type, p.current.Line, p.current.Column))
			p.nextToken()
		}
	}

	if !p.expect(TokenRBrace) {
		return nil
	}
	p.nextToken() // consume '}'

	return enum
}

// parseDatasource parses a datasource block (MVP: parse but don't use).
func (p *Parser) parseDatasource() *DatasourceDeclaration {
	pos := Position{Line: p.current.Line, Column: p.current.Column}
	p.nextToken() // consume 'datasource'

	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected datasource name at line %d:%d",
			p.current.Line, p.current.Column))
		return nil
	}

	ds := &DatasourceDeclaration{
		Position:   pos,
		Name:       p.current.Literal,
		Properties: []Property{},
	}
	p.nextToken() // consume datasource name

	if !p.expect(TokenLBrace) {
		return nil
	}
	p.nextToken() // consume '{'

	// Parse properties
	for p.current.Type != TokenRBrace && p.current.Type != TokenEOF {
		prop := p.parseProperty()
		if prop != nil {
			ds.Properties = append(ds.Properties, *prop)
		}
	}

	if !p.expect(TokenRBrace) {
		return nil
	}
	p.nextToken() // consume '}'

	return ds
}

// parseGenerator parses a generator block (MVP: parse but don't use).
func (p *Parser) parseGenerator() *GeneratorDeclaration {
	pos := Position{Line: p.current.Line, Column: p.current.Column}
	p.nextToken() // consume 'generator'

	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected generator name at line %d:%d",
			p.current.Line, p.current.Column))
		return nil
	}

	gen := &GeneratorDeclaration{
		Position:   pos,
		Name:       p.current.Literal,
		Properties: []Property{},
	}
	p.nextToken() // consume generator name

	if !p.expect(TokenLBrace) {
		return nil
	}
	p.nextToken() // consume '{'

	// Parse properties
	for p.current.Type != TokenRBrace && p.current.Type != TokenEOF {
		prop := p.parseProperty()
		if prop != nil {
			gen.Properties = append(gen.Properties, *prop)
		}
	}

	if !p.expect(TokenRBrace) {
		return nil
	}
	p.nextToken() // consume '}'

	return gen
}

// parseProperty parses a key = value property in datasource/generator blocks.
func (p *Parser) parseProperty() *Property {
	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected property key at line %d:%d",
			p.current.Line, p.current.Column))
		p.nextToken()
		return nil
	}

	key := p.current.Literal
	p.nextToken() // consume key

	if !p.expect(TokenEqual) {
		return nil
	}
	p.nextToken() // consume '='

	var value interface{}
	if p.current.Type == TokenString {
		value = p.current.Literal
		p.nextToken()
	} else if p.current.Type == TokenIdent {
		// Check if it's a function call like env("DATABASE_URL")
		funcName := p.current.Literal
		p.nextToken()
		if p.current.Type == TokenLParen {
			value = p.parseFunctionCall(funcName)
		} else {
			value = funcName
		}
	} else {
		p.addError(fmt.Sprintf("expected property value at line %d:%d",
			p.current.Line, p.current.Column))
		p.nextToken()
		return nil
	}

	return &Property{
		Key:   key,
		Value: value,
	}
}

// parseField parses a field declaration.
func (p *Parser) parseField() *Field {
	pos := Position{Line: p.current.Line, Column: p.current.Column}

	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected field name at line %d:%d",
			p.current.Line, p.current.Column))
		return nil
	}

	field := &Field{
		Position:   pos,
		Name:       p.current.Literal,
		Attributes: []FieldAttribute{},
	}
	p.nextToken() // consume field name

	// Parse field type
	field.Type = p.parseFieldType()

	// Parse field attributes
	for p.current.Type == TokenAt {
		attr := p.parseFieldAttribute()
		if attr != nil {
			field.Attributes = append(field.Attributes, *attr)
		}
	}

	return field
}

// parseFieldType parses a field type (e.g., String, Int?, User[]).
func (p *Parser) parseFieldType() FieldType {
	ft := FieldType{}

	// Parse base type
	if p.isTypeToken(p.current.Type) {
		ft.Name = p.current.Literal
		p.nextToken()
	} else if p.current.Type == TokenIdent {
		ft.Name = p.current.Literal
		p.nextToken()
	} else {
		p.addError(fmt.Sprintf("expected type at line %d:%d, got %s",
			p.current.Line, p.current.Column, p.current.Type))
		return ft
	}

	// Check for array type (e.g., String[])
	if p.current.Type == TokenLBracket {
		p.nextToken() // consume '['
		if !p.expect(TokenRBracket) {
			return ft
		}
		p.nextToken() // consume ']'
		ft.List = true
	}

	// Check for optional (e.g., String?)
	if p.current.Type == TokenQuestion {
		ft.Optional = true
		p.nextToken()
	}

	return ft
}

// isTypeToken checks if a token type represents a Prisma built-in type.
func (p *Parser) isTypeToken(t TokenType) bool {
	return t == TokenString_ || t == TokenInt_ || t == TokenBigInt_ ||
		t == TokenFloat_ || t == TokenDecimal_ || t == TokenBoolean_ ||
		t == TokenDateTime_ || t == TokenJson_ || t == TokenBytes_
}

// parseFieldAttribute parses a field-level attribute like @id, @default(value).
func (p *Parser) parseFieldAttribute() *FieldAttribute {
	pos := Position{Line: p.current.Line, Column: p.current.Column}
	p.nextToken() // consume '@'

	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected attribute name at line %d:%d",
			p.current.Line, p.current.Column))
		return nil
	}

	attr := &FieldAttribute{
		Position: pos,
		Name:     p.current.Literal,
		Args:     []Argument{},
	}
	p.nextToken() // consume attribute name

	// Parse arguments if present
	if p.current.Type == TokenLParen {
		attr.Args = p.parseArguments()
	}

	return attr
}

// parseModelAttribute parses a model-level attribute like @@map("table_name").
func (p *Parser) parseModelAttribute() *ModelAttribute {
	pos := Position{Line: p.current.Line, Column: p.current.Column}
	p.nextToken() // consume '@@'

	if p.current.Type != TokenIdent {
		p.addError(fmt.Sprintf("expected attribute name at line %d:%d",
			p.current.Line, p.current.Column))
		return nil
	}

	attr := &ModelAttribute{
		Position: pos,
		Name:     p.current.Literal,
		Args:     []Argument{},
	}
	p.nextToken() // consume attribute name

	// Parse arguments if present
	if p.current.Type == TokenLParen {
		attr.Args = p.parseArguments()
	}

	return attr
}

// parseArguments parses argument list in parentheses.
func (p *Parser) parseArguments() []Argument {
	p.nextToken() // consume '('

	args := []Argument{}

	for p.current.Type != TokenRParen && p.current.Type != TokenEOF {
		arg := p.parseArgument()
		if arg != nil {
			args = append(args, *arg)
		}

		if p.current.Type == TokenComma {
			p.nextToken() // consume ','
		} else if p.current.Type != TokenRParen {
			p.addError(fmt.Sprintf("expected ',' or ')' at line %d:%d",
				p.current.Line, p.current.Column))
			break
		}
	}

	if !p.expect(TokenRParen) {
		return args
	}
	p.nextToken() // consume ')'

	return args
}

// parseArgument parses a single argument (named or positional).
func (p *Parser) parseArgument() *Argument {
	arg := &Argument{}

	// Check for named argument (key: value)
	if p.current.Type == TokenIdent && p.peek.Type == TokenColon {
		arg.Name = p.current.Literal
		p.nextToken() // consume name
		p.nextToken() // consume ':'
	}

	// Parse argument value
	arg.Value = p.parseArgumentValue()

	return arg
}

// parseArgumentValue parses an argument value (string, number, bool, array, function).
func (p *Parser) parseArgumentValue() interface{} {
	switch p.current.Type {
	case TokenString:
		value := p.current.Literal
		p.nextToken()
		return value
	case TokenNumber:
		// Try to parse as int first, then float
		if intVal, err := strconv.Atoi(p.current.Literal); err == nil {
			p.nextToken()
			return intVal
		}
		if floatVal, err := strconv.ParseFloat(p.current.Literal, 64); err == nil {
			p.nextToken()
			return floatVal
		}
		p.addError(fmt.Sprintf("invalid number %q at line %d:%d",
			p.current.Literal, p.current.Line, p.current.Column))
		p.nextToken()
		return nil
	case TokenTrue:
		p.nextToken()
		return true
	case TokenFalse:
		p.nextToken()
		return false
	case TokenLBracket:
		return p.parseArrayValue()
	case TokenIdent:
		// Function call or identifier
		name := p.current.Literal
		p.nextToken()
		if p.current.Type == TokenLParen {
			return p.parseFunctionCall(name)
		}
		return name
	default:
		p.addError(fmt.Sprintf("unexpected token %s at line %d:%d in argument",
			p.current.Type, p.current.Line, p.current.Column))
		p.nextToken()
		return nil
	}
}

// parseArrayValue parses an array like [field1, field2].
func (p *Parser) parseArrayValue() []string {
	p.nextToken() // consume '['

	values := []string{}

	for p.current.Type != TokenRBracket && p.current.Type != TokenEOF {
		if p.current.Type == TokenIdent {
			values = append(values, p.current.Literal)
			p.nextToken()
		} else {
			p.addError(fmt.Sprintf("expected identifier in array at line %d:%d",
				p.current.Line, p.current.Column))
			p.nextToken()
		}

		if p.current.Type == TokenComma {
			p.nextToken() // consume ','
		} else if p.current.Type != TokenRBracket {
			break
		}
	}

	if !p.expect(TokenRBracket) {
		return values
	}
	p.nextToken() // consume ']'

	return values
}

// parseFunctionCall parses a function call like autoincrement() or uuid().
func (p *Parser) parseFunctionCall(name string) *FunctionCall {
	fc := &FunctionCall{
		Name: name,
		Args: []Argument{},
	}

	if p.current.Type == TokenLParen {
		fc.Args = p.parseArguments()
	}

	return fc
}

// expect checks if the current token matches the expected type.
func (p *Parser) expect(t TokenType) bool {
	if p.current.Type != t {
		p.addError(fmt.Sprintf("expected %s at line %d:%d, got %s",
			t, p.current.Line, p.current.Column, p.current.Type))
		return false
	}
	return true
}

// addError adds a parse error to the error list.
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

// Errors returns all parse errors encountered.
func (p *Parser) Errors() []string {
	return p.errors
}
