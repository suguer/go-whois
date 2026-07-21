package engine

import (
	"context"
	"github.com/suguer/go-whois/internal/model"
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
