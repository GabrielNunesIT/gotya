// Package gotya provides the public API for the library.
package gotya

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotya/gotya/ast"
	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
)

// ASTModule is the parsed abstract syntax tree of a YANG module.
// Use Parse or ParseFile to obtain an ASTModule.
type ASTModule struct {
	mod *ast.Module
}

// Name returns the YANG module name (the argument of the module statement).
func (m *ASTModule) Name() string {
	return m.mod.Argument()
}

// Keyword returns "module" or "submodule".
func (m *ASTModule) Keyword() string {
	return m.mod.Keyword()
}

// Argument returns the module name argument — identical to Name(), provided for
// compatibility with YANG AST conventions.
func (m *ASTModule) Argument() string {
	return m.mod.Argument()
}

// Module is a compiled YANG module. Use Compile to obtain a Module.
type Module struct {
	schema *schema.Module
}

// Schema returns the compiled schema.Module for use with generator packages.
func (m *Module) Schema() *schema.Module {
	return m.schema
}

// CompileOptions configures the behavior of Compile.
type CompileOptions struct {
	// MaxErrors is the maximum number of errors to accumulate before stopping.
	// Zero means use the default of 100.
	MaxErrors int
}

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
	return &ASTModule{mod: astMod}, nil
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

// Compile compiles one or more parsed YANG modules into a resolved schema.
// Pass nil opts to use defaults. The returned Module values provide access to
// the compiled schema via Schema().
func Compile(astModules []*ASTModule, opts *CompileOptions) ([]*Module, error) {
	compOpts := &compiler.Options{}
	if opts != nil {
		compOpts.MaxErrors = opts.MaxErrors
	}
	comp := compiler.New(compOpts)

	var compiled []*Module
	var errs []error
	for _, m := range astModules {
		schemaMod, err := comp.Compile(m.mod)
		if err != nil {
			errs = append(errs, fmt.Errorf("compile module %s: %w", m.mod.Argument(), err))
			continue
		}
		if schemaMod != nil {
			compiled = append(compiled, &Module{schema: schemaMod})
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return compiled, nil
}
