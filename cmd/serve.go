package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/suguer/go-whois/internal/api"
	"github.com/suguer/go-whois/internal/cache"
	"github.com/suguer/go-whois/internal/config"
	"github.com/suguer/go-whois/internal/engine"
	"github.com/suguer/go-whois/internal/service"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 HTTP 服务器",
	Long:  `启动 Go-WHOIS HTTP API 服务器`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
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

	// 创建处理器
	handler := api.NewHandler(lookupService)

	// 设置路由
	router := api.SetupRouter(handler)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
		IdleTimeout:  cfg.Server.HTTP.IdleTimeout,
	}

	// 启动服务器
	go func() {
		fmt.Printf("启动 HTTP 服务器，监听地址: %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "服务器启动失败: %v\n", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器关闭失败: %w", err)
	}

	fmt.Println("服务器已关闭")
	return nil
}
