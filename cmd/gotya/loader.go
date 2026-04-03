// Package main provides the gotya CLI tool and its loading utilities.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GabrielNunesIT/gotya/ast"
	"github.com/GabrielNunesIT/gotya/compiler"
	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/GabrielNunesIT/gotya/schema"
)

// DirectoryLoader implements compiler.Loader to resolve imports/includes from local directories.
//
// Why: When invoking `gotya` via CLI, paths are provided as strings specifying where to parse dependency YANG files from.
// Instead of hardcoding a directory logic inside the compiler, a loader acts as an intelligent file-system bridge that caches AST and Schemas.
type DirectoryLoader struct {
	paths       []string
	astCache    map[string]*ast.Module
	schemaCache map[string]*schema.Module
	inProgress  map[string]bool
}

// NewDirectoryLoader creates a loader referencing a list of directories
func NewDirectoryLoader(paths []string) *DirectoryLoader {
	return &DirectoryLoader{
		paths:       paths,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
		inProgress:  make(map[string]bool),
	}
}

// LoadAST locates a YANG file by name in the specified paths, reads it, and parses it into an AST module.
func (l *DirectoryLoader) LoadAST(name string) (*ast.Module, error) {
	if m, ok := l.astCache[name]; ok {
		return m, nil
	}

	var targetPath string
OuterLoop:
	for _, dir := range l.paths {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue // Skip unreadable directories
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			// Supported formats: 'modulename.yang' or 'modulename@revision.yang'
			if f.Name() == name+".yang" || strings.HasPrefix(f.Name(), name+"@") {
				targetPath = filepath.Join(dir, f.Name())
				break OuterLoop
			}
		}
	}

	if targetPath == "" {
		return nil, fmt.Errorf("module %s not found in provided paths", name)
	}

	cleanPath := filepath.Clean(targetPath)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", cleanPath, err)
	}

	lex := lexer.New(string(content))
	p := parser.New(lex)
	astMod := p.ParseModule()
	if astMod == nil {
		return nil, fmt.Errorf("failed to parse AST for module %s at %s", name, targetPath)
	}

	l.astCache[name] = astMod
	if astMod.Argument() != name {
		l.astCache[astMod.Argument()] = astMod
	}

	return astMod, nil
}

// Load compiles a resolved schema from the loaded AST module.
func (l *DirectoryLoader) Load(name string) (*schema.Module, error) {
	if m, ok := l.schemaCache[name]; ok {
		return m, nil
	}
	if l.inProgress[name] {
		return nil, fmt.Errorf("circular import detected: module %s is already being compiled", name)
	}
	l.inProgress[name] = true
	defer func() { delete(l.inProgress, name) }()

	astMod, err := l.LoadAST(name)
	if err != nil {
		return nil, err
	}

	comp := compiler.New(&compiler.Options{Loader: l})
	schemaMod, err := comp.Compile(astMod)

	// Only cache the schema module if compilation succeeded (no errors).
	// This prevents partial/invalid modules from being cached and reused.
	if err == nil && schemaMod != nil {
		l.schemaCache[name] = schemaMod
		if schemaMod.Name != name && schemaMod.Name != "" {
			l.schemaCache[schemaMod.Name] = schemaMod
		}
	}

	if err != nil {
		return schemaMod, fmt.Errorf("compile: %w", err)
	}

	return schemaMod, nil
}
