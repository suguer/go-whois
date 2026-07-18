package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-whois",
	Short: "Go-WHOIS 域名查询服务",
	Long:  `Go-WHOIS 是一个支持 WHOIS 和 RDAP 协议的域名查询服务，提供 CLI 和 HTTP API 两种访问方式`,
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
