// Package ast provides Abstract Syntax Tree node definitions for YANG modules.
//
//nolint:revive // package name matches purpose
package ast

// Node represents a node in the AST.
type Node interface {
	// TokenLiteral returns the literal value of the token associated with the node.
	TokenLiteral() string
	// String returns a string representation of the node for debugging/testing.
	String() string
}

// Statement represents a YANG statement (e.g., module, leaf, list).
// In YANG, almost everything is a statement consisting of a keyword, an optional argument, and a block or semicolon.
type Statement interface {
	Node
	// Keyword returns the statement keyword (e.g. "module", "description").
	Keyword() string
	// Argument returns the argument of the statement, if any.
	Argument() string
	// SubStatements returns the substatements (children) of this statement.
	SubStatements() []Statement
}

// BaseNode provides a common implementation for AST nodes.
type BaseNode struct {
	Key      string // The keyword (e.g. "module", "leaf")
	Arg      string // The optional argument
	TokenL   string // The literal token value (mostly useful for debugging)
	SubStmts []Statement
}

// TokenLiteral returns the literal text.
func (b *BaseNode) TokenLiteral() string { return b.TokenL }

// Keyword returns the statement keyword.
func (b *BaseNode) Keyword() string { return b.Key }

// Argument returns the statement argument.
func (b *BaseNode) Argument() string { return b.Arg }

// SubStatements returns the substatements.
func (b *BaseNode) SubStatements() []Statement { return b.SubStmts }

// String returns a generic string representation.
func (b *BaseNode) String() string {
	if b.Arg != "" {
		return b.Key + " " + b.Arg
	}
	return b.Key
}

// Module represents a complete YANG module.
type Module struct {
	*BaseNode
}

// String returns the string representation of the module.
func (m *Module) String() string {
	return "module " + m.Arg
}
