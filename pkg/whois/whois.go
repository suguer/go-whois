package whois

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/suguer/go-whois/pkg/model"
	"github.com/suguer/go-whois/pkg/validator"
	"gopkg.in/yaml.v3"
)

// 域名不存在关键词匹配
var domainNotFoundRegex = regexp.MustCompile(`(?i)(No match for|No matching record|The queried object does not exist:|DOMAIN NOT FOUND|No entries found|NOT FOUND|No Data Found|Status: AVAILABLE|Status: free|Domain Status: No Object Found)`)

// 查询限制关键词匹配
var queryLimitRegex = regexp.MustCompile(`(?i)(Queried interval is too short|Your access is too fast,please try again later|Query rate exceeded|please try again later|rate limit exceeded|Too many requests|Request denied|Query limit exceeded)`)

// checkWHOISResponseError 检查 WHOIS 响应中的错误关键词
// 返回错误类型和错误消息
func checkWHOISResponseError(rawResponse string) (code string, message string) {
	// 检查域名不存在
	if domainNotFoundRegex.MatchString(rawResponse) {
		return model.ErrCodeDomainNotFound, "域名未注册或不存在"
	}
	// 检查查询限制
	if queryLimitRegex.MatchString(rawResponse) {
		return model.ErrCodeQueryLimit, "查询频率受限，请稍后重试"
	}
	return "", ""
}

// TLDServerConfig 表示 TLD 服务器配置
type TLDServerConfig struct {
	Servers   map[string]string   `yaml:"servers"`
	Fallbacks map[string][]string `yaml:"fallback_servers"`
}

// WHOISClient 表示 WHOIS 查询客户端
type WHOISClient struct {
	servers    map[string]string
	fallbacks  map[string][]string
	timeout    time.Duration
	port       int
	logger     Logger
	configFile string
}

// NewWHOISClient 创建 WHOIS 客户端
func NewWHOISClient(opts ...WHOISOption) *WHOISClient {
	c := &WHOISClient{
		servers:   make(map[string]string),
		fallbacks: make(map[string][]string),
		timeout:   10 * time.Second,
		port:      43,
		logger:    newDefaultLogger(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	// 如果没有自定义服务器配置，尝试加载默认配置文件
	if len(c.servers) == 0 {
		c.loadDefaultConfig()
	}

	return c
}

// WHOISOption 定义 WHOIS 客户端配置选项
type WHOISOption func(*WHOISClient)

// WithWSTimeout 设置查询超时时间
func WithWSTimeout(timeout time.Duration) WHOISOption {
	return func(c *WHOISClient) {
		c.timeout = timeout
	}
}

// WithWSPort 设置 WHOIS 服务器端口
func WithWSPort(port int) WHOISOption {
	return func(c *WHOISClient) {
		c.port = port
	}
}

// WithWSLogger 设置自定义日志器
func WithWSLogger(logger Logger) WHOISOption {
	return func(c *WHOISClient) {
		c.logger = logger
	}
}

// WithWSServers 设置自定义 TLD 服务器映射
func WithWSServers(servers map[string]string) WHOISOption {
	return func(c *WHOISClient) {
		for k, v := range servers {
			c.servers[k] = v
		}
	}
}

// WithWSFallbacks 设置备用服务器映射
func WithWSFallbacks(fallbacks map[string][]string) WHOISOption {
	return func(c *WHOISClient) {
		for k, v := range fallbacks {
			c.fallbacks[k] = v
		}
	}
}

// WithWSConfigFile 设置本地 WHOIS 配置文件路径
// 指定后将从该文件加载 TLD -> WHOIS 服务器映射
func WithWSConfigFile(path string) WHOISOption {
	return func(c *WHOISClient) {
		c.configFile = path
	}
}

// loadDefaultConfig 加载默认配置文件
func (c *WHOISClient) loadDefaultConfig() {
	// 如果指定了配置文件，优先加载
	if c.configFile != "" {
		if err := c.loadConfigFromFile(c.configFile); err != nil {
			c.logger.Warn("从指定路径加载配置失败", "path", c.configFile, "error", err)
		} else {
			return
		}
	}

	// 尝试从相对路径加载
	configPaths := []string{
		"config/tld_whois_servers.yaml",
		"../config/tld_whois_servers.yaml",
		"../../config/tld_whois_servers.yaml",
		"../../../config/tld_whois_servers.yaml",
	}

	for _, path := range configPaths {
		if err := c.loadConfigFromFile(path); err != nil {
			continue
		}
		return
	}

	// 最后使用内嵌的默认配置
	c.LoadConfigFromBytes(defaultWHOISConfig)
}

// loadConfigFromFile 从指定文件加载配置
func (c *WHOISClient) loadConfigFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return c.LoadConfigFromBytes(data)
}

// LoadConfigFromBytes 从字节数据加载 WHOIS 服务器配置 (公开方法)
// YAML 格式须包含 servers 字段，可选 fallback_servers 字段
func (c *WHOISClient) LoadConfigFromBytes(data []byte) error {
	var tldConfig TLDServerConfig
	if err := yaml.Unmarshal(data, &tldConfig); err != nil {
		c.logger.Warn("解析 TLD 配置失败", "error", err)
		return err
	}

	// 加载服务器映射
	for k, v := range tldConfig.Servers {
		c.servers[k] = v
	}

	// 加载备用服务器
	for k, v := range tldConfig.Fallbacks {
		c.fallbacks[k] = v
	}

	c.logger.Info("已加载 TLD 服务器配置", "count", len(c.servers))
	return nil
}

// LoadServers 运行时加载 WHOIS 服务器映射 (TLD -> server)
// 会与已有配置合并，相同 TLD 覆盖旧值
func (c *WHOISClient) LoadServers(config map[string]string) {
	for k, v := range config {
		c.servers[k] = v
	}
	c.logger.Info("运行时加载 WHOIS 服务器配置", "count", len(config))
}

// GetServers 获取当前 WHOIS 服务器配置的副本 (TLD -> server 映射)
func (c *WHOISClient) GetServers() map[string]string {
	result := make(map[string]string, len(c.servers))
	for k, v := range c.servers {
		result[k] = v
	}
	return result
}

// Query 执行 WHOIS 查询
func (c *WHOISClient) Query(ctx context.Context, domain string) (*model.DomainInfo, error) {
	// 验证域名
	if err := validator.ValidateDomain(domain); err != nil {
		return nil, &model.Error{
			Code:    model.ErrCodeInvalidDomain,
			Message: "域名验证失败",
			Details: err.Error(),
		}
	}

	// 规范化域名
	domain = validator.NormalizeDomain(domain)

	// 获取 WHOIS 服务器
	server := c.getServer(domain)
	if server == "" {
		return nil, &model.Error{
			Code:    model.ErrCodeProtocolError,
			Message: fmt.Sprintf("未找到域名 %s 的 WHOIS 服务器", domain),
		}
	}

	// 执行查询
	rawResponse, err := c.queryServer(ctx, server, domain)
	if err != nil {
		// 尝试备用服务器
		tld := c.extractTLD(domain)
		if fallbacks, ok := c.fallbacks[tld]; ok {
			for _, fallback := range fallbacks {
				rawResponse, err = c.queryServer(ctx, fallback, domain)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			// 区分超时和其他网络错误
			code := model.ErrCodeProtocolError
			message := "WHOIS 查询失败"
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				code = model.ErrCodeQueryTimeout
				message = "WHOIS 查询超时"
			}
			return nil, &model.Error{
				Code:    code,
				Message: message,
				Details: err.Error(),
			}
		}
	}

	// 检查响应中的错误关键词
	if errCode, errMsg := checkWHOISResponseError(rawResponse); errCode != "" {
		return nil, &model.Error{
			Code:    errCode,
			Message: errMsg,
		}
	}

	// 创建结果
	result := &model.DomainInfo{
		DomainName:    domain,
		QueryProtocol: string(model.ProtocolWHOIS),
		QueryTime:     time.Now(),
		DataSource:    "live",
		RawResponse:   &rawResponse,
	}

	// 解析响应（简化版本，完整版本需要使用内部的 normalizer）
	if err := c.parseResponse(result, rawResponse); err != nil {
		c.logger.Warn("解析 WHOIS 响应失败", "error", err)
	}

	return result, nil
}

// getServer 获取域名对应的 WHOIS 服务器
func (c *WHOISClient) getServer(domain string) string {
	// 提取 TLD
	tld := c.extractTLD(domain)
	if tld == "" {
		return ""
	}

	// 查找服务器
	if server, ok := c.servers[tld]; ok {
		return server
	}

	// 默认使用 VeriSign 服务器
	if tld == ".com" || tld == ".net" {
		return "whois.verisign-grs.com"
	}

	return ""
}

// extractTLD 提取域名的顶级域名
func (c *WHOISClient) extractTLD(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	return "." + parts[len(parts)-1]
}

// queryServer 查询 WHOIS 服务器
func (c *WHOISClient) queryServer(ctx context.Context, server, domain string) (string, error) {
	// 建立 TCP 连接，支持 ctx 取消
	dialer := &net.Dialer{
		Timeout: c.timeout,
	}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", server, c.port))
	if err != nil {
		return "", fmt.Errorf("连接 WHOIS 服务器失败: %w", err)
	}
	defer conn.Close()

	// 设置超时
	conn.SetDeadline(time.Now().Add(c.timeout))

	// 发送查询请求
	query := domain + "\r\n"
	_, err = conn.Write([]byte(query))
	if err != nil {
		return "", fmt.Errorf("发送查询请求失败: %w", err)
	}

	// 读取响应
	var response strings.Builder
	scanner := bufio.NewScanner(conn)
	// 设置更大的缓冲区以处理大型 WHOIS 响应
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB
	for scanner.Scan() {
		// 检查 ctx 是否已取消
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		line := scanner.Text()
		response.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	return response.String(), nil
}

// parseResponse 解析 WHOIS 响应（简化版本）
func (c *WHOISClient) parseResponse(result *model.DomainInfo, rawResponse string) error {
	lines := strings.Split(rawResponse, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "domain name", "domainname":
			if result.DomainName == "" {
				result.DomainName = value
			}
		case "registrar", "sponsoring registrar":
			result.RegistrarName = value
		case "creation date", "created", "registration time":
			if t, err := time.Parse("2006-01-02T15:04:05Z", value); err == nil {
				result.RegistrationDate = &t
			} else if t, err := time.Parse("2006-01-02", value); err == nil {
				result.RegistrationDate = &t
			}
		case "registry expiry date", "expiry date", "expiration time":
			if t, err := time.Parse("2006-01-02T15:04:05Z", value); err == nil {
				result.ExpirationDate = &t
			} else if t, err := time.Parse("2006-01-02", value); err == nil {
				result.ExpirationDate = &t
			}
		case "updated date", "last modified":
			if t, err := time.Parse("2006-01-02T15:04:05Z", value); err == nil {
				result.LastUpdated = &t
			} else if t, err := time.Parse("2006-01-02", value); err == nil {
				result.LastUpdated = &t
			}
		case "name server", "nameserver", "nserver":
			if result.NameServers == nil {
				result.NameServers = make([]string, 0)
			}
			result.NameServers = append(result.NameServers, strings.ToLower(value))
		case "domain status", "status":
			if result.Status == nil {
				result.Status = make([]string, 0)
			}
			result.Status = append(result.Status, value)
		}
	}

	return nil
}

// Close 清理资源
func (c *WHOISClient) Close() error {
	// 目前没有需要清理的资源
	return nil
}
