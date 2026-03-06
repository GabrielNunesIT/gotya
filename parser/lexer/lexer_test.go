package lexer_test

import (
	"testing"

	"github.com/gotya/gotya/parser/lexer"
	"github.com/gotya/gotya/token"
	"github.com/stretchr/testify/assert"
)

func TestLexer_NextToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []token.Token
	}{
		{
			name:  "Basic statements and blocks",
			input: "module my-module { namespace \"urn:sys\"; }",
			expected: []token.Token{
				{Type: token.IDENTIFIER, Literal: "module", Pos: token.Position{Line: 1, Column: 1}},
				{Type: token.IDENTIFIER, Literal: "my-module", Pos: token.Position{Line: 1, Column: 8}},
				{Type: token.LBRACE, Literal: "{", Pos: token.Position{Line: 1, Column: 18}},
				{Type: token.IDENTIFIER, Literal: "namespace", Pos: token.Position{Line: 1, Column: 20}},
				{Type: token.STRING, Literal: "urn:sys", Pos: token.Position{Line: 1, Column: 30}},
				{Type: token.SEMI, Literal: ";", Pos: token.Position{Line: 1, Column: 39}},
				{Type: token.RBRACE, Literal: "}", Pos: token.Position{Line: 1, Column: 41}},
				{Type: token.EOF, Literal: "", Pos: token.Position{Line: 1, Column: 42}},
			},
		},
		{
			name: "Comments and mixed quotes",
			input: `
				// single line comment
				prefix 'my-pre'; /*
					multi line 
					comment 
				*/
				revision "2026-01-01" {
					description "Fix \"bug\" \n and \t other things";
				}
			`,
			expected: []token.Token{
				{Type: token.IDENTIFIER, Literal: "prefix", Pos: token.Position{Line: 3, Column: 5}},
				{Type: token.STRING, Literal: "my-pre", Pos: token.Position{Line: 3, Column: 12}},
				{Type: token.SEMI, Literal: ";", Pos: token.Position{Line: 3, Column: 20}},
				{Type: token.IDENTIFIER, Literal: "revision", Pos: token.Position{Line: 7, Column: 5}},
				{Type: token.STRING, Literal: "2026-01-01", Pos: token.Position{Line: 7, Column: 14}},
				{Type: token.LBRACE, Literal: "{", Pos: token.Position{Line: 7, Column: 27}},
				{Type: token.IDENTIFIER, Literal: "description", Pos: token.Position{Line: 8, Column: 6}},
				{Type: token.STRING, Literal: "Fix \"bug\" \n and \t other things", Pos: token.Position{Line: 8, Column: 18}},
				{Type: token.SEMI, Literal: ";", Pos: token.Position{Line: 8, Column: 54}},
				{Type: token.RBRACE, Literal: "}", Pos: token.Position{Line: 9, Column: 5}},
				{Type: token.EOF, Literal: "", Pos: token.Position{Line: 10, Column: 4}},
			},
		},
		{
			name:  "String concatenation and unquoted strings with slashes",
			input: `description "abc" + "def"; path /sys:node/sys:child;`,
			expected: []token.Token{
				{Type: token.IDENTIFIER, Literal: "description", Pos: token.Position{Line: 1, Column: 1}},
				{Type: token.STRING, Literal: "abc", Pos: token.Position{Line: 1, Column: 13}},
				{Type: token.PLUS, Literal: "+", Pos: token.Position{Line: 1, Column: 19}},
				{Type: token.STRING, Literal: "def", Pos: token.Position{Line: 1, Column: 21}},
				{Type: token.SEMI, Literal: ";", Pos: token.Position{Line: 1, Column: 26}},
				{Type: token.IDENTIFIER, Literal: "path", Pos: token.Position{Line: 1, Column: 28}},
				{Type: token.IDENTIFIER, Literal: "/sys:node/sys:child", Pos: token.Position{Line: 1, Column: 33}},
				{Type: token.SEMI, Literal: ";", Pos: token.Position{Line: 1, Column: 52}},
				{Type: token.EOF, Literal: "", Pos: token.Position{Line: 1, Column: 53}},
			},
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lex := lexer.New(tt.input)
			var tokens []token.Token
			for {
				tok := lex.NextToken()
				tokens = append(tokens, tok)
				if tok.Type == token.EOF {
					break
				}
			}

			// Validate length
			if len(tokens) != len(tt.expected) {
				t.Fatalf("Length mismatch. expected %d, got %d. Tokens: %v", len(tt.expected), len(tokens), tokens)
			}

			// Validate tokens
			for idx, exp := range tt.expected {
				assert.Equal(t, exp.Type, tokens[idx].Type, "Token type mismatch at index %d", idx)
				assert.Equal(t, exp.Literal, tokens[idx].Literal, "Token literal mismatch at index %d", idx)
				assert.Equal(t, exp.Pos.Line, tokens[idx].Pos.Line, "Token line mismatch at index %d", idx)
				assert.Equal(t, exp.Pos.Column, tokens[idx].Pos.Column, "Token col mismatch at index %d", idx)
			}
		})
	}
}
