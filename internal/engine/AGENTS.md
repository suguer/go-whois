# internal/engine

Query engines for RDAP + WHOIS. Implements `Engine` interface, used by `internal/service/LookupService`.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Engine interface | `engine.go` | `Name() Protocol`, `Query(ctx, domain) (*model.DomainInfo, error)`, `IsAvailable() bool` |
| Normalizer interface | `engine.go` | `NormalizeWHOIS(domain, raw)`, `NormalizeRDAP(domain, raw)` |
| Protocol type | `engine.go` | `ProtocolRDAP`, `ProtocolWHOIS`, `ProtocolAuto` consts (duplicated in `pkg/model`) |
| QueryRequest | `engine.go` | `{Domain string, Protocol Protocol}` - used by service layer, NOT pkg/model.LookupRequest |
| RDAP engine | `rdap.go` | `RDAP` struct, auto-refresh bootstrap goroutine |
| WHOIS engine | `whois.go` | `WHOIS` struct, TCP port 43, `net.DialTimeout` |
| Normalizer impl | `normalizer.go` | `DefaultNormalizer`, WHOIS regex patterns, RDAP parser |

## CONVENTIONS

- **Model import**: Uses `go-whois/internal/model.DomainInfo` (NOT `pkg/model.DomainInfo`) - the two are duplicate structs
- **RDAP auto-refresh**: `startAutoRefresh()` goroutine ticks on `cfg.BootstrapCacheTTL` (default 24h), mutates `endpoints` map under `sync.RWMutex`
- **RDAP default endpoints**: 18 TLDs hardcoded as fallback (com/net/org/info/io/co/me/asia/biz/app/dev + 7 more)
- **WHOIS protocol**: TCP port 43, sends `domain + "\r\n"`, reads via `bufio.Scanner` (5MB limit)
- **WHOIS fallback**: `fallbacks` map keyed by TLD (`.com` -> `[whois.verisign-grs.com, ...]`)
- **Status normalization**: 24-entry map in normalizer (e.g. `clientdeleteprohibited` -> `clientDeleteProhibited`)
- **Date parsing**: Tries `2006-01-02T15:04:05Z` then `2006-01-02` (RFC3339 then date-only)
- **DNSSEC**: Only sets `Signed`/`DelegationSigned` if `true` (never explicitly `false`, uses `*bool` pointer)
- **RDAP headers**: `User-Agent` + `Accept: application/rdap+json`
- **Error mapping**: 404 -> `DOMAIN_NOT_FOUND`, timeout -> `QUERY_TIMEOUT`, other non-200 -> `PROTOCOL_ERROR`

## ANTI-PATTERNS

- **DO NOT** add engines without implementing full `Engine` interface (Name/Query/IsAvailable)
- **DO NOT** bypass `Normalizer` - raw WHOIS text / RDAP JSON must be normalized to `DomainInfo`
- **DO NOT** import `pkg/model` here - use `internal/model.DomainInfo` exclusively
- **DO NOT** suppress RDAP auto-refresh goroutine - it keeps endpoints fresh
- **DO NOT** hardcode WHOIS servers - read from `config.RDAPConfig`/`WHOISConfig` passed at construction

## NOTES

- `IANABootstrapData` + `RDAPResponse` types are DUPLICATED in `pkg/whois/rdap.go` - parallel implementations, not shared
- `ParseRDAPResponse(data []byte) (*RDAPResponse, error)` is exported for testing but not used outside package
- WHOIS `parseResponse` regex varies by registrar - normalizer handles common patterns but edge cases exist
