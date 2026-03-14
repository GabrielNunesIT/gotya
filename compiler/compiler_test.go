package compiler_test

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gotya/gotya/compiler"
	"github.com/gotya/gotya/parser"
	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/schema"
	"github.com/stretchr/testify/assert"
)

// compile is a helper that runs lexer -> parser -> compiler on a YANG input string.
func compile(t *testing.T, input string) (*schema.Module, error) {
	t.Helper()
	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()
	comp := compiler.New(nil)
	return comp.Compile(astMod)
}

func TestCompiler_Compile(t *testing.T) {
	t.Parallel()

	input := `
		module interfaces {
			namespace "urn:interfaces";
			prefix "if";

			container interfaces {
				list interface {
					key "name";
					leaf name {
						type string;
					}
					leaf enabled {
						type boolean;
					}
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()
	assert.Empty(t, p.Errors())

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)
	assert.NotNil(t, schemaMod)
	assert.Equal(t, "interfaces", schemaMod.Name)
	assert.Equal(t, "urn:interfaces", schemaMod.Namespace)
	assert.Equal(t, "if", schemaMod.Prefix)

	// Verify top-level nodes
	assert.Len(t, schemaMod.Nodes, 1)
	assert.Contains(t, schemaMod.Nodes, "interfaces")

	cont, ok := schemaMod.Nodes["interfaces"].(*schema.Container)
	assert.True(t, ok)

	// Verify sub node "interface" inside container
	assert.Len(t, cont.GetChildren(), 1)
	assert.Contains(t, cont.GetChildren(), "interface")

	lst, ok := cont.GetChildren()["interface"].(*schema.List)
	assert.True(t, ok)
	assert.Equal(t, []string{"name"}, lst.Keys)

	// Verify leafs inside the list
	assert.Len(t, lst.GetChildren(), 2)
	assert.Contains(t, lst.GetChildren(), "name")
	assert.Contains(t, lst.GetChildren(), "enabled")

	leafName, ok := lst.GetChildren()["name"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Equal(t, "string", leafName.Type.Name)
}

func TestCompiler_Groupings(t *testing.T) {
	t.Parallel()

	input := `
		module test-group {
			namespace "urn:test";
			prefix "t";

			grouping common-fields {
				leaf id {
					type string;
				}
			}

			container item {
				uses common-fields;
				
				list sub-item {
					key "local-id";
					grouping local-group {
						leaf local-id {
							type int32;
						}
					}
					uses local-group;
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()
	assert.Empty(t, p.Errors())

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	cont, ok := schemaMod.Nodes["item"].(*schema.Container)
	assert.True(t, ok)

	// verify 'uses common-fields' in container 'item' generated 'leaf id'
	assert.Contains(t, cont.GetChildren(), "id")

	// verify 'list sub-item'
	assert.Contains(t, cont.GetChildren(), "sub-item")
	lst, ok := cont.GetChildren()["sub-item"].(*schema.List)
	assert.True(t, ok)

	// verify 'uses local-group' in list 'sub-item' generated 'leaf local-id'
	assert.Contains(t, lst.GetChildren(), "local-id")
}

func TestCompiler_Augment(t *testing.T) {
	t.Parallel()

	input := `
		module test-augment {
			namespace "urn:test";
			prefix "t";

			container base {
				leaf id {
					type string;
				}
			}

			augment "/t:base" {
				leaf extra {
					type int32;
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()
	assert.Empty(t, p.Errors())

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	cont, ok := schemaMod.Nodes["base"].(*schema.Container)
	assert.True(t, ok)

	// verify base string leaf exists
	assert.Contains(t, cont.GetChildren(), "id")

	// verify augmented int32 leaf exists
	assert.Contains(t, cont.GetChildren(), "extra")
}

func TestCompiler_Validation(t *testing.T) {
	t.Parallel()

	input := `
		module test-val {
			namespace "urn:test";
			prefix "t";

			list invalid-list {
				// No key but it's config data mapping to an error!
				leaf id {
					type string;
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list invalid-list must have at least one key")
}

func TestCompiler_IdentifierUniqueness(t *testing.T) {
	t.Parallel()

	input := `
		module test-unique {
			namespace "urn:test";
			prefix "t";

			container base {
				leaf id {
					type string;
				}
				leaf id {
					type int32;
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate identifier 'id'")
}

func TestCompiler_ConfigBoundary(t *testing.T) {
	t.Parallel()

	input := `
		module test-config {
			namespace "urn:test";
			prefix "t";

			container base {
				config false;
				leaf id {
					config true;
					type string;
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has 'config true' but its parent base has 'config false'")
}

func TestCompiler_TypeRestrictions(t *testing.T) {
	t.Parallel()

	input := `
		module test-type {
			namespace "urn:test";
			prefix "t";

			leaf my-string {
				type string {
					length "1..255";
					pattern "[a-zA-Z]+";
				}
			}

			leaf my-int {
				type int32 {
					range "1..100";
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	strLeaf, ok := schemaMod.Nodes["my-string"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Equal(t, "string", strLeaf.Type.Name)
	assert.Equal(t, []string{"1..255"}, strLeaf.Type.Length)
	assert.Equal(t, []string{"[a-zA-Z]+"}, strLeaf.Type.Pattern)

	intLeaf, ok := schemaMod.Nodes["my-int"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Equal(t, "int32", intLeaf.Type.Name)
	assert.Equal(t, []string{"1..100"}, intLeaf.Type.Range)
}

func TestCompiler_InvalidTypeRestrictions(t *testing.T) {
	t.Parallel()

	input := `
		module test-invalid-type {
			namespace "urn:test";
			prefix "t";

			leaf invalid-string {
				type string {
					range "1..100";
				}
			}

			leaf invalid-int {
				type int32 {
					length "1..10";
					pattern "[0-9]+";
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type string for node invalid-string cannot have range")
	assert.Contains(t, err.Error(), "type int32 for node invalid-int cannot have length")
	assert.Contains(t, err.Error(), "type int32 for node invalid-int cannot have pattern")
}

func TestCompiler_ListKeyValidation(t *testing.T) {
	t.Parallel()

	input := `
		module test-list-keys {
			namespace "urn:test";
			prefix "t";

			list bad-list-nonexistent {
				key "not-here";
				leaf there { type string; }
			}

			list bad-list-wrong-type {
				key "wrong-type";
				container wrong-type { }
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key 'not-here' not found in list 'bad-list-nonexistent'")
	assert.Contains(t, err.Error(), "key 'wrong-type' in list 'bad-list-wrong-type' must be a leaf")
}

func TestCompiler_CircularUses(t *testing.T) {
	t.Parallel()

	input := `
		module test-circular {
			namespace "urn:test";
			prefix "t";

			grouping A {
				uses B;
			}

			grouping B {
				uses A;
			}

			container item {
				uses A;
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency detected in uses")
}

func TestCompiler_MandatoryDefaultValidation(t *testing.T) {
	t.Parallel()

	input := `
		module test-mandatory {
			namespace "urn:test";
			prefix "t";

			leaf invalid-leaf {
				type string;
				mandatory true;
				default "hello";
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "leaf 'invalid-leaf' is mandatory and cannot have a default value")
}

func TestCompiler_MustAndPresence(t *testing.T) {
	t.Parallel()

	input := `
		module test-must-presence {
			namespace "urn:test";
			prefix "t";

			container my-container {
				presence "this container has a presence meaning";
				must "current() != ''";
				leaf my-leaf {
					type string;
					must "../my-container";
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	cont, ok := schemaMod.Nodes["my-container"].(*schema.Container)
	assert.True(t, ok)
	assert.NotNil(t, cont.Presence)
	assert.Equal(t, "this container has a presence meaning", *cont.Presence)
	assert.Equal(t, []string{"current() != ''"}, cont.Musts)

	leaf, ok := cont.GetChildren()["my-leaf"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Equal(t, []string{"../my-container"}, leaf.Musts)
}

func TestCompiler_AdvancedDataNodes(t *testing.T) {
	t.Parallel()

	input := `
		module test-adv {
			namespace "urn:test";
			prefix "t";

			choice my-choice {
				mandatory true;
				case explicit-case {
					leaf a { type string; }
				}
				leaf b { type int32; } // short-hand case 'b'
			}

			anyxml my-xml {
				mandatory true;
			}

			anydata my-data {
				mandatory false;
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	ch, ok := schemaMod.Nodes["my-choice"].(*schema.Choice)
	assert.True(t, ok)
	assert.True(t, ch.Mandatory)

	// "explicit-case"
	case1, ok := ch.GetChildren()["explicit-case"].(*schema.Case)
	assert.True(t, ok)
	_, hasA := case1.GetChildren()["a"].(*schema.Leaf)
	assert.True(t, hasA)

	// short-hand case "b"
	case2, ok := ch.GetChildren()["b"].(*schema.Case)
	assert.True(t, ok)
	_, hasB := case2.GetChildren()["b"].(*schema.Leaf)
	assert.True(t, hasB)

	ax, ok := schemaMod.Nodes["my-xml"].(*schema.AnyXML)
	assert.True(t, ok)
	assert.True(t, ax.Mandatory)

	ad, ok := schemaMod.Nodes["my-data"].(*schema.AnyData)
	assert.True(t, ok)
	assert.False(t, ad.Mandatory)
}

func TestCompiler_ProtocolNodes(t *testing.T) {
	t.Parallel()

	input := `
		module test-protocol {
			namespace "urn:test";
			prefix "t";

			rpc my-rpc {
				input {
					leaf a { type string; }
				}
				output {
					leaf b { type string; }
				}
			}

			notification my-notif {
				leaf x { type int32; }
			}

			container my-cont {
				action my-action {
					input {
						leaf z { type string; }
					}
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	rpc, ok := schemaMod.Nodes["my-rpc"].(*schema.RPC)
	assert.True(t, ok)
	assert.False(t, rpc.Config())

	in, ok := rpc.GetChildren()["input"].(*schema.Input)
	assert.True(t, ok)
	assert.False(t, in.Config())
	_, hasA := in.GetChildren()["a"].(*schema.Leaf)
	assert.True(t, hasA)

	out, ok := rpc.GetChildren()["output"].(*schema.Output)
	assert.True(t, ok)
	assert.False(t, out.Config())

	notif, ok := schemaMod.Nodes["my-notif"].(*schema.Notification)
	assert.True(t, ok)
	assert.False(t, notif.Config())

	cont, ok := schemaMod.Nodes["my-cont"].(*schema.Container)
	assert.True(t, ok)
	act, ok := cont.GetChildren()["my-action"].(*schema.Action)
	assert.True(t, ok)
	assert.False(t, act.Config())
}

func TestCompiler_Constraints(t *testing.T) {
	t.Parallel()

	input := `
		module test-constraints {
			namespace "urn:test";
			prefix "t";

			leaf a {
				type string;
				when "../b = 'true'";
			}

			list my-list {
				key "k1 k2";
				unique "u1 u2";
				unique "u3";
				min-elements 1;
				max-elements 10;
				ordered-by user;
				leaf k1 { type string; }
				leaf k2 { type string; }
				leaf u1 { type string; }
				leaf u2 { type string; }
				leaf u3 { type string; }
			}

			leaf-list my-leaf-list {
				type string;
				min-elements 2;
				max-elements unbounded;
				ordered-by system;
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	leafA, ok := schemaMod.Nodes["a"].(*schema.Leaf)
	assert.True(t, ok)
	assert.NotNil(t, leafA.When)
	assert.Equal(t, "../b = 'true'", *leafA.When)

	lst, ok := schemaMod.Nodes["my-list"].(*schema.List)
	assert.True(t, ok)
	assert.Equal(t, []string{"k1", "k2"}, lst.Keys)
	assert.Equal(t, [][]string{{"u1", "u2"}, {"u3"}}, lst.Unique)
	assert.NotNil(t, lst.MinElements)
	assert.Equal(t, uint32(1), *lst.MinElements)
	assert.NotNil(t, lst.MaxElements)
	assert.Equal(t, uint32(10), *lst.MaxElements)
	assert.NotNil(t, lst.OrderedBy)
	assert.Equal(t, "user", *lst.OrderedBy)

	ll, ok := schemaMod.Nodes["my-leaf-list"].(*schema.LeafList)
	assert.True(t, ok)
	assert.NotNil(t, ll.MinElements)
	assert.Equal(t, uint32(2), *ll.MinElements)
	assert.Nil(t, ll.MaxElements)
	assert.NotNil(t, ll.OrderedBy)
	assert.Equal(t, "system", *ll.OrderedBy)
}

func TestCompiler_Metadata(t *testing.T) {
	t.Parallel()

	input := `
		module test-meta {
			namespace "urn:test";
			prefix "t";

			feature my-feature {
				description "A test feature";
			}

			deviation /t:c {
				description "A test deviation";
			}

			container c {
				description "Container description";
				reference "RFC 1234";
				status deprecated;
				if-feature my-feature;

				leaf l {
					type int32;
					units "seconds";
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)

	assert.NoError(t, err)

	assert.Contains(t, schemaMod.Features, "my-feature")
	f := schemaMod.Features["my-feature"]
	assert.Equal(t, "A test feature", *f.Description)

	assert.Len(t, schemaMod.Deviations, 1)
	dev := schemaMod.Deviations[0]
	assert.Equal(t, "A test deviation", *dev.Description)

	c, ok := schemaMod.Nodes["c"].(*schema.Container)
	assert.True(t, ok)
	assert.Equal(t, "Container description", *c.Description)
	assert.Equal(t, "RFC 1234", *c.Reference)
	assert.Equal(t, "deprecated", *c.Status)
	assert.Equal(t, []string{"my-feature"}, c.IfFeatures)

	l, ok := c.GetChildren()["l"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Equal(t, "seconds", *l.Units)
}

func TestCompiler_UsesRefine(t *testing.T) {
	t.Parallel()

	inputLocal := `
		module test-local {
			namespace "urn:local";
			prefix "loc";

			grouping g1 {
				leaf a { type string; mandatory false; }
				container b {
					leaf c { type int32; }
				}
			}

			uses g1 {
				refine a {
					mandatory true;
					description "Refined!";
				}
				refine b/c {
					default "10";
				}
			}
		}
	`

	lex := lexer.New(inputLocal)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)
	assert.NoError(t, err)

	a, ok := schemaMod.Nodes["a"].(*schema.Leaf)
	assert.True(t, ok)
	assert.True(t, a.Mandatory) // should be refined to true
	assert.Equal(t, "Refined!", *a.Description)

	b, ok := schemaMod.Nodes["b"].(*schema.Container)
	assert.True(t, ok)
	c, ok := b.GetChildren()["c"].(*schema.Leaf)
	assert.True(t, ok)
	assert.NotNil(t, c.Default)
	assert.Equal(t, "10", *c.Default)
}

func TestCompiler_AdvancedTypes(t *testing.T) {
	t.Parallel()

	input := `
		module test-advanced-types {
			namespace "urn:test";
			prefix "t";

			identity id1;
			identity id2;

			leaf dec {
				type decimal64 {
					fraction-digits 2;
				}
			}

			leaf bid {
				type identityref {
					base id1;
					base id2;
				}
			}

			leaf mybits {
				type bits {
					bit b1;
					bit b2;
				}
			}

			leaf ref {
				type leafref {
					path "../a";
					require-instance false;
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)
	assert.NoError(t, err)

	dec, ok := schemaMod.Nodes["dec"].(*schema.Leaf)
	assert.True(t, ok)
	assert.NotNil(t, dec.Type.FractionDigits)
	assert.Equal(t, uint8(2), *dec.Type.FractionDigits)

	bid, ok := schemaMod.Nodes["bid"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Equal(t, []string{"id1", "id2"}, bid.Type.Bases)

	mybits, ok := schemaMod.Nodes["mybits"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Equal(t, []string{"b1", "b2"}, mybits.Type.Bits)

	ref, ok := schemaMod.Nodes["ref"].(*schema.Leaf)
	assert.True(t, ok)
	assert.NotNil(t, ref.Type.Path)
	assert.Equal(t, "../a", *ref.Type.Path)
	assert.NotNil(t, ref.Type.RequireInstance)
	assert.False(t, *ref.Type.RequireInstance)
}

func TestCompiler_IdentityrefValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Local Base",
			input: `
				module valid-id {
					namespace "urn:test";
					prefix "t";
					identity my-base;
					leaf id {
						type identityref {
							base my-base;
						}
					}
				}
			`,
			expectError: false,
		},
		{
			name: "Missing Local Base",
			input: `
				module missing-id {
					namespace "urn:test";
					prefix "t";
					leaf id {
						type identityref {
							base unknown-base;
						}
					}
				}
			`,
			expectError: true,
			errorMsg:    "invalid identityref base 'unknown-base' in id: identity not found locally",
		},
		{
			name: "No Base Statement",
			input: `
				module no-base {
					namespace "urn:test";
					prefix "t";
					leaf id {
						type identityref;
					}
				}
			`,
			expectError: true,
			errorMsg:    "identityref id must have at least one base statement",
		},
		{
			name: "Unknown Prefix",
			input: `
				module unknown-prefix {
					namespace "urn:test";
					prefix "t";
					leaf id {
						type identityref {
							base other:my-base;
						}
					}
				}
			`,
			expectError: true,
			errorMsg:    "invalid identityref base 'other:my-base' in id: unknown prefix other",
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lex := lexer.New(tt.input)
			p := parser.New(lex)
			astMod := p.ParseModule()

			comp := compiler.New(nil)
			_, err := comp.Compile(astMod)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompiler_FeaturePruning(t *testing.T) {
	t.Parallel()

	input := `
		module test-features {
			namespace "urn:test";
			prefix "t";

			feature f1;
			feature f2;

			leaf a {
				if-feature f1;
				type string;
			}

			leaf b {
				if-feature "not f1";
				type string;
			}

			leaf c {
				if-feature "f1 and f2";
				type string;
			}

			leaf d {
				if-feature "f1 or f2";
				type string;
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	// 1. Compile with f1 unsupported, f2 supported
	opts := &compiler.Options{
		SupportedFeatures: []string{"f2"},
	}
	comp := compiler.New(opts)
	schemaMod, err := comp.Compile(astMod)
	assert.NoError(t, err)

	_, hasA := schemaMod.Nodes["a"]
	assert.False(t, hasA, "a requires f1")

	_, hasB := schemaMod.Nodes["b"]
	assert.True(t, hasB, "b requires not f1")

	_, hasC := schemaMod.Nodes["c"]
	assert.False(t, hasC, "c requires f1 and f2")

	_, hasD := schemaMod.Nodes["d"]
	assert.True(t, hasD, "d requires f1 or f2")

	// 2. Compile with both supported
	optsBoth := &compiler.Options{
		SupportedFeatures: []string{"f1", "f2"},
	}
	compBoth := compiler.New(optsBoth)
	schemaBoth, err := compBoth.Compile(astMod)
	assert.NoError(t, err)

	_, hasA2 := schemaBoth.Nodes["a"]
	assert.True(t, hasA2)

	_, hasB2 := schemaBoth.Nodes["b"]
	assert.False(t, hasB2)

	_, hasC2 := schemaBoth.Nodes["c"]
	assert.True(t, hasC2)

	_, hasD2 := schemaBoth.Nodes["d"]
	assert.True(t, hasD2)
}

func TestCompiler_Deviations(t *testing.T) {
	t.Parallel()

	input := `
		module test-devs {
			namespace "urn:test";
			prefix "t";

			container base {
				leaf a { type string; mandatory true; }
				leaf b { type int32; default "5"; }
				leaf c { type string; }
				container sub {
					leaf d { type string; }
				}
			}

			deviation /t:base/t:a {
				deviate replace {
					mandatory false;
				}
			}

			deviation /t:base/t:b {
				deviate delete {
					default "5";
				}
			}

			deviation /t:base/t:c {
				deviate add {
					default "hello";
				}
			}

			deviation /t:base/t:sub {
				deviate not-supported;
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	schemaMod, err := comp.Compile(astMod)
	assert.NoError(t, err)

	base, ok := schemaMod.Nodes["base"].(*schema.Container)
	assert.True(t, ok)

	// a: mandatory true -> false
	a, ok := base.GetChildren()["a"].(*schema.Leaf)
	assert.True(t, ok)
	assert.False(t, a.Mandatory)

	// b: delete default
	b, ok := base.GetChildren()["b"].(*schema.Leaf)
	assert.True(t, ok)
	assert.Nil(t, b.Default)

	// c: add default
	c, ok := base.GetChildren()["c"].(*schema.Leaf)
	assert.True(t, ok)
	assert.NotNil(t, c.Default)
	assert.Equal(t, "hello", *c.Default)

	// sub: not-supported (pruned)
	_, hasSub := base.GetChildren()["sub"]
	assert.False(t, hasSub)
}

func TestCompiler_DefaultValues(t *testing.T) {
	t.Parallel()

	input := `
		module test-defaults {
			namespace "urn:test";
			prefix "t";

			leaf bad-bool {
				type boolean;
				default "True"; // should be "true"
			}

			leaf bad-int {
				type int32;
				default "12.5"; // should be integer
			}

			leaf bad-uint {
				type uint32;
				default "-5"; // should be unsigned
			}

			leaf bad-enum {
				type enumeration {
					enum "red";
					enum "blue";
				}
				default "green"; // should be red or blue
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid default 'True' for boolean leaf bad-bool")
	assert.Contains(t, err.Error(), "invalid default '12.5' for integer leaf bad-int")
	assert.Contains(t, err.Error(), "invalid default '-5' for unsigned integer leaf bad-uint")
	assert.Contains(t, err.Error(), "invalid default 'green' for enum leaf bad-enum")
}

func TestCompiler_XPathSyntax(t *testing.T) {
	t.Parallel()

	input := `
		module test-xpath {
			namespace "urn:test";
			prefix "t";

			leaf a {
				type string;
				when "../b = 'value"; // missing closing quote
			}

			leaf b {
				type string;
				must "current() = [1, 2"; // missing closing bracket
			}

			leaf c {
				type leafref {
					path "../d/e[f='val'"; // missing closing bracket
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatched quotes in when expression")
	assert.Contains(t, err.Error(), "mismatched brackets or parentheses in must expression")
	assert.Contains(t, err.Error(), "mismatched brackets or parentheses in path expression")
}

func TestCompiler_MalformedAugmentPath(t *testing.T) {
	t.Parallel()

	input := `
module test {
    namespace "urn:test"; prefix "t";
    augment "/nonexistent/path" {
        leaf x { type string; }
    }
}`
	_, err := compile(t, input)
	assert.Error(t, err, "augment to nonexistent path must return error")
}

func TestCompiler_MalformedRefinePath(t *testing.T) {
	t.Parallel()

	input := `
module test {
    namespace "urn:test"; prefix "t";
    grouping g { leaf x { type string; } }
    container c {
        uses g {
            refine "nonexistent-leaf" { mandatory true; }
        }
    }
}`
	_, err := compile(t, input)
	assert.Error(t, err, "refine to nonexistent path must return error")
}

func TestCompiler_CircularTypedef(t *testing.T) {
	t.Parallel()

	input := `
module test {
    namespace "urn:test"; prefix "t";
    typedef type-a { type type-b; }
    typedef type-b { type type-a; }
    leaf x { type type-a; }
}`
	_, err := compile(t, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular typedef")
}

func TestCompiler_UnresolvableAugment(t *testing.T) {
	t.Parallel()

	input := `
module test {
    namespace "urn:test"; prefix "t";
    augment "/does-not-exist/at-all" {
        leaf injected { type string; }
    }
}`
	_, err := compile(t, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "augment target not found")
}

func TestCompiler_DuplicateRPCInput(t *testing.T) {
	t.Parallel()

	input := `
module test {
    namespace "urn:test"; prefix "t";
    rpc do-thing {
        input { leaf x { type string; } }
        input { leaf y { type string; } }
    }
}`
	_, err := compile(t, input)
	assert.Error(t, err, "duplicate rpc input must return error")
}

func TestCompiler_NoDebugOutput(t *testing.T) {
	t.Parallel()

	runAndCapture := func(yangInput string) (string, string) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		oldStderr := os.Stderr
		re, we, _ := os.Pipe()
		os.Stderr = we

		lex := lexer.New(yangInput)
		p := parser.New(lex)
		astMod := p.ParseModule()
		comp := compiler.New(nil)
		_, _ = comp.Compile(astMod)

		w.Close()
		we.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr

		out, _ := io.ReadAll(r)
		errOut, _ := io.ReadAll(re)
		return string(out), string(errOut)
	}

	// Valid YANG
	validYANG := `module m { namespace "urn:m"; prefix "m"; }`
	out, errOut := runAndCapture(validYANG)
	assert.Empty(t, out, "stdout must be empty for valid YANG")
	assert.Empty(t, errOut, "stderr must be empty for valid YANG")

	// Malformed YANG (augment to nonexistent target triggers the previously-debug path)
	badYANG := `
module test {
    namespace "urn:test"; prefix "t";
    augment "/nonexistent/path" {
        leaf x { type string; }
    }
}`
	out2, errOut2 := runAndCapture(badYANG)
	assert.Empty(t, out2, "stdout must be empty for invalid YANG")
	assert.Empty(t, errOut2, "stderr must be empty for invalid YANG")
}

func TestCompiler_MaxErrors(t *testing.T) {
	t.Parallel()

	// Build a YANG list with 110 nonexistent keys to generate >100 errors
	var keys []string
	for i := 0; i < 110; i++ {
		keys = append(keys, fmt.Sprintf("k%d", i))
	}
	input := fmt.Sprintf(`
module test {
    namespace "urn:test"; prefix "t";
    list items {
        key "%s";
    }
}`, strings.Join(keys, " "))
	_, err := compile(t, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation stopped")
	// Count lines — should be at most ~101 (100 errors + 1 truncation)
	lines := strings.Split(err.Error(), "\n")
	assert.LessOrEqual(t, len(lines), 103, "error output should be bounded near MaxErrors")
}
