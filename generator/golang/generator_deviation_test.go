package golang_test

import (
	"bytes"
	"testing"

	"github.com/gotya/gotya/generator/golang"
	"github.com/gotya/gotya/schema"
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

	baseModule := &schema.Module{
		Name:      "base-module",
		Namespace: "urn:base-module",
		Prefix:    "bm",
		Nodes:     map[string]schema.Node{"config": baseContainer},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule}, &buf)
	require.NoError(t, err)

	t.Fatal("not yet implemented")
}

// TestDeviationReplace verifies that when a leaf type is replaced via
// "deviate replace", the generator uses the new type in the Go struct.
func TestDeviationReplace(t *testing.T) {
	t.Parallel()

	// Leaf starts as uint16, deviation replaces type with string.
	// At stub stage, manually represent the post-deviation schema state.
	replacedLeaf := schema.NewLeaf("mtu", &schema.TypeDefinition{Name: "string"})
	baseContainer := schema.NewContainer("config")
	require.NoError(t, baseContainer.AddChild(replacedLeaf))

	baseModule := &schema.Module{
		Name:      "base-module",
		Namespace: "urn:base-module",
		Prefix:    "bm",
		Nodes:     map[string]schema.Node{"config": baseContainer},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule}, &buf)
	require.NoError(t, err)

	t.Fatal("not yet implemented")
}

// TestDeviationAddDelete verifies that properties added or removed via
// "deviate add" / "deviate delete" are reflected in the generated Go struct.
func TestDeviationAddDelete(t *testing.T) {
	t.Parallel()

	// Leaf with a min-elements constraint added via deviate add.
	// At stub stage, represent post-deviation schema using a LeafList
	// with MinElements set (as deviate add min-elements applies to list/leaf-list).
	minVal := uint32(1)
	leafList := schema.NewLeafList("aliases", &schema.TypeDefinition{Name: "string"})
	leafList.MinElements = &minVal
	baseContainer := schema.NewContainer("config")
	require.NoError(t, baseContainer.AddChild(leafList))

	baseModule := &schema.Module{
		Name:      "base-module",
		Namespace: "urn:base-module",
		Prefix:    "bm",
		Nodes:     map[string]schema.Node{"config": baseContainer},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule}, &buf)
	require.NoError(t, err)

	t.Fatal("not yet implemented")
}
