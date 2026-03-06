package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotya/gotya/ast"
	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
)

// DirectoryLoader implements compiler.Loader to resolve imports/includes from local directories.
//
// Why: When invoking `gotya` via CLI, paths are provided as strings specifying where to parse dependency YANG files from.
// Instead of hardcoding a directory logic inside the compiler, a loader acts as an intelligent file-system bridge that caches AST and Schemas.
type DirectoryLoader struct {
	paths       []string
	astCache    map[string]*ast.Module
	schemaCache map[string]*schema.Module
}

// NewDirectoryLoader creates a loader referencing a list of directories
func NewDirectoryLoader(paths []string) *DirectoryLoader {
	return &DirectoryLoader{
		paths:       paths,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
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

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, err
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

	astMod, err := l.LoadAST(name)
	if err != nil {
		return nil, err
	}

	comp := compiler.New(&compiler.Options{Loader: l})
	schemaMod, err := comp.Compile(astMod)

	if schemaMod != nil {
		l.schemaCache[name] = schemaMod
		if schemaMod.Name != name && schemaMod.Name != "" {
			l.schemaCache[schemaMod.Name] = schemaMod
		}
	}

	return schemaMod, err
}
