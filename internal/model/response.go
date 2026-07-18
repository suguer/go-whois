package model

// APIResponse 表示统一 API 响应
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// APIError 表示 API 错误
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// BatchAPIResponse 表示批量查询响应
type BatchAPIResponse struct {
	Success bool          `json:"success"`
	Data    []*DomainInfo `json:"data"`
	Errors  []*BatchError `json:"errors,omitempty"`
}

// BatchError 表示批量查询中的单个错误
type BatchError struct {
	Domain string    `json:"domain"`
	Error  *APIError `json:"error"`
}

// HealthResponse 表示健康检查响应
type HealthResponse struct {
	Status  string       `json:"status"`
	Version string       `json:"version"`
	Uptime  int64        `json:"uptime_seconds"`
	Cache   CacheStats   `json:"cache"`
}

// CacheStats 表示缓存统计
type CacheStats struct {
	Enabled bool    `json:"enabled"`
	Size    int     `json:"size"`
	HitRate float64 `json:"hit_rate"`
}
