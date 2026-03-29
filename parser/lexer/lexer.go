// Package lexer implements a lexical analyzer for YANG 1.0 and 1.1 sources.
package lexer

import (
	"strings"
	"unicode"

	"github.com/GabrielNunesIT/gotya/token"
)

// Lexer produces a stream of tokens from a YANG input string.
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number
}

// New creates a new Lexer instance.
func New(input string) *Lexer {
	lex := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	lex.readChar()
	return lex
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	// Capture pos after skipping whitespace
	pos := token.Position{Line: l.line, Column: l.column}

	switch l.ch {
	case '{':
		tok = newToken(token.LBRACE, l.ch, pos)
	case '}':
		tok = newToken(token.RBRACE, l.ch, pos)
	case ';':
		tok = newToken(token.SEMI, l.ch, pos)
	case '+':
		tok = newToken(token.PLUS, l.ch, pos)
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readDQuoteString()
		tok.Pos = pos
		return tok // readDQuoteString advances past the closing quote
	case '\'':
		tok.Type = token.STRING
		tok.Literal = l.readSQuoteString()
		tok.Pos = pos
		return tok // readSQuoteString advances past the closing quote
	case '/':
		if l.peekChar() == '/' {
			l.skipSingleLineComment()
			return l.NextToken()
		} else if l.peekChar() == '*' {
			l.skipMultiLineComment()
			return l.NextToken()
		}
		// Fallback if just a slash inside an unquoted string
		tok.Type = token.IDENTIFIER
		tok.Literal = l.readUnquotedString()
		tok.Pos = pos
		return tok
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
		tok.Pos = pos
	default:
		// Read an unquoted string (identifier or keyword)
		tok.Literal = l.readUnquotedString()
		tok.Type = token.IDENTIFIER
		tok.Pos = pos
		return tok
	}

	l.readChar()
	return tok
}

func newToken(tokenType token.Type, ch byte, pos token.Position) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Pos: pos}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipSingleLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	l.skipWhitespace()
}

func (l *Lexer) skipMultiLineComment() {
	l.readChar() // skip '/'
	l.readChar() // skip '*'
	for l.ch != 0 {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // skip '*'
			l.readChar() // skip '/'
			break
		}
		l.readChar()
	}
	l.skipWhitespace()
}

func (l *Lexer) readDQuoteString() string {
	l.readChar() // skip opening quote
	var builder strings.Builder
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				builder.WriteByte('\n')
			case 't':
				builder.WriteByte('\t')
			case '"':
				builder.WriteByte('"')
			case '\\':
				builder.WriteByte('\\')
			default:
				// RFC 6020: A backslash followed by any other char matches that char
				builder.WriteByte(l.ch)
			}
		} else {
			builder.WriteByte(l.ch)
		}
		l.readChar()
	}
	l.readChar() // skip closing quote
	return builder.String()
}

func (l *Lexer) readSQuoteString() string {
	l.readChar() // skip opening quote
	position := l.position
	for l.ch != '\'' && l.ch != 0 {
		l.readChar()
	}
	res := l.input[position:l.position]
	l.readChar() // skip closing quote
	return res
}

func (l *Lexer) readUnquotedString() string {
	position := l.position
	for l.ch != 0 && !unicode.IsSpace(rune(l.ch)) && l.ch != ';' && l.ch != '{' && l.ch != '}' && l.ch != '"' && l.ch != '\'' && l.ch != '+' {
		// Stop if comment start
		if l.ch == '/' && (l.peekChar() == '/' || l.peekChar() == '*') {
			break
		}
		l.readChar()
	}
	return l.input[position:l.position]
}
