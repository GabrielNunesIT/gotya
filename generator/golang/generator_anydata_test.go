package golang_test

import (
	"bytes"
	"go/format"
	"testing"

	"github.com/GabrielNunesIT/gotya/compiler"
	"github.com/GabrielNunesIT/gotya/generator/golang"
	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/GabrielNunesIT/gotya/schema"
	"github.com/stretchr/testify/assert"
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

	out := buf.String()
	assert.Contains(t, out, "json.RawMessage", "anydata node must emit json.RawMessage field")
	assert.Contains(t, out, "Payload", "field name 'payload' should appear as 'Payload' in output")

	_, fmtErr := format.Source([]byte(out))
	assert.NoError(t, fmtErr, "generated output must be valid Go")
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

	out := buf.String()
	assert.Contains(t, out, "json.RawMessage", "anyxml node must emit json.RawMessage field")
	assert.Contains(t, out, "Payload", "field name 'payload' should appear as 'Payload' in output")

	_, fmtErr := format.Source([]byte(out))
	assert.NoError(t, fmtErr, "generated output must be valid Go")
}
