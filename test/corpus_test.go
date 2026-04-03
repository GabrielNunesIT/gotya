package test

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GabrielNunesIT/gotya/ast"
	"github.com/GabrielNunesIT/gotya/compiler"
	golanggenerator "github.com/GabrielNunesIT/gotya/generator/golang"
	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/GabrielNunesIT/gotya/schema"
)

var fixtureYANGModules = map[string]string{
	"test-types": `module test-types {
	namespace "urn:test-types";
	prefix tt;

	grouping common-fields {
		leaf id {
			type string;
		}
	}
}
`,
	"test-main": `module test-main {
	namespace "urn:test-main";
	prefix tm;

	import test-types { prefix tt; }

	container system {
		uses tt:common-fields;
		leaf enabled {
			type boolean;
		}
	}
}
`,
	"test-rpc": `module test-rpc {
	namespace "urn:test-rpc";
	prefix tr;

	rpc reboot {
		input {
			leaf reason {
				type string;
			}
		}
		output {
			leaf accepted {
				type boolean;
			}
		}
	}
}
`,
}

func writeFixtureModules(t *testing.T, dir string) []string {
	t.Helper()

	moduleNames := make([]string, 0, len(fixtureYANGModules))
	for moduleName, src := range fixtureYANGModules {
		path := filepath.Join(dir, moduleName+".yang")
		if err := os.WriteFile(path, []byte(src), 0600); err != nil {
			t.Fatalf("write fixture module %s: %v", moduleName, err)
		}
		moduleNames = append(moduleNames, moduleName)
	}

	return moduleNames
}

// corpusLoader is a copy of fileLoader from generate.go (which has //go:build ignore and
// cannot be imported). It implements schema.Loader for resolving cross-module imports.
type corpusLoader struct {
	dir         string
	astCache    map[string]*ast.Module
	schemaCache map[string]*schema.Module
}

// LoadAST parses an AST directly from the filesystem by searching for the appropriate file.
func (l *corpusLoader) LoadAST(name string) (*ast.Module, error) {
	if m, ok := l.astCache[name]; ok {
		return m, nil
	}

	files, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, err
	}

	var target string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), name+".yang") || strings.HasPrefix(f.Name(), name+"@") {
			target = filepath.Join(l.dir, f.Name())
			break
		}
	}

	if target == "" {
		return nil, fmt.Errorf("module %s not found in %s", name, l.dir)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}

	lex := lexer.New(string(content))
	p := parser.New(lex)
	astMod := p.ParseModule()
	if astMod == nil {
		return nil, fmt.Errorf("failed to parse module AST for file %s", name)
	}

	l.astCache[name] = astMod
	if astMod.Argument() != name {
		l.astCache[astMod.Argument()] = astMod
	}
	return astMod, nil
}

// Load compiles a resolved schema from the previously loaded AST.
func (l *corpusLoader) Load(name string) (*schema.Module, error) {
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

	if err != nil {
		return schemaMod, err
	}

	return schemaMod, nil
}

// namedModule pairs a module name with its compiled schema for use in parallel subtests.
type namedModule struct {
	name string
	mod  *schema.Module
}

// TestCorpus runs a representative in-memory fixture set through the full
// parse->compile->generate pipeline and asserts syntactically valid Go output.
func TestCorpus(t *testing.T) {
	yangsDir := t.TempDir()
	moduleNames := writeFixtureModules(t, yangsDir)

	loader := &corpusLoader{
		dir:         yangsDir,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
	}

	// Load all fixture modules sequentially to avoid concurrent map access in corpusLoader.
	var modules []namedModule
	for _, modName := range moduleNames {
		schemaMod, loadErr := loader.Load(modName)
		if loadErr != nil {
			t.Fatalf("module %s: compile failed: %v", modName, loadErr)
		}
		if schemaMod != nil {
			modules = append(modules, namedModule{name: modName, mod: schemaMod})
		}
	}

	if len(modules) == 0 {
		t.Fatal("corpus loaded zero modules — corpus directory may be missing or empty")
	}

	gen := golanggenerator.New(&golanggenerator.Options{
		PackageName: "corpus",
		RootName:    "Device",
	})

	// Run generation and format.Source assertions in parallel subtests.
	for _, nm := range modules {
		nm := nm // capture loop variable
		t.Run(nm.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if genErr := gen.Generate(nm.mod, &buf); genErr != nil {
				t.Errorf("Generate(%s): %v", nm.name, genErr)
				return
			}
			if buf.Len() == 0 {
				// Empty output is acceptable for modules with no generatable nodes.
				return
			}
			if _, fmtErr := format.Source(buf.Bytes()); fmtErr != nil {
				t.Errorf("Generate(%s) produced invalid Go syntax: %v", nm.name, fmtErr)
			}
		})
	}
}
