# Go-WHOIS

Go-WHOIS 是一个域名 WHOIS/RDAP 查询服务，使用 Go 语言开发。支持 CLI 命令行和 HTTP API 两种访问方式，输出标准化的 JSON 格式域名信息。同时可以作为第三方库集成到其他 Go 项目中使用。

## 功能特性

- **双协议支持**：同时支持 RDAP 和 WHOIS 协议查询
- **智能切换**：默认 RDAP 优先，失败自动回退到 WHOIS
- **TLD 级别配置**：支持为特定 TLD 配置优先使用的协议
- **IANA Bootstrap**：启动时从 IANA 加载 RDAP 端点配置，定时自动更新
- **自动更新 WHOIS 配置**：提供 CLI 命令从 IANA 官网获取所有 TLD 的 WHOIS 服务器信息
- **标准化输出**：统一的 JSON 输出格式，包含 ROID、注册商、注册日期、到期日期等信息
- **内存缓存**：支持 LRU 缓存，减少重复查询
- **CLI + HTTP API**：同时提供命令行工具和 RESTful API 服务
- **第三方库支持**：可作为 Go 模块被其他项目引用

## 支持的 TLD

- **RDAP**：1199 个 TLD（从 IANA Bootstrap 自动加载）
- **WHOIS**：801 个 TLD（通过 `update-whois` 命令自动更新）

包括主流域名：`.com`、`.net`、`.org`、`.cn`、`.asia`、`.io`、`.co`、`.me`、`.app`、`.dev` 等。

## 安装

### 从源码构建

```bash
# 克隆项目
git clone https://github.com/your-username/go-whois.git
cd go-whois

# 下载依赖
go mod download

# 构建
go build -o bin/go-whois.exe .
```

### 作为第三方库引入

```bash
go get go-whois
```

### 使用 Makefile

```bash
make build      # 构建项目
make test       # 运行测试
make lint       # 代码检查
```

## 使用方法

### CLI 命令行

```bash
# 查询域名信息（默认 RDAP 优先）
go-whois lookup example.com

# 指定协议查询
go-whois lookup --protocol rdap example.com
go-whois lookup --protocol whois example.com

# 更新 TLD WHOIS 服务器配置
go-whois update-whois

# 启动 HTTP 服务
go-whois serve
```

### HTTP API

```bash
# 启动服务
go-whois serve

# 查询域名
curl http://localhost:8080/api/v1/lookup/example.com

# 指定协议
curl http://localhost:8080/api/v1/lookup/example.com?protocol=whois

# 健康检查
curl http://localhost:8080/health
```

### 输出示例

```json
{
  "domain_name": "example.com",
  "roid": "2336788_DOMAIN_COM-VRSN",
  "query_protocol": "rdap",
  "query_time": "2026-07-17T17:22:05.8530837+08:00",
  "query_duration_ms": 574,
  "data_source": "live",
  "registrar_name": "RESERVED-Internet Assigned Numbers Authority",
  "registrar_iana_id": "376",
  "registration_date": "1995-08-14T04:00:00Z",
  "expiration_date": "2026-08-13T04:00:00Z",
  "status": [
    "client delete prohibited",
    "client transfer prohibited",
    "client update prohibited"
  ],
  "name_servers": [
    "a.iana-servers.net",
    "b.iana-servers.net"
  ],
  "dnssec": {
    "signed": true,
    "delegation_signed": true
  }
}
```

## 配置说明

配置文件位于 `config/config.yaml`，支持环境变量覆盖（前缀 `WHOIS_`）。

### 主要配置项

```yaml
server:
  http:
    host: "0.0.0.0"
    port: 8080

engine:
  rdap:
    enabled: true
    timeout: 10s
    bootstrap_url: "https://data.iana.org/rdap/dns.json"
    bootstrap_cache_ttl: 24h
  whois:
    enabled: true
    timeout: 10s
  priority:
    default: "rdap"        # 默认协议
    tld_override:           # TLD 级别协议覆盖
      # ".cn": "whois"

cache:
  memory:
    enabled: true
    max_size: 10000
    ttl: 1h

log:
  level: "info"
  format: "json"
```

### 环境变量

```bash
WHOIS_SERVER_HTTP_PORT=8080
WHOIS_ENGINE_RDAP_ENABLED=true
WHOIS_CACHE_MEMORY_MAX_SIZE=10000
```

## 命令行参数

### lookup 命令

```bash
go-whois lookup [domain] [flags]

Flags:
  -p, --protocol string   查询协议 (rdap|whois|auto) (default "auto")
  -v, --verbose           显示详细信息
  -h, --help              帮助信息
```

### update-whois 命令

```bash
go-whois update-whois [flags]

Flags:
  -c, --concurrency int   并发请求数 (default 20)
  -o, --output string     输出文件路径 (default "config/tld_whois_servers.yaml")
  -h, --help              帮助信息
```

### serve 命令

```bash
go-whois serve [flags]

Flags:
  -h, --help              帮助信息
```

## API 接口

### 单域名查询

```
GET /api/v1/lookup/{domain}?protocol=auto
```

**参数：**
- `domain`：域名（路径参数）
- `protocol`：查询协议，可选 `rdap`、`whois`、`auto`（默认 `auto`）

**响应：**
```json
{
  "success": true,
  "data": {
    "domain_name": "example.com",
    "query_protocol": "rdap",
    ...
  },
  "request_id": "req_abc123"
}
```

### 健康检查

```
GET /health
```

**响应：**
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

## 项目结构

```
go-whois/
├── cmd/                      # CLI 命令
│   ├── root.go               # 根命令
│   ├── lookup.go             # 域名查询命令
│   ├── serve.go              # HTTP 服务命令
│   └── update-whois.go       # 更新 WHOIS 配置命令（使用 pkg/whois 公共 API）
├── internal/                 # 内部业务逻辑
│   ├── config/               # 配置管理
│   ├── model/                # 数据模型
│   ├── errors/               # 错误处理
│   ├── engine/               # 查询引擎
│   │   ├── engine.go         # 引擎接口
│   │   ├── rdap.go           # RDAP 引擎
│   │   ├── whois.go          # WHOIS 引擎
│   │   └── normalizer.go     # 结果标准化
│   ├── cache/                # 缓存模块
│   ├── service/              # 查询服务
│   └── api/                  # HTTP API
├── pkg/                      # 公共工具包（可作为第三方库使用）
│   ├── model/                # 公共数据模型
│   │   └── domain.go         # DomainInfo, LookupRequest, LookupResponse, Error 等
│   ├── whois/                # 公共 WHOIS/RDAP 客户端
│   │   ├── client.go         # 高级客户端（支持缓存、自动协议选择、日志接口）
│   │   ├── rdap.go           # RDAP 查询实现
│   │   ├── whois.go          # WHOIS 查询实现
│   │   └── bootstrap.go      # RDAP Bootstrap 和 WHOIS 服务器信息获取
│   ├── validator/            # 域名验证
│   └── tld/                  # TLD 工具
├── examples/                 # 使用示例
│   └── usage.go              # 第三方库使用示例
├── config/                   # 配置文件
│   ├── config.yaml           # 主配置
│   └── tld_whois_servers.yaml # TLD WHOIS 服务器映射
├── data/                     # 数据文件
│   └── rdap_bootstrap.json   # RDAP Bootstrap 数据
├── main.go                   # 入口文件
├── Makefile                  # 构建脚本
├── Dockerfile                # Docker 配置
└── go.mod                    # Go 模块文件
```

## 作为第三方库使用

Go-WHOIS 可以作为第三方库集成到你的 Go 项目中，提供完整的域名查询、RDAP Bootstrap 获取和 WHOIS 服务器信息获取功能。

### 包结构

```
pkg/
├── model/          # 公共数据模型
│   └── domain.go   # DomainInfo, LookupRequest, LookupResponse, Error 等
└── whois/          # 公共客户端 API
    ├── client.go     # 高级客户端（支持缓存、自动协议选择、日志接口）
    ├── rdap.go       # RDAP 查询实现
    ├── whois.go      # WHOIS 查询实现
    └── bootstrap.go  # RDAP Bootstrap 和 WHOIS 服务器信息获取
```

### 公共 API 函数列表

#### 域名查询

| 函数 | 说明 |
|------|------|
| `NewClient(opts ...Option) *Client` | 创建高级客户端实例 |
| `Client.Lookup(domain string) (*DomainInfo, error)` | 查询域名信息（默认 RDAP 优先） |
| `Client.LookupWithContext(ctx context.Context, domain string) (*DomainInfo, error)` | 带上下文的域名查询 |
| `Client.LookupWithProtocol(domain string, protocol QueryProtocol) (*DomainInfo, error)` | 使用指定协议查询 |
| `Client.GetCacheStats() CacheStats` | 获取缓存统计信息 |
| `Client.ClearCache()` | 清空缓存 |
| `Client.Close() error` | 关闭客户端，清理资源 |
| `NewWHOISClient(opts ...WHOISOption) *WHOISClient` | 创建单独的 WHOIS 客户端 |
| `WHOISClient.Query(ctx context.Context, domain string) (*DomainInfo, error)` | 执行 WHOIS 查询 |

#### RDAP Bootstrap 获取

| 函数 | 说明 |
|------|------|
| `FetchRDAPBootstrap(bootstrapURL string) (map[string]string, error)` | 从 IANA 获取 RDAP Bootstrap 数据，返回 TLD 到端点的映射 |

#### WHOIS 服务器信息获取

| 函数 | 说明 |
|------|------|
| `FetchTLDList() ([]TLDInfo, error)` | 从 IANA 获取 TLD 列表 |
| `FetchWhoisServer(tld string) string` | 获取单个 TLD 的 WHOIS 服务器 |
| `FetchWhoisServers(tlds []TLDInfo, concurrency int, progressCallback func(int, int)) []TLDInfo` | 批量获取 WHOIS 服务器 |
| `FormatWhoisServersYAML(tlds []TLDInfo) string` | 将 TLD 信息格式化为 YAML 字符串 |
| `GetWhoisServersMap(tlds []TLDInfo) map[string]string` | 将 TLD 信息转换为 map 格式 |

### 快速开始

```go
package main

import (
    "fmt"
    "log"

    "go-whois/pkg/whois"
)

func main() {
    // 创建客户端
    client := whois.NewClient()
    defer client.Close()

    // 查询域名
    result, err := client.Lookup("example.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("域名: %s\n", result.DomainName)
    fmt.Printf("注册商: %s\n", result.RegistrarName)
    fmt.Printf("到期日期: %v\n", result.ExpirationDate)
    fmt.Printf("名称服务器: %v\n", result.NameServers)
}
```

### 配置选项

```go
import (
    "time"
    "go-whois/pkg/model"
    "go-whois/pkg/whois"
)

client := whois.NewClient(
    whois.WithProtocol(model.ProtocolRDAP),       // 设置查询协议 (rdap|whois|auto)
    whois.WithTimeout(15 * time.Second),          // 设置超时时间
    whois.WithCache(true, 1000, time.Hour),       // 启用缓存 (enabled, maxSize, ttl)
    whois.WithRDAPBootstrap("https://custom-url"), // 自定义 RDAP Bootstrap URL
    whois.WithUserAgent("my-app/1.0"),            // 设置 User-Agent
    whois.WithRawResponse(true),                  // 包含原始响应
    whois.WithLogger(&MyLogger{}),                // 设置自定义日志器
)
defer client.Close()
```

#### 客户端配置选项说明

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `WithProtocol(protocol QueryProtocol)` | 设置查询协议 | `model.ProtocolAuto` |
| `WithTimeout(timeout time.Duration)` | 设置查询超时时间 | `10s` |
| `WithCache(enabled bool, maxSize int, ttl time.Duration)` | 启用并配置缓存 | `true, 1000, 1h` |
| `WithRDAPBootstrap(url string)` | 设置 RDAP Bootstrap URL | IANA 官方 URL |
| `WithUserAgent(ua string)` | 设置 User-Agent | `go-whois/1.0` |
| `WithRawResponse(include bool)` | 是否包含原始响应 | `false` |
| `WithLogger(logger Logger)` | 设置自定义日志器 | 标准库 log |

### 单独使用 WHOIS 客户端

```go
whoisClient := whois.NewWHOISClient(
    whois.WithWSTimeout(10 * time.Second),
    whois.WithWSPort(43),
    whois.WithWSServers(map[string]string{
        ".com": "whois.verisign-grs.com",
        ".org": "whois.publicinterestregistry.org",
    }),
)
defer whoisClient.Close()

ctx := context.Background()
result, err := whoisClient.Query(ctx, "github.com")
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}
```

#### WHOIS 客户端配置选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `WithWSTimeout(timeout time.Duration)` | 设置查询超时时间 | `10s` |
| `WithWSPort(port int)` | 设置 WHOIS 服务器端口 | `43` |
| `WithWSServers(servers map[string]string)` | 设置 TLD 服务器映射 | 从配置文件加载 |
| `WithWSFallbacks(fallbacks map[string][]string)` | 设置备用服务器映射 | - |
| `WithWSLogger(logger Logger)` | 设置自定义日志器 | 标准库 log |

### 错误处理

```go
result, err := client.Lookup("invalid-domain")
if err != nil {
    // 检查错误类型
    if appErr, ok := err.(*model.Error); ok {
        switch appErr.Code {
        case model.ErrCodeInvalidDomain:
            fmt.Println("域名格式无效")
        case model.ErrCodeDomainNotFound:
            fmt.Println("域名未注册")
        case model.ErrCodeQueryTimeout:
            fmt.Println("查询超时")
        case model.ErrCodeProtocolError:
            fmt.Printf("协议错误: %s\n", appErr.Message)
        default:
            fmt.Printf("错误: %s - %s\n", appErr.Code, appErr.Message)
        }
    }
}
```

#### 错误代码说明

| 错误代码 | 说明 |
|----------|------|
| `INVALID_DOMAIN` | 域名格式无效 |
| `DOMAIN_NOT_FOUND` | 域名未注册 |
| `QUERY_TIMEOUT` | 查询超时 |
| `PROTOCOL_ERROR` | 协议错误 |
| `RATE_LIMITED` | 请求频率超限 |
| `INTERNAL_ERROR` | 内部错误 |
| `SERVICE_UNAVAILABLE` | 服务不可用 |

### 自定义日志

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, keysAndValues ...interface{}) {
    // 自定义 debug 日志
}

func (l *MyLogger) Info(msg string, keysAndValues ...interface{}) {
    // 自定义 info 日志
}

func (l *MyLogger) Warn(msg string, keysAndValues ...interface{}) {
    // 自定义 warn 日志
}

func (l *MyLogger) Error(msg string, keysAndValues ...interface{}) {
    // 自定义 error 日志
}

client := whois.NewClient(
    whois.WithLogger(&MyLogger{}),
)
```

### 获取 RDAP Bootstrap 数据

```go
// 从 IANA 获取 RDAP Bootstrap 数据
rdapEndpoints, err := whois.FetchRDAPBootstrap("https://data.iana.org/rdap/dns.json")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("获取到 %d 个 TLD 的 RDAP 端点\n", len(rdapEndpoints))
for tld, endpoint := range rdapEndpoints {
    fmt.Printf(".%s -> %s\n", tld, endpoint)
}
```

### 获取 WHOIS 服务器信息

```go
// 从 IANA 获取 TLD 列表
tlds, err := whois.FetchTLDList()
if err != nil {
    log.Fatal(err)
}

// 获取单个 TLD 的 WHOIS 服务器
server := whois.FetchWhoisServer("com")
fmt.Printf(".com WHOIS 服务器: %s\n", server)

// 批量获取 WHOIS 服务器
results := whois.FetchWhoisServers(tlds, 20, func(progress, total int) {
    fmt.Printf("\r进度: %d/%d", progress, total)
})

// 转换为 map 格式
serversMap := whois.GetWhoisServersMap(results)

// 格式化为 YAML
yamlContent := whois.FormatWhoisServersYAML(results)
```

### 运行示例

```bash
go run examples/usage.go
```

## 开发

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行单元测试
go test -short ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 代码检查

```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...

# 使用 golangci-lint
golangci-lint run
```

## Docker 部署

```bash
# 构建镜像
docker build -t go-whois .

# 运行容器
docker run -p 8080:8080 go-whois

# 使用 docker-compose
docker-compose up -d
```

## 技术栈

| 组件 | 技术选型 |
|------|----------|
| CLI 框架 | [cobra](https://github.com/spf13/cobra) |
| HTTP 框架 | [gin](https://github.com/gin-gonic/gin) |
| 配置管理 | [viper](https://github.com/spf13/viper) |
| 日志库 | [zap](https://go.uber.org/zap) |
| 缓存 | 内存 LRU |
| 测试 | [testify](https://github.com/stretchr/testify) |

## 协议规范

- **WHOIS**：[RFC 3912](https://tools.ietf.org/html/rfc3912)
- **RDAP**：[RFC 7480](https://tools.ietf.org/html/rfc7480) - [RFC 7483](https://tools.ietf.org/html/rfc7483)

## 许可证

MIT License
