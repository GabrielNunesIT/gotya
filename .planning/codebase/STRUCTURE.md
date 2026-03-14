# Codebase Structure

**Analysis Date:** 2026-03-14

## Directory Layout

```
gotya/
├── cmd/                          # CLI tools
│   └── gotya/                   # Main CLI tool for parsing and code generation
├── parser/                      # Lexical analysis and parsing
│   ├── lexer/                   # Tokenization (lexical analyzer)
│   └── parser.go                # Recursive descent parser for YANG statements
├── ast/                         # Abstract Syntax Tree definitions
│   └── ast.go                   # Node and Statement interfaces, BaseNode
├── token/                       # Token type definitions
│   └── token.go                 # Token types, constants, Position struct
├── compiler/                    # Semantic analysis and compilation
│   ├── compiler.go              # Main compiler orchestrator
│   └── validator.go             # Validation logic during compilation
├── schema/                      # Compiled semantic schema definitions
│   └── schema.go                # Node interface, BaseNode, Module, and all node types (Leaf, Container, List, etc.)
├── generator/                   # Code generation abstraction layer
│   ├── generator.go             # Generator interface definition
│   ├── golang/                  # Go code generator
│   │   ├── generator.go         # GoGenerator implementation
│   │   └── device.go            # Device struct generation logic
│   └── protobuf/                # Protobuf code generator
│       ├── generator.go         # ProtobufGenerator implementation
│       ├── device.go            # Proto message generation logic
│       └── validations.go       # CEL validation emission
├── codec/                       # Data encoding/decoding
│   └── rfc7951/                 # RFC 7951 JSON codec for YANG data
│       ├── decoder.go           # JSON → Go struct deserialization
│       └── encoder.go           # Go struct → JSON serialization
├── gotya.go                     # Public library API (Parse, ParseFile, Compile)
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency checksums
├── README.md                    # Project documentation and quick start
├── .golangci.yml                # Linting configuration
├── test/                        # Test assets and data
│   └── assets/yangs/            # Real YANG module files for testing
└── docs/                        # Documentation and specification files
```

## Directory Purposes

**cmd/ (CLI Tools):**
- Purpose: Entry point for command-line usage
- Contains: Main programs and their utilities
- Key files: `cmd/gotya/main.go` (CLI flag parsing, generator invocation), `cmd/gotya/loader.go` (DirectoryLoader implementation)

**parser/ (Parsing Pipeline):**
- Purpose: Lexical and syntactic analysis of YANG source
- Contains: Lexer, parser, and token definitions
- Key files: `parser/parser.go` (recursive descent parser), `parser/lexer/lexer.go` (tokenizer)

**ast/ (Abstract Syntax Tree):**
- Purpose: AST node structure definitions
- Contains: Node interface, Statement interface, BaseNode, Module
- Key files: `ast/ast.go`

**token/ (Token Definitions):**
- Purpose: Token type constants and position tracking
- Contains: Token type enum, Position struct, Token struct
- Key files: `token/token.go`

**compiler/ (Semantic Analysis):**
- Purpose: Transform AST into semantic schema, resolve references, validate
- Contains: Compiler orchestrator, validation logic
- Key files: `compiler/compiler.go`, `compiler/validator.go`

**schema/ (Compiled Schema Representation):**
- Purpose: Semantic data tree node definitions
- Contains: Node interface, BaseNode, Module struct, all typed nodes (Leaf, LeafList, Container, List, Choice, Case, AnyXML, AnyData, RPC, Notification, Augment, Refine, etc.)
- Key files: `schema/schema.go`

**generator/ (Code Generation):**
- Purpose: Abstract code generation and concrete implementations
- Contains: Generator interface, Go generator, Protobuf generator
- Key files: `generator/generator.go` (interface), `generator/golang/generator.go`, `generator/protobuf/generator.go`

**codec/ (Data Serialization):**
- Purpose: RFC 7951 JSON encoding/decoding for generated Go structures
- Contains: Decoder and encoder for module-prefixed JSON with YANG semantics
- Key files: `codec/rfc7951/decoder.go`, `codec/rfc7951/encoder.go`

**test/ (Test Assets):**
- Purpose: YANG module files and other test data for integration testing
- Contains: Real YANG modules (ALB, IETF modules, etc.)
- Generated: No, committed test fixtures

## Key File Locations

**Entry Points:**
- `gotya.go`: Public library API - Parse(), ParseFile(), Compile()
- `cmd/gotya/main.go`: CLI tool entry point with flag parsing and generator dispatch
- `parser/lexer/lexer.go`: Lexical analysis starting point
- `parser/parser.go`: Syntactic analysis starting point

**Configuration:**
- `.golangci.yml`: Linting rules and settings
- `go.mod`: Module definition and dependencies

**Core Logic:**
- `parser/parser.go`: YANG statement parsing (lines 1-200+ for parseStatement)
- `compiler/compiler.go`: AST-to-schema transformation (lines 1-300+ for Compile method)
- `schema/schema.go`: Node definitions and composite structure
- `generator/golang/generator.go`: Go struct generation strategy
- `generator/protobuf/generator.go`: Protobuf message generation strategy

**Testing:**
- `parser/parser_test.go`: Parser unit tests
- `parser/lexer/lexer_test.go`: Lexer unit tests
- `compiler/compiler_test.go`: Compiler integration tests
- `generator/golang/generator_test.go`: Go code generation tests
- `generator/golang/device_test.go`: Device struct generation tests
- `generator/protobuf/generator_test.go`: Protobuf generation tests
- `generator/protobuf/device_test.go`: Protobuf device generation tests
- `codec/rfc7951/codec_test.go`: JSON codec tests
- `test/assets/yangs/`: Real YANG module fixtures

## Naming Conventions

**Files:**
- Implementation files: `{component}.go` (e.g., `parser.go`, `compiler.go`, `lexer.go`)
- Test files: `{component}_test.go` (e.g., `parser_test.go`, `compiler_test.go`)
- Sub-package files in nested dirs: Same pattern (e.g., `generator/golang/generator.go`)

**Directories:**
- Package names match directory names (Go convention)
- Nested packages use descriptive names (e.g., `parser/lexer`, `generator/golang`, `codec/rfc7951`)
- CLI tool lives in `cmd/{binary_name}` (e.g., `cmd/gotya`)

**Go Types & Interfaces:**
- PascalCase for exported types (e.g., `Parser`, `Compiler`, `Module`, `Node`)
- camelCase for unexported types (e.g., `baseNode`)
- Interface names: Usually nouns ending in "-er" or "-or" (e.g., `Generator`, `ModuleLoader`)

**Functions & Methods:**
- PascalCase for exported functions (e.g., `Parse`, `Compile`, `New`)
- camelCase for unexported methods (e.g., `parseStatement`, `nextToken`)

## Where to Add New Code

**New Feature (YANG language support):**
- Primary code: Extend `parser/parser.go` for new statement types, add validation in `compiler/compiler.go`
- Tests: Add to `parser/parser_test.go` or `compiler/compiler_test.go`
- Test data: Add YANG files to `test/assets/yangs/`

**New Generator Target (e.g., TypeScript):**
- Implementation: Create new package `generator/typescript/generator.go` implementing the `Generator` interface
- CLI support: Add case in `cmd/gotya/main.go` switch statement to invoke new generator
- Tests: Create `generator/typescript/generator_test.go`

**New Code Generation Feature (e.g., validation methods):**
- Location: `generator/{language}/generator.go` or separate file like `generator/{language}/validations.go`
- Pattern: Add option to `Options` struct, implement in Generate/GenerateDevice methods, follow existing patterns

**New Validation Rule:**
- Location: `compiler/validator.go` or add helper in `compiler/compiler.go`
- Pattern: Call during Compile() pass, accumulate errors in Compiler.errors slice

**New Codec Format (e.g., XML):**
- Location: Create `codec/{format}/encoder.go` and `codec/{format}/decoder.go`
- Tests: Create `codec/{format}/codec_test.go`
- Pattern: Follow same interface pattern as rfc7951 (Encode/Decode functions)

**Shared Utilities:**
- Location: Create utility package in root if cross-package, or in relevant package if localized
- Pattern: Keep utilities focused and testable

## Special Directories

**cmd/gotya/:**
- Purpose: Command-line tool binary
- Generated: No (source code)
- Committed: Yes
- Note: Contains both main.go and loader.go; DirectoryLoader is CLI-specific implementation of compiler.ModuleLoader interface

**test/assets/yangs/:**
- Purpose: Real YANG module files for integration testing
- Generated: No (manually curated fixtures)
- Committed: Yes
- Note: Contains complete YANG modules including ALB (Adaptive Load Balancer) and IETF standard modules for realistic testing

**parser/lexer/:**
- Purpose: Lexical analysis package nested inside parser
- Generated: No
- Committed: Yes
- Note: Separated from recursive descent parser because lexical analysis is self-contained concern

**codec/rfc7951/:**
- Purpose: RFC 7951 JSON codec implementation
- Generated: No
- Committed: Yes
- Note: Uses reflection to generically handle any generated Go struct, decoupled from generation logic

**generator/{golang,protobuf}/:**
- Purpose: Language-specific code generation implementations
- Generated: No (source code)
- Committed: Yes
- Note: Separate packages for clean separation; both implement same Generator interface

---

*Structure analysis: 2026-03-14*
