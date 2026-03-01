package protobuf_test

import (
	"bytes"
	"testing"

	"github.com/gotya/gotya/adapters/generator/protobuf"
	"github.com/gotya/gotya/adapters/parser/lexer"
	"github.com/gotya/gotya/adapters/parser/parser"
	"github.com/gotya/gotya/usecases/compiler"
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
		}
		list interface {
			key "name";
			leaf name {
				type string;
			}
			leaf enabled {
				type boolean;
			}
			leaf mtu {
				type uint16;
			}
			leaf status {
				type enumeration {
					enum UP;
					enum DOWN;
				}
			}
			choice protocol {
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

	gen := protobuf.New("test_package")
	var buf bytes.Buffer
	err = gen.Generate(schemaMod, &buf)
	require.NoError(t, err)

	out := buf.String()

	// High level assert structures generated
	assert.Contains(t, out, "package test_package;")
	assert.Contains(t, out, "message System {")
	assert.Contains(t, out, "string hostname = 1;")
	assert.Contains(t, out, "repeated Interface interface = 2;")

	assert.Contains(t, out, "message Interface {")
	assert.Contains(t, out, "repeated string aliases = 1;")
	assert.Contains(t, out, "bool enabled = 2;")
	assert.Contains(t, out, "uint32 mtu = 3;")
	assert.Contains(t, out, "string name = 4;")
	assert.Contains(t, out, "oneof protocol {")
	assert.Contains(t, out, "HttpCase http = 5;")
	assert.Contains(t, out, "HttpsCase https = 6;")
	assert.Contains(t, out, "Status status = 7;")

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
