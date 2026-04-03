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

// TestRPCServiceBlock verifies that a *schema.RPC node emits a service block with rpc method,
// and both Input and Output message definitions with module-scoped names.
func TestRPCServiceBlock(t *testing.T) {

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
	assert.Contains(t, out, "rpc ResetCounters(TestRpcResetCountersInput) returns (TestRpcResetCountersOutput)", "rpc signature must use module-scoped request/response types")
	assert.Contains(t, out, "message TestRpcResetCountersInput", "Input message must be defined with module scope")
	assert.Contains(t, out, "message TestRpcResetCountersOutput", "Output message must be defined with module scope")
}

// TestActionMessageOnly verifies that a *schema.Action node emits Input/Output messages
// but does not emit a gRPC service method.
func TestActionMessageOnly(t *testing.T) {

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
	assert.NotContains(t, out, "service TestActionService", "action-only modules must not emit service blocks")
	assert.NotContains(t, out, "rpc Ping(", "actions must not be emitted as grpc methods")
	assert.Contains(t, out, "message Network", "parent container message must be emitted")
	assert.Contains(t, out, "Ping ping = 1;", "action must be represented as a field in its parent message using normal naming strategy")
	assert.Contains(t, out, "message Ping", "action wrapper message must use action name")
	assert.Contains(t, out, "message PingInput", "action input message must use action name")
	assert.Contains(t, out, "message PingOutput", "action output message must use action name")
}

// TestActionContextUniqueNames verifies that actions with the same name in different
// YANG contexts produce distinct Input/Output message names.
func TestActionContextUniqueNames(t *testing.T) {
	t.Parallel()
	yangSource := `
module test-action-context {
	namespace "urn:test-action-context";
	prefix tac;

	container access {
		action reset {
			input { leaf reason { type string; } }
			output { leaf ok { type boolean; } }
		}
	}

	container transport {
		action reset {
			input { leaf reason { type string; } }
			output { leaf ok { type boolean; } }
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
	assert.Contains(t, out, "message Access", "access container message must be emitted")
	assert.Contains(t, out, "message Transport", "transport container message must be emitted")
	assert.Contains(t, out, "Reset reset = 1;", "action must be represented as a field in parent containers")
	assert.Contains(t, out, "message Reset", "action wrapper message must be emitted")
	assert.Contains(t, out, "message ResetInput")
	assert.Contains(t, out, "message ResetOutput")
	assert.NotContains(t, out, "service TestActionContextService", "action-only modules must not emit services")
}

// TestNotificationMessage verifies that a *schema.Notification node emits a standalone message
// and does NOT emit a service block rpc entry for the notification.
func TestNotificationMessage(t *testing.T) {

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
