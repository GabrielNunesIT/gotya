// Package gotya provides the public API for the library.
package gotya

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotya/gotya/ast"
	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
)

// ASTModule is an alias for the parsed abstract syntax tree of a YANG module.
// We export this so consumers of the parser API know what type they are working with.
type ASTModule = ast.Module

// Diagnostic holds structured information about a single parse error.
type Diagnostic struct {
	File    string
	Line    int
	Column  int
	Message string
}

// ParseError is returned by Parse and ParseFile when one or more parse errors occur.
// Use errors.As to access the full Errors slice.
type ParseError struct {
	Errors []Diagnostic
}

// Error implements the error interface. It returns a human-readable summary.
func (e *ParseError) Error() string {
	if len(e.Errors) == 0 {
		return "parse error"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("parse error: %s", e.Errors[0].Message)
	}
	return fmt.Sprintf("%d parse errors: %s", len(e.Errors), e.Errors[0].Message)
}

// parseContent is the shared implementation used by Parse and ParseFile.
// filename is set to the file path when called from ParseFile, or "" when called from Parse.
func parseContent(content, filename string) (*ASTModule, error) {
	l := lexer.New(content)
	p := parser.New(l)
	astMod := p.ParseModule()
	if diags := p.Diagnostics(); len(diags) > 0 {
		pe := &ParseError{}
		for _, d := range diags {
			pe.Errors = append(pe.Errors, Diagnostic{
				File:    filename,
				Line:    d.Line,
				Column:  d.Column,
				Message: d.Message,
			})
		}
		return nil, pe
	}
	return astMod, nil
}

// Parse parses a YANG module from src and returns the AST representation.
// If the source contains syntax errors, Parse returns a *ParseError containing
// all diagnostics. Use errors.As to access file, line, column, and message fields.
func Parse(src string) (*ASTModule, error) {
	return parseContent(src, "")
}

// ParseFile reads the YANG file at path and returns the AST representation.
// If the file contains syntax errors, ParseFile returns a *ParseError where
// each Diagnostic.File is set to the cleaned path.
func ParseFile(path string) (*ASTModule, error) {
	cleanPath := filepath.Clean(path)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", cleanPath, err)
	}
	return parseContent(string(content), cleanPath)
}

// Compile takes one or more AST modules and compiles them into a fully resolved
// semantic schema consisting of interlinked nodes.
//
// Why: After parsing a single module, all 'include', 'import', and 'uses' statements are
// simply textual references. A compiler pass is required to validate types, resolve groups,
// build the true node hierarchy, and apply augments across multiple interconnected YANG files.
func Compile(astModules []*ASTModule, opts *compiler.Options) ([]*schema.Module, error) {
	if opts == nil {
		opts = &compiler.Options{}
	}
	comp := compiler.New(opts)

	var compiled []*schema.Module
	for _, astMod := range astModules {
		mod, err := comp.Compile(astMod)
		if err != nil {
			return nil, fmt.Errorf("compile module %s: %w", astMod.Argument(), err)
		}
		if mod != nil {
			compiled = append(compiled, mod)
		}
	}
	return compiled, nil
}
