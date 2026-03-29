package test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GabrielNunesIT/gotya/ast"
	protobuf "github.com/GabrielNunesIT/gotya/generator/protobuf"
	"github.com/GabrielNunesIT/gotya/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// update is shared with corpus_test.go if defined there; declare only if not already declared.
// Since both files live in the same package "test", we must avoid duplicate var declarations.
// The -update flag is declared here for proto golden files only.
var updateProto = flag.Bool("update-proto", false, "regenerate golden .proto files in test/out/proto/")

// TestGoldenProto runs every YANG file in test/assets/yangs/ through the full
// parse→compile→GenerateDevice pipeline for the proto generator and compares output
// to stored golden files in test/out/proto/.
//
// Run with -update-proto to regenerate all golden files.
func TestGoldenProto(t *testing.T) {
	yangsDir := filepath.Join("assets", "yangs")
	loader := &corpusLoader{
		dir:         yangsDir,
		astCache:    make(map[string]*ast.Module),
		schemaCache: make(map[string]*schema.Module),
	}

	entries, err := os.ReadDir(yangsDir)
	if err != nil {
		t.Fatalf("read corpus dir %s: %v", yangsDir, err)
	}

	// Load all modules sequentially to avoid concurrent map access in corpusLoader.
	var modules []namedModule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yang") {
			continue
		}
		modName := strings.TrimSuffix(name, ".yang")

		schemaMod, loadErr := loader.Load(modName)
		if loadErr != nil {
			t.Logf("module %s: compiler errors (not a test failure): %v", modName, loadErr)
		}
		if schemaMod != nil {
			modules = append(modules, namedModule{name: modName, mod: schemaMod})
		}
	}

	if len(modules) == 0 {
		t.Fatal("corpus loaded zero modules — corpus directory may be missing or empty")
	}

	goldenDir := filepath.Join("out", "proto")

	// Create golden directory before launching parallel subtests to avoid race on mkdir.
	if *updateProto {
		if mkdirErr := os.MkdirAll(goldenDir, 0750); mkdirErr != nil {
			t.Fatalf("create golden dir %s: %v", goldenDir, mkdirErr)
		}
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

			goldenPath := filepath.Join(goldenDir, nm.name+".proto")

			if *updateProto {
				if writeErr := os.WriteFile(goldenPath, buf.Bytes(), 0600); writeErr != nil {
					t.Fatalf("write golden %s: %v", goldenPath, writeErr)
				}
				return
			}

			golden, readErr := os.ReadFile(goldenPath)
			if os.IsNotExist(readErr) {
				t.Errorf("golden file missing for %s — run with -update-proto to create", nm.name)
				return
			}
			require.NoError(t, readErr, "read golden file %s", goldenPath)
			assert.Equal(t, string(golden), buf.String(),
				"proto output for %s does not match golden file — run with -update-proto to regenerate", nm.name)
		})
	}
}
