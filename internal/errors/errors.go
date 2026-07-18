package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 表示错误代码
type ErrorCode string

const (
	// 业务错误码
	ErrCodeInvalidDomain     ErrorCode = "INVALID_DOMAIN"
	ErrCodeDomainNotFound    ErrorCode = "DOMAIN_NOT_FOUND"
	ErrCodeQueryTimeout      ErrorCode = "QUERY_TIMEOUT"
	ErrCodeProtocolError     ErrorCode = "PROTOCOL_ERROR"
	ErrCodeRateLimited       ErrorCode = "RATE_LIMITED"
	ErrCodeBatchSizeExceeded ErrorCode = "BATCH_SIZE_EXCEEDED"

	// 系统错误码
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeConfigError        ErrorCode = "CONFIG_ERROR"
)

// AppError 表示应用错误
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
	Err        error     `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// 预定义错误
var (
	ErrInvalidDomain = &AppError{
		Code:       ErrCodeInvalidDomain,
		Message:    "域名格式无效",
		HTTPStatus: http.StatusBadRequest,
	}

	ErrDomainNotFound = &AppError{
		Code:       ErrCodeDomainNotFound,
		Message:    "域名未注册",
		HTTPStatus: http.StatusNotFound,
	}

	ErrQueryTimeout = &AppError{
		Code:       ErrCodeQueryTimeout,
		Message:    "查询超时",
		HTTPStatus: http.StatusGatewayTimeout,
	}

	ErrProtocolError = &AppError{
		Code:       ErrCodeProtocolError,
		Message:    "协议查询失败",
		HTTPStatus: http.StatusBadGateway,
	}

	ErrRateLimited = &AppError{
		Code:       ErrCodeRateLimited,
		Message:    "请求过于频繁",
		HTTPStatus: http.StatusTooManyRequests,
	}

	ErrInternalError = &AppError{
		Code:       ErrCodeInternalError,
		Message:    "内部服务器错误",
		HTTPStatus: http.StatusInternalServerError,
	}
)

// NewInvalidDomainError 创建域名无效错误
func NewInvalidDomainError(domain string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeInvalidDomain,
		Message:    "域名格式无效",
		Details:    fmt.Sprintf("域名 '%s' 不符合规范", domain),
		HTTPStatus: http.StatusBadRequest,
		Err:        err,
	}
}

// NewDomainNotFoundError 创建域名未找到错误
func NewDomainNotFoundError(domain string) *AppError {
	return &AppError{
		Code:       ErrCodeDomainNotFound,
		Message:    "域名未注册",
		Details:    fmt.Sprintf("域名 '%s' 未注册或不存在", domain),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewQueryTimeoutError 创建查询超时错误
func NewQueryTimeoutError(domain string, protocol string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeQueryTimeout,
		Message:    "查询超时",
		Details:    fmt.Sprintf("查询域名 '%s' 使用协议 '%s' 超时", domain, protocol),
		HTTPStatus: http.StatusGatewayTimeout,
		Err:        err,
	}
}

// NewProtocolError 创建协议错误
func NewProtocolError(domain string, protocol string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeProtocolError,
		Message:    "协议查询失败",
		Details:    fmt.Sprintf("查询域名 '%s' 使用协议 '%s' 失败", domain, protocol),
		HTTPStatus: http.StatusBadGateway,
		Err:        err,
	}
}

// WrapInternalError 包装内部错误
func WrapInternalError(err error) *AppError {
	return &AppError{
		Code:       ErrCodeInternalError,
		Message:    "内部服务器错误",
		Details:    "请联系管理员",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}
