package whois

import (
	_ "embed"
)

// defaultWHOISConfig 内嵌的默认 WHOIS 服务器配置
// 用于第三方库在没有外部配置文件时的默认配置
//
//go:embed default_whois_servers.yaml
var defaultWHOISConfig []byte
