package whois

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	} `json:"entities"`
	SecureDNS struct {
		DelegationSigned bool `json:"delegationSigned"`
	} `json:"secureDNS"`
}

// loadRDAPBootstrap 从 IANA 加载 RDAP Bootstrap 数据
func (c *Client) loadRDAPBootstrap() {
	resp, err := c.httpClient.Get(c.options.rdapBootstrap)
	if err != nil {
		c.logger.Warn("加载 IANA RDAP Bootstrap 失败", "error", err)
		c.loadDefaultRDAP()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("IANA RDAP Bootstrap 返回错误状态码", "status", resp.StatusCode)
		c.loadDefaultRDAP()
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warn("读取 RDAP Bootstrap 响应失败", "error", err)
		c.loadDefaultRDAP()
		return
	}

	var data IANABootstrapData
	if err := json.Unmarshal(body, &data); err != nil {
		c.logger.Warn("解析 RDAP Bootstrap JSON 失败", "error", err)
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

	c.logger.Info("成功加载 IANA RDAP Bootstrap", "count", len(c.rdapCache))
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
		return nil, &model.Error{
			Code:    model.ErrCodeQueryTimeout,
			Message: "RDAP 查询超时",
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

	// 读取响应
	body, err := io.ReadAll(resp.Body)
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
				name, url, ianaID := parseRegistrarVCard(entity.VCardArray)
				result.RegistrarName = name
				result.RegistrarURL = url
				result.RegistrarIANAID = ianaID
				if entity.Handle != "" && result.RegistrarIANAID == "" {
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

// normalizeStatus 规范化域名状态
func normalizeStatus(status string) string {
	statusMap := map[string]string{
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

	lower := strings.ToLower(status)
	if normalized, ok := statusMap[lower]; ok {
		return normalized
	}
	return status
}
