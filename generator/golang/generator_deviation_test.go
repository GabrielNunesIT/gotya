package golang_test

import (
	"bytes"
	"go/format"
	"testing"

	"github.com/gotya/gotya/ast"
	"github.com/gotya/gotya/generator/golang"
	"github.com/gotya/gotya/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeviationNotSupported verifies that when a leaf is removed via
// "deviate not-supported", the generator omits it from the Go struct.
// Uses manually constructed schema.Module objects because cross-module
// deviation resolution is not yet supported by the compiler at stub stage.
func TestDeviationNotSupported(t *testing.T) {
	t.Parallel()

	mtuLeaf := schema.NewLeaf("mtu", &schema.TypeDefinition{Name: "uint16"})
	baseContainer := schema.NewContainer("config")
	require.NoError(t, baseContainer.AddChild(mtuLeaf))

	// Deviation: deviate not-supported on /config/mtu
	dev := schema.NewDeviation("/config/mtu")
	dev.Deviates["not-supported"] = nil // presence of key triggers removal

	baseModule := &schema.Module{
		Name:       "base-module",
		Namespace:  "urn:base-module",
		Prefix:     "bm",
		Nodes:      map[string]schema.Node{"config": baseContainer},
		Deviations: []*schema.Deviation{dev},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule}, &buf)
	require.NoError(t, err)

	out := buf.String()
	// The deviated leaf "mtu" should not appear as a field in generated output.
	assert.NotContains(t, out, "Mtu", "deviated leaf must be absent from generated Go")
}

// TestDeviationReplace verifies that when a leaf type is replaced via
// "deviate replace", the generator uses the new type in the Go struct.
func TestDeviationReplace(t *testing.T) {
	t.Parallel()

	// Leaf starts as uint16; deviation replaces type with string.
	mtuLeaf := schema.NewLeaf("mtu", &schema.TypeDefinition{Name: "uint16"})
	baseContainer := schema.NewContainer("config")
	require.NoError(t, baseContainer.AddChild(mtuLeaf))

	// Construct a minimal ast.Statement for "type string"
	typeStmt := &ast.BaseNode{Key: "type", Arg: "string"}

	// Deviation: deviate replace type string on /config/mtu
	dev := schema.NewDeviation("/config/mtu")
	dev.Deviates["replace"] = []ast.Statement{typeStmt}

	baseModule := &schema.Module{
		Name:       "base-module",
		Namespace:  "urn:base-module",
		Prefix:     "bm",
		Nodes:      map[string]schema.Node{"config": baseContainer},
		Deviations: []*schema.Deviation{dev},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule}, &buf)
	require.NoError(t, err)

	out := buf.String()
	// After type replacement, the field should use *string, not *uint16.
	assert.Contains(t, out, "*string", "replaced type must appear in generated Go")
}

// TestDeviationAddDelete verifies that properties added or removed via
// "deviate add" / "deviate delete" do not cause panics or errors in the generator.
// For v1, add/delete affect only constraints not emitted as Go — the observable
// effect is that GenerateDevice succeeds and produces valid Go.
func TestDeviationAddDelete(t *testing.T) {
	t.Parallel()

	// LeafList with a deviation adding min-elements constraint.
	// The deviate add / delete are no-ops for v1 generation.
	leafList := schema.NewLeafList("aliases", &schema.TypeDefinition{Name: "string"})
	baseContainer := schema.NewContainer("config")
	require.NoError(t, baseContainer.AddChild(leafList))

	// Construct a minimal ast.Statement for "min-elements 1"
	minElemStmt := &ast.BaseNode{Key: "min-elements", Arg: "1"}

	// Deviation: deviate add min-elements 1 on /config/aliases
	dev := schema.NewDeviation("/config/aliases")
	dev.Deviates["add"] = []ast.Statement{minElemStmt}

	baseModule := &schema.Module{
		Name:       "base-module",
		Namespace:  "urn:base-module",
		Prefix:     "bm",
		Nodes:      map[string]schema.Node{"config": baseContainer},
		Deviations: []*schema.Deviation{dev},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule}, &buf)
	require.NoError(t, err)

	// Verify the output is valid Go source.
	out := buf.Bytes()
	_, fmtErr := format.Source(out)
	assert.NoError(t, fmtErr, "generated Go must be valid syntax")
}
