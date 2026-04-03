package golang_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GabrielNunesIT/gotya/ast"
	"github.com/GabrielNunesIT/gotya/compiler"
	"github.com/GabrielNunesIT/gotya/generator/golang"
	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/GabrielNunesIT/gotya/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLoader struct {
	dir         string
	astCache    map[string]*ast.Module
	schemaCache map[string]*schema.Module
}

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

func writeGeneratorFixtures(t *testing.T, dir string) []string {
	t.Helper()

	moduleNames := make([]string, 0, len(generatorFixtureModules))
	for moduleName, src := range generatorFixtureModules {
		path := filepath.Join(dir, moduleName+".yang")
		err := os.WriteFile(path, []byte(src), 0600)
		require.NoError(t, err)
		moduleNames = append(moduleNames, moduleName)
	}

	return moduleNames
}

func (l *testLoader) LoadAST(name string) (*ast.Module, error) {
	if m, ok := l.astCache[name]; ok {
		return m, nil
	}

	files, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", l.dir, err)
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

	cleanTarget := filepath.Clean(target)
	content, err := os.ReadFile(cleanTarget)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", cleanTarget, err)
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

func (l *testLoader) Load(name string) (*schema.Module, error) {
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
		return schemaMod, fmt.Errorf("compile: %w", err)
	}

	return schemaMod, nil
}

func TestGenerateDeviceFromTestAssets(t *testing.T) {
	t.Parallel()

	yangsDir := t.TempDir()
	moduleNames := writeGeneratorFixtures(t, yangsDir)

	loader := &testLoader{
		dir:         yangsDir,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
	}

	var modules []*schema.Module
	for _, modName := range moduleNames {
		astMod, astErr := loader.LoadAST(modName)
		if astErr != nil {
			fmt.Printf("Failed to load AST for %s: %v\n", modName, astErr)
			continue
		}

		if astMod.Keyword() == "submodule" {
			continue
		}

		fmt.Printf("Loading module %s\n", modName)
		schemaMod, loadErr := loader.Load(modName)
		if loadErr != nil {
			fmt.Printf("Module %s compiled with errors: %v\n", modName, loadErr)
		}
		if schemaMod != nil {
			modules = append(modules, schemaMod)
		}
	}

	require.NotEmpty(t, modules, "Expected to successfully parse at least one module")
	gen := golang.New(&golang.Options{
		PackageName:      "device",
		RootName:         "Device",
		GenerateFakeroot: true,
	})
	var buf bytes.Buffer
	err := gen.GenerateDevice(modules, &buf)
	assert.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "type DeviceConfig struct {")
	assert.Contains(t, out, "type DeviceState struct {")

	outDir := filepath.Join("..", "..", "test", "out")
	err = os.MkdirAll(outDir, 0750)
	require.NoError(t, err)

	outPath := filepath.Join(outDir, "device.go")
	err = os.WriteFile(outPath, []byte(out), 0600)
	require.NoError(t, err)

	t.Logf("Device struct successfully written to %s", outPath)
}
