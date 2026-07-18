package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-whois/internal/cache"
	"go-whois/internal/config"
	"go-whois/internal/engine"
	"go-whois/internal/service"

	"github.com/spf13/cobra"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup [domain]",
	Short: "查询域名 WHOIS/RDAP 信息",
	Long:  `查询指定域名的 WHOIS 或 RDAP 注册信息，并以 JSON 格式输出`,
	Args:  cobra.ExactArgs(1),
	RunE:  runLookup,
}

var (
	protocol string
	verbose  bool
)

func init() {
	rootCmd.AddCommand(lookupCmd)
	lookupCmd.Flags().StringVarP(&protocol, "protocol", "p", "auto", "查询协议 (rdap|whois|auto)")
	lookupCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示详细信息")
}

func runLookup(cmd *cobra.Command, args []string) error {
	domain := args[0]

	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建引擎
	engines := createEngines(cfg)

	// 创建缓存
	cacheManager := cache.NewMemoryCache(cfg.Cache.Memory.MaxSize)

	// 创建查询服务
	lookupService := service.NewLookupService(engines, engine.NewNormalizer(), cacheManager, cfg)

	// 创建查询请求
	req := &engine.QueryRequest{
		Domain:   domain,
		Protocol: engine.Protocol(protocol),
	}

	// 执行查询
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := lookupService.Lookup(ctx, req)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}

	// 输出结果
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化结果失败: %w", err)
	}

	fmt.Println(string(output))

	return nil
}

// createEngines 创建查询引擎
func createEngines(cfg *config.Config) map[engine.Protocol]engine.Engine {
	engines := make(map[engine.Protocol]engine.Engine)

	// 创建 RDAP 引擎
	if cfg.Engine.RDAP.Enabled {
		rdapEngine := engine.NewRDAP(&cfg.Engine.RDAP)
		engines[engine.ProtocolRDAP] = rdapEngine
	}

	// 创建 WHOIS 引擎
	if cfg.Engine.WHOIS.Enabled {
		whoisEngine := engine.NewWHOIS(&cfg.Engine.WHOIS, nil, nil)
		engines[engine.ProtocolWHOIS] = whoisEngine
	}

	return engines
}
