package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-whois/internal/config"
	"go-whois/internal/model"
	"go-whois/pkg/validator"
)

// IANABootstrapData 表示 IANA RDAP Bootstrap JSON 结构
// services 格式: [[[tld1, tld2], [endpoint1, endpoint2]], ...]
type IANABootstrapData struct {
	Services    [][][]string `json:"services"`
	Version     string       `json:"version"`
	Publication string       `json:"publication"`
}

// RDAP 表示 RDAP 查询引擎
type RDAP struct {
	config       *config.RDAPConfig
	client       *http.Client
	mu           sync.RWMutex
	endpoints    map[string]string
	bootstrapURL string
	lastUpdate   time.Time
}

// NewRDAP 创建新的 RDAP 引擎
func NewRDAP(cfg *config.RDAPConfig) *RDAP {
	r := &RDAP{
		config:       cfg,
		client:       &http.Client{Timeout: cfg.Timeout},
		endpoints:    make(map[string]string),
		bootstrapURL: cfg.BootstrapURL,
	}

	// 启动时加载 Bootstrap 数据
	if err := r.loadBootstrap(); err != nil {
		log.Printf("警告: 加载 IANA RDAP Bootstrap 数据失败: %v, 使用内置默认配置", err)
		r.loadDefaults()
	}

	// 启动定时更新协程
	go r.startAutoRefresh()

	return r
}

// loadBootstrap 从 IANA 加载 Bootstrap 数据
func (r *RDAP) loadBootstrap() error {
	resp, err := r.client.Get(r.bootstrapURL)
	if err != nil {
		return fmt.Errorf("请求 IANA Bootstrap 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IANA Bootstrap 返回状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取 Bootstrap 响应失败: %w", err)
	}

	var data IANABootstrapData
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("解析 Bootstrap JSON 失败: %w", err)
	}

	// 解析 services 数据
	// 格式: [[[tld1, tld2], [endpoint1, endpoint2]], ...]
	// 每个 service 是 [][]string: [tlds, endpoints]
	newEndpoints := make(map[string]string)
	for _, service := range data.Services {
		if len(service) < 2 || len(service[0]) == 0 || len(service[1]) == 0 {
			continue
		}
		tlds := service[0]       // TLD 列表
		endpoints := service[1]  // 端点列表
		endpoint := endpoints[0] // 使用第一个端点
		// 确保 endpoint 以 / 结尾
		if !strings.HasSuffix(endpoint, "/") {
			endpoint += "/"
		}
		for _, tld := range tlds {
			newEndpoints[tld] = endpoint
		}
	}

	// 更新内存中的端点映射
	r.mu.Lock()
	r.endpoints = newEndpoints
	r.lastUpdate = time.Now()
	r.mu.Unlock()

	log.Printf("成功加载 IANA RDAP Bootstrap 数据, 共 %d 个 TLD", len(newEndpoints))
	return nil
}

// loadDefaults 加载内置默认配置
func (r *RDAP) loadDefaults() {
	defaults := map[string]string{
		"com":  "https://rdap.verisign.com/com/v1/",
		"net":  "https://rdap.verisign.com/net/v1/",
		"org":  "https://rdap.publicinterestregistry.org/rdap/",
		"info": "https://rdap.identitydigital.services/rdap/",
		"io":   "https://rdap.identitydigital.services/rdap/",
		"co":   "https://rdap.nic.co/",
		"me":   "https://rdap.identitydigital.services/rdap/",
		"asia": "https://rdap.identitydigital.services/rdap/",
		"biz":  "https://rdap.identitydigital.services/rdap/",
		"name": "https://rdap.identitydigital.services/rdap/",
		"pro":  "https://rdap.identitydigital.services/rdap/",
		"mobi": "https://rdap.identitydigital.services/rdap/",
		"tel":  "https://rdap.identitydigital.services/rdap/",
		"app":  "https://rdap.nic.google/",
		"dev":  "https://rdap.nic.google/",
		"page": "https://rdap.nic.google/",
		"xyz":  "https://rdap.nic.xyz/",
		"club": "https://rdap.nic.club/",
	}

	r.mu.Lock()
	r.endpoints = defaults
	r.lastUpdate = time.Now()
	r.mu.Unlock()
}

// startAutoRefresh 启动定时刷新协程
func (r *RDAP) startAutoRefresh() {
	ticker := time.NewTicker(r.config.BootstrapCacheTTL)
	defer ticker.Stop()

	for range ticker.C {
		if err := r.loadBootstrap(); err != nil {
			log.Printf("警告: 刷新 IANA RDAP Bootstrap 数据失败: %v", err)
		}
	}
}

// GetEndpointsCount 获取当前加载的 TLD 端点数量 (用于健康检查)
func (r *RDAP) GetEndpointsCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.endpoints)
}

// GetLastUpdateTime 获取最后更新时间 (用于健康检查)
func (r *RDAP) GetLastUpdateTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastUpdate
}

// Name 返回引擎名称
func (r *RDAP) Name() Protocol {
	return ProtocolRDAP
}

// IsAvailable 检查引擎是否可用
func (r *RDAP) IsAvailable() bool {
	return r.config.Enabled
}

// Query 执行 RDAP 查询
func (r *RDAP) Query(ctx context.Context, domain string) (*model.DomainInfo, error) {
	// 验证域名
	if err := validator.ValidateDomain(domain); err != nil {
		return nil, fmt.Errorf("域名验证失败: %w", err)
	}

	// 规范化域名
	domain = validator.NormalizeDomain(domain)

	// 获取 RDAP 端点
	endpoint, err := r.getEndpoint(domain)
	if err != nil {
		return nil, fmt.Errorf("获取 RDAP 端点失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("%sdomain/%s", endpoint, domain)

	// 执行查询
	rawData, err := r.doRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("RDAP 查询失败: %w", err)
	}

	// 使用标准化器解析
	normalizer := NewNormalizer()
	normalizedResult, err := normalizer.NormalizeRDAP(domain, rawData)
	if err != nil {
		// 如果解析失败，返回基本结果
		return &model.DomainInfo{
			DomainName:    domain,
			QueryProtocol: string(ProtocolRDAP),
			QueryTime:     time.Now(),
			DataSource:    "live",
		}, nil
	}

	return normalizedResult, nil
}

// getEndpoint 获取域名对应的 RDAP 端点
func (r *RDAP) getEndpoint(domain string) (string, error) {
	// 提取 TLD
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("无效的域名格式")
	}
	tld := parts[len(parts)-1]

	r.mu.RLock()
	endpoint, ok := r.endpoints[tld]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("未找到 TLD %s 的 RDAP 端点", tld)
	}

	return endpoint, nil
}

// doRequest 执行 HTTP 请求
func (r *RDAP) doRequest(ctx context.Context, url string) ([]byte, error) {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 User-Agent
	req.Header.Set("User-Agent", r.config.UserAgent)
	req.Header.Set("Accept", "application/rdap+json")

	// 执行请求
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("域名未注册")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("请求过于频繁")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return body, nil
}

// RDAPResponse 表示 RDAP 响应
type RDAPResponse struct {
	ObjectClassName string   `json:"objectClassName"`
	Handle          string   `json:"handle"`
	LDHName         string   `json:"ldhName"`
	Status          []string `json:"status"`
	Events          []struct {
		EventAction string    `json:"eventAction"`
		EventDate   time.Time `json:"eventDate"`
	} `json:"events"`
	Nameservers []struct {
		LDHName string `json:"ldhName"`
	} `json:"nameservers"`
	Entities []struct {
		ObjectClassName string        `json:"objectClassName"`
		Handle          string        `json:"handle"`
		Roles           []string      `json:"roles"`
		VCardArray      []interface{} `json:"vcardArray"`
	} `json:"entities"`
	SecureDNS struct {
		DelegationSigned bool `json:"delegationSigned"`
	} `json:"secureDNS"`
}

// ParseRDAPResponse 解析 RDAP 响应
func ParseRDAPResponse(data []byte) (*RDAPResponse, error) {
	var resp RDAPResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析 RDAP 响应失败: %w", err)
	}
	return &resp, nil
}
