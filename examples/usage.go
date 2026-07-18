// 示例：如何使用 go-whois 作为第三方库
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-whois/pkg/model"
	"go-whois/pkg/whois"
)

func main() {
	// 基本用法
	basicUsage()

	// 高级配置用法
	advancedUsage()

	// 使用 WHOIS 客户端
	whoisClientUsage()

	// 获取 RDAP Bootstrap 数据
	rdapBootstrapUsage()

	// 获取 WHOIS 服务器信息
	whoisServersUsage()
}

// 基本用法演示
func basicUsage() {
	fmt.Println("=== 基本用法 ===")

	// 创建客户端（使用默认配置）
	client := whois.NewClient()
	defer client.Close()

	// 查询域名
	result, err := client.Lookup("example.com")
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}

	// 输出结果
	fmt.Printf("域名: %s\n", result.DomainName)
	fmt.Printf("注册商: %s\n", result.RegistrarName)
	fmt.Printf("注册日期: %v\n", result.RegistrationDate)
	fmt.Printf("到期日期: %v\n", result.ExpirationDate)
	fmt.Printf("名称服务器: %v\n", result.NameServers)
	fmt.Println()
}

// 高级配置用法演示
func advancedUsage() {
	fmt.Println("=== 高级配置用法 ===")

	// 创建带自定义配置的客户端
	client := whois.NewClient(
		// 设置查询协议为 RDAP
		whois.WithProtocol(model.ProtocolRDAP),
		// 设置超时时间
		whois.WithTimeout(15 * time.Second),
		// 启用缓存
		whois.WithCache(true, 500, 30*time.Minute),
		// 设置自定义 RDAP Bootstrap URL
		whois.WithRDAPBootstrap("https://data.iana.org/rdap/dns.json"),
		// 设置 User-Agent
		whois.WithUserAgent("my-app/1.0"),
		// 包含原始响应
		whois.WithRawResponse(true),
	)
	defer client.Close()

	// 使用上下文查询
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := client.LookupWithContext(ctx, "google.com")
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}

	// 输出结果
	fmt.Printf("域名: %s\n", result.DomainName)
	fmt.Printf("ROID: %s\n", result.ROID)
	fmt.Printf("协议: %s\n", result.QueryProtocol)
	fmt.Printf("注册商: %s\n", result.RegistrarName)
	fmt.Printf("注册商 IANA ID: %s\n", result.RegistrarIANAID)
	fmt.Printf("状态: %v\n", result.Status)
	fmt.Println()
}

// WHOIS 客户端使用演示
func whoisClientUsage() {
	fmt.Println("=== WHOIS 客户端用法 ===")

	// 创建单独的 WHOIS 客户端
	whoisClient := whois.NewWHOISClient(
		whois.WithWSTimeout(10 * time.Second),
		whois.WithWSPort(43),
		whois.WithWSServers(map[string]string{
			".com": "whois.verisign-grs.com",
			".net": "whois.verisign-grs.com",
			".org": "whois.publicinterestregistry.org",
		}),
	)
	defer whoisClient.Close()

	// 使用 WHOIS 协议查询
	ctx := context.Background()
	result, err := whoisClient.Query(ctx, "github.com")
	if err != nil {
		log.Printf("WHOIS 查询失败: %v", err)
		return
	}

	// 输出结果
	fmt.Printf("域名: %s\n", result.DomainName)
	fmt.Printf("注册商: %s\n", result.RegistrarName)
	fmt.Printf("名称服务器: %v\n", result.NameServers)
	fmt.Printf("状态: %v\n", result.Status)
	fmt.Println()

	// 获取缓存统计（使用高级客户端）
	client := whois.NewClient()
	defer client.Close()

	stats := client.GetCacheStats()
	fmt.Printf("缓存统计: 启用=%v, 大小=%d\n", stats.Enabled, stats.Size)
}

// RDAP Bootstrap 使用演示
func rdapBootstrapUsage() {
	fmt.Println("=== RDAP Bootstrap 用法 ===")

	// 从 IANA 获取 RDAP Bootstrap 数据
	rdapEndpoints, err := whois.FetchRDAPBootstrap("https://data.iana.org/rdap/dns.json")
	if err != nil {
		log.Printf("获取 RDAP Bootstrap 失败: %v", err)
		return
	}

	fmt.Printf("获取到 %d 个 TLD 的 RDAP 端点\n", len(rdapEndpoints))

	// 显示部分端点
	count := 0
	for tld, endpoint := range rdapEndpoints {
		if count >= 5 {
			fmt.Println("...")
			break
		}
		fmt.Printf("  .%s -> %s\n", tld, endpoint)
		count++
	}
	fmt.Println()
}

// WHOIS 服务器信息获取演示
func whoisServersUsage() {
	fmt.Println("=== WHOIS 服务器信息获取 ===")

	// 从 IANA 获取 TLD 列表
	tlds, err := whois.FetchTLDList()
	if err != nil {
		log.Printf("获取 TLD 列表失败: %v", err)
		return
	}

	fmt.Printf("获取到 %d 个 TLD\n", len(tlds))

	// 获取单个 TLD 的 WHOIS 服务器
	server := whois.FetchWhoisServer("com")
	fmt.Printf("  .com WHOIS 服务器: %s\n", server)

	// 批量获取 WHOIS 服务器（限制并发数为 5）
	limitedTLDs := tlds
	if len(limitedTLDs) > 10 {
		limitedTLDs = tlds[:10] // 只获取前 10 个作为示例
	}

	fmt.Printf("批量获取 %d 个 TLD 的 WHOIS 服务器...\n", len(limitedTLDs))
	results := whois.FetchWhoisServers(limitedTLDs, 5, func(progress, total int) {
		fmt.Printf("\r进度: %d/%d", progress, total)
	})
	fmt.Println()

	// 显示结果
	whoisCount := 0
	for _, info := range results {
		if info.WhoisServer != "" {
			whoisCount++
			fmt.Printf("  .%s -> %s\n", info.TLD, info.WhoisServer)
		}
	}
	fmt.Printf("其中 %d 个有 WHOIS 服务器\n", whoisCount)

	// 获取 map 格式
	serversMap := whois.GetWhoisServersMap(results)
	fmt.Printf("Map 格式: %d 个条目\n", len(serversMap))
}