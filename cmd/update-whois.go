package cmd

import (
	"fmt"
	"os"

	"go-whois/pkg/whois"

	"github.com/spf13/cobra"
)

var updateWhoisCmd = &cobra.Command{
	Use:   "update-whois",
	Short: "更新 TLD WHOIS 服务器配置",
	Long:  `从 IANA 官网获取所有 TLD 的 WHOIS 服务器信息，更新配置文件`,
	RunE:  runUpdateWhois,
}

var (
	outputFile  string
	concurrency int
)

func init() {
	rootCmd.AddCommand(updateWhoisCmd)
	updateWhoisCmd.Flags().StringVarP(&outputFile, "output", "o", "config/tld_whois_servers.yaml", "输出文件路径")
	updateWhoisCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 20, "并发请求数")
}

func runUpdateWhois(cmd *cobra.Command, args []string) error {
	fmt.Println("正在从 IANA 获取 TLD 列表...")

	// 获取 TLD 列表
	tlds, err := whois.FetchTLDList()
	if err != nil {
		return fmt.Errorf("获取 TLD 列表失败: %w", err)
	}

	fmt.Printf("找到 %d 个 TLD，开始获取 WHOIS 服务器信息...\n", len(tlds))

	// 并发获取 WHOIS 服务器信息
	results := whois.FetchWhoisServers(tlds, concurrency, func(progress, total int) {
		if progress%50 == 0 || progress == total {
			fmt.Printf("\r进度: %d/%d", progress, total)
		}
	})

	// 统计
	whoisCount := 0
	for _, info := range results {
		if info.WhoisServer != "" {
			whoisCount++
		}
	}
	fmt.Printf("\n完成！共 %d 个 TLD，其中 %d 个有 WHOIS 服务器\n", len(results), whoisCount)

	// 生成 YAML 内容
	yamlContent := whois.FormatWhoisServersYAML(results)

	// 写入配置文件
	if err := os.WriteFile(outputFile, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	fmt.Printf("配置已写入: %s\n", outputFile)
	return nil
}
