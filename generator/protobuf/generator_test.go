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

func TestProtoGenerator_Generate(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-module {
	namespace "urn:test";
	prefix t;

	container system {
		leaf hostname {
			type string;
			default "localhost";
		}
		list interface {
			key "name";
			leaf name {
				type string;
			}
			leaf enabled {
				type boolean;
				default "true";
			}
			leaf mtu {
				type uint16;
			}
			leaf status {
				type enumeration {
					enum UP;
					enum DOWN;
				}
				default "UP";
			}
			choice protocol {
				default "http";
				case http {
					leaf port { type uint16; }
				}
				case https {
					leaf secure-port { type uint16; }
				}
			}
			leaf-list aliases {
				type string;
			}
		}
	}
}
`
	l := lexer.New(yangSource)
	p := parser.New(l)
	astMod := p.ParseModule()
	require.NotNil(t, astMod)

	c := compiler.New(&compiler.Options{})
	schemaMod, err := c.Compile(astMod)
	require.NoError(t, err)
	gen := protobuf.New(&protobuf.Options{
		PackageName:             "test_package",
		GeneratePopulateDefault: true,
	})
	var buf bytes.Buffer
	err = gen.Generate(schemaMod, &buf)
	require.NoError(t, err)

	out := buf.String()

	// High level assert structures generated
	assert.Contains(t, out, "package test_package;")
	assert.Contains(t, out, "message System {")
	// Assert defaults generated locally
	assert.Contains(t, out, "import \"google/protobuf/descriptor.proto\";")
	assert.Contains(t, out, "extend google.protobuf.FieldOptions {")
	assert.Contains(t, out, "string gotya_default = 50000;")

	// Assert generated fields with extensions
	assert.Contains(t, out, "string hostname = 1 [(gotya_default) = \"localhost\"];")
	assert.Contains(t, out, "repeated Interface interface = 2;")

	assert.Contains(t, out, "message Interface {")
	assert.Contains(t, out, "repeated string aliases = 1;")
	assert.Contains(t, out, "bool enabled = 2 [(gotya_default) = \"true\"];")
	assert.Contains(t, out, "uint32 mtu = 3;")
	assert.Contains(t, out, "string name = 4;")

	assert.Contains(t, out, "oneof protocol {")
	assert.Contains(t, out, "option (gotya_oneof_default) = \"http\";")
	assert.Contains(t, out, "HttpCase http = 5;")
	assert.Contains(t, out, "HttpsCase https = 6;")
	assert.Contains(t, out, "Status status = 7 [(gotya_default) = \"UP\"];")

	// Check if enums were generated properly
	assert.Contains(t, out, "enum Status {")
	assert.Contains(t, out, "STATUS_UP = 0;")
	assert.Contains(t, out, "STATUS_DOWN = 1;")

	// Check if cases were generated properly as messages
	assert.Contains(t, out, "message HttpCase {")
	assert.Contains(t, out, "uint32 port = 1;")
	assert.Contains(t, out, "message HttpsCase {")
	assert.Contains(t, out, "uint32 secure_port = 1;")
}

func TestProtoGenerator_GenerateDevice(t *testing.T) {
	t.Parallel()
	yangSource1 := `
module test-module-1 {
	namespace "urn:test1";
	prefix t1;
	container sys {
		leaf id { type string; }
	}
}`
	yangSource2 := `
module test-module-2 {
	namespace "urn:test2";
	prefix t2;
	container net {
		leaf port { type uint16; }
	}
}`
	l1 := lexer.New(yangSource1)
	p1 := parser.New(l1)
	astMod1 := p1.ParseModule()
	require.NotNil(t, astMod1)

	l2 := lexer.New(yangSource2)
	p2 := parser.New(l2)
	astMod2 := p2.ParseModule()
	require.NotNil(t, astMod2)

	c := compiler.New(&compiler.Options{})
	schemaMod1, err := c.Compile(astMod1)
	require.NoError(t, err)
	schemaMod2, err := c.Compile(astMod2)
	require.NoError(t, err)

	gen := protobuf.New(&protobuf.Options{
		PackageName:           "test_pkg",
		GenerateFakeroot:      true,
		GenerateCELValidation: true,
	})
	var buf bytes.Buffer
	protoGen, ok := gen.(*protobuf.ProtoGenerator)
	require.True(t, ok)
	err = protoGen.GenerateDevice([]*schema.Module{schemaMod1, schemaMod2}, &buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "message Device {")
	assert.Contains(t, out, "TestModule1 test_module_1 = 1;")
	assert.Contains(t, out, "TestModule2 test_module_2 = 2;")
	assert.Contains(t, out, "message TestModule1 {")
	assert.Contains(t, out, "message Sys {")
}

func TestNew(t *testing.T) {
	t.Parallel()
	gen := protobuf.New(nil)
	assert.NotNil(t, gen)

	gen2 := protobuf.New(&protobuf.Options{})
	assert.NotNil(t, gen2)
}
