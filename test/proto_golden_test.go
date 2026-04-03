package test

import (
	"bytes"
	"testing"

	"github.com/GabrielNunesIT/gotya/ast"
	protobuf "github.com/GabrielNunesIT/gotya/generator/protobuf"
	"github.com/GabrielNunesIT/gotya/schema"
	"github.com/stretchr/testify/assert"
)

// TestGoldenProto runs the fixture modules through parse->compile->GenerateDevice
// and asserts deterministic, non-empty proto output without external golden assets.
func TestGoldenProto(t *testing.T) {
	yangsDir := t.TempDir()
	moduleNames := writeFixtureModules(t, yangsDir)

	loader := &corpusLoader{
		dir:         yangsDir,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
	}

	// Load all modules sequentially to avoid concurrent map access in corpusLoader.
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

	gen := protobuf.New(&protobuf.Options{
		PackageName: "corpus",
		RootName:    "Device",
	})

	for _, nm := range modules {
		nm := nm // capture loop variable
		t.Run(nm.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if genErr := gen.GenerateDevice([]*schema.Module{nm.mod}, &buf); genErr != nil {
				t.Errorf("GenerateDevice(%s): %v", nm.name, genErr)
				return
			}

			out := buf.String()
			assert.NotEmpty(t, out, "proto output for %s should not be empty", nm.name)
			assert.Contains(t, out, "syntax = \"proto3\";", "proto output for %s should declare proto3 syntax", nm.name)
			assert.Contains(t, out, "package corpus;", "proto output for %s should include the configured package", nm.name)

			var second bytes.Buffer
			if genErr := gen.GenerateDevice([]*schema.Module{nm.mod}, &second); genErr != nil {
				t.Errorf("second GenerateDevice(%s): %v", nm.name, genErr)
				return
			}
			assert.Equal(t, out, second.String(), "proto output for %s should be deterministic", nm.name)
		})
	}
}
