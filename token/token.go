// Package token provides the definitions for lexical tokens in the gotya YANG parser.
//
//nolint:revive // package name matches purpose
package token

// Type represents the type of a token.
type Type string

const (
	// ILLEGAL indicates an invalid character or sequence.
	ILLEGAL Type = "ILLEGAL"
	// EOF indicates the end of the input file or stream.
	EOF Type = "EOF"

	// IDENTIFIER representing identifiers and literals.
	IDENTIFIER Type = "IDENTIFIER" // e.g., module, my-node, prefix:name
	// STRING representing string literals.
	STRING Type = "STRING" // e.g., "hello world", 'test'
	// NUMBER representing numeric literals.
	NUMBER Type = "NUMBER" // e.g., 42, -10.5
	// COMMENT representing single or multi-line comments.
	COMMENT Type = "COMMENT" // e.g., /* comment */, // comment

	// PLUS representing the string concatenation operator in YANG 1.1.
	PLUS Type = "+"

	// LBRACE representing the left curly brace.
	LBRACE Type = "{"
	// RBRACE representing the right curly brace.
	RBRACE Type = "}"
	// SEMI representing the semicolon delimiter.
	SEMI Type = ";"
)

// Position holds the line and column of a token in the source file.
type Position struct {
	Line   int
	Column int
}

// Token represents a lexical token in the parsed YANG source.
type Token struct {
	Type    Type
	Literal string
	Pos     Position
}

// String maps token types to their string representation.
func (t Type) String() string {
	return string(t)
}
