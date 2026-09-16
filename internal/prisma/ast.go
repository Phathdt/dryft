package prisma

// AST node types for Prisma schema language.

// Node is the base interface for all AST nodes.
type Node interface {
	node()
	Pos() Position
}

// Position represents a source location in the Prisma schema.
type Position struct {
	Line   int
	Column int
}

// Schema represents the root AST node for a complete Prisma schema.
type Schema struct {
	Declarations []Declaration
}

func (s *Schema) node()         {}
func (s *Schema) Pos() Position { return Position{Line: 1, Column: 1} }

// Declaration represents a top-level declaration (model, enum, datasource, generator).
type Declaration interface {
	Node
	declaration()
}

// ModelDeclaration represents a Prisma model block.
type ModelDeclaration struct {
	Position   Position
	Name       string
	Fields     []Field
	Attributes []ModelAttribute
}

func (m *ModelDeclaration) node()         {}
func (m *ModelDeclaration) Pos() Position { return m.Position }
func (m *ModelDeclaration) declaration()  {}

// EnumDeclaration represents a Prisma enum block.
type EnumDeclaration struct {
	Position Position
	Name     string
	Values   []EnumValue
}

func (e *EnumDeclaration) node()         {}
func (e *EnumDeclaration) Pos() Position { return e.Position }
func (e *EnumDeclaration) declaration()  {}

// DatasourceDeclaration represents a datasource block (parsed but not used in MVP).
type DatasourceDeclaration struct {
	Position   Position
	Name       string
	Properties []Property
}

func (d *DatasourceDeclaration) node()         {}
func (d *DatasourceDeclaration) Pos() Position { return d.Position }
func (d *DatasourceDeclaration) declaration()  {}

// GeneratorDeclaration represents a generator block (parsed but not used in MVP).
type GeneratorDeclaration struct {
	Position   Position
	Name       string
	Properties []Property
}

func (g *GeneratorDeclaration) node()         {}
func (g *GeneratorDeclaration) Pos() Position { return g.Position }
func (g *GeneratorDeclaration) declaration()  {}

// Field represents a field in a model.
type Field struct {
	Position   Position
	Name       string
	Type       FieldType
	Attributes []FieldAttribute
}

func (f *Field) node()         {}
func (f *Field) Pos() Position { return f.Position }

// FieldType represents the type of a field.
type FieldType struct {
	Name     string // Base type name (e.g., "String", "Int", "User")
	Optional bool   // true if field ends with "?"
	List     bool   // true if field ends with "[]"
}

// EnumValue represents a value in an enum declaration.
type EnumValue struct {
	Position Position
	Name     string
}

func (e *EnumValue) node()         {}
func (e *EnumValue) Pos() Position { return e.Position }

// FieldAttribute represents a field-level attribute like @id, @unique, @default.
type FieldAttribute struct {
	Position Position
	Name     string
	Args     []Argument
}

func (a *FieldAttribute) node()         {}
func (a *FieldAttribute) Pos() Position { return a.Position }

// ModelAttribute represents a model-level attribute like @@map, @@index, @@unique, @@id.
type ModelAttribute struct {
	Position Position
	Name     string
	Args     []Argument
}

func (a *ModelAttribute) node()         {}
func (a *ModelAttribute) Pos() Position { return a.Position }

// Argument represents an argument to an attribute.
type Argument struct {
	Name  string      // empty for positional arguments
	Value interface{} // string, int, bool, []string, FunctionCall
}

// FunctionCall represents a function call in an attribute (e.g., autoincrement(), now()).
type FunctionCall struct {
	Name string
	Args []Argument
}

// Property represents a key-value property in datasource/generator blocks.
type Property struct {
	Key   string
	Value interface{} // string, FunctionCall
}
