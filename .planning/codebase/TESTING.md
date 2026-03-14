# Testing Patterns

**Analysis Date:** 2026-03-14

## Test Framework

**Runner:**
- Go's standard `testing` package (Go 1.25.5)
- No external test runner (not using ginkgo, testify runner)

**Assertion Library:**
- github.com/stretchr/testify v1.11.1
- Functions: `assert.NoError()`, `assert.Error()`, `assert.Equal()`, `assert.Len()`, `assert.Contains()`, `assert.True()`, `assert.False()`, `assert.Nil()`, `assert.NotNil()`
- Import as: `"github.com/stretchr/testify/assert"`

**Run Commands:**
```bash
go test ./...              # Run all tests
go test -v ./...           # Run with verbose output
go test -run TestName ./...  # Run specific test
go test -count=1 ./...     # Run without cache
go test -timeout 30s ./... # Set timeout
```

**Coverage:**
- No enforced coverage requirement detected
- Run with: `go test -cover ./...`

## Test File Organization

**Location:**
- Co-located with source code in same package
- Pattern: `<name>_test.go` in same directory as `<name>.go`
- Examples:
  - `parser/parser_test.go` tests `parser/parser.go`
  - `parser/lexer/lexer_test.go` tests `parser/lexer/lexer.go`
  - `compiler/compiler_test.go` tests `compiler/compiler.go`

**Naming:**
- Test files: `_test.go` suffix
- Test functions: `TestName(t *testing.T)` format
- Sub-tests: `t.Run("subtest name", func(t *testing.T) {...})`
- Test packages: `<name>_test` (different from implementation package)
- Examples: `parser_test`, `lexer_test`, `compiler_test`, `golang_test`

**Structure:**
```
parser/
├── parser.go
├── parser_test.go
└── lexer/
    ├── lexer.go
    └── lexer_test.go
```

## Test Structure

**Suite Organization:**

Test functions use table-driven testing pattern for multiple cases:

```go
func TestCompiler_IdentifierUniqueness(t *testing.T) {
	t.Parallel()

	input := `...`

	lex := lexer.New(input)
	p := parser.New(lex)
	astMod := p.ParseModule()

	comp := compiler.New(nil)
	_, err := comp.Compile(astMod)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate identifier 'id'")
}
```

More complex example with test cases:

```go
func TestCompiler_IdentifierrefValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Local Base",
			input: `...`,
			expectError: false,
		},
		{
			name: "Missing Local Base",
			input: `...`,
			expectError: true,
			errorMsg: "invalid identityref base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// test implementation
		})
	}
}
```

**Patterns:**

- **t.Parallel():** Called at start of every test function for concurrent execution
- **Setup:** Create test input (inline string) → create lexer → create parser → parse
- **Assertions:** Inline assertions after each operation
- **Error testing:** Both positive (NoError) and negative (Error, Contains) paths tested

## Test Types

**Unit Tests:**
- Scope: Individual functions/methods
- Coverage: Core logic, error conditions, edge cases
- Examples:
  - `TestParseModule`: Tests module parsing
  - `TestCompiler_Compile`: Tests compilation
  - `TestLexer_NextToken`: Tests lexical analysis

**Integration Tests:**
- Scope: Multiple components together (parser → compiler → generator)
- Pattern: Parse YANG input → compile to schema → generate code
- Example: `TestGoGenerator_Generate` tests full pipeline
- Located in: `generator/golang/generator_test.go`, `generator/protobuf/generator_test.go`

**Validation Tests:**
- Scope: Error conditions and validation rules
- Pattern: Test invalid input produces expected errors
- Examples:
  - `TestCompiler_Validation`: List without keys
  - `TestCompiler_InvalidTypeRestrictions`: Type mismatches
  - `TestCompiler_DefaultValues`: Invalid default values

## Test Data and Fixtures

**Test Data:**
- Inline YANG source as raw strings in test functions
- No separate fixtures directory
- Pattern: Multi-line string literals for complex YANG modules
- Example:
  ```go
  input := `
    module test-module {
      namespace "urn:test";
      prefix "t";
      // ... YANG content
    }
  `
  ```

**Factories:**
- No explicit factory pattern
- Inline creation using constructor functions: `lexer.New(input)`, `parser.New(lex)`
- Common setup pattern repeated in tests (not extracted to helper)

## Mocking

**Framework:**
- Mockery (v0.0.0, specified in .mockery.yaml)
- Configuration: `.mockery.yaml` with in-package generation

**Mock Configuration:**

From `.mockery.yaml`:
```yaml
with-expecter: true
packages:
  github.com/gotya/gotya/...:
    config:
      dir: "{{.InterfaceDir}}"
      mockname: "Mock{{.InterfaceName}}"
      outpkg: "{{.PackageName}}"
      filename: "mock_{{.InterfaceName | lower}}.go"
      inpackage: true
```

**Patterns:**
- Mockery generates mocks in-package
- Mock naming: `Mock<InterfaceName>` (e.g., `MockModuleLoader`)
- Expectation-based assertions with `.On()` chains

**What to Mock:**
- External interfaces: `ModuleLoader` (interfaces for dependency injection)
- File I/O dependencies
- External service calls

**What NOT to Mock:**
- Internal structs (parse results, AST nodes)
- Standard library functions (use real ones)
- Testing package itself (use real testing.T)

## Common Patterns

**Async Testing:**
Not used in this codebase (no goroutines in tests)

**Error Testing:**
```go
// Negative case
assert.Error(t, err)
assert.Contains(t, err.Error(), "expected error message substring")

// Positive case
assert.NoError(t, err)
assert.NotNil(t, result)
```

**Type Assertions in Tests:**
```go
cont, ok := schemaMod.Nodes["interfaces"].(*schema.Container)
assert.True(t, ok)  // Verify the cast succeeded
assert.Len(t, cont.GetChildren(), 1)
```

**Assertion Ordering:**
1. Check operation succeeded (NoError, NotNil)
2. Verify return values match expectations
3. Verify nested/derived values

## Test Examples from Codebase

**Parser Test** (`parser/parser_test.go`):
- Tests lexical → AST conversion
- Validates keyword recognition, arguments, nested structures
- Uses string constants for YANG source

**Compiler Test** (`compiler/compiler_test.go`):
- 30+ test functions covering:
  - Basic compilation (containers, lists, leaves)
  - Grouping resolution and circular dependency detection
  - Type validation and restrictions
  - Augment processing
  - Features and deviations
  - Complex data structures (choices, cases, RPC, notifications)
- Each test follows: Parse → Compile → Assert
- Comprehensive error case coverage

**Generator Test** (`generator/golang/generator_test.go`):
- Tests YANG → Go struct generation
- Validates struct tags, field types, method generation
- Uses string matching on generated output

---

*Testing analysis: 2026-03-14*
