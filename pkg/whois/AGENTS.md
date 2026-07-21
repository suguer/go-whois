# pkg/whois

Public 3rd-party client API. Self-contained - does NOT depend on `internal/` packages. Parallel implementation to `internal/engine/` (dual-layer architecture).

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Client + Option pattern | `client.go` | `NewClient(opts...)`, functional options, `Logger` interface |
| RDAP query | `rdap.go` | `queryRDAP()`, `parseRDAPResponse()`, 11 default TLD endpoints |
| WHOIS query (standalone) | `whois.go` | `WHOISClient` struct, `WHOISOption` pattern (WS prefix) |
| IANA fetchers | `bootstrap.go` | `FetchRDAPBootstrap`, `FetchTLDList`, `FetchWhoisServers` |
| Public model | `pkg/model/domain.go` | `DomainInfo`, `QueryProtocol`, `Error`, error codes |

## CONVENTIONS

- **Option pattern**: `type Option func(*options)` - `WithProtocol`, `WithTimeout`, `WithCache`, `WithRDAPBootstrap`, `WithRDAPBootstrapFile`, `WithWHOISConfigFile`, `WithUserAgent`, `WithLogger`, `WithRawResponse`
- **WHOISOption pattern**: Separate `type WHOISOption func(*whoisOptions)` with WS prefix (`WithWSTimeout`, `WithWSPort`, `WithWSServers`, `WithWSFallbacks`, `WithWSLogger`, `WithWSConfigFile`) - avoids name collision with `Option`
- **Logger interface**: `Debug/Info/Warn/Error(msg string, keysAndValues ...interface{})` - default impl uses stdlib `log` to stderr with `[go-whois] ` prefix, Debug is no-op
- **Model import**: Uses `go-whois/pkg/model.DomainInfo` (NOT `internal/model.DomainInfo`)
- **Cache key format**: `string(protocol) + ":" + domain` (matches internal/ convention)
- **Auto mode**: RDAP first, on error falls back to WHOIS (logs warning)
- **WHOISClient config loading**: If `configFile` is set via `WithWSConfigFile`, loads from that file first; otherwise tries `config/tld_whois_servers.yaml` then `../config/`, `../../config/`, `../../../config/` (relative path fallback chain)
- **RDAP Bootstrap loading**: If `rdapBootstrapFile` is set via `WithRDAPBootstrapFile`, loads from local file first; falls back to URL fetch on failure
- **Config download functions**: `DownloadRDAPBootstrap(destPath)` and `DownloadWHOISConfig(destPath, concurrency, progressCallback)` download configs to specified paths
- **FetchWhoisServers concurrency**: Uses `sync.WaitGroup` + buffered channel semaphore + writes to `&results[idx]` (index-isolated, safe)
- **Bootstrap fetch**: `FetchRDAPBootstrap` parses IANA `dns.json`; `FetchTLDList`/`FetchWhoisServer` scrape IANA HTML pages via regex

## ANTI-PATTERNS

- **DO NOT** import `internal/` packages here - `pkg/whois` must remain standalone for external consumers
- **DO NOT** trust `validateDomain` in `client.go` - it's simplified (empty + length > 253 check only), use `pkg/validator.ValidateDomain` for full regex
- **DO NOT** assume cache is LRU - `evictOldest` evicts earliest-expiry-first (misnamed, NOT true LRU)
- **DO NOT** expect RDAP endpoints available immediately after `NewClient()` - `loadRDAPBootstrap()` runs async in goroutine; first queries may miss endpoints
- **DO NOT** add dependencies on `internal/` - if sharing needed, move shared code to `pkg/`

## NOTES

- **Dual-layer architecture**: `pkg/whois/` (1374 lines) is a PARALLEL implementation to `internal/engine/` (900 lines). Same protocols, separate code. Bug fixes must be applied to both.
- **`cmd/update-whois.go`** uses this public API; `cmd/lookup.go` and `cmd/serve.go` use `internal/` packages - architectural inconsistency
- **`FetchWhoisServer` regex**: `(?i)<b>WHOIS Server:</b>\s*([^<\s]+)` - scrapes IANA TLD detail page HTML
- **`FormatWhoisServersYAML`**: Groups TLDs by type (generic/sponsored/country-code/infrastructure)
- **`WHOISClient.fallbacks`**: Keyed by FULL domain (not TLD) - likely a bug, should probably key by TLD
