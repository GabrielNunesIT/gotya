# External Integrations

**Analysis Date:** 2026-03-14

## APIs & External Services

**None Detected**

This project is a self-contained YANG parser and compiler library with no integration to external APIs or cloud services. It operates entirely on local YANG module files and generates code/proto definitions as output.

## Data Storage

**Databases:**
- None - Project operates entirely in-memory with no persistence layer

**File Storage:**
- Local filesystem only - Reads YANG `.yang` files from specified directories
- Writes generated output files (Go source or Protobuf definitions) to configured output directory
- No cloud storage integration

**Caching:**
- In-memory caching only (AST and Schema module caches in `DirectoryLoader`)
- No external cache service integration (Redis, Memcached, etc.)

## Authentication & Identity

**Auth Provider:**
- Not applicable - This is a compiler library with no user/identity system

## Monitoring & Observability

**Error Tracking:**
- Not integrated - Errors are returned as `error` types through Go's standard error handling

**Logs:**
- Standard Go logging via `log` package
- CLI tool uses `log.Fatalf()` for fatal errors and `log.Printf()` for informational messages
- No structured logging framework (logrus, slog, zap) integrated

**Location:** `cmd/gotya/main.go` - Standard log outputs to stderr/stdout

## CI/CD & Deployment

**Hosting:**
- GitHub-hosted project (github.com/gotya/gotya)
- Library package distributed via `go get`

**CI Pipeline:**
- GitHub Actions (`.github/workflows/ci.yml`)
- Triggers: Pull requests to main, pushes to main and release/* branches
- Pipeline steps:
  1. Checkout code (`actions/checkout@v4`)
  2. Set up Go (`actions/setup-go@v6`) - Uses go-version from `go.mod`
  3. Run golangci-lint (`golangci/golangci-lint-action@v9.2.0`) - Latest version
  4. Run tests (`go test -v ./...`)

**Build Artifacts:**
- CLI binary compiled from `cmd/gotya/main.go`
- Library artifacts: Go modules available via `go get github.com/gotya/gotya`

## Environment Configuration

**Required env vars:**
- None required for library operation
- None required for CLI operation (all behavior controlled via flags)

**Secrets location:**
- Not applicable - No API keys, credentials, or secrets used

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## Module Loading

**File Resolution Strategy:**
- CLI uses `DirectoryLoader` (`cmd/gotya/loader.go`) to resolve YANG module imports/includes
- Searches for modules by name in configured paths
- Supports both `modulename.yang` and `modulename@revision.yang` naming patterns
- Caches loaded modules in memory to avoid duplicate parsing

**YANG Module Dependencies:**
- YANG modules can import/include other YANG modules
- Loader resolves these dependencies from local filesystem paths
- No network-based module repository integration

---

*Integration audit: 2026-03-14*
