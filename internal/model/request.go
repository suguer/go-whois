package model

// LookupRequest 表示查询请求
type LookupRequest struct {
	Domain   string `json:"domain" binding:"required"`
	Protocol string `json:"protocol" binding:"omitempty,oneof=rdap whois auto"`
}

// BatchLookupRequest 表示批量查询请求
type BatchLookupRequest struct {
	Domains  []string `json:"domains" binding:"required,min=1,max=50"`
	Protocol string   `json:"protocol" binding:"omitempty,oneof=rdap whois auto"`
}
