# Technology Stack

**Analysis Date:** 2026-03-14

## Languages

**Primary:**
- Go 1.25.5 - The entire codebase is written in Go, used for parser, compiler, and code generators

## Runtime

**Environment:**
- Go runtime 1.25.5

**Package Manager:**
- Go modules (go.mod/go.sum)
- Lockfile: Present (go.sum)

## Frameworks

**Core:**
- No external web/application frameworks - This is a library project providing YANG parsing and compilation APIs

**Testing:**
- testify v1.11.1 - Assertion and mocking framework for unit tests
- mockery - Mock generation tool for Go interfaces (configured in `.mockery.yaml`)

**Build/Dev:**
- golangci-lint v2 - Multi-linter aggregator for Go code quality (configured in `.golangci.yml`)
- goimports - Import formatting and sorting

## Key Dependencies

**Critical:**
- github.com/stretchr/testify v1.11.1 - Testing utilities (assert, require, mock)
  - Transitive: github.com/davecgh/go-spew v1.1.1 (debug printing)
  - Transitive: github.com/pmezard/go-difflib v1.0.0 (diff calculations)
  - Transitive: gopkg.in/yaml.v3 v3.0.1 (YAML support for testify)

**No Database Drivers:**
- Project operates on in-memory YANG AST and schema objects; no database integration

**No External Service Clients:**
- Project is self-contained; all functionality is local to the library

## Configuration

**Environment:**
- No environment variables required for core functionality
- No `.env` file pattern used

**Build:**
- Standard `go build` (no custom build config file)
- `go.mod` specifies module path: `github.com/gotya/gotya`
- `.golangci.yml` - Linter configuration with strict quality standards
- `.mockery.yaml` - Mock generation configuration for `github.com/gotya/gotya/...` packages

**CLI Tool Configuration:**
- CLI entry point: `cmd/gotya/main.go`
- Flags for configurable behavior:
  - `-paths` - Comma-separated directories to search for YANG imports/includes
  - `-outdir` - Output directory for generated files (default: "out")
  - `-format` - Output format: "go" or "pb" (default: "go")
  - `-package_name` - Generated package name
  - `-fakeroot_name` - Root entity name
  - `-generate_fakeroot` - Generate fake root element
  - `-skip_deprecated` - Skip deprecated nodes in generation
  - `-skip_obsolete` - Skip obsolete nodes in generation
  - `-add_annotations` - Add metadata field to structs
  - `-generate_getters` - Generate getter methods
  - `-generate_setters` - Generate setter methods
  - `-generate_populate_default` - Generate PopulateDefaults method
  - `-generate_ordered_maps` - Generate lists as slices instead of maps
  - `-proto_cel` - Enable bufbuild/protovalidate CEL constraints for protobuf output

## Platform Requirements

**Development:**
- Go 1.25.5 or compatible version
- golangci-lint v2 for code linting
- mockery for mock code generation
- Standard Unix-like environment (Linux/macOS/Windows with WSL tested)

**Production:**
- Standalone Go binary compiled from `cmd/gotya/main.go`
- Runs on any platform with Go runtime or as precompiled binary
- No external dependencies at runtime beyond Go stdlib

## Output Generation

**Code Generators:**
- `generator/golang/` - Generates Go structs from compiled YANG modules
- `generator/protobuf/` - Generates Protobuf v3 definitions from compiled YANG modules
- Both generators support extensive customization through Options struct

## Codec Support

**RFC 7951:**
- `codec/rfc7951/` - Implements RFC 7951 JSON encoding for YANG instance data
- Supports JSON marshaling/unmarshaling of generated YANG structures

---

*Stack analysis: 2026-03-14*
