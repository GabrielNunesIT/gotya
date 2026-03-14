package golang_test

import (
	"bytes"
	"testing"

	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/generator/golang"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
	"github.com/stretchr/testify/require"
)

func TestIdentityref(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-identityref {
	namespace "urn:test-identityref";
	prefix ti;

	identity base-identity;
	identity derived-one {
		base base-identity;
	}

	container config {
		leaf my-leaf {
			type identityref {
				base base-identity;
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

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	require.NoError(t, err)

	t.Fatal("not yet implemented")
}

func TestIdentityrefCrossModule(t *testing.T) {
	t.Parallel()

	// Cross-module identityref base resolution requires a loader for import resolution.
	// The compiler does not support resolving prefixed bases (bt:base-identity) without
	// a loader. Use manually constructed schema.Module objects to represent the compiled
	// state for stub purposes — the generator stub test fails on t.Fatal below.

	baseIdent := schema.NewIdentity("base-identity")
	derivedIdent := schema.NewIdentity("derived-one")
	derivedIdent.Bases = []string{"base-identity"}

	baseModule := &schema.Module{
		Name:       "base-types",
		Namespace:  "urn:base-types",
		Prefix:     "bt",
		Identities: map[string]*schema.Identity{
			"base-identity": baseIdent,
			"derived-one":   derivedIdent,
		},
	}

	leafType := schema.TypeDefinition{
		Name:  "identityref",
		Bases: []string{"base-types:base-identity"},
	}
	myLeaf := schema.NewLeaf("my-leaf", &leafType)
	configContainer := schema.NewContainer("config")
	require.NoError(t, configContainer.AddChild(myLeaf))

	consumerModule := &schema.Module{
		Name:      "consumer-module",
		Namespace: "urn:consumer-module",
		Prefix:    "cm",
		Nodes:     map[string]schema.Node{"config": configContainer},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule, consumerModule}, &buf)
	require.NoError(t, err)

	t.Fatal("not yet implemented")
}
