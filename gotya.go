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

// Parse takes a raw YANG string and returns an interpreted AST module.
//
// Why: Providing a simple string-based parser allows for easy integration when
// YANG models are dynamically generated or loaded from a database, bypassing the filesystem.
func Parse(content string) (*ASTModule, error) {
	l := lexer.New(content)
	p := parser.New(l)
	astMod := p.ParseModule()
	// NOTE: We could gather parser errors here and return them.
	// For now, if the module is nil, it failed completely.
	if astMod == nil {
		return nil, os.ErrInvalid // Simplistic error for facade demonstration
	}
	return astMod, nil
}

// ParseFile takes a filesystem path to a .yang file, reads it,
// and returns an interpreted AST module.
//
// Why: Standard use cases revolve around working with local files. This provides
// a convenient wrapper so users do not have to handle the file I/O boilerplate themselves.
func ParseFile(path string) (*ast.Module, error) {
	cleanPath := filepath.Clean(path)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", cleanPath, err)
	}
	return Parse(string(content))
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
