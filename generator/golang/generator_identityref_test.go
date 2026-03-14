package golang_test

import (
	"bytes"
	"go/format"
	"testing"

	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/generator/golang"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
	"github.com/stretchr/testify/assert"
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

	out := buf.String()

	// Identity type must be emitted (typed string alias).
	assert.Contains(t, out, "Identity", "expected identity type in output")

	// identityref leaf must NOT fall back to *string.
	assert.NotContains(t, out, `*string`, "identityref must not emit *string")

	// Derived const value must be present.
	assert.Contains(t, out, "derived-one", "expected derived identity const value")

	// String() method must be emitted.
	assert.Contains(t, out, "func (v", "expected String() method in output")

	// Generated output must be valid Go.
	_, fmtErr := format.Source([]byte(out))
	assert.NoError(t, fmtErr, "generated output must be valid Go source")
}

func TestIdentityrefCrossModule(t *testing.T) {
	t.Parallel()

	// Build the base module with an identity hierarchy.
	baseIdent := schema.NewIdentity("base-identity")
	derivedIdent := schema.NewIdentity("derived-one")
	derivedIdent.Bases = []string{"base-identity"}

	baseModule := &schema.Module{
		Name:      "base-types",
		Namespace: "urn:base-types",
		Prefix:    "bt",
		Identities: map[string]*schema.Identity{
			"base-identity": baseIdent,
			"derived-one":   derivedIdent,
		},
	}

	// Build the consumer module referencing the base via a prefixed identityref.
	leafType := schema.TypeDefinition{
		Name:  "identityref",
		Bases: []string{"bt:base-identity"},
	}
	myLeaf := schema.NewLeaf("my-leaf", &leafType)
	configContainer := schema.NewContainer("config")
	require.NoError(t, configContainer.AddChild(myLeaf))

	consumerModule := &schema.Module{
		Name:      "consumer-module",
		Namespace: "urn:consumer-module",
		Prefix:    "cm",
		Nodes:     map[string]schema.Node{"config": configContainer},
		// Imports maps the prefix used in identityref bases to the module name.
		Imports: map[string]string{"bt": "base-types"},
	}

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err := gen.GenerateDevice([]*schema.Module{baseModule, consumerModule}, &buf)
	require.NoError(t, err)

	out := buf.String()

	// Identity type must be present in generated output.
	assert.Contains(t, out, "Identity", "expected identity type in cross-module output")

	// Derived identity value from the base module must appear as a const.
	assert.Contains(t, out, "derived-one", "expected derived identity in cross-module output")

	// identityref field must NOT fall back to *string.
	assert.NotContains(t, out, `*string`, "identityref must not emit *string in cross-module output")

	// Generated output must be valid Go.
	_, fmtErr := format.Source([]byte(out))
	assert.NoError(t, fmtErr, "generated cross-module output must be valid Go source")
}
