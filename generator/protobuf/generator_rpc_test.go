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

// TestRPCServiceBlock verifies that a *schema.RPC node emits a service block with rpc method,
// and both Input and Output message definitions.
func TestRPCServiceBlock(t *testing.T) {
	t.Fatal("not yet implemented")

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

	gen := protobuf.New(&protobuf.Options{PackageName: "test", RootName: "Device"})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "service", "rpc node must emit a service block")
	assert.Contains(t, out, "Service", "service block must be named after the module with a 'Service' suffix")
	assert.Contains(t, out, "rpc ", "service block must contain an rpc method")
	assert.Contains(t, out, "ResetCountersInput", "rpc node must emit an Input message type")
	assert.Contains(t, out, "ResetCountersOutput", "rpc node must emit an Output message type")
	assert.Contains(t, out, "message ResetCountersInput", "Input message must be defined")
	assert.Contains(t, out, "message ResetCountersOutput", "Output message must be defined")
}

// TestActionServiceBlock verifies that a *schema.Action node emits a service block with
// rpc method, and both Input and Output message definitions.
func TestActionServiceBlock(t *testing.T) {
	t.Fatal("not yet implemented")

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

	gen := protobuf.New(&protobuf.Options{PackageName: "test", RootName: "Device"})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "service", "action node must emit a service block")
	assert.Contains(t, out, "Service", "service block must be named after the module with a 'Service' suffix")
	assert.Contains(t, out, "rpc ", "service block must contain an rpc method entry")
	assert.Contains(t, out, "PingInput", "action node must emit an Input message type")
	assert.Contains(t, out, "PingOutput", "action node must emit an Output message type")
	assert.Contains(t, out, "message PingInput", "Input message must be defined")
	assert.Contains(t, out, "message PingOutput", "Output message must be defined")
}

// TestNotificationMessage verifies that a *schema.Notification node emits a standalone message
// and does NOT emit a service block rpc entry for the notification.
func TestNotificationMessage(t *testing.T) {
	t.Fatal("not yet implemented")

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

	gen := protobuf.New(&protobuf.Options{PackageName: "test", RootName: "Device"})
	var buf bytes.Buffer
	err = gen.GenerateDevice([]*schema.Module{schemaMod}, &buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "message LinkUpNotification", "notification must emit a standalone message named <Name>Notification")
	assert.NotContains(t, out, "rpc LinkUp", "notification must not appear as an rpc method in a service block")
	assert.NotContains(t, out, "LinkUpInput", "notification must not emit an Input message")
	assert.NotContains(t, out, "LinkUpOutput", "notification must not emit an Output message")
}
