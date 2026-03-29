package parser_test

import (
	"testing"

	"github.com/GabrielNunesIT/gotya/parser"
	"github.com/GabrielNunesIT/gotya/parser/lexer"
	"github.com/stretchr/testify/assert"
)

func TestParseModule(t *testing.T) {
	t.Parallel()

	input := `
		module my-module {
			namespace "urn:sys";
			prefix 'sys' + '-pre';

			revision "2026-01-01" {
				description "Initial";
			}

			container system {
				leaf host {
					type string;
				}
			}
		}
	`

	lex := lexer.New(input)
	p := parser.New(lex)
	module := p.ParseModule()

	assert.Empty(t, p.Errors(), "Parser should not have any errors")
	assert.NotNil(t, module, "Module should not be nil")

	assert.Equal(t, "module", module.Keyword())
	assert.Equal(t, "my-module", module.Argument())

	stmts := module.SubStatements()
	assert.Len(t, stmts, 4, "Expected 4 top-level statements: namespace, prefix, revision, container")

	// Verify namespace
	assert.Equal(t, "namespace", stmts[0].Keyword())
	assert.Equal(t, "urn:sys", stmts[0].Argument())

	// Verify prefix
	assert.Equal(t, "prefix", stmts[1].Keyword())
	assert.Equal(t, "sys-pre", stmts[1].Argument())

	// Verify revision
	assert.Equal(t, "revision", stmts[2].Keyword())
	assert.Equal(t, "2026-01-01", stmts[2].Argument())
	assert.Len(t, stmts[2].SubStatements(), 1)
	assert.Equal(t, "description", stmts[2].SubStatements()[0].Keyword())

	// Verify container
	assert.Equal(t, "container", stmts[3].Keyword())
	assert.Equal(t, "system", stmts[3].Argument())

	containerStmts := stmts[3].SubStatements()
	assert.Len(t, containerStmts, 1)
	assert.Equal(t, "leaf", containerStmts[0].Keyword())
	assert.Equal(t, "host", containerStmts[0].Argument())

	leafStmts := containerStmts[0].SubStatements()
	assert.Len(t, leafStmts, 1)
	assert.Equal(t, "type", leafStmts[0].Keyword())
	assert.Equal(t, "string", leafStmts[0].Argument())
}
