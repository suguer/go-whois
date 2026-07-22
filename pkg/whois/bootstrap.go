package whois

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// TLDInfo 表示 TLD 信息
type TLDInfo struct {
	TLD         string `json:"tld"`
	Type        string `json:"type"`
	WhoisServer string `json:"whois_server,omitempty"`
}

// DownloadRDAPConfig 从 IANA 获取 RDAP Bootstrap 数据
// 返回 TLD 到 RDAP 端点的映射
func DownloadRDAPConfig(bootstrapURL string) (map[string]string, error) {
	if bootstrapURL == "" {
		bootstrapURL = "https://data.iana.org/rdap/dns.json"
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(bootstrapURL)
	if err != nil {
		return nil, fmt.Errorf("获取 RDAP Bootstrap 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RDAP Bootstrap 返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 限制 10MB
	if err != nil {
		return nil, fmt.Errorf("读取 RDAP Bootstrap 响应失败: %w", err)
	}

	var data IANABootstrapData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("解析 RDAP Bootstrap JSON 失败: %w", err)
	}

	result := make(map[string]string)
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
			result[tld] = endpoint
		}
	}

	return result, nil
}

// FetchTLDList 从 IANA 获取 TLD 列表
func FetchTLDList() ([]TLDInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://www.iana.org/domains/root/db")
	if err != nil {
		return nil, fmt.Errorf("获取 TLD 列表失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 限制 10MB
	if err != nil {
		return nil, fmt.Errorf("读取 TLD 列表响应失败: %w", err)
	}
	content := string(body)

	var tlds []TLDInfo

	// HTML 结构示例:
	// <tr>
	//   <td><span class="domain tld"><a href="/domains/root/db/aaa.html">.aaa</a></span></td>
	//   <td>generic</td>
	//   <td>Manager Name</td>
	// </tr>

	trRegex := regexp.MustCompile(`(?s)<tr>(.*?)</tr>`)
	tldLinkRegex := regexp.MustCompile(`<a href="/domains/root/db/([a-z0-9-]+)\.html">`)
	typeRegex := regexp.MustCompile(`(?i)<td>\s*(generic|country-code|sponsored|generic-restricted|infrastructure)\s*</td>`)

	rows := trRegex.FindAllStringSubmatch(content, -1)
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		rowContent := row[1]

		tldMatch := tldLinkRegex.FindStringSubmatch(rowContent)
		if len(tldMatch) < 2 {
			continue
		}
		tldName := tldMatch[1]

		tldType := "unknown"
		typeMatch := typeRegex.FindStringSubmatch(rowContent)
		if len(typeMatch) >= 2 {
			tldType = strings.ToLower(typeMatch[1])
		}

		tlds = append(tlds, TLDInfo{
			TLD:  tldName,
			Type: tldType,
		})
	}

	return tlds, nil
}

// FetchWhoisServers 并发获取 TLD 的 WHOIS 服务器信息
// concurrency: 并发请求数
// progressCallback: 进度回调函数 (可选)
func FetchWhoisServers(tlds []TLDInfo, concurrency int, progressCallback func(progress, total int)) []TLDInfo {
	if concurrency <= 0 {
		concurrency = 20
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	results := make([]TLDInfo, len(tlds))
	copy(results, tlds)

	progress := 0
	total := len(tlds)
	mu := sync.Mutex{}

	for i := range results {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			tld := &results[idx]
			server := FetchWhoisServer(tld.TLD)
			tld.WhoisServer = server

			mu.Lock()
			progress++
			if progressCallback != nil {
				progressCallback(progress, total)
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	return results
}

// FetchWhoisServer 获取单个 TLD 的 WHOIS 服务器
func FetchWhoisServer(tld string) string {
	url := fmt.Sprintf("https://www.iana.org/domains/root/db/%s.html", tld)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 限制 1MB
	if err != nil {
		return ""
	}
	content := string(body)

	// 匹配 WHOIS Server: xxx <br>
	whoisRegex := regexp.MustCompile(`(?i)<b>WHOIS Server:</b>\s*([^<\s]+)`)
	matches := whoisRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// FormatWhoisServersYAML 将 TLD 信息格式化为 YAML 字符串
func FormatWhoisServersYAML(tlds []TLDInfo) string {
	var sb strings.Builder
	sb.WriteString("# TLD WHOIS 服务器映射\n")
	sb.WriteString("# 由 FetchWhoisServers 自动生成\n")
	sb.WriteString(fmt.Sprintf("# 更新时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	sb.WriteString("servers:\n")
	// 按类型分组输出
	groups := map[string][]TLDInfo{
		"generic":            {},
		"country-code":       {},
		"sponsored":          {},
		"generic-restricted": {},
		"infrastructure":     {},
	}

	for _, tld := range tlds {
		if tld.WhoisServer != "" {
			groups[tld.Type] = append(groups[tld.Type], tld)
		}
	}

	// 通用顶级域名
	if len(groups["generic"]) > 0 {
		sb.WriteString("  # 通用顶级域名 (gTLD)\n")
		for _, tld := range groups["generic"] {
			sb.WriteString(fmt.Sprintf("  \".%s\": \"%s\"\n", tld.TLD, tld.WhoisServer))
		}
		sb.WriteString("\n")
	}

	// 受限顶级域名
	if len(groups["generic-restricted"]) > 0 {
		sb.WriteString("  # 受限顶级域名\n")
		for _, tld := range groups["generic-restricted"] {
			sb.WriteString(fmt.Sprintf("  \".%s\": \"%s\"\n", tld.TLD, tld.WhoisServer))
		}
		sb.WriteString("\n")
	}

	// 赞助顶级域名
	if len(groups["sponsored"]) > 0 {
		sb.WriteString("  # 赞助顶级域名 (sTLD)\n")
		for _, tld := range groups["sponsored"] {
			sb.WriteString(fmt.Sprintf("  \".%s\": \"%s\"\n", tld.TLD, tld.WhoisServer))
		}
		sb.WriteString("\n")
	}

	// 国家代码顶级域名
	if len(groups["country-code"]) > 0 {
		sb.WriteString("  # 国家代码顶级域名 (ccTLD)\n")
		for _, tld := range groups["country-code"] {
			sb.WriteString(fmt.Sprintf("  \".%s\": \"%s\"\n", tld.TLD, tld.WhoisServer))
		}
		sb.WriteString("\n")
	}

	// 基础设施顶级域名
	if len(groups["infrastructure"]) > 0 {
		sb.WriteString("  # 基础设施顶级域名\n")
		for _, tld := range groups["infrastructure"] {
			sb.WriteString(fmt.Sprintf("  \".%s\": \"%s\"\n", tld.TLD, tld.WhoisServer))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetWhoisServersMap 将 TLD 信息转换为 map 格式
func GetWhoisServersMap(tlds []TLDInfo) map[string]string {
	result := make(map[string]string)
	for _, tld := range tlds {
		if tld.WhoisServer != "" {
			key := "." + tld.TLD
			result[key] = tld.WhoisServer
		}
	}
	return result
}

// DownloadRDAPBootstrap 从 IANA 下载 RDAP Bootstrap 数据并保存到本地文件
// destPath: 目标文件路径 (例如: "config/rdap_bootstrap.json")
func DownloadRDAPBootstrap(destPath string) error {
	return DownloadRDAPBootstrapFromURL("https://data.iana.org/rdap/dns.json", destPath)
}

// DownloadRDAPBootstrapFromURL 从指定 URL 下载 RDAP Bootstrap 数据并保存到本地文件
// url: 下载地址
// destPath: 目标文件路径
func DownloadRDAPBootstrapFromURL(url, destPath string) error {
	// 确保目标目录存在
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 下载文件
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载 RDAP Bootstrap 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 RDAP Bootstrap 返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 限制 10MB
	if err != nil {
		return fmt.Errorf("读取 RDAP Bootstrap 响应失败: %w", err)
	}

	// 验证 JSON 格式
	var data IANABootstrapData
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("验证 RDAP Bootstrap JSON 失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(destPath, body, 0644); err != nil {
		return fmt.Errorf("写入 RDAP Bootstrap 文件失败: %w", err)
	}

	return nil
}

// DownloadWHOISConfig 从 IANA 获取 TLD 的 WHOIS 服务器信息并保存到本地 YAML 文件
// destPath: 目标文件路径 (例如: "config/tld_whois_servers.yaml")
// concurrency: 并发请求数 (建议 10-20)
// progressCallback: 进度回调函数 (可选)
func DownloadWHOISConfig(destPath string, concurrency int, progressCallback func(progress, total int)) error {
	// 确保目标目录存在
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 获取 TLD 列表
	tlds, err := FetchTLDList()
	if err != nil {
		return fmt.Errorf("获取 TLD 列表失败: %w", err)
	}

	// 并发获取 WHOIS 服务器信息
	results := FetchWhoisServers(tlds, concurrency, progressCallback)

	// 格式化为 YAML
	yamlContent := FormatWhoisServersYAML(results)

	// 写入文件
	if err := os.WriteFile(destPath, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("写入 WHOIS 配置文件失败: %w", err)
	}

	return nil
}
