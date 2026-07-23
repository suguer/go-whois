// Package model 定义了 Go-WHOIS 的公共数据模型。
// 这些类型可以被第三方开发者直接使用。
package model

import "time"

// DomainInfo 表示域名查询结果
type DomainInfo struct {
	// 域名名称
	DomainName string `json:"domain_name"`
	// ROID (Registry Object ID)
	ROID string `json:"roid,omitempty"`
	// 查询协议 (rdap/whois)
	QueryProtocol string `json:"query_protocol"`
	// 查询时间
	QueryTime time.Time `json:"query_time"`
	// 查询耗时(毫秒)
	QueryDuration int64 `json:"query_duration_ms"`
	// 数据来源 (live/cache)
	DataSource string `json:"data_source"`
	// 注册商名称
	RegistrarName string `json:"registrar_name,omitempty"`
	// 注册商网址
	RegistrarURL string `json:"registrar_url,omitempty"`
	// 注册商 IANA ID
	RegistrarIANAID string `json:"registrar_iana_id,omitempty"`
	// 注册人名称
	RegistrantName string `json:"registrant_name,omitempty"`
	// 注册日期
	RegistrationDate *time.Time `json:"registration_date,omitempty"`
	// 到期日期
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`
	// 最后更新日期
	LastUpdated *time.Time `json:"last_updated,omitempty"`
	// 域名状态列表
	Status []string `json:"status"`
	// 名称服务器列表
	NameServers []string `json:"name_servers"`
	// DNSSEC 信息
	DNSSEC DNSSECInfo `json:"dnssec"`
	// 原始响应 (可选，调试用)
	RawResponse *string `json:"raw_response,omitempty"`
}

// DNSSECInfo 表示 DNSSEC 信息
type DNSSECInfo struct {
	// 是否已签名
	Signed *bool `json:"signed,omitempty"`
	// 委派是否已签名
	DelegationSigned *bool `json:"delegation_signed,omitempty"`
}

// QueryProtocol 查询协议类型
type QueryProtocol string

const (
	// ProtocolRDAP 使用 RDAP 协议
	ProtocolRDAP QueryProtocol = "rdap"
	// ProtocolWHOIS 使用 WHOIS 协议
	ProtocolWHOIS QueryProtocol = "whois"
	// ProtocolAuto 自动选择协议 (RDAP 优先，失败回退 WHOIS)
	ProtocolAuto QueryProtocol = "auto"
)

// DataSource 数据来源类型
type DataSource string

const (
	// DataSourceLive 实时查询
	DataSourceLive DataSource = "live"
	// DataSourceCache 缓存数据
	DataSourceCache DataSource = "cache"
)

// LookupRequest 表示查询请求
type LookupRequest struct {
	// 域名
	Domain string `json:"domain"`
	// 查询协议
	Protocol QueryProtocol `json:"protocol,omitempty"`
}

// LookupResponse 表示查询响应
type LookupResponse struct {
	// 是否成功
	Success bool `json:"success"`
	// 域名信息
	Data *DomainInfo `json:"data,omitempty"`
	// 错误信息
	Error *Error `json:"error,omitempty"`
	// 请求 ID
	RequestID string `json:"request_id,omitempty"`
}

// Error 表示错误信息
type Error struct {
	// 错误代码
	Code string `json:"code"`
	// 错误消息
	Message string `json:"message"`
	// 错误详情
	Details string `json:"details,omitempty"`
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Details != "" {
		return e.Code + ": " + e.Message + " - " + e.Details
	}
	return e.Code + ": " + e.Message
}

// 错误代码常量
const (
	ErrCodeInvalidDomain       = "INVALID_DOMAIN"
	ErrCodeDomainNotFound      = "DOMAIN_NOT_FOUND"
	ErrCodeQueryTimeout        = "QUERY_TIMEOUT"
	ErrCodeQueryLimit          = "QUERY_LIMIT"
	ErrCodeProtocolError       = "PROTOCOL_ERROR"
	ErrCodeRateLimited         = "RATE_LIMITED"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeUnsupportedProtocol = "UNSUPPORTED_PROTOCOL"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Enabled bool    `json:"enabled"`
	Size    int     `json:"size"`
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	HitRate float64 `json:"hit_rate"`
}
