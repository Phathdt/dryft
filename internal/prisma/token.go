package prisma

import "fmt"

// TokenType represents the type of token in the Prisma schema.
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenIllegal
	TokenComment

	// Identifiers and literals
	TokenIdent
	TokenString
	TokenNumber
	TokenTrue
	TokenFalse

	// Keywords
	TokenModel
	TokenEnum
	TokenTypeKeyword
	TokenDatasource
	TokenGenerator
	TokenView

	// Delimiters
	TokenLBrace   // {
	TokenRBrace   // }
	TokenLParen   // (
	TokenRParen   // )
	TokenLBracket // [
	TokenRBracket // ]
	TokenComma    // ,
	TokenColon    // :
	TokenEqual    // =
	TokenQuestion // ?
	TokenAt       // @
	TokenAtAt     // @@

	// Built-in types
	TokenString_   // String
	TokenInt_      // Int
	TokenBigInt_   // BigInt
	TokenFloat_    // Float
	TokenDecimal_  // Decimal
	TokenBoolean_  // Boolean
	TokenDateTime_ // DateTime
	TokenJson_     // Json
	TokenBytes_    // Bytes
)

var tokenNames = map[TokenType]string{
	TokenEOF:         "EOF",
	TokenIllegal:     "ILLEGAL",
	TokenComment:     "COMMENT",
	TokenIdent:       "IDENT",
	TokenString:      "STRING",
	TokenNumber:      "NUMBER",
	TokenTrue:        "true",
	TokenFalse:       "false",
	TokenModel:       "model",
	TokenEnum:        "enum",
	TokenTypeKeyword: "type",
	TokenDatasource:  "datasource",
	TokenGenerator:   "generator",
	TokenView:        "view",
	TokenLBrace:      "{",
	TokenRBrace:      "}",
	TokenLParen:      "(",
	TokenRParen:      ")",
	TokenLBracket:    "[",
	TokenRBracket:    "]",
	TokenComma:       ",",
	TokenColon:       ":",
	TokenEqual:       "=",
	TokenQuestion:    "?",
	TokenAt:          "@",
	TokenAtAt:        "@@",
	TokenString_:     "String",
	TokenInt_:        "Int",
	TokenBigInt_:     "BigInt",
	TokenFloat_:      "Float",
	TokenDecimal_:    "Decimal",
	TokenBoolean_:    "Boolean",
	TokenDateTime_:   "DateTime",
	TokenJson_:       "Json",
	TokenBytes_:      "Bytes",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// Token represents a lexical token in the Prisma schema.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q) at %d:%d", t.Type, t.Literal, t.Line, t.Column)
}

// keywords maps Prisma keywords to their token types.
var keywords = map[string]TokenType{
	"model":      TokenModel,
	"enum":       TokenEnum,
	"type":       TokenTypeKeyword,
	"datasource": TokenDatasource,
	"generator":  TokenGenerator,
	"view":       TokenView,
	"true":       TokenTrue,
	"false":      TokenFalse,
	"String":     TokenString_,
	"Int":        TokenInt_,
	"BigInt":     TokenBigInt_,
	"Float":      TokenFloat_,
	"Decimal":    TokenDecimal_,
	"Boolean":    TokenBoolean_,
	"DateTime":   TokenDateTime_,
	"Json":       TokenJson_,
	"Bytes":      TokenBytes_,
}

// LookupIdent checks if an identifier is a keyword and returns the appropriate token type.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TokenIdent
}
