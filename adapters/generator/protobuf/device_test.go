package protobuf_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotya/gotya/adapters/generator/protobuf"
	"github.com/gotya/gotya/adapters/parser/lexer"
	"github.com/gotya/gotya/adapters/parser/parser"
	"github.com/gotya/gotya/domain/schema"
	"github.com/gotya/gotya/usecases/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDeviceFromTestAssetsProto(t *testing.T) {
	t.Parallel()

	yangsDir := filepath.Join("..", "..", "..", "test", "assets", "yangs")

	files, err := os.ReadDir(yangsDir)
	require.NoError(t, err)

	var modules []*schema.Module
	opts := &compiler.Options{}
	comp := compiler.New(opts)

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yang" {
			continue
		}

		path := filepath.Join(yangsDir, file.Name())
		content, err := os.ReadFile(path)
		require.NoError(t, err)

		l := lexer.New(string(content))
		p := parser.New(l)
		astMod := p.ParseModule()
		if astMod == nil {
			t.Logf("Failed to parse module AST for file %s", file.Name())
			continue
		}

		schemaMod, err := comp.Compile(astMod)
		if err != nil {
			fmt.Printf("Module %s compiled with errors: %v\n", file.Name(), err)
		}
		if schemaMod != nil {
			modules = append(modules, schemaMod)
		}
	}

	require.NotEmpty(t, modules, "Expected to successfully parse at least one module")

	gen := protobuf.New("device")
	var buf bytes.Buffer
	err = gen.GenerateDevice(modules, &buf)
	assert.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "message Device {")

	outDir := filepath.Join("..", "..", "..", "test", "out")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	outPath := filepath.Join(outDir, "device.proto")
	err = os.WriteFile(outPath, []byte(out), 0644)
	require.NoError(t, err)

	t.Logf("Proto Device struct successfully written to %s", outPath)
}
