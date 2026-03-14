# Architecture

**Analysis Date:** 2026-03-14

## Pattern Overview

**Overall:** Multi-stage compiler architecture with lexer → parser → AST → compiler → schema pipeline, followed by pluggable code generators.

**Key Characteristics:**
- Clean separation of concerns across distinct compilation stages
- Pluggable generator interface for multiple output formats (Go, Protobuf)
- Interface-based design for extensibility (ModuleLoader, Generator)
- Public API facade (`gotya.go`) hiding internal pipeline complexity
- CLI tool for standalone usage separate from library API

## Layers

**Lexical Analysis Layer:**
- Purpose: Tokenize raw YANG source input into discrete tokens
- Location: `github.com/gotya/gotya/parser/lexer` (`parser/lexer/lexer.go`)
- Contains: Lexer state machine that reads YANG syntax character-by-character
- Depends on: `token` package for token type definitions
- Used by: Parser layer

**Parsing Layer:**
- Purpose: Convert token stream into Abstract Syntax Tree (AST) structure
- Location: `github.com/gotya/gotya/parser` (`parser/parser.go`)
- Contains: Recursive descent parser implementing YANG 1.0/1.1 statement grammar
- Depends on: Lexer, token, ast packages
- Used by: Compiler layer, public API

**Token Definition Layer:**
- Purpose: Define lexical token types and position tracking
- Location: `github.com/gotya/gotya/token` (`token/token.go`)
- Contains: Token type constants (IDENTIFIER, STRING, LBRACE, RBRACE, SEMI, etc.), Position struct
- Depends on: None (foundational)
- Used by: Lexer, Parser

**AST Layer:**
- Purpose: Define the Abstract Syntax Tree node structure
- Location: `github.com/gotya/gotya/ast` (`ast/ast.go`)
- Contains: Node interface, Statement interface, BaseNode struct, Module struct
- Depends on: None (foundational)
- Used by: Parser, Compiler, public API

**Compilation Layer:**
- Purpose: Transform AST into semantic schema with type resolution, validation, and reference linking
- Location: `github.com/gotya/gotya/compiler` (`compiler/compiler.go`, `compiler/validator.go`)
- Contains: Compiler orchestrator, ModuleLoader interface, validation logic
- Depends on: AST, schema packages
- Used by: Public API, CLI, code generators

**Schema Layer:**
- Purpose: Define the compiled semantic data tree representation
- Location: `github.com/gotya/gotya/schema` (`schema/schema.go`)
- Contains: Node interface, BaseNode, Module struct, typed node definitions (Leaf, LeafList, Container, List, etc.)
- Depends on: AST (for storing raw groupings/typedefs)
- Used by: Compiler, generators, codec

**Generator Interface Layer:**
- Purpose: Abstract code generation across multiple target formats
- Location: `github.com/gotya/gotya/generator` (`generator/generator.go`)
- Contains: Generator interface with Generate() and GenerateDevice() methods
- Depends on: schema package
- Used by: Concrete generators (Go, Protobuf), CLI

**Go Generator Layer:**
- Purpose: Emit Go source code from compiled YANG schema
- Location: `github.com/gotya/gotya/generator/golang` (`generator/golang/generator.go`, `generator/golang/device.go`)
- Contains: GoGenerator implementation, Options config
- Depends on: schema, generator packages
- Used by: CLI tool

**Protobuf Generator Layer:**
- Purpose: Emit Protobuf .proto files from compiled YANG schema
- Location: `github.com/gotya/gotya/generator/protobuf` (`generator/protobuf/generator.go`, `generator/protobuf/device.go`, `generator/protobuf/validations.go`)
- Contains: ProtobufGenerator implementation, CEL validation support, Options config
- Depends on: schema, generator packages
- Used by: CLI tool

**Codec Layer:**
- Purpose: Provide RFC 7951-compliant JSON encoding/decoding for generated Go structs
- Location: `github.com/gotya/gotya/codec/rfc7951` (`codec/rfc7951/decoder.go`, `codec/rfc7951/encoder.go`)
- Contains: Decode/Encode functions handling module-prefixed JSON, string-encoded integers, empty types
- Depends on: Reflection, schema knowledge
- Used by: Applications consuming generated code

**Public API Facade:**
- Purpose: Simple string/file-based entry points hiding internal pipeline
- Location: `github.com/gotya/gotya` (`gotya.go`)
- Contains: Parse(), ParseFile(), Compile() functions
- Depends on: Lexer, Parser, Compiler packages
- Used by: Library consumers, CLI tool

**CLI Tool Layer:**
- Purpose: Standalone command-line interface for YANG parsing, compilation, and code generation
- Location: `github.com/gotya/gotya/cmd/gotya` (`cmd/gotya/main.go`, `cmd/gotya/loader.go`)
- Contains: DirectoryLoader implementation (ModuleLoader interface), flag parsing, generator invocation
- Depends on: Public API, all generator packages
- Used by: End users via binary invocation

## Data Flow

**String-based Parsing:**

1. User calls `gotya.Parse(yangString)`
2. Lexer tokenizes string → token stream
3. Parser builds AST from tokens → *ast.Module
4. Returns AST to caller or as input to compilation

**File-based Parsing:**

1. User calls `gotya.ParseFile(path)`
2. Read file from filesystem → string content
3. Pass to Parse() → same as string-based flow

**Compilation with Dependency Resolution:**

1. User calls `gotya.Compile([]*astModules, opts)`
2. Compiler created with optional ModuleLoader
3. For each AST module:
   - Extract namespace, prefix, features, identities
   - Resolve imports/includes via loader (if provided)
   - Process typedefs, groupings, augments
   - Build semantic schema tree by transforming AST nodes
   - Link cross-module references
4. Return []*schema.Module

**CLI Processing Pipeline:**

1. CLI parses flags and input YANG file arguments
2. DirectoryLoader created with search paths
3. Load all input files:
   - ParseFile() each → ast.Module
   - Register in loader cache
4. Compile each AST module:
   - Loader resolves import/include references
   - Compiler builds schema tree
   - Cache compiled schema
5. Invoke appropriate generator (Go or Protobuf)
6. Write generated code to output file

**State Management:**

- **Parser state:** Current token + peek token (2-token lookahead)
- **Compiler state:** Per-module groupings, typedefs, imports, augments, features
- **Loader state (CLI):** AST cache and schema cache for deduplication and cross-module linking
- **Generator state:** Visited node tracking to avoid duplicate struct generation

## Key Abstractions

**Node Interface (`schema.Node`):**
- Purpose: Represents a data node in the semantic schema
- Examples: `schema.Leaf`, `schema.LeafList`, `schema.Container`, `schema.List`, `schema.AnyXML`, `schema.AnyData`, `schema.Case`, `schema.Choice`
- Pattern: Composite pattern where containers have children, leaves are terminal

**Statement Interface (`ast.Statement`):**
- Purpose: Represents a YANG statement in the AST
- Examples: Module, leaf, container, list, typedef, grouping, augment, import, include
- Pattern: Visitor pattern for traversing AST structure

**ModuleLoader Interface:**
- Purpose: Abstract loading of external YANG modules and submodules
- Examples: `DirectoryLoader` (CLI), custom loaders can be injected via compiler.Options
- Pattern: Strategy pattern for dependency resolution

**Generator Interface:**
- Purpose: Abstract code generation across formats
- Examples: `GoGenerator`, `ProtobufGenerator`, future custom generators
- Pattern: Strategy pattern for pluggable code emission

**BaseNode Struct:**
- Purpose: Common implementation for all AST nodes
- Contains: Keyword, argument, substatements, token literal
- Pattern: Composite node with common metadata

## Entry Points

**Library Entry Point (gotya.Parse):**
- Location: `gotya.go:Parse()`
- Triggers: Direct library import usage
- Responsibilities: Create lexer → parser → parse module → return AST

**Library Entry Point (gotya.ParseFile):**
- Location: `gotya.go:ParseFile()`
- Triggers: File-based library usage
- Responsibilities: Read file → decode string → call Parse()

**Library Entry Point (gotya.Compile):**
- Location: `gotya.go:Compile()`
- Triggers: Semantic analysis phase in library workflow
- Responsibilities: Create compiler → compile AST modules → resolve dependencies → return schema

**CLI Entry Point:**
- Location: `cmd/gotya/main.go:main()`
- Triggers: Binary invocation with YANG files and flags
- Responsibilities: Parse flags → load files → compile → generate output

## Error Handling

**Strategy:** Errors propagate up layers with context wrapping. Parser errors return nil module. Compiler errors are collected and returned as error strings slice.

**Patterns:**
- Lexer/Parser: Return nil module if parsing fails completely
- Compiler: Collect error strings, return non-nil compiled module if partial success
- CLI: Log fatal errors and exit with status 1
- Generators: Return wrapped errors with context

## Cross-Cutting Concerns

**Logging:** CLI tool uses standard `log` package for debug messages and errors. Library provides no logging (application responsibility).

**Validation:** Compiler validates during schema building - type resolution, reference checking, augment compatibility, feature filtering.

**Error Collection:** Compiler accumulates errors rather than failing fast to support partial compilation diagnostics.

**Caching:** DirectoryLoader caches both AST and compiled schema modules to avoid redundant processing during cross-module resolution.

---

*Architecture analysis: 2026-03-14*
