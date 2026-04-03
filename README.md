# gotya

A modern, fast, and strict YANG parser and compiler written in Go.

`gotya` is designed to provide a robust foundation for building YANG-driven applications, complete with clean architecture, strict validation, and an easy-to-use public API.

## Installation

```bash
go get github.com/GabrielNunesIT/gotya
```

## Quick Start

`gotya` provides a simple top-level package to parse and compile YANG modules.

```go
package main

import (
 "fmt"
 "log"

 "github.com/GabrielNunesIT/gotya"
)

func main() {
 yangContent := `
 module test {
  namespace "http://test.com";
  prefix "t";
  
  container system {
   leaf hostname {
    type string;
   }
  }
 }`

 // 1. Parse the string into an AST module
 astModule, err := gotya.Parse(yangContent)
 if err != nil {
  log.Fatalf("Failed to parse: %v", err)
 }

 // 2. Compile the AST module into a semantic schema module
 modules, err := gotya.Compile([]*gotya.ASTModule{astModule}, nil)
 if err != nil {
  log.Fatalf("Failed to compile: %v", err)
 }

 for _, mod := range modules {
  fmt.Printf("Successfully compiled module: %s\n", mod.Schema().Name)
 }
}
```

## Testing

`gotya` uses reproducible inline test fixtures to ensure test reliability and portability:

- **Test Fixtures**: YANG module fixtures are defined as inline strings in test files, written to temporary directories at test runtime
- **No External Dependencies**: Tests do not depend on external YANG files or assets, making them fully self-contained
- **Reproducibility**: All tests produce deterministic results and can run in any environment without additional setup
- **Running Tests**: Execute `go test ./...` to run the full test suite

This approach ensures tests are fast, reliable, and can be run in parallel without interference.

## Development

`gotya` enforces a strict quality standard.

- **Linting**: We use `golangci-lint` (v2).
- **Testing**: `go test ./...` with `testify`.
- **Mocking**: `mockery` integration.

See the `.golangci.yml` in the root of the project for our enforced rules.
