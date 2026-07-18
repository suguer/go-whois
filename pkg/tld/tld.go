package tld

import (
	"strings"
)

// ExtractTLD 从域名中提取顶级域名
func ExtractTLD(domain string) string {
	// 转换为小写
	domain = strings.ToLower(domain)

	// 去除末尾的点号
	domain = strings.TrimSuffix(domain, ".")

	// 按点号分割
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}

	// 返回最后一段作为 TLD
	return "." + parts[len(parts)-1]
}

// ExtractSLD 从域名中提取二级域名
func ExtractSLD(domain string) string {
	// 转换为小写
	domain = strings.ToLower(domain)

	// 去除末尾的点号
	domain = strings.TrimSuffix(domain, ".")

	// 按点号分割
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}

	// 返回倒数第二段
	return parts[len(parts)-2]
}

// IsGTLD 检查是否为通用顶级域名
func IsGTLD(tld string) bool {
	gtlds := map[string]bool{
		".com":  true,
		".net":  true,
		".org":  true,
		".info": true,
		".biz":  true,
		".name": true,
		".pro":  true,
		".mobi": true,
		".tel":  true,
		".asia": true,
	}

	return gtlds[tld]
}

// IsCCTLD 检查是否为国家代码顶级域名
func IsCCTLD(tld string) bool {
	// 国家代码顶级域名通常是两个字母
	if len(tld) == 3 && strings.HasPrefix(tld, ".") {
		code := tld[1:]
		// 检查是否为两个字母
		if len(code) == 2 {
			for _, c := range code {
				if c < 'a' || c > 'z' {
					return false
				}
			}
			return true
		}
	}
	return false
}

// GetTLDType 获取顶级域名类型
func GetTLDType(tld string) string {
	if IsGTLD(tld) {
		return "gtld"
	}
	if IsCCTLD(tld) {
		return "cctld"
	}
	return "other"
}
