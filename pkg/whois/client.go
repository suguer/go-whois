// Package whois 提供了 Go-WHOIS 的公共客户端 API。
// 第三方开发者可以通过这个包来查询域名的 WHOIS/RDAP 信息。
//
// 基本用法:
//
//	client := whois.NewClient()
//	result, err := client.Lookup("example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Domain: %s, Registrar: %s\n", result.DomainName, result.RegistrarName)
//
// 使用选项配置:
//
//	client := whois.NewClient(
//	    whois.WithProtocol(model.ProtocolRDAP),
//	    whois.WithTimeout(15*time.Second),
//	    whois.WithCache(true, 1000, time.Hour),
//	)
//
// 自定义日志:
//
//	client := whois.NewClient(
//	    whois.WithLogger(myLogger),
//	)
package whois

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/suguer/go-whois/pkg/model"
)

// Logger 定义日志接口，允许用户自定义日志实现
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// defaultLogger 使用标准库 log 的默认实现
type defaultLogger struct {
	logger *log.Logger
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		logger: log.New(os.Stderr, "[go-whois] ", log.LstdFlags),
	}
}

func (l *defaultLogger) Debug(msg string, keysAndValues ...interface{}) {
	// 默认不输出 debug
}

func (l *defaultLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("INFO: %s %v", msg, keysAndValues)
}

func (l *defaultLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("WARN: %s %v", msg, keysAndValues)
}

func (l *defaultLogger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("ERROR: %s %v", msg, keysAndValues)
}

// Client 是 Go-WHOIS 的公共客户端
type Client struct {
	mu          sync.RWMutex
	options     *options
	httpClient  *http.Client
	rdapCache   map[string]string // TLD -> RDAP endpoint
	whoisCache  map[string]string // TLD -> WHOIS server
	whoisClient *WHOISClient      // WHOIS 查询客户端
	resultCache map[string]*cacheEntry
	logger      Logger
}

// cacheEntry 缓存条目
type cacheEntry struct {
	data      *model.DomainInfo
	expiresAt time.Time
}

// options 客户端配置选项
type options struct {
	protocol        model.QueryProtocol
	timeout         time.Duration
	cacheEnabled    bool
	cacheMaxSize    int
	cacheTTL        time.Duration
	rdapBootstrap   string
	userAgent       string
	logger          Logger
	includeRaw      bool
}

// Option 定义配置选项函数类型
type Option func(*options)

// WithProtocol 设置查询协议
func WithProtocol(protocol model.QueryProtocol) Option {
	return func(o *options) {
		o.protocol = protocol
	}
}

// WithTimeout 设置查询超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// WithCache 启用并配置缓存
func WithCache(enabled bool, maxSize int, ttl time.Duration) Option {
	return func(o *options) {
		o.cacheEnabled = enabled
		o.cacheMaxSize = maxSize
		o.cacheTTL = ttl
	}
}

// WithRDAPBootstrap 设置 RDAP Bootstrap URL
func WithRDAPBootstrap(url string) Option {
	return func(o *options) {
		o.rdapBootstrap = url
	}
}

// WithUserAgent 设置 User-Agent
func WithUserAgent(ua string) Option {
	return func(o *options) {
		o.userAgent = ua
	}
}

// WithLogger 设置自定义日志器
func WithLogger(logger Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithRawResponse 是否包含原始响应
func WithRawResponse(include bool) Option {
	return func(o *options) {
		o.includeRaw = include
	}
}

// NewClient 创建新的客户端实例
func NewClient(opts ...Option) *Client {
	o := &options{
		protocol:      model.ProtocolAuto,
		timeout:       10 * time.Second,
		cacheEnabled:  true,
		cacheMaxSize:  1000,
		cacheTTL:      time.Hour,
		rdapBootstrap: "https://data.iana.org/rdap/dns.json",
		userAgent:     "go-whois/1.0",
		logger:        newDefaultLogger(),
		includeRaw:    false,
	}

	for _, opt := range opts {
		opt(o)
	}

	c := &Client{
		options:     o,
		httpClient:  &http.Client{Timeout: o.timeout},
		rdapCache:   make(map[string]string),
		whoisCache:  make(map[string]string),
		resultCache: make(map[string]*cacheEntry),
		logger:      o.logger,
	}

	// 初始化 WHOIS 客户端
	c.whoisClient = NewWHOISClient(
		WithWSTimeout(o.timeout),
		WithWSLogger(o.logger),
	)

	// 异步加载 RDAP Bootstrap
	go c.loadRDAPBootstrap()

	return c
}

// Lookup 查询域名信息
// 默认使用 RDAP 协议，失败时自动回退到 WHOIS
func (c *Client) Lookup(domain string) (*model.DomainInfo, error) {
	return c.LookupWithContext(context.Background(), domain)
}

// LookupWithContext 带上下文的域名查询
func (c *Client) LookupWithContext(ctx context.Context, domain string) (*model.DomainInfo, error) {
	return c.lookupWithProtocol(ctx, domain, c.options.protocol)
}

// LookupWithProtocol 使用指定协议查询域名信息
func (c *Client) LookupWithProtocol(domain string, protocol model.QueryProtocol) (*model.DomainInfo, error) {
	return c.lookupWithProtocol(context.Background(), domain, protocol)
}

// lookupWithProtocol 内部查询实现
func (c *Client) lookupWithProtocol(ctx context.Context, domain string, protocol model.QueryProtocol) (*model.DomainInfo, error) {
	// 验证域名
	if err := validateDomain(domain); err != nil {
		return nil, &model.Error{
			Code:    model.ErrCodeInvalidDomain,
			Message: "域名格式无效",
			Details: err.Error(),
		}
	}

	// 规范化域名
	domain = normalizeDomain(domain)

	// 检查缓存
	cacheKey := string(protocol) + ":" + domain
	if c.options.cacheEnabled {
		if entry, ok := c.getFromCache(cacheKey); ok {
			c.logger.Debug("缓存命中", "domain", domain)
			return entry, nil
		}
	}

	// 执行查询
	var result *model.DomainInfo
	var err error

	switch protocol {
	case model.ProtocolRDAP:
		result, err = c.queryRDAP(ctx, domain)
	case model.ProtocolWHOIS:
		result, err = c.queryWHOIS(ctx, domain)
	case model.ProtocolAuto:
		// RDAP 优先
		result, err = c.queryRDAP(ctx, domain)
		if err != nil {
			c.logger.Warn("RDAP 查询失败，回退到 WHOIS", "domain", domain, "error", err)
			result, err = c.queryWHOIS(ctx, domain)
		}
	default:
		return nil, &model.Error{
			Code:    model.ErrCodeInvalidDomain,
			Message: "不支持的协议",
			Details: string(protocol),
		}
	}

	if err != nil {
		return nil, err
	}

	// 移除原始响应（如果不需要）
	if !c.options.includeRaw {
		result.RawResponse = nil
	}

	// 写入缓存
	if c.options.cacheEnabled {
		c.setCache(cacheKey, result)
	}

	return result, nil
}

// GetCacheStats 获取缓存统计信息
func (c *Client) GetCacheStats() model.CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return model.CacheStats{
		Enabled: c.options.cacheEnabled,
		Size:    len(c.resultCache),
	}
}

// ClearCache 清空缓存
func (c *Client) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.resultCache = make(map[string]*cacheEntry)
	c.logger.Info("缓存已清空")
}

// getFromCache 从缓存获取
func (c *Client) getFromCache(key string) (*model.DomainInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.resultCache[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.data, true
}

// setCache 写入缓存
func (c *Client) setCache(key string, data *model.DomainInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查缓存大小
	if len(c.resultCache) >= c.options.cacheMaxSize {
		c.evictOldest()
	}

	c.resultCache[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.options.cacheTTL),
	}
}

// evictOldest 淘汰最旧的缓存
func (c *Client) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.resultCache {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.resultCache, oldestKey)
	}
}

// queryWHOIS 执行 WHOIS 查询
func (c *Client) queryWHOIS(ctx context.Context, domain string) (*model.DomainInfo, error) {
	if c.whoisClient == nil {
		return nil, &model.Error{
			Code:    model.ErrCodeInternalError,
			Message: "WHOIS 客户端未初始化",
		}
	}

	result, err := c.whoisClient.Query(ctx, domain)
	if err != nil {
		return nil, &model.Error{
			Code:    model.ErrCodeProtocolError,
			Message: "WHOIS 查询失败",
			Details: err.Error(),
		}
	}

	// 保存原始响应（如果需要）
	if c.options.includeRaw {
		// 原始响应已经在 whoisClient.Query 中设置
	} else {
		result.RawResponse = nil
	}

	return result, nil
}

// validateDomain 验证域名格式
func validateDomain(domain string) error {
	if len(domain) == 0 {
		return fmt.Errorf("域名不能为空")
	}
	if len(domain) > 253 {
		return fmt.Errorf("域名长度超过253字符")
	}
	return nil
}

// normalizeDomain 规范化域名
func normalizeDomain(domain string) string {
	// 转换为小写
	domain = strings.ToLower(domain)
	// 去除末尾的点
	domain = strings.TrimSuffix(domain, ".")
	return domain
}

// Close 清理客户端资源
func (c *Client) Close() error {
	// 清空缓存
	c.ClearCache()
	return nil
}
