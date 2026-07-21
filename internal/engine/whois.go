package engine

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/suguer/go-whois/internal/config"
	"github.com/suguer/go-whois/internal/model"
	"github.com/suguer/go-whois/pkg/validator"

	"gopkg.in/yaml.v3"
)

// TLDServerConfig 表示 TLD 服务器配置
type TLDServerConfig struct {
	Servers   map[string]string   `yaml:"servers"`
	Fallbacks map[string][]string `yaml:"fallback_servers"`
}

// WHOIS 表示 WHOIS 查询引擎
type WHOIS struct {
	config    *config.WHOISConfig
	servers   map[string]string
	fallbacks map[string][]string
}

// NewWHOIS 创建新的 WHOIS 引擎
func NewWHOIS(cfg *config.WHOISConfig, servers map[string]string, fallbacks map[string][]string) *WHOIS {
	w := &WHOIS{
		config:    cfg,
		servers:   make(map[string]string),
		fallbacks: make(map[string][]string),
	}

	// 从配置文件加载
	w.loadFromFile()

	// 合并传入的配置
	for k, v := range servers {
		w.servers[k] = v
	}
	for k, v := range fallbacks {
		w.fallbacks[k] = v
	}

	return w
}

// loadFromFile 从配置文件加载 TLD 服务器映射
func (w *WHOIS) loadFromFile() {
	// 尝试加载配置文件
	configPaths := []string{
		"config/tld_whois_servers.yaml",
		"../config/tld_whois_servers.yaml",
		"../../config/tld_whois_servers.yaml",
	}

	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var tldConfig TLDServerConfig
		if err := yaml.Unmarshal(data, &tldConfig); err != nil {
			continue
		}

		// 加载服务器映射
		for k, v := range tldConfig.Servers {
			w.servers[k] = v
		}

		// 加载备用服务器
		for k, v := range tldConfig.Fallbacks {
			w.fallbacks[k] = v
		}

		break
	}
}

// Name 返回引擎名称
func (w *WHOIS) Name() Protocol {
	return ProtocolWHOIS
}

// IsAvailable 检查引擎是否可用
func (w *WHOIS) IsAvailable() bool {
	return w.config.Enabled
}

// Query 执行 WHOIS 查询
func (w *WHOIS) Query(ctx context.Context, domain string) (*model.DomainInfo, error) {
	// 验证域名
	if err := validator.ValidateDomain(domain); err != nil {
		return nil, fmt.Errorf("域名验证失败: %w", err)
	}

	// 规范化域名
	domain = validator.NormalizeDomain(domain)

	// 获取 WHOIS 服务器
	server := w.getServer(domain)
	if server == "" {
		return nil, fmt.Errorf("未找到域名 %s 的 WHOIS 服务器", domain)
	}

	// 执行查询
	rawResponse, err := w.queryServer(ctx, server, domain)
	if err != nil {
		// 尝试备用服务器
		if fallbacks, ok := w.fallbacks[domain]; ok {
			for _, fallback := range fallbacks {
				rawResponse, err = w.queryServer(ctx, fallback, domain)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("WHOIS 查询失败: %w", err)
		}
	}

	// 解析响应
	result := &model.DomainInfo{
		DomainName:    domain,
		QueryProtocol: string(ProtocolWHOIS),
		QueryTime:     time.Now(),
		DataSource:    "live",
		RawResponse:   &rawResponse,
	}

	// 使用标准化器解析
	normalizer := NewNormalizer()
	normalizedResult, err := normalizer.NormalizeWHOIS(domain, rawResponse)
	if err != nil {
		// 如果解析失败，返回原始响应
		return result, nil
	}

	return normalizedResult, nil
}

// getServer 获取域名对应的 WHOIS 服务器
func (w *WHOIS) getServer(domain string) string {
	// 提取 TLD
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	tld := "." + parts[len(parts)-1]

	// 查找服务器
	if server, ok := w.servers[tld]; ok {
		return server
	}

	// 默认使用 VeriSign 服务器
	if tld == ".com" || tld == ".net" {
		return "whois.verisign-grs.com"
	}

	return ""
}

// queryServer 查询 WHOIS 服务器
func (w *WHOIS) queryServer(ctx context.Context, server, domain string) (string, error) {
	// 建立 TCP 连接
	addr := fmt.Sprintf("%s:%d", server, w.config.DefaultPort)
	conn, err := net.DialTimeout("tcp", addr, w.config.Timeout)
	if err != nil {
		return "", fmt.Errorf("连接 WHOIS 服务器失败: %w", err)
	}
	defer conn.Close()

	// 设置超时
	conn.SetDeadline(time.Now().Add(w.config.Timeout))

	// 发送查询请求
	query := domain + "\r\n"
	_, err = conn.Write([]byte(query))
	if err != nil {
		return "", fmt.Errorf("发送查询请求失败: %w", err)
	}

	// 读取响应
	var response strings.Builder
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		response.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	return response.String(), nil
}
