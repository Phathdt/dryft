package prisma

import (
	"testing"
)

func TestLexer_BasicTokens(t *testing.T) {
	input := `{ } ( ) [ ] , : = ? @ @@`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenLBrace, "{"},
		{TokenRBrace, "}"},
		{TokenLParen, "("},
		{TokenRParen, ")"},
		{TokenLBracket, "["},
		{TokenRBracket, "]"},
		{TokenComma, ","},
		{TokenColon, ":"},
		{TokenEqual, "="},
		{TokenQuestion, "?"},
		{TokenAt, "@"},
		{TokenAtAt, "@@"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Keywords(t *testing.T) {
	input := `model enum type datasource generator view`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenModel, "model"},
		{TokenEnum, "enum"},
		{TokenTypeKeyword, "type"},
		{TokenDatasource, "datasource"},
		{TokenGenerator, "generator"},
		{TokenView, "view"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_BuiltinTypes(t *testing.T) {
	input := `String Int BigInt Float Decimal Boolean DateTime Json Bytes`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenString_, "String"},
		{TokenInt_, "Int"},
		{TokenBigInt_, "BigInt"},
		{TokenFloat_, "Float"},
		{TokenDecimal_, "Decimal"},
		{TokenBoolean_, "Boolean"},
		{TokenDateTime_, "DateTime"},
		{TokenJson_, "Json"},
		{TokenBytes_, "Bytes"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Identifiers(t *testing.T) {
	input := `User user_id created_at Role123 _private`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIdent, "User"},
		{TokenIdent, "user_id"},
		{TokenIdent, "created_at"},
		{TokenIdent, "Role123"},
		{TokenIdent, "_private"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Numbers(t *testing.T) {
	input := `42 -10 3.14 -2.5 0`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenNumber, "42"},
		{TokenNumber, "-10"},
		{TokenNumber, "3.14"},
		{TokenNumber, "-2.5"},
		{TokenNumber, "0"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Strings(t *testing.T) {
	input := `"hello" "world with spaces" "escaped \"quote\""`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenString, "hello"},
		{TokenString, "world with spaces"},
		{TokenString, `escaped \"quote\"`},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Comments(t *testing.T) {
	input := `// This is a comment
model User {
  // Field comment
  id Int
}`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenComment, "// This is a comment"},
		{TokenModel, "model"},
		{TokenIdent, "User"},
		{TokenLBrace, "{"},
		{TokenComment, "// Field comment"},
		{TokenIdent, "id"},
		{TokenInt_, "Int"},
		{TokenRBrace, "}"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_ModelBlock(t *testing.T) {
	input := `model User {
  id    Int    @id @default(autoincrement())
  email String @unique
  name  String?
}`

	l := NewLexer(input)
	tokens := l.AllTokens()

	if len(tokens) == 0 {
		t.Fatal("expected tokens, got none")
	}

	if tokens[0].Type != TokenModel {
		t.Errorf("expected first token to be TokenModel, got %q", tokens[0].Type)
	}

	if tokens[len(tokens)-1].Type != TokenEOF {
		t.Errorf("expected last token to be TokenEOF, got %q", tokens[len(tokens)-1].Type)
	}
}

func TestLexer_EnumBlock(t *testing.T) {
	input := `enum Role {
  USER
  ADMIN
  MODERATOR
}`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenEnum, "enum"},
		{TokenIdent, "Role"},
		{TokenLBrace, "{"},
		{TokenIdent, "USER"},
		{TokenIdent, "ADMIN"},
		{TokenIdent, "MODERATOR"},
		{TokenRBrace, "}"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_LineAndColumn(t *testing.T) {
	input := `model User {
  id Int
}`

	l := NewLexer(input)

	// model - line 1
	tok := l.NextToken()
	if tok.Line != 1 {
		t.Errorf("expected line 1, got %d", tok.Line)
	}

	// User - line 1
	tok = l.NextToken()
	if tok.Line != 1 {
		t.Errorf("expected line 1, got %d", tok.Line)
	}

	// { - line 1
	tok = l.NextToken()
	if tok.Line != 1 {
		t.Errorf("expected line 1, got %d", tok.Line)
	}

	// id - line 2
	tok = l.NextToken()
	if tok.Line != 2 {
		t.Errorf("expected line 2, got %d", tok.Line)
	}
}

func TestLexer_BooleanLiterals(t *testing.T) {
	input := `true false`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenTrue, "true"},
		{TokenFalse, "false"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_ComplexSchema(t *testing.T) {
	input := `generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id        String   @id @default(uuid()) @db.Uuid
  email     String   @unique
  name      String?
  createdAt DateTime @default(now())
  posts     Post[]

  @@map("users")
}

enum Status {
  ACTIVE
  INACTIVE
}`

	l := NewLexer(input)
	tokens := l.AllTokens()

	// Basic sanity checks
	if len(tokens) < 10 {
		t.Fatalf("expected many tokens, got %d", len(tokens))
	}

	// Check that we have the major keywords
	hasGenerator := false
	hasDatasource := false
	hasModel := false
	hasEnum := false

	for _, tok := range tokens {
		switch tok.Type {
		case TokenGenerator:
			hasGenerator = true
		case TokenDatasource:
			hasDatasource = true
		case TokenModel:
			hasModel = true
		case TokenEnum:
			hasEnum = true
		}
	}

	if !hasGenerator {
		t.Error("expected to find generator keyword")
	}
	if !hasDatasource {
		t.Error("expected to find datasource keyword")
	}
	if !hasModel {
		t.Error("expected to find model keyword")
	}
	if !hasEnum {
		t.Error("expected to find enum keyword")
	}
}
