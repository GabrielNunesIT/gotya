package protobuf_test

import (
	"bytes"
	"testing"

	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/generator/protobuf"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnyDataField verifies that a *schema.AnyData node emits a google.protobuf.Any field.
func TestAnyDataField(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-anydata {
	namespace "urn:test-anydata";
	prefix ta;

	anydata payload;
}
`
	l := lexer.New(yangSource)
	p := parser.New(l)
	astMod := p.ParseModule()
	require.NotNil(t, astMod)

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)
	require.NoError(t, err)

	gen := protobuf.New(&protobuf.Options{PackageName: "test", RootName: "Device"})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "google.protobuf.Any", "anydata node must emit google.protobuf.Any field type")
	assert.Contains(t, out, "payload", "anydata node field name must appear in output")
}

// TestAnyXMLField verifies that a *schema.AnyXML node emits a google.protobuf.Any field.
func TestAnyXMLField(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-anyxml {
	namespace "urn:test-anyxml";
	prefix tx;

	anyxml payload;
}
`
	l := lexer.New(yangSource)
	p := parser.New(l)
	astMod := p.ParseModule()
	require.NotNil(t, astMod)

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)
	require.NoError(t, err)

	gen := protobuf.New(&protobuf.Options{PackageName: "test", RootName: "Device"})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "google.protobuf.Any", "anyxml node must emit google.protobuf.Any field type")
	assert.Contains(t, out, "payload", "anyxml node field name must appear in output")
}

// TestAnyDataConditionalImport verifies that the google/protobuf/any.proto import is added
// only when the module contains anydata or anyxml nodes.
func TestAnyDataConditionalImport(t *testing.T) {
	t.Parallel()

	anydataSource := `
module test-with-anydata {
	namespace "urn:test-with-anydata";
	prefix twa;

	anydata payload;
}
`
	noAnydataSource := `
module test-no-anydata {
	namespace "urn:test-no-anydata";
	prefix tna;

	leaf name {
		type string;
	}
}
`

	comp := compiler.New(nil)

	l1 := lexer.New(anydataSource)
	p1 := parser.New(l1)
	astMod1 := p1.ParseModule()
	require.NotNil(t, astMod1)
	schemaMod1, err := comp.Compile(astMod1)
	require.NoError(t, err)

	l2 := lexer.New(noAnydataSource)
	p2 := parser.New(l2)
	astMod2 := p2.ParseModule()
	require.NotNil(t, astMod2)
	schemaMod2, err := comp.Compile(astMod2)
	require.NoError(t, err)

	// Module WITH anydata must include the any.proto import.
	gen := protobuf.New(&protobuf.Options{PackageName: "test", RootName: "Device"})
	var buf1 bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod1}, &buf1)
	require.NoError(t, err)
	assert.Contains(t, buf1.String(), `import "google/protobuf/any.proto";`,
		"module with anydata must include google/protobuf/any.proto import")

	// Module WITHOUT anydata must NOT include the any.proto import.
	var buf2 bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod2}, &buf2)
	require.NoError(t, err)
	assert.NotContains(t, buf2.String(), `import "google/protobuf/any.proto";`,
		"module without anydata must not include google/protobuf/any.proto import")
}
