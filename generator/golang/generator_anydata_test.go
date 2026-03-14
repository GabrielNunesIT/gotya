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

func TestAnyData(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-anydata {
	namespace "urn:test-anydata";
	prefix ta;

	container state {
		anydata payload;
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

func TestAnyXML(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-anyxml {
	namespace "urn:test-anyxml";
	prefix tx;

	container state {
		anyxml payload;
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
