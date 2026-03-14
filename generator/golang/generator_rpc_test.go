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

func TestRPC(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-rpc {
	namespace "urn:test-rpc";
	prefix tr;

	rpc reset-counters {
		input {
			leaf target {
				type string;
			}
		}
		output {
			leaf status {
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

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)
	require.NoError(t, err)

	gen := golang.New(&golang.Options{PackageName: "testpkg"})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	require.NoError(t, err)

	t.Fatal("not yet implemented")
}

func TestAction(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-action {
	namespace "urn:test-action";
	prefix ta;

	container network {
		action ping {
			input {
				leaf host {
					type string;
				}
			}
			output {
				leaf result {
					type string;
				}
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

func TestNotification(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-notification {
	namespace "urn:test-notification";
	prefix tn;

	notification link-up {
		leaf interface-name {
			type string;
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
