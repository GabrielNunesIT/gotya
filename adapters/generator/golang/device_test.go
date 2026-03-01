package golang_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotya/gotya/adapters/generator/golang"
	"github.com/gotya/gotya/adapters/parser/lexer"
	"github.com/gotya/gotya/adapters/parser/parser"
	"github.com/gotya/gotya/domain/ast"
	"github.com/gotya/gotya/domain/schema"
	"github.com/gotya/gotya/usecases/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLoader struct {
	dir         string
	astCache    map[string]*ast.Module
	schemaCache map[string]*schema.Module
}

func (l *testLoader) LoadAST(name string) (*ast.Module, error) {
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
		return schemaMod, err
	}

	return schemaMod, nil
}

func TestGenerateDeviceFromTestAssets(t *testing.T) {
	t.Parallel()

	yangsDir := filepath.Join("..", "..", "..", "test", "assets", "yangs")

	// Read all .yang files in the directory
	files, err := os.ReadDir(yangsDir)
	require.NoError(t, err)

	loader := &testLoader{
		dir:         yangsDir,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
	}

	var modules []*schema.Module
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yang" {
			continue
		}

		modName := file.Name()
		if idx := strings.Index(modName, "@"); idx != -1 {
			modName = modName[:idx]
		} else {
			modName = strings.TrimSuffix(modName, ".yang")
		}

		astMod, err := loader.LoadAST(modName)
		if err != nil {
			fmt.Printf("Failed to load AST for %s: %v\n", modName, err)
			continue
		}

		if astMod.Keyword() == "submodule" {
			continue
		}

		fmt.Printf("Loading module %s\n", modName)
		schemaMod, err := loader.Load(modName)
		if err != nil {
			fmt.Printf("Module %s compiled with errors: %v\n", modName, err)
		}
		if schemaMod != nil {
			modules = append(modules, schemaMod)
		}
	}

	require.NotEmpty(t, modules, "Expected to successfully parse at least one module")

	gen := golang.New("device")
	var buf bytes.Buffer
	err = gen.GenerateDevice(modules, &buf)
	assert.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "type DeviceConfig struct {")
	assert.Contains(t, out, "type DeviceState struct {")

	outDir := filepath.Join("..", "..", "..", "test", "out")
	err = os.MkdirAll(outDir, 0750)
	require.NoError(t, err)

	outPath := filepath.Join(outDir, "device.go")
	err = os.WriteFile(outPath, []byte(out), 0600)
	require.NoError(t, err)

	t.Logf("Device struct successfully written to %s", outPath)
}
