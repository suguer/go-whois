# Go-WHOIS 域名查询服务系统 - 开发文档

> 文档版本：v1.0  
> 创建日期：2026-07-17  
> 基于需求文档：REQUIREMENTS.md  
> 适用范围：开发团队编码规范与实现指南

---

## 目录

1. [项目概述](#1-项目概述)
2. [项目目录结构设计](#2-项目目录结构设计)
3. [核心模块代码结构和接口定义](#3-核心模块代码结构和接口定义)
4. [数据模型定义](#4-数据模型定义)
5. [配置文件结构](#5-配置文件结构)
6. [错误处理规范](#6-错误处理规范)
7. [日志规范](#7-日志规范)
8. [测试规范](#8-测试规范)
9. [代码风格和命名规范](#9-代码风格和命名规范)
10. [依赖管理策略](#10-依赖管理策略)
11. [开发环境搭建指南](#11-开发环境搭建指南)

---

## 1. 项目概述

### 1.1 项目信息

| 项目 | 说明 |
|------|------|
| 项目名称 | go-whois |
| 编程语言 | Go 1.21+ |
| 项目类型 | CLI + HTTP API 服务 |
| 核心功能 | 域名 WHOIS/RDAP 查询，JSON 标准化输出 |
| 技术栈 | cobra (CLI) + gin (HTTP) + viper (配置) + zap (日志) |

### 1.2 架构分层

```
+------------------------------------------------------------------+
|                          调用层 (Clients)                         |
+------------------------------------------------------------------+
|        CLI客户端              |         HTTP客户端                |
+-------------------------------+----------------------------------+
                                |
+------------------------------------------------------------------+
|                        接口层 (API Layer)                         |
+------------------------------------------------------------------+
|   CLI命令解析 (cobra)         |    HTTP服务器 (gin)              |
+-------------------------------+----------------------------------+
                                |
+------------------------------------------------------------------+
|                       服务层 (Service Layer)                      |
+------------------------------------------------------------------+
|                    LookupService (查询调度器)                     |
+------------------------------------------------------------------+
                                |
+------------------------------------------------------------------+
|                       引擎层 (Engine Layer)                       |
+------------------------------------------------------------------+
|   RDAP Engine               |        WHOIS Engine               |
+-------------------------------+----------------------------------+
                                |
+------------------------------------------------------------------+
|                       缓存层 (Cache Layer)                        |
+------------------------------------------------------------------+
|                    CacheManager (缓存管理器)                      |
+------------------------------------------------------------------+
                                |
+------------------------------------------------------------------+
|                      基础设施层 (Infrastructure)                   |
+------------------------------------------------------------------+
|   配置管理        |   日志系统        |   指标监控               |
|   (viper)         |   (zap)           |   (prometheus)           |
+-------------------+-------------------+-------------------------+
```

---

## 2. 项目目录结构设计

### 2.1 完整目录结构

```
go-whois/
├── cmd/                                    # 命令行入口
│   ├── root.go                             # 根命令定义
│   ├── lookup.go                           # lookup 子命令
│   └── serve.go                            # serve 子命令（HTTP服务）
│
├── internal/                               # 内部包（不对外暴露）
│   ├── config/                             # 配置管理模块
│   │   ├── config.go                       # 配置结构体定义
│   │   ├── loader.go                       # 配置加载器
│   │   └── defaults.go                     # 默认配置值
│   │
│   ├── service/                            # 业务服务层
│   │   ├── lookup.go                       # 查询服务接口
│   │   └── lookup_impl.go                  # 查询服务实现
│   │
│   ├── engine/                             # 查询引擎层
│   │   ├── engine.go                       # 引擎接口定义
│   │   ├── rdap.go                         # RDAP 查询引擎
│   │   ├── rdap_bootstrap.go               # RDAP Bootstrap 处理
│   │   ├── whois.go                        # WHOIS 查询引擎
│   │   ├── whois_server.go                 # WHOIS 服务器管理
│   │   └── normalizer.go                   # 结果标准化器
│   │
│   ├── cache/                              # 缓存模块
│   │   ├── cache.go                        # 缓存接口定义
│   │   ├── memory.go                       # 内存缓存实现
│   │   └── redis.go                        # Redis 缓存实现（可选）
│   │
│   ├── api/                                # HTTP API 模块
│   │   ├── handler.go                      # 请求处理器
│   │   ├── router.go                       # 路由定义
│   │   ├── middleware/                     # 中间件
│   │   │   ├── logger.go                   # 日志中间件
│   │   │   ├── ratelimit.go               # 限流中间件
│   │   │   ├── cors.go                     # CORS 中间件
│   │   │   └── recovery.go                # 异常恢复中间件
│   │   └── response/                       # 响应工具
│   │       └── response.go                 # 统一响应格式
│   │
│   ├── model/                              # 数据模型
│   │   ├── domain.go                       # 域名信息模型
│   │   ├── request.go                      # 请求模型
│   │   └── response.go                     # 响应模型
│   │
│   └── errors/                             # 错误定义
│       └── errors.go                       # 自定义错误类型
│
├── pkg/                                    # 可复用的工具包
│   ├── validator/                          # 域名验证工具
│   │   ├── validator.go                    # 验证接口
│   │   └── domain.go                       # 域名验证实现
│   │
│   ├── tld/                                # TLD 工具
│   │   ├── tld.go                          # TLD 解析工具
│   │   └── data.go                         # TLD 数据
│   │
│   └── netutil/                            # 网络工具
│       └── tcp.go                          # TCP 连接工具
│
├── config/                                 # 配置文件目录
│   ├── config.yaml                         # 主配置文件
│   ├── config.example.yaml                 # 配置示例文件
│   └── tld_whois_servers.yaml              # TLD WHOIS 服务器映射
│
├── data/                                   # 数据文件目录
│   └── rdap_bootstrap.json                 # RDAP Bootstrap 数据
│
├── scripts/                                # 脚本目录
│   ├── build.sh                            # 构建脚本
│   ├── build.ps1                           # Windows 构建脚本
│   └── install.sh                          # 安装脚本
│
├── test/                                   # 测试数据目录
│   ├── fixtures/                           # 测试固件
│   │   ├── whois_response.txt              # WHOIS 响应样例
│   │   └── rdap_response.json              # RDAP 响应样例
│   └── integration/                        # 集成测试
│       └── lookup_test.go                  # 查询集成测试
│
├── main.go                                 # 程序入口
├── go.mod                                  # Go 模块定义
├── go.sum                                  # 依赖校验和
├── Makefile                                # 构建脚本
├── Dockerfile                              # Docker 构建文件
├── .golangci.yml                           # golangci-lint 配置
├── .gitignore                              # Git 忽略文件
└── README.md                               # 项目说明文档
```

### 2.2 目录说明

| 目录 | 用途 | 访问权限 |
|------|------|----------|
| `cmd/` | CLI 命令入口，每个文件对应一个子命令 | 公开 |
| `internal/` | 内部业务逻辑，不对外暴露 | 私有 |
| `pkg/` | 可复用的工具包，可被外部引用 | 公开 |
| `config/` | 配置文件，随项目发布 | 公开 |
| `data/` | 静态数据文件 | 公开 |
| `scripts/` | 构建和部署脚本 | 公开 |
| `test/` | 测试数据和集成测试 | 公开 |

### 2.3 文件命名规范

| 文件类型 | 命名规则 | 示例 |
|----------|----------|------|
| Go 源文件 | 小写字母 + 下划线分隔 | `lookup_service.go` |
| 测试文件 | 源文件名 + `_test` 后缀 | `lookup_service_test.go` |
| 接口文件 | 模块名 + `.go` | `engine.go` |
| 实现文件 | 模块名 + `_impl.go` 或具体实现名 | `memory.go` |
| 配置文件 | 小写字母 + 下划线分隔 | `config.yaml` |

---

## 3. 核心模块代码结构和接口定义

### 3.1 引擎接口 (`internal/engine/engine.go`)

```go
package engine

import (
    "context"
    "go-whois/internal/model"
)

// Protocol 表示查询协议类型
type Protocol string

const (
    ProtocolRDAP  Protocol = "rdap"
    ProtocolWHOIS Protocol = "whois"
    ProtocolAuto  Protocol = "auto"
)

// QueryRequest 表示查询请求
type QueryRequest struct {
    Domain   string   `json:"domain"`
    Protocol Protocol `json:"protocol"`
}

// Engine 定义查询引擎接口
type Engine interface {
    // Name 返回引擎名称
    Name() Protocol

    // Query 执行域名查询
    Query(ctx context.Context, domain string) (*model.DomainInfo, error)

    // IsAvailable 检查引擎是否可用
    IsAvailable() bool
}

// Normalizer 定义结果标准化接口
type Normalizer interface {
    // NormalizeWHOIS 标准化 WHOIS 响应
    NormalizeWHOIS(domain string, rawResponse string) (*model.DomainInfo, error)

    // NormalizeRDAP 标准化 RDAP 响应
    NormalizeRDAP(domain string, rawData []byte) (*model.DomainInfo, error)
}
```

### 3.2 查询服务接口 (`internal/service/lookup.go`)

```go
package service

import (
    "context"
    "go-whois/internal/engine"
    "go-whois/internal/model"
)

// LookupService 定义查询服务接口
type LookupService interface {
    // Lookup 执行单域名查询
    Lookup(ctx context.Context, req *engine.QueryRequest) (*model.DomainInfo, error)

    // BatchLookup 执行批量域名查询
    BatchLookup(ctx context.Context, reqs []*engine.QueryRequest) ([]*model.DomainInfo, error)
}

// LookupServiceImpl 实现 LookupService
type LookupServiceImpl struct {
    engines  map[engine.Protocol]engine.Engine
    normalizer engine.Normalizer
    cache    CacheManager
    config   *Config
}

// NewLookupService 创建新的查询服务实例
func NewLookupService(
    engines map[engine.Protocol]engine.Engine,
    normalizer engine.Normalizer,
    cache CacheManager,
    config *Config,
) LookupService {
    return &LookupServiceImpl{
        engines:    engines,
        normalizer: normalizer,
        cache:      cache,
        config:     config,
    }
}

// Lookup 实现单域名查询
func (s *LookupServiceImpl) Lookup(ctx context.Context, req *engine.QueryRequest) (*model.DomainInfo, error) {
    // 1. 检查缓存
    // 2. 确定查询协议
    // 3. 执行查询
    // 4. 标准化结果
    // 5. 写入缓存
    // 6. 返回结果
}
```

### 3.3 缓存接口 (`internal/cache/cache.go`)

```go
package cache

import (
    "context"
    "time"
    "go-whois/internal/model"
)

// CacheManager 定义缓存管理接口
type CacheManager interface {
    // Get 获取缓存
    Get(ctx context.Context, key string) (*model.DomainInfo, error)

    // Set 设置缓存
    Set(ctx context.Context, key string, value *model.DomainInfo, ttl time.Duration) error

    // Delete 删除缓存
    Delete(ctx context.Context, key string) error

    // Clear 清空缓存
    Clear(ctx context.Context) error

    // Stats 获取缓存统计
    Stats() CacheStats
}

// CacheStats 缓存统计信息
type CacheStats struct {
    Hits      int64 `json:"hits"`
    Misses    int64 `json:"misses"`
    Size      int   `json:"size"`
    HitRate   float64 `json:"hit_rate"`
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(protocol, domain string) string {
    return protocol + ":" + domain
}
```

### 3.4 HTTP API 接口 (`internal/api/handler.go`)

```go
package api

import (
    "github.com/gin-gonic/gin"
    "go-whois/internal/service"
)

// Handler 定义 HTTP 请求处理器
type Handler struct {
    lookupService service.LookupService
}

// NewHandler 创建新的处理器实例
func NewHandler(lookupService service.LookupService) *Handler {
    return &Handler{
        lookupService: lookupService,
    }
}

// Lookup 单域名查询处理
func (h *Handler) Lookup(c *gin.Context) {
    // 1. 解析请求参数
    // 2. 验证域名格式
    // 3. 调用查询服务
    // 4. 返回 JSON 响应
}

// BatchLookup 批量查询处理
func (h *Handler) BatchLookup(c *gin.Context) {
    // 1. 解析批量请求
    // 2. 验证请求数量
    // 3. 并发调用查询服务
    // 4. 返回 JSON 响应
}

// HealthCheck 健康检查处理
func (h *Handler) HealthCheck(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "ok",
        "service": "go-whois",
    })
}
```

### 3.5 CLI 命令结构 (`cmd/lookup.go`)

```go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "go-whois/internal/service"
)

var lookupCmd = &cobra.Command{
    Use:   "lookup [domain]",
    Short: "查询域名 WHOIS/RDAP 信息",
    Long:  `查询指定域名的 WHOIS 或 RDAP 注册信息，并以 JSON 格式输出`,
    Args:  cobra.ExactArgs(1),
    RunE:  runLookup,
}

var (
    protocol string
    output   string
    verbose  bool
)

func init() {
    rootCmd.AddCommand(lookupCmd)
    lookupCmd.Flags().StringVarP(&protocol, "protocol", "p", "auto", "查询协议 (rdap|whois|auto)")
    lookupCmd.Flags().StringVarP(&output, "output", "o", "json", "输出格式 (json|text)")
    lookupCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示详细信息")
}

func runLookup(cmd *cobra.Command, args []string) error {
    domain := args[0]
    // 1. 初始化配置
    // 2. 创建查询服务
    // 3. 执行查询
    // 4. 格式化输出
    return nil
}
```

---

## 4. 数据模型定义

### 4.1 域名信息模型 (`internal/model/domain.go`)

```go
package model

import "time"

// DomainInfo 表示域名查询结果
type DomainInfo struct {
    DomainName    string        `json:"domain_name"`
    QueryProtocol string        `json:"query_protocol"`
    QueryTime     time.Time     `json:"query_time"`
    QueryDuration int64         `json:"query_duration_ms"`
    DataSource    string        `json:"data_source"`
    Registration  Registration  `json:"registration"`
    Status        []string      `json:"status"`
    NameServers   []string      `json:"name_servers"`
    DNSSEC        DNSSECInfo    `json:"dnssec"`
    RawResponse   *string       `json:"raw_response,omitempty"`
}

// Registration 表示注册信息
type Registration struct {
    Registrar        Registrar    `json:"registrar"`
    Registrant       Registrant   `json:"registrant"`
    RegistrationDate *time.Time   `json:"registration_date,omitempty"`
    ExpirationDate   *time.Time   `json:"expiration_date,omitempty"`
    LastUpdated      *time.Time   `json:"last_updated,omitempty"`
}

// Registrar 表示注册商信息
type Registrar struct {
    Name   string `json:"name,omitempty"`
    URL    string `json:"url,omitempty"`
    IANAID string `json:"iana_id,omitempty"`
}

// Registrant 表示注册人信息
type Registrant struct {
    Name         string `json:"name,omitempty"`
    Organization string `json:"organization,omitempty"`
    Country      string `json:"country,omitempty"`
    State        string `json:"state,omitempty"`
    City         string `json:"city,omitempty"`
}

// DNSSECInfo 表示 DNSSEC 信息
type DNSSECInfo struct {
    Signed           *bool `json:"signed,omitempty"`
    DelegationSigned *bool `json:"delegation_signed,omitempty"`
}
```

### 4.2 请求模型 (`internal/model/request.go`)

```go
package model

// LookupRequest 表示查询请求
type LookupRequest struct {
    Domain   string `json:"domain" binding:"required"`
    Protocol string `json:"protocol" binding:"omitempty,oneof=rdap whois auto"`
}

// BatchLookupRequest 表示批量查询请求
type BatchLookupRequest struct {
    Domains  []string `json:"domains" binding:"required,min=1,max=100"`
    Protocol string   `json:"protocol" binding:"omitempty,oneof=rdap whois auto"`
}
```

### 4.3 响应模型 (`internal/model/response.go`)

```go
package model

// APIResponse 表示统一 API 响应
type APIResponse struct {
    Success   bool        `json:"success"`
    Data      interface{} `json:"data,omitempty"`
    Error     *APIError   `json:"error,omitempty"`
    RequestID string      `json:"request_id,omitempty"`
}

// APIError 表示 API 错误
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

// BatchAPIResponse 表示批量查询响应
type BatchAPIResponse struct {
    Success bool              `json:"success"`
    Data    []*DomainInfo     `json:"data"`
    Errors  []*BatchError     `json:"errors,omitempty"`
}

// BatchError 表示批量查询中的单个错误
type BatchError struct {
    Domain  string    `json:"domain"`
    Error   *APIError `json:"error"`
}
```

---

## 5. 配置文件结构

### 5.1 主配置文件 (`config/config.yaml`)

```yaml
# Go-WHOIS 主配置文件
# 环境变量可覆盖配置项，格式：WHOIS_大写配置项名

server:
  # HTTP 服务器配置
  http:
    host: "0.0.0.0"
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 60s
    shutdown_timeout: 10s

  # CLI 配置
  cli:
    default_protocol: "auto"
    default_output: "json"
    verbose: false

# 查询引擎配置
engine:
  # RDAP 配置
  rdap:
    enabled: true
    timeout: 10s
    max_retries: 3
    retry_delay: 1s
    bootstrap_url: "https://data.iana.org/rdap/dns.json"
    bootstrap_cache_ttl: 24h
    user_agent: "go-whois/1.0"

  # WHOIS 配置
  whois:
    enabled: true
    timeout: 10s
    max_retries: 3
    retry_delay: 1s
    default_port: 43
    user_agent: "go-whois/1.0"

  # 协议优先级配置
  priority:
    default: "rdap"
    # TLD 级别协议配置
    tld_override:
      # 某些 TLD 只支持 WHOIS
      # ".cn": "whois"
      # ".ru": "whois"

# 缓存配置
cache:
  # 内存缓存
  memory:
    enabled: true
    max_size: 10000
    ttl: 1h
    cleanup_interval: 10m

  # Redis 缓存（可选）
  redis:
    enabled: false
    host: "localhost"
    port: 6379
    password: ""
    db: 0
    ttl: 1h
    max_retries: 3

# 日志配置
log:
  level: "info"           # debug, info, warn, error
  format: "json"          # json, console
  output: "stdout"        # stdout, stderr, file
  file_path: "logs/go-whois.log"
  max_size: 100           # MB
  max_backups: 3
  max_age: 7              # days
  compress: true

# 限流配置
ratelimit:
  enabled: true
  rate: 100               # 每秒请求数
  burst: 200              # 突发请求数
  cleanup_interval: 1m

# 监控配置
metrics:
  enabled: true
  path: "/metrics"
  port: 9090

# 健康检查配置
health:
  path: "/health"
  timeout: 5s
```

### 5.2 TLD WHOIS 服务器映射 (`config/tld_whois_servers.yaml`)

```yaml
# TLD WHOIS 服务器映射
# 格式: tld: server_address

servers:
  # 通用顶级域名 (gTLD)
  ".com": "whois.verisign-grs.com"
  ".net": "whois.verisign-grs.com"
  ".org": "whois.pir.org"
  ".info": "whois.afilias.net"
  ".biz": "whois.neulevel.biz"
  ".name": "whois.nic.name"
  ".pro": "whois.registrypro.pro"

  # 国家代码顶级域名 (ccTLD)
  ".cn": "whois.cnnic.cn"
  ".uk": "whois.nic.uk"
  ".de": "whois.denic.de"
  ".fr": "whois.nic.fr"
  ".jp": "whois.jprs.jp"
  ".au": "whois.auda.org.au"
  ".ca": "whois.cira.ca"
  ".ru": "whois.tcinet.ru"
  ".br": "whois.registro.br"
  ".in": "whois.inregistry.net"

  # 新通用顶级域名 (new gTLD)
  ".app": "whois.nic.google"
  ".dev": "whois.nic.google"
  ".io": "whois.nic.io"
  ".co": "whois.nic.co"
  ".me": "whois.nic.me"

# 备用服务器映射
fallback_servers:
  ".com":
    - "whois.verisign-grs.com"
    - "whois.internic.net"
  ".net":
    - "whois.verisign-grs.com"
    - "whois.internic.net"
```

### 5.3 配置加载器 (`internal/config/loader.go`)

```go
package config

import (
    "strings"
    "github.com/spf13/viper"
)

// Config 表示应用配置
type Config struct {
    Server    ServerConfig    `mapstructure:"server"`
    Engine    EngineConfig    `mapstructure:"engine"`
    Cache     CacheConfig     `mapstructure:"cache"`
    Log       LogConfig       `mapstructure:"log"`
    RateLimit RateLimitConfig `mapstructure:"ratelimit"`
    Metrics   MetricsConfig   `mapstructure:"metrics"`
    Health    HealthConfig    `mapstructure:"health"`
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
    v := viper.New()

    // 设置默认值
    setDefaults(v)

    // 读取配置文件
    if configPath != "" {
        v.SetConfigFile(configPath)
    } else {
        v.SetConfigName("config")
        v.SetConfigType("yaml")
        v.AddConfigPath("./config")
        v.AddConfigPath("$HOME/.go-whois")
        v.AddConfigPath("/etc/go-whois")
    }

    // 读取环境变量
    v.SetEnvPrefix("WHOIS")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    // 读取配置
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
    }

    // 解析配置
    var config Config
    if err := v.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

---

## 6. 错误处理规范

### 6.1 错误类型定义 (`internal/errors/errors.go`)

```go
package errors

import (
    "fmt"
    "net/http"
)

// ErrorCode 表示错误代码
type ErrorCode string

const (
    // 业务错误码
    ErrCodeInvalidDomain    ErrorCode = "INVALID_DOMAIN"
    ErrCodeDomainNotFound   ErrorCode = "DOMAIN_NOT_FOUND"
    ErrCodeQueryTimeout     ErrorCode = "QUERY_TIMEOUT"
    ErrCodeProtocolError    ErrorCode = "PROTOCOL_ERROR"
    ErrCodeRateLimited      ErrorCode = "RATE_LIMITED"
    ErrCodeBatchSizeExceeded ErrorCode = "BATCH_SIZE_EXCEEDED"

    // 系统错误码
    ErrCodeInternalError    ErrorCode = "INTERNAL_ERROR"
    ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
    ErrCodeConfigError      ErrorCode = "CONFIG_ERROR"
)

// AppError 表示应用错误
type AppError struct {
    Code       ErrorCode `json:"code"`
    Message    string    `json:"message"`
    Details    string    `json:"details,omitempty"`
    HTTPStatus int       `json:"-"`
    Err        error     `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
    return e.Err
}

// 预定义错误
var (
    ErrInvalidDomain = &AppError{
        Code:       ErrCodeInvalidDomain,
        Message:    "域名格式无效",
        HTTPStatus: http.StatusBadRequest,
    }

    ErrDomainNotFound = &AppError{
        Code:       ErrCodeDomainNotFound,
        Message:    "域名未注册",
        HTTPStatus: http.StatusNotFound,
    }

    ErrQueryTimeout = &AppError{
        Code:       ErrCodeQueryTimeout,
        Message:    "查询超时",
        HTTPStatus: http.StatusGatewayTimeout,
    }

    ErrProtocolError = &AppError{
        Code:       ErrCodeProtocolError,
        Message:    "协议查询失败",
        HTTPStatus: http.StatusBadGateway,
    }

    ErrRateLimited = &AppError{
        Code:       ErrCodeRateLimited,
        Message:    "请求过于频繁",
        HTTPStatus: http.StatusTooManyRequests,
    }

    ErrInternalError = &AppError{
        Code:       ErrCodeInternalError,
        Message:    "内部服务器错误",
        HTTPStatus: http.StatusInternalServerError,
    }
)

// NewInvalidDomainError 创建域名无效错误
func NewInvalidDomainError(domain string, err error) *AppError {
    return &AppError{
        Code:       ErrCodeInvalidDomain,
        Message:    "域名格式无效",
        Details:    fmt.Sprintf("域名 '%s' 不符合规范", domain),
        HTTPStatus: http.StatusBadRequest,
        Err:        err,
    }
}

// NewQueryTimeoutError 创建查询超时错误
func NewQueryTimeoutError(domain string, protocol string, err error) *AppError {
    return &AppError{
        Code:       ErrCodeQueryTimeout,
        Message:    "查询超时",
        Details:    fmt.Sprintf("查询域名 '%s' 使用协议 '%s' 超时", domain, protocol),
        HTTPStatus: http.StatusGatewayTimeout,
        Err:        err,
    }
}

// WrapInternalError 包装内部错误
func WrapInternalError(err error) *AppError {
    return &AppError{
        Code:       ErrCodeInternalError,
        Message:    "内部服务器错误",
        Details:    "请联系管理员",
        HTTPStatus: http.StatusInternalServerError,
        Err:        err,
    }
}
```

### 6.2 错误处理最佳实践

```go
// 1. 使用自定义错误类型
func (e *RDAP) Query(ctx context.Context, domain string) (*model.DomainInfo, error) {
    if err := validator.ValidateDomain(domain); err != nil {
        return nil, errors.NewInvalidDomainError(domain, err)
    }

    // 执行查询...
    if err != nil {
        return nil, errors.NewQueryTimeoutError(domain, "rdap", err)
    }

    return result, nil
}

// 2. 错误包装和传递
func (s *LookupServiceImpl) Lookup(ctx context.Context, req *engine.QueryRequest) (*model.DomainInfo, error) {
    result, err := s.engines[req.Protocol].Query(ctx, req.Domain)
    if err != nil {
        // 包装错误，添加上下文信息
        return nil, fmt.Errorf("查询失败: %w", err)
    }
    return result, nil
}

// 3. HTTP 错误响应
func (h *Handler) handleAppError(c *gin.Context, err error) {
    var appErr *errors.AppError
    if errors.As(err, &appErr) {
        c.JSON(appErr.HTTPStatus, model.APIResponse{
            Success: false,
            Error: &model.APIError{
                Code:    string(appErr.Code),
                Message: appErr.Message,
                Details: appErr.Details,
            },
            RequestID: c.GetString("request_id"),
        })
        return
    }

    // 未知错误
    c.JSON(http.StatusInternalServerError, model.APIResponse{
        Success: false,
        Error: &model.APIError{
            Code:    string(errors.ErrCodeInternalError),
            Message: "内部服务器错误",
        },
        RequestID: c.GetString("request_id"),
    })
}
```

---

## 7. 日志规范

### 7.1 日志配置 (`internal/config/config.go`)

```go
package config

// LogConfig 表示日志配置
type LogConfig struct {
    Level          string `mapstructure:"level"`
    Format         string `mapstructure:"format"`
    Output         string `mapstructure:"output"`
    FilePath       string `mapstructure:"file_path"`
    MaxSize        int    `mapstructure:"max_size"`
    MaxBackups     int    `mapstructure:"max_backups"`
    MaxAge         int    `mapstructure:"max_age"`
    Compress       bool   `mapstructure:"compress"`
}
```

### 7.2 日志初始化 (`internal/infra/logger/logger.go`)

```go
package logger

import (
    "os"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinish/lumberjack.v2"
    "go-whois/internal/config"
)

var log *zap.Logger

// Init 初始化日志
func Init(cfg config.LogConfig) error {
    // 设置日志级别
    level := parseLevel(cfg.Level)

    // 设置编码器
    encoderConfig := zap.NewProductionEncoderConfig()
    encoderConfig.TimeKey = "timestamp"
    encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

    var encoder zapcore.Encoder
    if cfg.Format == "json" {
        encoder = zapcore.NewJSONEncoder(encoderConfig)
    } else {
        encoder = zapcore.NewConsoleEncoder(encoderConfig)
    }

    // 设置输出
    var writeSyncer zapcore.WriteSyncer
    if cfg.Output == "file" {
        writer := &lumberjack.Logger{
            Filename:   cfg.FilePath,
            MaxSize:    cfg.MaxSize,
            MaxBackups: cfg.MaxBackups,
            MaxAge:     cfg.MaxAge,
            Compress:   cfg.Compress,
        }
        writeSyncer = zapcore.AddSync(writer)
    } else {
        writeSyncer = zapcore.AddSync(os.Stdout)
    }

    // 创建核心
    core := zapcore.NewCore(encoder, writeSyncer, level)
    log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

    return nil
}

// GetLogger 获取日志实例
func GetLogger() *zap.Logger {
    return log
}

// WithContext 添加上下文字段
func WithContext(fields ...zap.Field) *zap.Logger {
    return log.With(fields...)
}
```

### 7.3 日志使用规范

```go
// 1. 结构化日志
logger.Info("查询开始",
    zap.String("domain", domain),
    zap.String("protocol", string(protocol)),
    zap.String("request_id", requestID),
)

// 2. 错误日志
logger.Error("查询失败",
    zap.String("domain", domain),
    zap.String("protocol", string(protocol)),
    zap.Error(err),
    zap.Duration("duration", duration),
)

// 3. 性能日志
logger.Info("查询完成",
    zap.String("domain", domain),
    zap.String("protocol", string(protocol)),
    zap.Int64("duration_ms", duration.Milliseconds()),
    zap.String("data_source", dataSource),
)

// 4. 调试日志
logger.Debug("RDAP Bootstrap 查询",
    zap.String("tld", tld),
    zap.String("endpoint", endpoint),
)
```

### 7.4 日志字段规范

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `domain` | string | 查询的域名 | `"example.com"` |
| `protocol` | string | 查询协议 | `"rdap"` / `"whois"` |
| `request_id` | string | 请求 ID | `"req_abc123"` |
| `duration_ms` | int64 | 查询耗时（毫秒） | `1234` |
| `data_source` | string | 数据来源 | `"live"` / `"cache"` |
| `error` | error | 错误信息 | `<nil>` |
| `status_code` | int | HTTP 状态码 | `200` |
| `client_ip` | string | 客户端 IP | `"192.168.1.1"` |
| `user_agent` | string | 用户代理 | `"go-whois/1.0"` |

---

## 8. 测试规范

### 8.1 测试文件组织

```
internal/
├── engine/
│   ├── engine.go
│   ├── engine_test.go           # 接口测试
│   ├── rdap.go
│   ├── rdap_test.go             # RDAP 引擎单元测试
│   ├── rdap_test_helpers.go     # 测试辅助函数
│   ├── whois.go
│   └── whois_test.go            # WHOIS 引擎单元测试
├── service/
│   ├── lookup.go
│   └── lookup_test.go           # 查询服务单元测试
└── cache/
    ├── cache.go
    ├── memory.go
    └── memory_test.go           # 内存缓存单元测试

test/
├── fixtures/                    # 测试固件
│   ├── whois_response.txt
│   └── rdap_response.json
└── integration/                 # 集成测试
    └── lookup_test.go
```

### 8.2 单元测试规范

```go
package engine

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go-whois/internal/model"
)

// TestRDAPQuery 测试 RDAP 查询
func TestRDAPQuery(t *testing.T) {
    // 使用表驱动测试
    tests := []struct {
        name     string
        domain   string
        wantErr  bool
        errMsg   string
    }{
        {
            name:    "有效域名",
            domain:  "example.com",
            wantErr: false,
        },
        {
            name:    "无效域名格式",
            domain:  "invalid-domain",
            wantErr: true,
            errMsg:  "域名格式无效",
        },
        {
            name:    "空域名",
            domain:  "",
            wantErr: true,
            errMsg:  "域名不能为空",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 准备
            engine := NewRDAP(config, logger)
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()

            // 执行
            result, err := engine.Query(ctx, tt.domain)

            // 断言
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tterrMsg)
                assert.Nil(t, result)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, result)
                assert.Equal(t, tt.domain, result.DomainName)
            }
        })
    }
}

// TestRDAPQueryWithMock 测试使用 Mock
func TestRDAPQueryWithMock(t *testing.T) {
    // 创建 Mock HTTP 客户端
    mockClient := &MockHTTPClient{
        Response: &http.Response{
            StatusCode: 200,
            Body:       io.NopCloser(strings.NewReader(`{"domain":"example.com"}`)),
        },
    }

    engine := &RDAP{
        client: mockClient,
        logger: zaptest.NewLogger(t),
    }

    // 执行测试
    result, err := engine.Query(context.Background(), "example.com")

    // 断言
    require.NoError(t, err)
    assert.Equal(t, "example.com", result.DomainName)
    assert.Equal(t, "rdap", result.QueryProtocol)
}
```

### 8.3 集成测试规范

```go
package integration

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go-whois/internal/service"
)

// TestLookupIntegration 测试查询服务集成
func TestLookupIntegration(t *testing.T) {
    // 跳过短测试
    if testing.Short() {
        t.Skip("跳过集成测试")
    }

    // 初始化服务
    svc, err := setupTestService(t)
    require.NoError(t, err)

    // 测试 RDAP 查询
    t.Run("RDAP查询", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        result, err := svc.Lookup(ctx, &engine.QueryRequest{
            Domain:   "google.com",
            Protocol: engine.ProtocolRDAP,
        })

        require.NoError(t, err)
        assert.Equal(t, "google.com", result.DomainName)
        assert.Equal(t, "rdap", result.QueryProtocol)
        assert.NotEmpty(t, result.Registration.Registrar.Name)
    })

    // 测试 WHOIS 查询
    t.Run("WHOIS查询", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        result, err := svc.Lookup(ctx, &engine.QueryRequest{
            Domain:   "example.com",
            Protocol: engine.ProtocolWHOIS,
        })

        require.NoError(t, err)
        assert.Equal(t, "example.com", result.DomainName)
        assert.Equal(t, "whois", result.QueryProtocol)
    })
}
```

### 8.4 测试工具函数

```go
// test/helpers_test.go
package test

import (
    "os"
    "path/filepath"
    "testing"
)

// LoadFixture 加载测试固件
func LoadFixture(t *testing.T, filename string) string {
    t.Helper()

    path := filepath.Join("fixtures", filename)
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("加载固件失败: %v", err)
    }

    return string(data)
}

// SetupTestDB 设置测试数据库（如果需要）
func SetupTestDB(t *testing.T) {
    t.Helper()
    // 测试数据库设置
}

// CleanupTestDB 清理测试数据库
func CleanupTestDB(t *testing.T) {
    t.Helper()
    // 测试数据库清理
}
```

### 8.5 测试覆盖率要求

| 模块 | 最低覆盖率 | 目标覆盖率 |
|------|------------|------------|
| `internal/engine` | 80% | 90% |
| `internal/service` | 75% | 85% |
| `internal/cache` | 70% | 80% |
| `internal/api` | 70% | 80% |
| `pkg/` | 60% | 70% |

### 8.6 测试运行命令

```bash
# 运行所有测试
go test ./...

# 运行单元测试（跳过集成测试）
go test -short ./...

# 运行集成测试
go test -tags=integration ./test/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 运行基准测试
go test -bench=. ./internal/engine/...
```

---

## 9. 代码风格和命名规范

### 9.1 通用规范

#### 9.1.1 文件编码
- 所有源代码文件使用 UTF-8 编码
- 文件末尾保留一个空行

#### 9.1.2 缩进和空格
- 使用 Tab 缩进，不要使用空格
- 运算符两侧加空格
- 逗号、分号后加空格

#### 9.1.3 行长度
- 建议每行不超过 120 字符
- 超长行在合理位置换行

### 9.2 命名规范

#### 9.2.1 包名
```go
// 推荐：小写单词，简短有意义
package engine
package cache
package validator

// 避免：下划线、混合大小写
package engine_pool      // 不推荐
package EnginePool       // 不推荐
```

#### 9.2.2 结构体和接口
```go
// 结构体：驼峰命名
type DomainInfo struct { ... }
type LookupService struct { ... }

// 接口：动词 + 名词
type Engine interface { ... }
type CacheManager interface { ... }
type Normalizer interface { ... }
```

#### 9.2.3 函数和方法
```go
// 函数：驼峰命名
func NewLookupService(...) *LookupService { ... }
func GenerateCacheKey(...) string { ... }

// 方法：驼峰命名
func (s *LookupService) Lookup(...) (*DomainInfo, error) { ... }
func (c *MemoryCache) Get(...) (*DomainInfo, error) { ... }
```

#### 9.2.4 变量
```go
// 局部变量：驼峰命名
domainName := "example.com"
queryTimeout := 30 * time.Second

// 全局变量：驼峰命名
var DefaultTimeout = 30 * time.Second
var ErrInvalidDomain = errors.New("invalid domain")
```

#### 9.2.5 常量
```go
// 常量：全大写 + 下划线
const (
    ProtocolRDAP  = "rdap"
    ProtocolWHOIS = "whois"
    ProtocolAuto  = "auto"
)

// 枚举常量：类型 + 值
type Protocol string

const (
    ProtocolRDAP  Protocol = "rdap"
    ProtocolWHOIS Protocol = "whois"
)
```

#### 9.2.6 数据库字段
```go
// 数据库字段：蛇形命名
type Domain struct {
    DomainName    string    `db:"domain_name"`
    RegistrarName string    `db:"registrar_name"`
    CreatedAt     time.Time `db:"created_at"`
    UpdatedAt     time.Time `db:"updated_at"`
}
```

### 9.3 注释规范

```go
// 1. 包注释
// Package engine 实现域名 WHOIS/RDAP 查询引擎。
//
// 该包提供了域名注册信息的查询功能，支持 RDAP 和 WHOIS 两种协议。
// RDAP 是推荐的查询协议，WHOIS 作为备用方案。
package engine

// 2. 结构体注释
// DomainInfo 表示域名查询结果。
//
// 包含域名的基本注册信息、注册商信息、域名状态等。
// 所有时间字段使用 UTC 时区。
type DomainInfo struct { ... }

// 3. 函数注释
// Lookup 执行单域名查询。
//
// 根据请求中的协议参数选择查询引擎，如果未指定则使用默认策略（RDAP 优先）。
// 查询结果会自动标准化为统一的 JSON 格式。
//
// 参数：
//   - ctx: 上下文，用于控制超时和取消
//   - req: 查询请求，包含域名和协议参数
//
// 返回：
//   - *DomainInfo: 查询结果
//   - error: 查询失败时返回错误
func (s *LookupServiceImpl) Lookup(ctx context.Context, req *QueryRequest) (*DomainInfo, error) { ... }

// 4. 行内注释
func (s *LookupServiceImpl) Lookup(ctx context.Context, req *QueryRequest) (*DomainInfo, error) {
    // 检查缓存
    cacheKey := GenerateCacheKey(req.Protocol, req.Domain)
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        return cached, nil
    }

    // 执行查询
    result, err := s.engines[req.Protocol].Query(ctx, req.Domain)
    if err != nil {
        return nil, fmt.Errorf("查询失败: %w", err)
    }

    // 写入缓存
    if err := s.cache.Set(ctx, cacheKey, result, s.config.Cache.TTL); err != nil {
        logger.Warn("写入缓存失败", zap.Error(err))
    }

    return result, nil
}
```

### 9.4 代码组织

```go
// 文件内代码组织顺序：
// 1. 包声明
// 2. 导入声明（分组：标准库、第三方库、内部包）
// 3. 常量定义
// 4. 类型定义
// 5. 全局变量
// 6. init 函数
// 7. 构造函数
// 8. 公开方法
// 9. 私有方法

package engine

import (
    // 标准库
    "context"
    "fmt"
    "time"

    // 第三方库
    "go.uber.org/zap"

    // 内部包
    "go-whois/internal/config"
    "go-whois/internal/model"
)

const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)

type RDAP struct {
    client *http.Client
    logger *zap.Logger
    config *config.RDAPConfig
}

func NewRDAP(config *config.RDAPConfig, logger *zap.Logger) *RDAP {
    return &RDAP{
        client: &http.Client{Timeout: config.Timeout},
        logger: logger,
        config: config,
    }
}

func (r *RDAP) Query(ctx context.Context, domain string) (*model.DomainInfo, error) {
    // 实现...
}

func (r *RDAP) getBootstrapURL(tld string) (string, error) {
    // 私有方法实现...
}
```

### 9.5 代码格式化工具

```bash
# 使用 gofmt 格式化代码
gofmt -w .

# 使用 goimports 管理导入
goimports -w .

# 使用 golangci-lint 进行代码检查
golangci-lint run
```

### 9.6 golangci-lint 配置 (`.golangci.yml`)

```yaml
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gocritic
    - gocyclo
    - gosec
    - misspell
    - unconvert
    - whitespace
    - bodyclose
    - contextcheck
    - durationcheck
    - errname
    - errorlint
    - exhaustive
    - exportloopref
    - makezero
    - nilerr
    - prealloc
    - predeclared
    - promlinter
    - revive
    - rowserrcheck
    - sqlclosecheck
    - tparallel
    - unparam

linters-settings:
  gocyclo:
    min-complexity: 15
  govet:
    check-shadowing: true
  revive:
    rules:
      - name: blank-imports
      - name: context-as-argument
      - name: dot-imports
      - name: error-return
      - name: error-strings
      - name: error-naming
      - name: exported
      - name: if-return
      - name: increment-decrement
      - name: var-naming
      - name: package-comments

issues:
  max-issues-per-linter: 50
  max-same-issues: 5
  exclude-rules:
    - path: _test\.go
      linters:
        - gocritic
        - gocyclo
```

---

## 10. 依赖管理策略

### 10.1 go.mod 配置

```go
module go-whois

go 1.21

require (
    // CLI 框架
    github.com/spf13/cobra v1.8.0

    // HTTP 框架
    github.com/gin-gonic/gin v1.9.1

    // 配置管理
    github.com/spf13/viper v1.18.2

    // 日志库
    go.uber.org/zap v1.26.0
    gopkg.in/natefinish/lumberjack.v2 v2.2.1

    // 缓存（可选）
    github.com/allegro/bigcache/v3 v3.1.0

    // 测试
    github.com/stretchr/testify v1.8.4

    // 工具库
    github.com/google/uuid v1.5.0
)

require (
    // 间接依赖...
)
```

### 10.2 依赖分类

| 类别 | 包名 | 用途 | 是否必须 |
|------|------|------|----------|
| CLI | `github.com/spf13/cobra` | CLI 框架 | 是 |
| HTTP | `github.com/gin-gonic/gin` | HTTP 框架 | 是 |
| 配置 | `github.com/spf13/viper` | 配置管理 | 是 |
| 日志 | `go.uber.org/zap` | 结构化日志 | 是 |
| 日志 | `gopkg.in/natefinish/lumberjack.v2` | 日志轮转 | 是 |
| 缓存 | `github.com/allegro/bigcache/v3` | 内存缓存 | 可选 |
| 测试 | `github.com/stretchr/testify` | 测试断言 | 开发 |
| 工具 | `github.com/google/uuid` | UUID 生成 | 是 |

### 10.3 依赖版本策略

1. **主版本升级**：谨慎升级，需要全面测试
2. **次版本升级**：定期升级，确保兼容性
3. **补丁版本**：及时升级，修复安全漏洞

### 10.4 依赖更新命令

```bash
# 查看可更新的依赖
go list -m -u all

# 更新单个依赖
go get github.com/spf13/cobra@v1.8.0

# 更新所有依赖
go get -u ./...

# 清理未使用的依赖
go mod tidy

# 验证依赖
go mod verify

# 下载依赖到本地缓存
go mod download
```

### 10.5 依赖安全检查

```bash
# 使用 govulncheck 检查安全漏洞
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# 使用 go-mod-outdated 检查过时依赖
go install github.com/psampaz/go-mod-outdated@latest
go-mod-outdated -update
```

---

## 11. 开发环境搭建指南

### 11.1 环境要求

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 编程语言 |
| Git | 2.30+ | 版本控制 |
| Make | 4.0+ | 构建工具（可选） |
| Docker | 20.10+ | 容器化（可选） |
| golangci-lint | 1.55+ | 代码检查工具 |

### 11.2 安装 Go

#### Windows
```powershell
# 使用 Chocolatey 安装
choco install golang

# 或者下载安装包
# https://go.dev/dl/
```

#### Linux/macOS
```bash
# 使用官方安装脚本
wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz

# 添加到 PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

### 11.3 安装开发工具

```bash
# 安装 golangci-lint
# Windows (使用 Chocolatey)
choco install golangci-lint

# Linux/macOS
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# 安装 goimports
go install golang.org/x/tools/cmd/goimports@latest

# 安装 govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# 安装 go-mod-outdated
go install github.com/psampaz/go-mod-outdated@latest
```

### 11.4 项目初始化

```bash
# 克隆项目
git clone https://github.com/your-org/go-whois.git
cd go-whois

# 安装依赖
go mod download

# 验证依赖
go mod verify

# 运行测试
go test ./...

# 构建项目
go build -o bin/go-whois .

# 运行项目
./bin/go-whois lookup example.com
```

### 11.5 IDE 配置

#### VS Code

安装扩展：
- Go (golang.go)
- Go Test Explorer
- Go Doc
- Error Lens

配置 `.vscode/settings.json`：
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "--fast"
    ],
    "go.testFlags": [
        "-v",
        "-count=1"
    ],
    "go.coverOnSave": true,
    "go.coverageDecorator": {
        "type": "highlight",
        "coveredColor": "rgba(64,128,64,0.2)",
        "uncoveredColor": "rgba(128,64,64,0.2)"
    },
    "editor.formatOnSave": true,
    "[go]": {
        "editor.defaultFormatter": "golang.go"
    }
}
```

配置 `.vscode/launch.json`：
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Lookup",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}",
            "args": [
                "lookup",
                "example.com"
            ]
        },
        {
            "name": "Launch Server",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}",
            "args": [
                "serve"
            ]
        }
    ]
}
```

#### GoLand

1. 打开项目目录
2. 配置 Go SDK：File -> Project Structure -> SDKs -> + -> Go SDK
3. 配置代码风格：Editor -> Code Style -> Go
4. 配置运行配置：Run -> Edit Configurations

### 11.6 Makefile

```makefile
# Go-WHOIS Makefile

# 变量
APP_NAME := go-whois
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | cut -d ' ' -f 3)

# 构建标志
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test lint fmt vet run help

all: clean lint test build

## build: 构建项目
build:
	@echo "构建 $(APP_NAME)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/$(APP_NAME) .
	@echo "构建完成: bin/$(APP_NAME)"

## clean: 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -rf bin/
	@go clean

## test: 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

## test-short: 运行单元测试（跳过集成测试）
test-short:
	@echo "运行单元测试..."
	@go test -short -v ./...

## test-integration: 运行集成测试
test-integration:
	@echo "运行集成测试..."
	@go test -tags=integration -v ./test/...

## coverage: 生成覆盖率报告
coverage:
	@echo "生成覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告: coverage.html"

## lint: 代码检查
lint:
	@echo "运行代码检查..."
	@golangci-lint run

## fmt: 格式化代码
fmt:
	@echo "格式化代码..."
	@gofmt -w .
	@goimports -w .

## vet: 静态分析
vet:
	@echo "运行静态分析..."
	@go vet ./...

## run: 运行项目
run: build
	@./bin/$(APP_NAME) $(ARGS)

## serve: 启动 HTTP 服务
serve: build
	@./bin/$(APP_NAME) serve

## lookup: 查询域名
lookup: build
	@./bin/$(APP_NAME) lookup $(DOMAIN)

## docker-build: 构建 Docker 镜像
docker-build:
	@echo "构建 Docker 镜像..."
	@docker build -t $(APP_NAME):$(VERSION) .

## docker-run: 运行 Docker 容器
docker-run:
	@echo "运行 Docker 容器..."
	@docker run -p 8080:8080 $(APP_NAME):$(VERSION)

## deps: 下载依赖
deps:
	@echo "下载依赖..."
	@go mod download

## deps-update: 更新依赖
deps-update:
	@echo "更新依赖..."
	@go get -u ./...
	@go mod tidy

## deps-check: 检查依赖更新
deps-check:
	@echo "检查依赖更新..."
	@go list -m -u all

## vuln: 安全漏洞检查
vuln:
	@echo "安全漏洞检查..."
	@govulncheck ./...

## help: 显示帮助
help:
	@echo "Go-WHOIS Makefile"
	@echo ""
	@echo "用法: make [target]"
	@echo ""
	@echo "目标:"
	@echo "  all              清理、检查、测试、构建"
	@echo "  build            构建项目"
	@echo "  clean            清理构建产物"
	@echo "  test             运行测试"
	@echo "  test-short       运行单元测试"
	@echo "  test-integration 运行集成测试"
	@echo "  coverage         生成覆盖率报告"
	@echo "  lint             代码检查"
	@echo "  fmt              格式化代码"
	@echo "  vet              静态分析"
	@echo "  run              运行项目"
	@echo "  serve            启动 HTTP 服务"
	@echo "  lookup           查询域名"
	@echo "  docker-build     构建 Docker 镜像"
	@echo "  docker-run       运行 Docker 容器"
	@echo "  deps             下载依赖"
	@echo "  deps-update      更新依赖"
	@echo "  deps-check       检查依赖更新"
	@echo "  vuln             安全漏洞检查"
	@echo "  help             显示帮助"
```

### 11.7 Docker 支持

#### Dockerfile
```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/go-whois .

# 运行阶段
FROM alpine:latest

# 安装 CA 证书
RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/go-whois .

# 复制配置文件
COPY config/ ./config/

# 暴露端口
EXPOSE 8080

# 运行
ENTRYPOINT ["./go-whois"]
CMD ["serve"]
```

#### docker-compose.yml
```yaml
version: '3.8'

services:
  go-whois:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./config:/app/config
    environment:
      - WHOIS_SERVER_HTTP_PORT=8080
      - WHOIS_LOG_LEVEL=info
      - WHOIS_LOG_FORMAT=json
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

### 11.8 开发工作流

```bash
# 1. 创建功能分支
git checkout -b feature/rdap-bootstrap

# 2. 开发功能
# 编写代码...

# 3. 运行测试
make test-short

# 4. 代码检查
make lint
make fmt

# 5. 提交代码
git add .
git commit -m "feat(engine): 实现 RDAP Bootstrap 查询"

# 6. 推送分支
git push origin feature/rdap-bootstrap

# 7. 创建 Pull Request
# 在 GitHub/GitLab 上创建 PR

# 8. 代码审查
# 等待审查通过

# 9. 合并到主分支
# 审查通过后合并
```

---

## 附录

### A. 参考资料

1. [Go 官方文档](https://go.dev/doc/)
2. [Effective Go](https://go.dev/doc/effective_go)
3. [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
4. [Cobra 文档](https://cobra.dev/)
5. [Gin 文档](https://gin-gonic.com/)
6. [Viper 文档](https://github.com/spf13/viper)
7. [Zap 文档](https://pkg.go.dev/go.uber.org/zap)
8. [RFC 7483 - RDAP](https://tools.ietf.org/html/rfc7483)
9. [RFC 3912 - WHOIS](https://tools.ietf.org/html/rfc3912)

### B. 常见问题

#### Q1: 如何添加新的 TLD WHOIS 服务器？
A1: 编辑 `config/tld_whois_servers.yaml` 文件，在 `servers` 部分添加新的映射。

#### Q2: 如何切换日志级别？
A2: 修改 `config/config.yaml` 中的 `log.level` 配置，或设置环境变量 `WHOIS_LOG_LEVEL`。

#### Q3: 如何启用 Redis 缓存？
A3: 在 `config/config.yaml` 中设置 `cache.redis.enabled: true`，并配置 Redis 连接信息。

#### Q4: 如何添加新的查询协议？
A4: 在 `internal/engine/` 目录下创建新的引擎实现，实现 `Engine` 接口，然后在 `LookupService` 中注册。

---

> 文档维护：本文档应随项目开发持续更新，确保与代码实现保持一致。
