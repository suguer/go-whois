package whois

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/suguer/go-whois/pkg/model"
)

// IANABootstrapData 表示 IANA RDAP Bootstrap JSON 结构
type IANABootstrapData struct {
	Services    [][][]string `json:"services"`
	Version     string       `json:"version"`
	Publication string       `json:"publication"`
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
		PublicIDs       []struct {
			Type       string `json:"type"`
			Identifier string `json:"identifier"`
		} `json:"publicIds"`
	} `json:"entities"`
	SecureDNS struct {
		DelegationSigned bool `json:"delegationSigned"`
	} `json:"secureDNS"`
}

// loadRDAPBootstrap 加载 RDAP Bootstrap 数据
// 优先从本地文件加载，其次从 URL 加载
func (c *Client) loadRDAPBootstrap() {
	defer c.readyOnce.Do(func() {
		close(c.readyCh)
	})

	// 如果指定了本地文件，优先从文件加载
	if c.options.rdapBootstrapFile != "" {
		if err := c.loadRDAPFromFile(c.options.rdapBootstrapFile); err != nil {
			c.logger.Warn("从本地文件加载 RDAP Bootstrap 失败", "path", c.options.rdapBootstrapFile, "error", err)
		} else {
			return
		}
	}

	resp, err := c.httpClient.Get(c.options.rdapBootstrap)
	if err != nil {
		c.logger.Warn("加载 IANA RDAP Bootstrap 失败", "error", err)
		c.loadEmbeddedRDAP()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("IANA RDAP Bootstrap 返回错误状态码", "status", resp.StatusCode)
		c.loadEmbeddedRDAP()
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warn("读取 RDAP Bootstrap 响应失败", "error", err)
		c.loadEmbeddedRDAP()
		return
	}

	var data IANABootstrapData
	if err := json.Unmarshal(body, &data); err != nil {
		c.logger.Warn("解析 RDAP Bootstrap JSON 失败", "error", err)
		c.loadEmbeddedRDAP()
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, service := range data.Services {
		if len(service) < 2 || len(service[0]) == 0 || len(service[1]) == 0 {
			continue
		}
		tlds := service[0]
		endpoint := service[1][0]
		if !strings.HasSuffix(endpoint, "/") {
			endpoint += "/"
		}
		for _, tld := range tlds {
			c.rdapCache[tld] = endpoint
		}
	}

	c.logger.Info("成功加载 IANA RDAP Bootstrap", "count", len(c.rdapCache))
}

// loadEmbeddedRDAP 从内嵌的默认配置加载 RDAP 端点
// 用于第三方库在无法下载 IANA 数据时的回退方案
func (c *Client) loadEmbeddedRDAP() {
	var data IANABootstrapData
	if err := json.Unmarshal(defaultRDAPBootstrap, &data); err != nil {
		c.logger.Warn("解析内嵌 RDAP Bootstrap 失败，使用硬编码默认值", "error", err)
		c.loadDefaultRDAP()
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, service := range data.Services {
		if len(service) < 2 || len(service[0]) == 0 || len(service[1]) == 0 {
			continue
		}
		tlds := service[0]
		endpoint := service[1][0]
		if !strings.HasSuffix(endpoint, "/") {
			endpoint += "/"
		}
		for _, tld := range tlds {
			c.rdapCache[tld] = endpoint
		}
	}

	c.logger.Info("从内嵌配置加载 RDAP Bootstrap", "count", len(c.rdapCache))
}

// loadDefaultRDAP 加载默认 RDAP 端点
func (c *Client) loadDefaultRDAP() {
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
		"app":  "https://rdap.nic.google/",
		"dev":  "https://rdap.nic.google/",
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.rdapCache = defaults
	c.logger.Info("加载默认 RDAP 端点", "count", len(defaults))
}

// loadRDAPFromFile 从本地文件加载 RDAP Bootstrap 数据
func (c *Client) loadRDAPFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 RDAP Bootstrap 文件失败: %w", err)
	}

	var bootstrapData IANABootstrapData
	if err := json.Unmarshal(data, &bootstrapData); err != nil {
		return fmt.Errorf("解析 RDAP Bootstrap JSON 失败: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, service := range bootstrapData.Services {
		if len(service) < 2 || len(service[0]) == 0 || len(service[1]) == 0 {
			continue
		}
		tlds := service[0]
		endpoint := service[1][0]
		if !strings.HasSuffix(endpoint, "/") {
			endpoint += "/"
		}
		for _, tld := range tlds {
			c.rdapCache[tld] = endpoint
		}
	}

	c.logger.Info("从本地文件加载 RDAP Bootstrap", "path", path, "count", len(c.rdapCache))
	return nil
}

// getRDAPEndpoint 获取 TLD 的 RDAP 端点
func (c *Client) getRDAPEndpoint(tld string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	endpoint, ok := c.rdapCache[tld]
	return endpoint, ok
}

// queryRDAP 执行 RDAP 查询
func (c *Client) queryRDAP(ctx context.Context, domain string) (*model.DomainInfo, error) {
	// 提取 TLD
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return nil, &model.Error{
			Code:    model.ErrCodeInvalidDomain,
			Message: "无效的域名格式",
		}
	}
	tld := parts[len(parts)-1]

	// 获取 RDAP 端点
	endpoint, ok := c.getRDAPEndpoint(tld)
	if !ok {
		return nil, &model.Error{
			Code:    model.ErrCodeProtocolError,
			Message: fmt.Sprintf("未找到 TLD %s 的 RDAP 端点", tld),
		}
	}

	// 构建请求 URL
	url := fmt.Sprintf("%sdomain/%s", endpoint, domain)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, &model.Error{
			Code:    model.ErrCodeInternalError,
			Message: "创建请求失败",
			Details: err.Error(),
		}
	}

	req.Header.Set("User-Agent", c.options.userAgent)
	req.Header.Set("Accept", "application/rdap+json")

	// 执行请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// 区分超时和其他网络错误
		code := model.ErrCodeProtocolError
		message := "RDAP 查询失败"
		// 使用 net.Error 接口或 errors.Is 检查超时
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			code = model.ErrCodeQueryTimeout
			message = "RDAP 查询超时"
		}
		return nil, &model.Error{
			Code:    code,
			Message: message,
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	duration := time.Since(startTime).Milliseconds()

	// 检查状态码
	if resp.StatusCode == http.StatusNotFound {
		return nil, &model.Error{
			Code:    model.ErrCodeDomainNotFound,
			Message: "域名未注册",
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &model.Error{
			Code:    model.ErrCodeProtocolError,
			Message: fmt.Sprintf("RDAP 查询失败，状态码: %d", resp.StatusCode),
		}
	}

	// 读取响应（限制 10MB）
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, &model.Error{
			Code:    model.ErrCodeInternalError,
			Message: "读取响应失败",
			Details: err.Error(),
		}
	}

	// 解析响应
	return c.parseRDAPResponse(domain, body, duration)
}

// parseRDAPResponse 解析 RDAP 响应
func (c *Client) parseRDAPResponse(domain string, data []byte, duration int64) (*model.DomainInfo, error) {
	var resp RDAPResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, &model.Error{
			Code:    model.ErrCodeInternalError,
			Message: "解析 RDAP 响应失败",
			Details: err.Error(),
		}
	}

	result := &model.DomainInfo{
		DomainName:    domain,
		ROID:          resp.Handle,
		QueryProtocol: "rdap",
		QueryTime:     time.Now(),
		QueryDuration: duration,
		DataSource:    "live",
		Status:        make([]string, 0),
		NameServers:   make([]string, 0),
	}

	// 解析状态
	for _, status := range resp.Status {
		result.Status = append(result.Status, normalizeStatus(status))
	}

	// 解析事件（日期）
	for _, event := range resp.Events {
		switch event.EventAction {
		case "registration":
			t := event.EventDate
			result.RegistrationDate = &t
		case "expiration":
			t := event.EventDate
			result.ExpirationDate = &t
		case "last changed":
			t := event.EventDate
			result.LastUpdated = &t
		}
	}

	// 解析名称服务器
	for _, ns := range resp.Nameservers {
		result.NameServers = append(result.NameServers, strings.ToLower(ns.LDHName))
	}

	// 解析实体（注册商、注册人）
	for _, entity := range resp.Entities {
		for _, role := range entity.Roles {
			switch role {
			case "registrar":
				name, url, _ := parseRegistrarVCard(entity.VCardArray)
				result.RegistrarName = name
				result.RegistrarURL = url
				// 优先从 publicIds 获取 IANA Registrar ID
				for _, pid := range entity.PublicIDs {
					if pid.Type == "IANA Registrar ID" {
						result.RegistrarIANAID = pid.Identifier
						break
					}
				}
				// 如果 publicIds 没有，使用 Handle 作为备选
				if result.RegistrarIANAID == "" && entity.Handle != "" {
					result.RegistrarIANAID = entity.Handle
				}
			case "registrant":
				result.RegistrantName = parseRegistrantVCard(entity.VCardArray)
			}
		}
	}

	// 解析 DNSSEC
	if resp.SecureDNS.DelegationSigned {
		signed := true
		result.DNSSEC = model.DNSSECInfo{
			Signed:           &signed,
			DelegationSigned: &signed,
		}
	}

	// 保存原始响应
	if c.options.includeRaw {
		rawStr := string(data)
		result.RawResponse = &rawStr
	}

	return result, nil
}

// parseRegistrarVCard 解析注册商 vCard
func parseRegistrarVCard(vcard []interface{}) (name, url, ianaID string) {
	if len(vcard) < 2 {
		return
	}

	fields, ok := vcard[1].([]interface{})
	if !ok {
		return
	}

	for _, item := range fields {
		field, ok := item.([]interface{})
		if !ok || len(field) < 4 {
			continue
		}
		key, _ := field[0].(string)
		value, _ := field[3].(string)
		switch key {
		case "fn":
			name = value
		case "url":
			url = value
		}
	}

	return
}

// parseRegistrantVCard 解析注册人 vCard
func parseRegistrantVCard(vcard []interface{}) string {
	if len(vcard) < 2 {
		return ""
	}

	fields, ok := vcard[1].([]interface{})
	if !ok {
		return ""
	}

	for _, item := range fields {
		field, ok := item.([]interface{})
		if !ok || len(field) < 4 {
			continue
		}
		key, _ := field[0].(string)
		if key == "fn" {
			value, _ := field[3].(string)
			return value
		}
	}

	return ""
}

// statusMap 域名状态规范化映射表
var statusMap = map[string]string{
	"clientdeleteprohibited":   "clientDeleteProhibited",
	"clienthold":               "clientHold",
	"clientrenewprohibited":    "clientRenewProhibited",
	"clienttransferprohibited": "clientTransferProhibited",
	"clientupdateprohibited":   "clientUpdateProhibited",
	"serverdeleteprohibited":   "serverDeleteProhibited",
	"serverhold":               "serverHold",
	"serverrenewprohibited":    "serverRenewProhibited",
	"servertransferprohibited": "serverTransferProhibited",
	"serverupdateprohibited":   "serverUpdateProhibited",
	"ok":                       "ok",
	"active":                   "active",
	"inactive":                 "inactive",
	"locked":                   "locked",
	"pendingcreate":            "pendingCreate",
	"pendingdelete":            "pendingDelete",
	"pendingrenew":             "pendingRenew",
	"pendingtransfer":          "pendingTransfer",
	"pendingupdate":            "pendingUpdate",
	"redemptionperiod":         "redemptionPeriod",
	"renewperiod":              "renewPeriod",
	"transferperiod":           "transferPeriod",
	"addperiod":                "addPeriod",
	"autorenewperiod":          "autoRenewPeriod",
}

// normalizeStatus 规范化域名状态
func normalizeStatus(status string) string {
	lower := strings.ToLower(status)
	if normalized, ok := statusMap[lower]; ok {
		return normalized
	}
	return status
}
