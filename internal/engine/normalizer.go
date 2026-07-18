package engine

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"go-whois/internal/model"
)

// DefaultNormalizer 表示结果标准化器
type DefaultNormalizer struct {
	whoisPatterns map[string]*regexp.Regexp
}

// NewNormalizer 创建新的标准化器
func NewNormalizer() Normalizer {
	n := &DefaultNormalizer{
		whoisPatterns: make(map[string]*regexp.Regexp),
	}
	n.initPatterns()
	return n
}

// initPatterns 初始化 WHOIS 解析模式
func (n *DefaultNormalizer) initPatterns() {
	// ROID - 标准格式和 .cn 格式
	n.whoisPatterns["roid"] = regexp.MustCompile(`(?i)roid:\s*(.+)`)
	n.whoisPatterns["registry_domain_id"] = regexp.MustCompile(`(?i)registry domain id:\s*(.+)`)

	// 注册商 - 标准格式
	n.whoisPatterns["registrar"] = regexp.MustCompile(`(?i)registrar:\s*(.+)`)
	n.whoisPatterns["registrar_url"] = regexp.MustCompile(`(?i)registrar url:\s*(.+)`)
	n.whoisPatterns["registrar_iana_id"] = regexp.MustCompile(`(?i)registrar iana id:\s*(.+)`)

	// 注册商 - .cn 格式
	n.whoisPatterns["sponsoring_registrar"] = regexp.MustCompile(`(?i)sponsoring registrar:\s*(.+)`)

	// 注册日期 - 标准格式
	n.whoisPatterns["creation_date"] = regexp.MustCompile(`(?i)creation date:\s*(.+)`)
	n.whoisPatterns["expiration_date"] = regexp.MustCompile(`(?i)registrar registration expiration date:\s*(.+)`)
	n.whoisPatterns["updated_date"] = regexp.MustCompile(`(?i)updated date:\s*(.+)`)

	// 注册日期 - .cn 格式
	n.whoisPatterns["registration_time"] = regexp.MustCompile(`(?i)registration time:\s*(.+)`)
	n.whoisPatterns["expiration_time"] = regexp.MustCompile(`(?i)expiration time:\s*(.+)`)

	// 域名状态 - 标准格式和 .cn 格式
	n.whoisPatterns["status"] = regexp.MustCompile(`(?i)domain status:\s*(.+)`)

	// 名称服务器 - 标准格式和 .cn 格式
	n.whoisPatterns["name_server"] = regexp.MustCompile(`(?i)name server:\s*(.+)`)

	// 注册人 - .cn 格式
	n.whoisPatterns["registrant"] = regexp.MustCompile(`(?i)registrant:\s*(.+)`)
}

// NormalizeWHOIS 标准化 WHOIS 响应
func (n *DefaultNormalizer) NormalizeWHOIS(domain string, rawResponse string) (*model.DomainInfo, error) {
	result := &model.DomainInfo{
		DomainName:    domain,
		QueryProtocol: "whois",
		QueryTime:     time.Now(),
		DataSource:    "live",
		Status:        make([]string, 0),
		NameServers:   make([]string, 0),
		RawResponse:   &rawResponse,
	}

	// 解析 ROID
	if matches := n.whoisPatterns["roid"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		result.ROID = strings.TrimSpace(matches[1])
	}
	if result.ROID == "" {
		if matches := n.whoisPatterns["registry_domain_id"].FindStringSubmatch(rawResponse); len(matches) > 1 {
			result.ROID = strings.TrimSpace(matches[1])
		}
	}

	// 解析注册商信息
	if matches := n.whoisPatterns["registrar"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		result.RegistrarName = strings.TrimSpace(matches[1])
	}
	// .cn 格式的注册商
	if result.RegistrarName == "" {
		if matches := n.whoisPatterns["sponsoring_registrar"].FindStringSubmatch(rawResponse); len(matches) > 1 {
			result.RegistrarName = strings.TrimSpace(matches[1])
		}
	}
	if matches := n.whoisPatterns["registrar_url"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		result.RegistrarURL = strings.TrimSpace(matches[1])
	}
	if matches := n.whoisPatterns["registrar_iana_id"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		result.RegistrarIANAID = strings.TrimSpace(matches[1])
	}

	// 解析注册人信息
	if matches := n.whoisPatterns["registrant"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		result.RegistrantName = strings.TrimSpace(matches[1])
	}

	// 解析日期 - 标准格式
	if matches := n.whoisPatterns["creation_date"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		if t, err := parseDate(strings.TrimSpace(matches[1])); err == nil {
			result.RegistrationDate = &t
		}
	}
	if matches := n.whoisPatterns["expiration_date"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		if t, err := parseDate(strings.TrimSpace(matches[1])); err == nil {
			result.ExpirationDate = &t
		}
	}
	if matches := n.whoisPatterns["updated_date"].FindStringSubmatch(rawResponse); len(matches) > 1 {
		if t, err := parseDate(strings.TrimSpace(matches[1])); err == nil {
			result.LastUpdated = &t
		}
	}

	// 解析日期 - .cn 格式
	if result.RegistrationDate == nil {
		if matches := n.whoisPatterns["registration_time"].FindStringSubmatch(rawResponse); len(matches) > 1 {
			if t, err := parseDate(strings.TrimSpace(matches[1])); err == nil {
				result.RegistrationDate = &t
			}
		}
	}
	if result.ExpirationDate == nil {
		if matches := n.whoisPatterns["expiration_time"].FindStringSubmatch(rawResponse); len(matches) > 1 {
			if t, err := parseDate(strings.TrimSpace(matches[1])); err == nil {
				result.ExpirationDate = &t
			}
		}
	}

	// 解析域名状态
	statusMatches := n.whoisPatterns["status"].FindAllStringSubmatch(rawResponse, -1)
	for _, match := range statusMatches {
		if len(match) > 1 {
			status := strings.TrimSpace(match[1])
			// 移除 URL 部分
			if idx := strings.Index(status, " "); idx > 0 {
				status = status[:idx]
			}
			result.Status = append(result.Status, status)
		}
	}

	// 解析名称服务器
	nsMatches := n.whoisPatterns["name_server"].FindAllStringSubmatch(rawResponse, -1)
	for _, match := range nsMatches {
		if len(match) > 1 {
			ns := strings.TrimSpace(strings.ToLower(match[1]))
			result.NameServers = append(result.NameServers, ns)
		}
	}

	return result, nil
}

// NormalizeRDAP 标准化 RDAP 响应
func (n *DefaultNormalizer) NormalizeRDAP(domain string, rawData []byte) (*model.DomainInfo, error) {
	// 解析 RDAP 响应
	rdapResp, err := ParseRDAPResponse(rawData)
	if err != nil {
		return nil, fmt.Errorf("解析 RDAP 响应失败: %w", err)
	}

	result := &model.DomainInfo{
		DomainName:    domain,
		ROID:          rdapResp.Handle,
		QueryProtocol: "rdap",
		QueryTime:     time.Now(),
		DataSource:    "live",
		Status:        make([]string, 0),
		NameServers:   make([]string, 0),
	}

	// 解析状态
	for _, status := range rdapResp.Status {
		result.Status = append(result.Status, normalizeStatus(status))
	}

	// 解析事件（日期）
	for _, event := range rdapResp.Events {
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
	for _, ns := range rdapResp.Nameservers {
		result.NameServers = append(result.NameServers, strings.ToLower(ns.LDHName))
	}

	// 解析实体（注册商、注册人）
	for _, entity := range rdapResp.Entities {
		for _, role := range entity.Roles {
			switch role {
			case "registrar":
				registrarName, registrarURL, registrarIANAID := n.parseRegistrar(entity)
				result.RegistrarName = registrarName
				result.RegistrarURL = registrarURL
				result.RegistrarIANAID = registrarIANAID
			case "registrant":
				result.RegistrantName = n.parseRegistrantName(entity)
			}
		}
	}

	// 解析 DNSSEC
	if rdapResp.SecureDNS.DelegationSigned {
		signed := true
		result.DNSSEC = model.DNSSECInfo{
			Signed:           &signed,
			DelegationSigned: &signed,
		}
	}

	// 设置原始响应
	rawStr := string(rawData)
	result.RawResponse = &rawStr

	return result, nil
}

// parseRegistrar 解析注册商信息
func (n *DefaultNormalizer) parseRegistrar(entity struct {
	ObjectClassName string        `json:"objectClassName"`
	Handle          string        `json:"handle"`
	Roles           []string      `json:"roles"`
	VCardArray      []interface{} `json:"vcardArray"`
}) (name, url, ianaID string) {
	ianaID = entity.Handle

	// 解析 vCard
	if len(entity.VCardArray) > 1 {
		if vcard, ok := entity.VCardArray[1].([]interface{}); ok {
			for _, item := range vcard {
				if field, ok := item.([]interface{}); ok && len(field) >= 4 {
					key, _ := field[0].(string)
					value, _ := field[3].(string)
					switch key {
					case "fn":
						name = value
					case "url":
						url = value
					}
				}
			}
		}
	}

	return name, url, ianaID
}

// parseRegistrantName 解析注册人名称
func (n *DefaultNormalizer) parseRegistrantName(entity struct {
	ObjectClassName string        `json:"objectClassName"`
	Handle          string        `json:"handle"`
	Roles           []string      `json:"roles"`
	VCardArray      []interface{} `json:"vcardArray"`
}) string {
	// 解析 vCard
	if len(entity.VCardArray) > 1 {
		if vcard, ok := entity.VCardArray[1].([]interface{}); ok {
			for _, item := range vcard {
				if field, ok := item.([]interface{}); ok && len(field) >= 4 {
					key, _ := field[0].(string)
					if key == "fn" {
						value, _ := field[3].(string)
						return value
					}
				}
			}
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

// parseDate 解析日期字符串
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02-Jan-2006",
		"2006.01.02 15:04:05",
		"20060102",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析日期: %s", dateStr)
}
