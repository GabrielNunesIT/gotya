package protobuf_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/GabrielNunesIT/gotya/compiler"
	"github.com/GabrielNunesIT/gotya/generator/protobuf"
	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/GabrielNunesIT/gotya/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var protobufFixtureModules = map[string]string{
	"test-main": `module test-main {
  namespace "urn:test-main";
  prefix tm;

  container system {
	    leaf id {
	      type string;
	    }
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

func compileFixtureModules(t *testing.T) []*schema.Module {
	t.Helper()

	yangsDir := t.TempDir()
	for moduleName, src := range protobufFixtureModules {
		cleanPath := filepath.Clean(filepath.Join(yangsDir, moduleName+".yang"))
		require.NoError(t, os.WriteFile(cleanPath, []byte(src), 0600))
	}

	files, err := os.ReadDir(yangsDir)
	require.NoError(t, err)

	var modules []*schema.Module
	comp := compiler.New(&compiler.Options{})

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yang" {
			continue
		}

		cleanPath := filepath.Clean(filepath.Join(yangsDir, file.Name()))
		content, readErr := os.ReadFile(cleanPath)
		require.NoError(t, readErr)

		l := lexer.New(string(content))
		p := parser.New(l)
		astMod := p.ParseModule()
		if astMod == nil {
			t.Fatalf("failed to parse fixture module %s", file.Name())
		}

		schemaMod, compErr := comp.Compile(astMod)
		require.NoError(t, compErr)
		if schemaMod != nil {
			modules = append(modules, schemaMod)
		}
	}

	require.NotEmpty(t, modules, "Expected to successfully parse at least one module")
	return modules
}

func TestGenerateDeviceFromTestAssetsProto(t *testing.T) {
	t.Parallel()
	modules := compileFixtureModules(t)
	gen := protobuf.New(&protobuf.Options{
		PackageName:      "device",
		GenerateFakeroot: true,
	})
	var buf bytes.Buffer
	err := gen.GenerateDevice(modules, &buf)
	assert.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "message Device {")

	outDir := filepath.Join("..", "..", "test", "out")
	err = os.MkdirAll(outDir, 0750)
	require.NoError(t, err)

	outPath := filepath.Join(outDir, "device.proto")
	err = os.WriteFile(outPath, []byte(out), 0600)
	require.NoError(t, err)

	t.Logf("Proto Device struct successfully written to %s", outPath)
}

func TestGenerateDeviceFromTestAssetsProtoWithCEL(t *testing.T) {
	t.Parallel()
	modules := compileFixtureModules(t)
	gen := protobuf.New(&protobuf.Options{
		PackageName:           "device",
		GenerateFakeroot:      true,
		GenerateCELValidation: true,
	})
	var buf bytes.Buffer
	err := gen.GenerateDevice(modules, &buf)
	assert.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "message Device {")
	assert.Contains(t, out, "buf/validate/validate.proto")

	outDir := filepath.Join("..", "..", "test", "out")
	err = os.MkdirAll(outDir, 0750)
	require.NoError(t, err)

	outPath := filepath.Join(outDir, "device_cel.proto")
	err = os.WriteFile(outPath, []byte(out), 0600)
	require.NoError(t, err)

	t.Logf("Proto Device struct with CEL successfully written to %s", outPath)
}
