# PROJECT KNOWLEDGE BASE

**Generated:** 2026-07-18
**Status:** Development Phase (MVP functional, tests/CI missing)

## OVERVIEW

Go-WHOIS is a domain WHOIS/RDAP lookup service. CLI + HTTP API, JSON output. Stack: cobra (CLI) + gin (HTTP) + viper (config). Uses stdlib `log`/`fmt` for output (NOT zap despite older docs).

## STRUCTURE

```
go-whois/
├── cmd/                  # CLI: root, lookup, serve, update-whois
├── internal/             # Private business logic (used by CLI/HTTP)
│   ├── config/           # Config structs + viper loader
│   ├── service/          # LookupService (query orchestrator)
│   ├── engine/           # RDAP + WHOIS engines + normalizer (see ./AGENTS.md)
│   ├── cache/            # MemoryCache (Redis planned, not impl)
│   ├── api/              # Gin handlers + router (middleware/ + response/ EMPTY)
│   ├── model/            # Domain/Request/Response structs (DUPLICATE of pkg/model)
│   └── errors/           # AppError type + error codes
├── pkg/                  # Public 3rd-party API (self-contained, see pkg/whois/AGENTS.md)
│   ├── whois/            # Public Client + Option pattern (parallel to internal/engine)
│   ├── model/            # Public DomainInfo (DUPLICATE of internal/model)
│   ├── validator/        # Domain regex validation
│   └── tld/              # TLD extraction
├── config/               # config.yaml + tld_whois_servers.yaml (auto-generated)
├── data/                 # rdap_bootstrap.json (IANA official)
├── docs/                 # DEVELOPMENT.md (spec) + REQUIREMENTS.md
├── examples/             # usage.go (standalone demo, own package main)
├── Makefile              # build/test/lint/docker targets
├── Dockerfile            # Multi-stage alpine (WARNING: pinned to golang:1.21, go.mod needs 1.25)
└── main.go               # Thin entry: cmd.Execute()
```

**Empty placeholder dirs** (exist but no files): `scripts/`, `internal/api/middleware/`, `internal/api/response/`, `test/fixtures/`, `test/integration/`

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Domain validation | `pkg/validator/domain.go` | Regex: `^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$` |
| TLD extraction | `pkg/tld/tld.go` | Parse TLD from domain string |
| WHOIS servers | `config/tld_whois_servers.yaml` | `.com` -> `whois.verisign-grs.com.com` (auto-generated, 801 TLDs) |
| RDAP bootstrap | `data/rdap_bootstrap.json` | IANA official data (1199 TLDs) |
| Query orchestration | `internal/service/lookup.go` | LookupService decides protocol, calls engine |
| Engine interface | `internal/engine/engine.go` | `Engine` interface: `Query(ctx, domain) -> DomainInfo` |
| Result normalization | `internal/engine/normalizer.go` | WHOIS text -> JSON, RDAP JSON -> standardized |
| HTTP handlers | `internal/api/handler.go` | `Lookup()`, `HealthCheck()` (BatchLookup NOT impl) |
| CLI commands | `cmd/{root,lookup,serve,update-whois}.go` | cobra commands; `createEngines()` helper lives in lookup.go |
| Error codes | `internal/errors/errors.go` | `INVALID_DOMAIN`, `DOMAIN_NOT_FOUND`, `QUERY_TIMEOUT`, etc. |
| Config loading | `internal/config/loader.go` | viper with env var prefix `WHOIS_` |
| Build targets | `Makefile` | build/test/lint/docker/vuln (LDFLAGS broken - see NOTES) |
| Spec docs | `docs/DEVELOPMENT.md` | Full spec (sections 1-9), test conventions in section 8 |
| Public client API | `pkg/whois/client.go` | `NewClient(opts...)`, Option pattern, `Logger` interface |

## CONVENTIONS

- **Go version**: 1.25.0 (per `go.mod`; Dockerfile pinned to 1.21 - mismatch)
- **Package naming**: lowercase, single word (`engine`, `cache`, `validator`)
- **File naming**: lowercase + underscore (`lookup_service.go`, `rdap_bootstrap.go`)
- **Interface naming**: verb + noun (`Engine`, `CacheManager`, `Normalizer`)
- **Error handling**: Custom `AppError` with code, message, HTTP status
- **Logging**: stdlib `log`/`fmt.Printf` (NOT zap - docs are aspirational; `Logger` interface in `pkg/whois` allows custom loggers)
- **Config env vars**: `WHOIS_` prefix, uppercase, underscore for dots (`WHOIS_SERVER_PORT`)
- **Imports**: stdlib -> third-party -> internal (`go-whois/internal/...`)

## ANTI-PATTERNS (THIS PROJECT)

- **DO NOT** expose `internal/` packages - they're private by Go convention
- **DO NOT** hardcode WHOIS servers - use `config/tld_whois_servers.yaml`
- **DO NOT** cache errors (except domain not found)
- **DO NOT** suppress protocol auto-switch in `auto` mode - it's a core feature
- **DO NOT** use `fmt.Println` for output - use structured zap logs
- **DO NOT** skip domain validation before queries
- **NEVER** return raw WHOIS text to API consumers - always normalize to JSON

## UNIQUE STYLES

- **Protocol priority**: RDAP first, WHOIS fallback (configurable per TLD)
- **Cache key format**: `{protocol}:{normalized_domain}`
- **Batch queries**: Max 50 domains, concurrent execution, per-domain error isolation (SPEC - not yet implemented)
- **Rate limiting**: 100 req/s default, configurable (CONFIG exists, middleware NOT implemented)
- **Metrics**: Prometheus format at `/metrics` (CONFIG exists, endpoint NOT implemented)

## COMMANDS

```bash
# CLI usage
go-whois lookup example.com                    # Single query
go-whois lookup --protocol rdap example.com    # Force RDAP
go-whois lookup --file domains.txt             # Batch from file
go-whois serve                                 # Start HTTP server
go-whois update-whois                          # Refresh config/tld_whois_servers.yaml from IANA

# Development
go build -o bin/go-whois .                     # Build
go test ./...                                  # Run tests (ZERO test files exist currently)
go test -short ./...                           # Unit tests only
go test -coverprofile=coverage.out ./...       # Coverage
golangci-lint run                              # Lint (no .golangci.yml exists - uses defaults)
make build test lint docker vuln               # Makefile targets
```

## DEPENDENCIES

Direct deps in `go.mod` (only 3):

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` v1.10.2 | CLI framework |
| `github.com/gin-gonic/gin` v1.12.0 | HTTP framework |
| `github.com/spf13/viper` v1.21.0 | Config management |

**NOT actually used** (despite older docs claiming): `go.uber.org/zap`, `gopkg.in/natefinish/lumberjack.v2`, `github.com/stretchr/testify`. Code uses stdlib `log`/`fmt`. testify is only transitive (via gin/cobra) in go.sum.

## NOTES

- **MVP Functional**: Core RDAP/WHOIS query, CLI, HTTP API work. Tests/CI/batch/rate-limit/metrics NOT implemented.
- **RFC compliance**: WHOIS (RFC 3912), RDAP (RFC 7480-7483)
- **Protocol switching**: RDAP 5xx/timeout → auto-switch to WHOIS (only in `auto` mode)
- **WHOIS parsing**: Format varies by TLD — normalizer must handle edge cases
- **Privacy**: Some fields redacted by registrars — mark as `REDACTED FOR PRIVACY`
- **Dual-layer architecture**: `pkg/whois/` (public, 1374 lines) and `internal/engine/` (private, 900 lines) are PARALLEL implementations - not shared. Bug fixes must be applied to both.
- **Module path**: `go-whois` (NOT VCS-qualified like `github.com/user/go-whois`) - `go get go-whois` won't work externally despite README claim
- **Makefile LDFLAGS broken**: Injects `main.Version`/`main.BuildTime` but `main.go` doesn't declare these vars
- **Dockerfile mismatch**: Pinned to `golang:1.21-alpine` but `go.mod` requires 1.25.0 - Docker build WILL FAIL
- **Empty dirs**: `scripts/`, `internal/api/middleware/`, `internal/api/response/`, `test/fixtures/`, `test/integration/` exist but contain no files

## THIRD-PARTY LIBRARY USAGE (3rd-party client API)

Go-WHOIS can be used as a third-party library. Full API docs + code examples in **`README.md`** and **`examples/usage.go`**. Key packages: `pkg/whois` (Client + Option pattern, see `pkg/whois/AGENTS.md`), `pkg/model` (DomainInfo, QueryProtocol, Error codes). Quick start: `client := whois.NewClient(); result, err := client.Lookup("example.com")`.

