package protobuf_test

import (
	"bytes"
	"testing"

	"github.com/GabrielNunesIT/gotya/compiler"
	"github.com/GabrielNunesIT/gotya/generator/protobuf"
	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/GabrielNunesIT/gotya/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCELPathValidationValid verifies that GenerateDevice with GenerateCELValidation=true
// returns nil when all annotated leaf fields exist in the schema.
func TestCELPathValidationValid(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-cel-valid {
	namespace "urn:test-cel-valid";
	prefix tcv;

	container config {
		leaf name {
			type string {
				length "1..64";
			}
		}
		leaf count {
			type uint32 {
				range "1..100";
			}
		}
	}
}
`
	l := lexer.New(yangSource)
	p := parser.New(l)
	astMod := p.ParseModule()
	require.NotNil(t, astMod)

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)
	require.NoError(t, err)

	gen := protobuf.New(&protobuf.Options{
		PackageName:           "test",
		RootName:              "Device",
		GenerateCELValidation: true,
	})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	assert.NoError(t, err, "GenerateDevice must return nil error when all CEL annotation paths are valid")
}

// TestCELPathValidationInvalid verifies that GenerateDevice with GenerateCELValidation=true
// returns a non-nil error containing "CEL path validation failed" when an annotated leaf path
// has no corresponding schema node.
func TestCELPathValidationInvalid(t *testing.T) {
	t.Parallel()
	// Manually construct a schema module where a leaf has a CEL annotation path that
	// does not correspond to any real schema node. We do this by creating a module
	// with a leaf whose type has constraints (triggering buildValidateOptions), then
	// poisoning the schema so the generated proto field path does not exist.
	mod := &schema.Module{
		Name:      "test-cel-invalid",
		Namespace: "urn:test-cel-invalid",
		Nodes:     make(map[string]schema.Node),
	}
	// Add a leaf with a length constraint so buildValidateOptions emits a CEL annotation.
	leaf := schema.NewLeaf("valid-name", &schema.TypeDefinition{
		Name:   "string",
		Length: []string{"1..64"},
	})
	require.NoError(t, mod.AddNode(leaf))

	gen := protobuf.New(&protobuf.Options{
		PackageName:           "test",
		RootName:              "Device",
		GenerateCELValidation: true,
	})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{mod}, &buf)
	assert.Error(t, err, "GenerateDevice must return an error when a CEL annotation path is invalid")
	assert.Contains(t, err.Error(), "CEL path validation failed",
		"error must mention 'CEL path validation failed'")
}

// TestCELPathValidationAllErrors verifies that when multiple invalid CEL paths exist,
// all of them are reported in the error (not just the first).
func TestCELPathValidationAllErrors(t *testing.T) {
	t.Parallel()
	// Construct a module with two leaves that have type constraints (triggering CEL annotations)
	// and verify that if both paths are invalid, both appear in the error message.
	mod := &schema.Module{
		Name:      "test-cel-multi-invalid",
		Namespace: "urn:test-cel-multi-invalid",
		Nodes:     make(map[string]schema.Node),
	}
	leaf1 := schema.NewLeaf("first-field", &schema.TypeDefinition{
		Name:   "string",
		Length: []string{"1..32"},
	})
	leaf2 := schema.NewLeaf("second-field", &schema.TypeDefinition{
		Name:    "int32",
		Range:   []string{"1..100"},
	})
	require.NoError(t, mod.AddNode(leaf1))
	require.NoError(t, mod.AddNode(leaf2))

	gen := protobuf.New(&protobuf.Options{
		PackageName:           "test",
		RootName:              "Device",
		GenerateCELValidation: true,
	})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{mod}, &buf)
	require.Error(t, err, "GenerateDevice must return an error when CEL annotation paths are invalid")
	assert.Contains(t, err.Error(), "first_field",
		"error must reference the first invalid field path")
	assert.Contains(t, err.Error(), "second_field",
		"error must reference the second invalid field path — all errors must be collected, not just the first")
}
