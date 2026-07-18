package validator

import (
	"regexp"
	"strings"
)

// domainRegex 域名正则表达式
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// ValidateDomain 验证域名格式
func ValidateDomain(domain string) error {
	if domain == "" {
		return ErrEmptyDomain
	}

	// 转换为小写
	domain = strings.ToLower(domain)

	// 去除末尾的点号
	domain = strings.TrimSuffix(domain, ".")

	// 检查长度
	if len(domain) > 253 {
		return ErrDomainTooLong
	}

	// 检查格式
	if !domainRegex.MatchString(domain) {
		return ErrInvalidDomainFormat
	}

	// 检查每段长度
	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if len(part) > 63 {
			return ErrLabelTooLong
		}
		if len(part) == 0 {
			return ErrEmptyLabel
		}
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return ErrInvalidLabelFormat
		}
	}

	return nil
}

// NormalizeDomain 规范化域名
func NormalizeDomain(domain string) string {
	// 转换为小写
	domain = strings.ToLower(domain)

	// 去除末尾的点号
	domain = strings.TrimSuffix(domain, ".")

	// 去除首尾空格
	domain = strings.TrimSpace(domain)

	return domain
}

// 错误定义
var (
	ErrEmptyDomain         = &ValidationError{Message: "域名不能为空"}
	ErrDomainTooLong       = &ValidationError{Message: "域名长度超过253字符限制"}
	ErrInvalidDomainFormat = &ValidationError{Message: "域名格式无效"}
	ErrLabelTooLong        = &ValidationError{Message: "域名标签长度超过63字符限制"}
	ErrEmptyLabel          = &ValidationError{Message: "域名标签不能为空"}
	ErrInvalidLabelFormat  = &ValidationError{Message: "域名标签格式无效，不能以连字符开头或结尾"}
)

// ValidationError 表示验证错误
type ValidationError struct {
	Message string
}

// Error 实现 error 接口
func (e *ValidationError) Error() string {
	return e.Message
}
