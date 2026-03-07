// Package parser implements a recursive descent parser for YANG 1.0 and 1.1 modules.
//
//nolint:revive // package name matches purpose
package parser

import (
	"fmt"

	"github.com/gotya/gotya/ast"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/token"
)

// Parser holds the state for the recursive descent parser.
type Parser struct {
	lex       *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string
}

// New creates a new Parser instance from an initialized lexer.
func New(lex *lexer.Lexer) *Parser {
	p := &Parser{
		lex:    lex,
		errors: []string{},
	}
	// Read two tokens, so curToken and peekToken are both set.
	p.nextToken()
	p.nextToken()
	return p
}

// Errors returns any parsing errors encountered.
func (p *Parser) Errors() []string {
	return p.errors
}

// nextToken advances the tokens.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lex.NextToken()
}

// ParseModule parses a complete YANG module or submodule.
func (p *Parser) ParseModule() *ast.Module {
	module := &ast.Module{
		BaseNode: &ast.BaseNode{},
	}

	// We expect the first statement to be "module" or "submodule"
	if p.curToken.Type != token.IDENTIFIER || (p.curToken.Literal != "module" && p.curToken.Literal != "submodule") {
		p.addError("expected 'module' or 'submodule', got " + p.curToken.Literal)
		return nil
	}

	stmt := p.parseStatement()
	if stmt != nil {
		if bn, ok := stmt.(*ast.BaseNode); ok {
			module.BaseNode = bn
		}
	}

	return module
}

// parseStatement parses a generic YANG statement.
func (p *Parser) parseStatement() ast.Statement {
	if p.curToken.Type != token.IDENTIFIER {
		p.addError(fmt.Sprintf("expected keyword at line %d, got %s", p.curToken.Pos.Line, p.curToken.Type))
		return nil
	}

	stmt := &ast.BaseNode{
		Key:    p.curToken.Literal,
		TokenL: p.curToken.Literal,
	}
	p.nextToken()

	// Parse optional argument string
	// Argument can be IDENTIFIER, STRING, or multiple STRINGs concatenated by PLUS
	switch p.curToken.Type {
	case token.IDENTIFIER:
		stmt.Arg = p.curToken.Literal
		p.nextToken()
	case token.STRING:
		stmt.Arg = p.parseStringArgument()
	default:
		// Not an expected argument type
	}

	// Now we either expect a ';' or '{'
	if p.curToken.Type == token.SEMI {
		p.nextToken() // consume ';'
		return stmt
	}

	if p.curToken.Type == token.LBRACE {
		p.nextToken() // consume '{'
		stmt.SubStmts = p.parseBlock()
		if p.curToken.Type != token.RBRACE {
			p.addError(fmt.Sprintf("expected '}' at line %d, got %s", p.curToken.Pos.Line, p.curToken.Type))
		} else {
			p.nextToken() // consume '}'
		}
		return stmt
	}

	p.addError(fmt.Sprintf("expected ';' or '{' after statement %s at line %d, got %s", stmt.Key, p.curToken.Pos.Line, p.curToken.Literal))
	return nil
}

// parseStringArgument parses strings, including YANG 1.1 + concatenation.
func (p *Parser) parseStringArgument() string {
	res := p.curToken.Literal
	p.nextToken()

	for p.curToken.Type == token.PLUS {
		p.nextToken() // consume '+'
		if p.curToken.Type != token.STRING && p.curToken.Type != token.IDENTIFIER {
			p.addError(fmt.Sprintf("expected string after '+' at line %d", p.curToken.Pos.Line))
			return res
		}
		res += p.curToken.Literal
		p.nextToken()
	}

	return res
}

// parseBlock parses a sequence of statements inside braces.
func (p *Parser) parseBlock() []ast.Statement {
	var stmts []ast.Statement

	for p.curToken.Type != token.RBRACE && p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		} else {
			// On error, we try to recover by skipping tokens until semicolon or brace
			p.recoverStatement()
		}
	}

	return stmts
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

func (p *Parser) recoverStatement() {
	for p.curToken.Type != token.SEMI && p.curToken.Type != token.RBRACE && p.curToken.Type != token.EOF {
		p.nextToken()
	}
	if p.curToken.Type == token.SEMI {
		p.nextToken()
	}
}
