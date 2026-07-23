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
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/suguer/go-whois/pkg/model"
	"github.com/suguer/go-whois/pkg/validator"
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
	readyCh     chan struct{} // RDAP Bootstrap 加载完成信号
	readyOnce   sync.Once
	closed      bool
}

// cacheEntry 缓存条目
type cacheEntry struct {
	data      *model.DomainInfo
	expiresAt time.Time
}

// options 客户端配置选项
type options struct {
	protocol          model.QueryProtocol
	timeout           time.Duration
	cacheEnabled      bool
	cacheMaxSize      int
	cacheTTL          time.Duration
	rdapBootstrap     string
	rdapBootstrapFile string
	whoisConfigFile   string
	userAgent         string
	logger            Logger
	includeRaw        bool
	httpClient        *http.Client
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

// WithRDAPBootstrapFile 设置本地 RDAP Bootstrap 文件路径
// 优先从本地文件加载，不再从网络下载
func WithRDAPBootstrapFile(path string) Option {
	return func(o *options) {
		o.rdapBootstrapFile = path
	}
}

// WithWHOISConfigFile 设置本地 WHOIS 服务器配置文件路径
// 指定后将从该文件加载 TLD -> WHOIS 服务器映射
func WithWHOISConfigFile(path string) Option {
	return func(o *options) {
		o.whoisConfigFile = path
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

// WithHTTPClient 设置自定义 HTTP 客户端
// 可用于配置代理、TLS、连接池等
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.httpClient = client
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

	// 创建 HTTP 客户端
	httpClient := o.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: o.timeout}
	}

	c := &Client{
		options:     o,
		httpClient:  httpClient,
		rdapCache:   make(map[string]string),
		whoisCache:  make(map[string]string),
		resultCache: make(map[string]*cacheEntry),
		logger:      o.logger,
		readyCh:     make(chan struct{}),
	}

	// 初始化 WHOIS 客户端
	whoisOpts := []WHOISOption{
		WithWSTimeout(o.timeout),
		WithWSLogger(o.logger),
	}
	if o.whoisConfigFile != "" {
		whoisOpts = append(whoisOpts, WithWSConfigFile(o.whoisConfigFile))
	}
	c.whoisClient = NewWHOISClient(whoisOpts...)

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

// WaitForReady 等待客户端就绪（RDAP Bootstrap 加载完成）
// 在首次查询前调用可确保 RDAP 端点已加载，避免降级到 WHOIS
func (c *Client) WaitForReady(ctx context.Context) error {
	select {
	case <-c.readyCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// BatchResult 表示批量查询的单个结果
type BatchResult struct {
	Domain string
	Result *model.DomainInfo
	Error  error
}

// BatchLookup 批量查询域名信息
// maxConcurrency: 最大并发数 (建议 5-10)
// 返回结果顺序与输入域名顺序一致
func (c *Client) BatchLookup(ctx context.Context, domains []string, maxConcurrency int) []BatchResult {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}

	results := make([]BatchResult, len(domains))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, domain := range domains {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, d string) {
			defer wg.Done()
			defer func() { <-sem }()

			result, err := c.LookupWithContext(ctx, d)
			results[idx] = BatchResult{
				Domain: d,
				Result: result,
				Error:  err,
			}
		}(i, domain)
	}

	wg.Wait()
	return results
}

// lookupWithProtocol 内部查询实现
func (c *Client) lookupWithProtocol(ctx context.Context, domain string, protocol model.QueryProtocol) (*model.DomainInfo, error) {
	// 检查客户端是否已关闭
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, &model.Error{
			Code:    model.ErrCodeInternalError,
			Message: "客户端已关闭",
		}
	}
	c.mu.RUnlock()

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
		// RDAP 优先，仅在 5xx/超时/协议错误时回退到 WHOIS
		result, err = c.queryRDAP(ctx, domain)
		if err != nil {
			// 检查是否应该回退到 WHOIS
			if modelErr, ok := err.(*model.Error); ok {
				switch modelErr.Code {
				case model.ErrCodeQueryTimeout, model.ErrCodeProtocolError, model.ErrCodeServiceUnavailable:
					c.logger.Warn("RDAP 查询失败，回退到 WHOIS", "domain", domain, "error", err)
					result, err = c.queryWHOIS(ctx, domain)
				}
				// DOMAIN_NOT_FOUND 等其他错误直接返回
			}
		}
	default:
		return nil, &model.Error{
			Code:    model.ErrCodeUnsupportedProtocol,
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

// getFromCache 从缓存获取（返回副本，避免共享指针）
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

	// 返回浅拷贝，避免调用方修改缓存数据
	data := *entry.data
	data.DataSource = string(model.DataSourceCache)
	return &data, true
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
		// 如果已经是 *model.Error，直接返回（保留原始错误码）
		if modelErr, ok := err.(*model.Error); ok {
			return nil, modelErr
		}
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
	return validator.ValidateDomain(domain)
}

// normalizeDomain 规范化域名
func normalizeDomain(domain string) string {
	return validator.NormalizeDomain(domain)
}

// Close 清理客户端资源
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// 等待 RDAP Bootstrap 加载完成（防止 goroutine 泄漏）
	<-c.readyCh

	// 清空缓存
	c.mu.Lock()
	c.resultCache = make(map[string]*cacheEntry)
	c.mu.Unlock()

	// 关闭 HTTP 连接
	c.httpClient.CloseIdleConnections()

	// 关闭 WHOIS 客户端
	if c.whoisClient != nil {
		c.whoisClient.Close()
	}

	c.logger.Info("客户端已关闭")
	return nil
}
