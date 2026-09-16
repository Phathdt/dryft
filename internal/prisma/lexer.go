package prisma

import (
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes Prisma schema input.
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number (1-indexed)
	column       int  // current column number (1-indexed)
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position.
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

// peekChar returns the next character without advancing position.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case 0:
		tok.Type = TokenEOF
		tok.Literal = ""
	case '{':
		tok = l.newToken(TokenLBrace, l.ch)
	case '}':
		tok = l.newToken(TokenRBrace, l.ch)
	case '(':
		tok = l.newToken(TokenLParen, l.ch)
	case ')':
		tok = l.newToken(TokenRParen, l.ch)
	case '[':
		tok = l.newToken(TokenLBracket, l.ch)
	case ']':
		tok = l.newToken(TokenRBracket, l.ch)
	case ',':
		tok = l.newToken(TokenComma, l.ch)
	case ':':
		tok = l.newToken(TokenColon, l.ch)
	case '=':
		tok = l.newToken(TokenEqual, l.ch)
	case '?':
		tok = l.newToken(TokenQuestion, l.ch)
	case '@':
		if l.peekChar() == '@' {
			ch := l.ch
			l.readChar()
			tok.Type = TokenAtAt
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok = l.newToken(TokenAt, l.ch)
		}
	case '"':
		tok.Type = TokenString
		tok.Literal = l.readString()
		l.readChar() // advance past closing quote
		return tok
	case '/':
		if l.peekChar() == '/' {
			tok.Type = TokenComment
			tok.Literal = l.readLineComment()
			return tok
		}
		tok = l.newToken(TokenIllegal, l.ch)
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			tok.Type = TokenNumber
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = l.newToken(TokenIllegal, l.ch)
		}
	}

	l.readChar()
	return tok
}

// newToken creates a new token with the given type and character.
func (l *Lexer) newToken(tokenType TokenType, ch byte) Token {
	return Token{
		Type:    tokenType,
		Literal: string(ch),
		Line:    l.line,
		Column:  l.column,
	}
}

// readIdentifier reads an identifier from the input.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '.' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or decimal) from the input.
func (l *Lexer) readNumber() string {
	position := l.position

	// Handle negative sign
	if l.ch == '-' {
		l.readChar()
	}

	// Read digits
	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

// readString reads a double-quoted string from the input.
func (l *Lexer) readString() string {
	position := l.position + 1 // skip opening quote
	for {
		l.readChar()
		if l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			l.readChar() // skip escaped character
			continue
		}
		if l.ch == '"' {
			break
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
	}
	str := l.input[position:l.position]
	return str
}

// readLineComment reads a line comment (// ...) from the input.
func (l *Lexer) readLineComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

// skipWhitespace skips whitespace characters.
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

// isLetter returns true if the character is a letter.
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit returns true if the character is a digit.
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// AllTokens returns all tokens from the input for testing purposes.
func (l *Lexer) AllTokens() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens
}

// Position returns the current line and column.
func (l *Lexer) Position() (line, column int) {
	return l.line, l.column
}

// isIdentRune returns true if the rune can be part of an identifier.
func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// isWhitespace returns true if the rune is whitespace.
func isWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}

// runeAt returns the rune at the given position in the input.
func (l *Lexer) runeAt(pos int) (rune, int) {
	if pos >= len(l.input) {
		return 0, 0
	}
	r, size := utf8.DecodeRuneInString(l.input[pos:])
	return r, size
}
