# Coding Conventions

**Analysis Date:** 2026-03-14

## Naming Patterns

**Files:**
- Go files follow standard package structure: `<package>/<file>.go`
- Test files: `<name>_test.go` (located in same package as code under test)
- Examples: `parser.go`, `parser_test.go`, `lexer.go`, `lexer_test.go`

**Functions:**
- Exported functions: PascalCase (e.g., `Parse`, `ParseFile`, `Compile`, `New`, `Errors`)
- Unexported functions: camelCase (e.g., `nextToken`, `parseStatement`, `addError`, `recoverStatement`)
- Constructor functions named `New` (e.g., `New(lex *lexer.Lexer) *Parser`)
- Receiver methods follow function naming rules above

**Variables:**
- Package-level receiver names: short, typically 1 letter (e.g., `p *Parser`, `l *Lexer`, `c *Compiler`, `g *GoGenerator`)
- Local variables: camelCase (e.g., `curToken`, `peekToken`, `astMod`, `schemaMod`)
- Constants: UPPER_CASE with underscores for exported, camelCase for unexported
- Interface implementations use standard Go naming (no suffix)

**Types:**
- Exported structs: PascalCase (e.g., `Parser`, `Lexer`, `Compiler`, `Module`)
- Exported interfaces: PascalCase ending with `er` or noun form (e.g., `ModuleLoader`, `Node`, `Statement`)
- Field names: PascalCase when exported (e.g., `Key`, `Arg`, `TokenL`, `SubStmts`)

## Code Style

**Formatting:**
- goimports: Used for import organization (configured in .golangci.yml)
- Follows standard Go formatting rules (enforced by gofmt)
- Line length: No strict limit, but readability prioritized

**Linting:**
- golangci-lint v2 with custom configuration in `.golangci.yml`
- Key enabled linters:
  - errcheck: Requires checking error returns
  - govet: Detects vet errors including variable shadowing
  - staticcheck, unused: Code quality checks
  - errname, wrapcheck: Error handling standards
  - revive: Code style enforcement

**Comments:**
- Package-level comments: Describe purpose of entire package, e.g., `// Package parser implements a recursive descent parser for YANG 1.0 and 1.1 modules.`
- Function/method comments: Describe purpose and behavior before the function, e.g., `// Parse takes a raw YANG string and returns an interpreted AST module.`
- Exported symbols must have doc comments
- Comments explain WHY, not WHAT (what is clear from code)
- Special pattern: Include rationale for design decisions in doc comments (see examples in `gotya.go` with "Why:" explanations)
- Inline comments: Clarify non-obvious logic (e.g., `// Read two tokens, so curToken and peekToken are both set.`)
- Directive comments: `//nolint:` used selectively (e.g., `//nolint:revive // package name matches purpose`)

**JSDoc/Godoc:**
- Follows Go doc comment conventions
- First sentence is summary (appears in godoc listing)
- Additional details follow after blank line
- Deprecated symbols can be marked with `// Deprecated:` prefix

## Import Organization

**Order:**
1. Standard library imports (fmt, os, etc.)
2. Third-party imports (github.com/...)
3. Local package imports (relative to module path)
4. Organized with blank lines separating groups

**Example:**
```go
import (
	"fmt"
	"os"

	"github.com/stretchr/testify/assert"

	"github.com/gotya/gotya/ast"
	"github.com/gotya/gotya/compiler"
)
```

**Path Aliases:**
- Not commonly used; full import paths preferred
- When needed, use meaningful aliases matching package name (e.g., `import lex "github.com/gotya/gotya/parser/lexer"`)

## Error Handling

**Patterns:**
- Errors returned as final return value: `func Foo() (result, error)`
- Early returns on error: Check error immediately after function call
- Error wrapping with context: Use `fmt.Errorf("operation context: %w", err)` pattern
- Example from codebase:
  ```go
  if err != nil {
    return nil, fmt.Errorf("compile module %s: %w", astMod.Argument(), err)
  }
  ```
- Errors collected during parsing: Parser stores errors in slice (`p.errors []string`), accessible via `.Errors()` method
- Nil checks before operations: Explicit nil checks guard against invalid input
  ```go
  if astMod == nil {
    return nil, errors.New("ast module cannot be nil")
  }
  ```

## Function Design

**Size:**
- Relatively compact functions (most 5-40 lines)
- Larger functions extract helper methods (e.g., `generateNode`, `compileDataNode`)

**Parameters:**
- Receiver-based: Methods use receiver for state (e.g., `(p *Parser)`)
- Dependency injection: Constructor takes dependencies (e.g., `New(lex *lexer.Lexer)`)
- Options structs for multiple configuration: `Options` struct with fields, passed as pointer
  ```go
  type Options struct {
    Loader            ModuleLoader
    SupportedFeatures []string
  }
  ```

**Return Values:**
- Standard: (value, error) pair
- Multiple return types: Explicitly named in code, not in signature
- Early returns on error

## Module Design

**Exports:**
- Explicit control of what's exported
- Public interfaces: `ModuleLoader`, `Node`, `Statement`
- Public structs: `Parser`, `Compiler`, `Module`, etc.
- Private helpers: `nextToken()`, `parseStatement()`, `compileDataNode()`

**Barrel Files:**
- Type aliases at package level to re-export from sub-packages (e.g., `type ASTModule = ast.Module` in `gotya.go`)
- Package facade: `gotya.go` provides public API wrapping internal packages

**Initialization:**
- No `init()` functions in codebase
- Constructor pattern preferred: `New()` functions with dependency injection

---

*Convention analysis: 2026-03-14*
