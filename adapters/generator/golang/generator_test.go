package golang_test

import (
	"bytes"
	"testing"

	"github.com/gotya/gotya/adapters/generator/golang"
	"github.com/gotya/gotya/adapters/parser/lexer"
	"github.com/gotya/gotya/adapters/parser/parser"
	"github.com/gotya/gotya/usecases/compiler"
	"github.com/stretchr/testify/assert"
)

func TestGoGenerator_Generate(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-module {
	namespace "urn:test";
	prefix tm;

	container system {
		leaf hostname {
			type string;
		}
		choice protocol {
			case http {
				leaf port {
					type uint16;
				}
			}
			case https {
				leaf secure-port {
					type uint16;
				}
			}
		}
		list interface {
			key "name";
			config true;
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
	assert.NotNil(t, astMod)

	opts := &compiler.Options{}
	comp := compiler.New(opts)
	schemaMod, err := comp.Compile(astMod)
	assert.NoError(t, err)

	gen := golang.New("testpkg")
	var buf bytes.Buffer
	err = gen.Generate(schemaMod, &buf)
	assert.NoError(t, err)

	out := buf.String()

	// Verify package statement
	assert.Contains(t, out, "package testpkg")

	// Verify System container
	assert.Contains(t, out, "type System struct {\n")
	assert.Contains(t, out, "\tHostname *string `json:\"hostname,omitempty\"`\n")
	assert.Contains(t, out, "\tInterface map[string]*Interface `json:\"interface,omitempty\"`\n")
	assert.Contains(t, out, "\tPort *uint16 `json:\"port,omitempty\"`\n")
	assert.Contains(t, out, "\tSecurePort *uint16 `json:\"secure-port,omitempty\"`\n")

	// Verify Interface list
	assert.Contains(t, out, "type Interface struct {\n")
	assert.Contains(t, out, "\tAliases []*string `json:\"aliases,omitempty\"`\n")
	assert.Contains(t, out, "\tEnabled *bool `json:\"enabled,omitempty\"`\n")
	assert.Contains(t, out, "\tMtu *uint16 `json:\"mtu,omitempty\"`\n")
	assert.Contains(t, out, "\tName *string `json:\"name,omitempty\"`\n")
	assert.Contains(t, out, "\tStatus *string `json:\"status,omitempty\"`\n")

	// Verify Validate methods
	assert.Contains(t, out, "func (s *System) Validate() error {\n")
	assert.Contains(t, out, "func (s *Interface) Validate() error {\n")
}
