package prisma

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestConverter_SimpleRelation_GeneratesForeignKey(t *testing.T) {
	input := `
model Post {
  id     Int  @id
  userId Int  @map("user_id")
  user   User @relation(fields: [userId], references: [id])
}

model User {
  id    Int    @id
  posts Post[]
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Find Post table
	var postTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "Post" {
			postTable = &s.Tables[i]
			break
		}
	}

	if postTable == nil {
		t.Fatal("Post table not found")
	}

	// Verify FK was generated
	if len(postTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(postTable.ForeignKeys))
	}

	fk := postTable.ForeignKeys[0]

	// Verify FK details
	if len(fk.Columns) != 1 || fk.Columns[0] != "user_id" {
		t.Errorf("expected FK column [user_id], got %v", fk.Columns)
	}

	if fk.RefTable != "User" {
		t.Errorf("expected RefTable User, got %s", fk.RefTable)
	}

	if len(fk.RefColumns) != 1 || fk.RefColumns[0] != "id" {
		t.Errorf("expected RefColumns [id], got %v", fk.RefColumns)
	}

	// Default actions should be NoAction
	if fk.OnDelete != schema.ActionNoAction {
		t.Errorf("expected OnDelete NoAction, got %v", fk.OnDelete)
	}
}

func TestConverter_CompositeFK_GeneratesCorrectly(t *testing.T) {
	input := `
model Order {
  tenantId Int @map("tenant_id")
  userId   Int @map("user_id")
  user     User @relation(fields: [tenantId, userId], references: [tenantId, id])

  @@id([tenantId, userId])
}

model User {
  tenantId Int
  id       Int
  orders   Order[]

  @@id([tenantId, id])
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Find Order table
	var orderTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "Order" {
			orderTable = &s.Tables[i]
			break
		}
	}

	if orderTable == nil {
		t.Fatal("Order table not found")
	}

	// Verify composite FK
	if len(orderTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(orderTable.ForeignKeys))
	}

	fk := orderTable.ForeignKeys[0]

	if len(fk.Columns) != 2 {
		t.Fatalf("expected 2 FK columns, got %d", len(fk.Columns))
	}

	// Verify column order matches
	if fk.Columns[0] != "tenant_id" || fk.Columns[1] != "user_id" {
		t.Errorf("expected FK columns [tenant_id, user_id], got %v", fk.Columns)
	}

	if fk.RefColumns[0] != "tenantId" || fk.RefColumns[1] != "id" {
		t.Errorf("expected RefColumns [tenantId, id], got %v", fk.RefColumns)
	}
}

func TestConverter_ReferentialActions_ParsedCorrectly(t *testing.T) {
	input := `
model Comment {
  id     Int  @id
  postId Int  @map("post_id")
  post   Post @relation(fields: [postId], references: [id], onDelete: Cascade, onUpdate: Restrict)
}

model Post {
  id       Int       @id
  comments Comment[]
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Find Comment table
	var commentTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "Comment" {
			commentTable = &s.Tables[i]
			break
		}
	}

	if commentTable == nil {
		t.Fatal("Comment table not found")
	}

	if len(commentTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(commentTable.ForeignKeys))
	}

	fk := commentTable.ForeignKeys[0]

	if fk.OnDelete != schema.ActionCascade {
		t.Errorf("expected OnDelete Cascade, got %v", fk.OnDelete)
	}

	if fk.OnUpdate != schema.ActionRestrict {
		t.Errorf("expected OnUpdate Restrict, got %v", fk.OnUpdate)
	}
}

func TestConverter_NullableRelation_OptionalField(t *testing.T) {
	input := `
model Post {
  id       Int   @id
  authorId Int?  @map("author_id")
  author   User? @relation(fields: [authorId], references: [id])
}

model User {
  id    Int    @id
  posts Post[]
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Find Post table
	var postTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "Post" {
			postTable = &s.Tables[i]
			break
		}
	}

	if postTable == nil {
		t.Fatal("Post table not found")
	}

	// FK should still be generated
	if len(postTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(postTable.ForeignKeys))
	}

	// Verify column is nullable
	var authorIdCol *schema.Column
	for i := range postTable.Columns {
		if postTable.Columns[i].Name == "author_id" {
			authorIdCol = &postTable.Columns[i]
			break
		}
	}

	if authorIdCol == nil {
		t.Fatal("author_id column not found")
	}

	if !authorIdCol.Nullable {
		t.Error("expected author_id to be nullable")
	}
}

func TestConverter_BackReference_NoFK(t *testing.T) {
	input := `
model User {
  id    Int    @id
  posts Post[]
}

model Post {
  id Int @id
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// User table should have no FKs (posts is passive side)
	for i := range s.Tables {
		if s.Tables[i].Name == "User" {
			if len(s.Tables[i].ForeignKeys) > 0 {
				t.Errorf("User table should not have foreign keys, got %d", len(s.Tables[i].ForeignKeys))
			}
		}
	}
}

func TestConverter_ListFieldWithFields_ReturnsError(t *testing.T) {
	input := `
model User {
  id    Int    @id
  posts Post[] @relation(fields: [id], references: [userId])
}

model Post {
  id     Int @id
  userId Int
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should return error
	if err == nil {
		t.Fatal("expected error for list field with fields argument, got nil")
	}

	// Error should mention list fields cannot have fields
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestConverter_MismatchedFieldsReferences_ReturnsError(t *testing.T) {
	input := `
model Post {
  id     Int  @id
  userId Int
  user   User @relation(fields: [userId], references: [id, email])
}

model User {
  id    Int    @id
  email String
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should return error about count mismatch
	if err == nil {
		t.Fatal("expected error for fields/references count mismatch, got nil")
	}
}

func TestConverter_ReferencedModelWithMap_UsesTableName(t *testing.T) {
	input := `
model Post {
  id     Int  @id
  userId Int  @map("user_id")
  user   User @relation(fields: [userId], references: [id])

  @@map("posts")
}

model User {
  id    Int    @id
  posts Post[]

  @@map("users")
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Find posts table
	var postsTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "posts" {
			postsTable = &s.Tables[i]
			break
		}
	}

	if postsTable == nil {
		t.Fatal("posts table not found")
	}

	if len(postsTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 FK, got %d", len(postsTable.ForeignKeys))
	}

	fk := postsTable.ForeignKeys[0]

	// CRITICAL: RefTable should be "users" (mapped table name), NOT "User" (model name)
	if fk.RefTable != "users" {
		t.Errorf("expected RefTable 'users' (mapped table name), got %q", fk.RefTable)
	}

	// Verify column mapping also works
	if len(fk.Columns) != 1 || fk.Columns[0] != "user_id" {
		t.Errorf("expected FK column [user_id], got %v", fk.Columns)
	}
}

func TestConverter_InvalidReferenceColumn_ReturnsError(t *testing.T) {
	input := `
model Post {
  id     Int  @id
  userId Int
  user   User @relation(fields: [userId], references: [nonExistentColumn])
}

model User {
  id    Int    @id
  posts Post[]
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should return error about non-existent reference column
	if err == nil {
		t.Fatal("expected error for non-existent reference column, got nil")
	}

	if !strings.Contains(err.Error(), "nonExistentColumn") {
		t.Errorf("error should mention the non-existent column name, got: %v", err)
	}
}
