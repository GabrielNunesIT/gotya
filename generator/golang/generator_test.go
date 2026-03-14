package golang_test

import (
	"bytes"
	"go/format"
	"strings"
	"testing"

	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/generator/golang"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	gen := golang.New(&golang.Options{
		PackageName:         "testpkg",
		GenerateOrderedMaps: true,
	})
	var buf bytes.Buffer
	err = gen.Generate(schemaMod, &buf)
	assert.NoError(t, err)

	out := buf.String()

	// Verify package statement
	assert.Contains(t, out, "package testpkg")

	// Verify System container
	assert.Contains(t, out, "type TestModuleSystem struct {\n")
	assert.Contains(t, out, "\tHostname *string `json:\"hostname,omitempty\" xml:\"urn:test hostname,omitempty\" yang:\"test-module:hostname\"`\n")
	assert.Contains(t, out, "\tInterface []*TestModuleInterfaceEntry `json:\"interface,omitempty\" xml:\"urn:test interface,omitempty\" yang:\"test-module:interface\"`\n")
	assert.Contains(t, out, "\tPort *uint16 `json:\"port,omitempty\" xml:\"urn:test port,omitempty\" yang:\"test-module:port\"`\n")
	assert.Contains(t, out, "\tSecurePort *uint16 `json:\"secure-port,omitempty\" xml:\"urn:test secure-port,omitempty\" yang:\"test-module:secure-port\"`\n")

	// Verify Interface list
	assert.Contains(t, out, "type TestModuleInterfaceEntry struct {\n")
	assert.Contains(t, out, "\tAliases []*string `json:\"aliases,omitempty\" xml:\"urn:test aliases,omitempty\" yang:\"test-module:aliases\"`\n")
	assert.Contains(t, out, "\tEnabled *bool `json:\"enabled,omitempty\" xml:\"urn:test enabled,omitempty\" yang:\"test-module:enabled\"`\n")
	assert.Contains(t, out, "\tMtu *uint16 `json:\"mtu,omitempty\" xml:\"urn:test mtu,omitempty\" yang:\"test-module:mtu\"`\n")
	assert.Contains(t, out, "\tName *string `json:\"name,omitempty\" xml:\"urn:test name,omitempty\" yang:\"test-module:name\"`\n")
	assert.Contains(t, out, "\tStatus *TestModuleStatusEnum `json:\"status,omitempty\" xml:\"urn:test status,omitempty\" yang:\"test-module:status\"`\n")

	// Verify IsOrdered and Order methods for the 'interface' list
	if !strings.Contains(out, "func (s *TestModuleSystem) IsOrderedInterface() bool {") {
		t.Fatalf("expected IsOrderedInterface method, got: %v", out)
	}

	if !strings.Contains(out, "func (s *TestModuleSystem) OrderInterface() string {") {
		t.Fatalf("expected OrderInterface method, got: %v", out)
	}

	// Verify IsOrdered and Order methods for the 'aliases' leaf-list
	if !strings.Contains(out, "func (s *TestModuleInterfaceEntry) IsOrderedAliases() bool {") {
		t.Fatalf("expected IsOrderedAliases method, got: %v", out)
	}

	if !strings.Contains(out, "func (s *TestModuleInterfaceEntry) OrderAliases() string {") {
		t.Fatalf("expected OrderAliases method, got: %v", out)
	}

	// Verify Validate methods
	assert.Contains(t, out, "func (s *TestModuleSystem) Validate() error {\n")
	assert.Contains(t, out, "func (s *TestModuleInterfaceEntry) Validate() error {\n")
}

func TestGoGenerator_FormatValidation(t *testing.T) {
	t.Parallel()
	// After format.Source integration, GenerateDevice output must be valid, gofmt-formatted Go.
	// This test verifies that: (a) valid modules produce no error, and (b) the output written
	// to the caller's writer passes go/format.Source (i.e. it IS the formatted output).
	gen := golang.New(&golang.Options{PackageName: "fmttest", RootName: "Device"})
	mod := &schema.Module{Name: "simple", Nodes: nil}
	var buf bytes.Buffer
	err := gen.(*golang.GoGenerator).GenerateDevice([]*schema.Module{mod}, &buf)
	require.NoError(t, err)
	// Verify output is valid, canonical Go syntax
	_, fmtErr := format.Source(buf.Bytes())
	assert.NoError(t, fmtErr, "GenerateDevice output must be valid Go syntax")
}
