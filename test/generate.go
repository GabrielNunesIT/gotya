//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/GabrielNunesIT/gotya/ast"
	"github.com/GabrielNunesIT/gotya/compiler"
	"github.com/GabrielNunesIT/gotya/generator/golang"
	"github.com/GabrielNunesIT/gotya/generator/protobuf"
	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/GabrielNunesIT/gotya/schema"
)

var generatorFixtureModules = map[string]string{
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

// fileLoader is a custom loader for the true local ast/schema from a directory
type fileLoader struct {
	dir         string
	astCache    map[string]*ast.Module
	schemaCache map[string]*schema.Module
}

// LoadAST parses an AST directly from the filesystem by searching for the appropriate file.
//
// Why: The compiler needs continuous module resolution for import and include statements.
// A custom loader manages parsing dependencies automatically from the local yangs directory.
func (l *fileLoader) LoadAST(name string) (*ast.Module, error) {
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

// Load compiles a resolved schema from the previously loaded AST, using recursion.
//
// Why: Converting raw syntax to a semantic schema happens at compile time. Doing this lazily
// keeps memory footprint lower and prevents cyclic load explosions.
func (l *fileLoader) Load(name string) (*schema.Module, error) {
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

// main is the entrypoint to generate the Go structures from YANG models.
//
// Why: Having a standalone binary file allows go:generate to be easily used locally or on CI/CD
// without distributing multiple complex scripts.
func main() {
	outDir := "out"
	tmpDir, err := os.MkdirTemp("", "gotya-fixtures-")
	if err != nil {
		log.Fatalf("Failed to create temporary fixture directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	moduleNames := make([]string, 0, len(generatorFixtureModules))
	for name, src := range generatorFixtureModules {
		if err := os.WriteFile(filepath.Join(tmpDir, name+".yang"), []byte(src), 0600); err != nil {
			log.Fatalf("Failed to write fixture module %s: %v", name, err)
		}
		moduleNames = append(moduleNames, name)
	}

	loader := &fileLoader{
		dir:         tmpDir,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
	}

	var modules []*schema.Module
	for _, modName := range moduleNames {
		astMod, err := loader.LoadAST(modName)
		if err != nil {
			log.Printf("Failed to load AST for %s: %v", modName, err)
			continue
		}

		if astMod.Keyword() == "submodule" {
			continue
		}

		schemaMod, err := loader.Load(modName)
		if err != nil {
			log.Printf("Module %s compiled with errors: %v", modName, err)
		}
		if schemaMod != nil {
			modules = append(modules, schemaMod)
		}
	}

	if len(modules) == 0 {
		log.Fatalf("Expected to successfully parse at least one module")
	}
	fmt.Println("Initializing Go code generator...")
	gen := golang.New(&golang.Options{
		PackageName:             "device",
		RootName:                "Device",
		GenerateFakeroot:        true,
		AddAnnotations:          true,
		GenerateGetters:         true,
		GenerateSetters:         true,
		GeneratePopulateDefault: true,
		GenerateOrderedMaps:     true, // Preserve slice output compatibility
	})

	if err := os.MkdirAll(outDir, 0750); err != nil {
		log.Fatalf("Failed to create out directory: %v", err)
	}

	outPath := filepath.Join(outDir, "device.go")
	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("Failed to create output file %s: %v", outPath, err)
	}
	defer outFile.Close()

	err = gen.GenerateDevice(modules, outFile)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	log.Printf("Successfully generated device.go in %s", outPath)

	fmt.Println("Initializing Protobuf generator...")
	protoGen := protobuf.New(&protobuf.Options{
		PackageName:           "device",
		GenerateCELValidation: true,
	})

	protoOutPath := filepath.Join(outDir, "device.proto")
	protoOutFile, err := os.Create(protoOutPath)
	if err != nil {
		log.Fatalf("Failed to create output file %s: %v", protoOutPath, err)
	}
	defer protoOutFile.Close()

	err = protoGen.GenerateDevice(modules, protoOutFile)
	if err != nil {
		log.Fatalf("Proto generation failed: %v", err)
	}
	log.Printf("Successfully generated device.proto in %s", protoOutPath)
}
