package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.go-whois")
		v.AddConfigPath("/etc/go-whois")
	}

	// 读取环境变量
	v.SetEnvPrefix("WHOIS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// 服务器配置
	v.SetDefault("server.http.host", "0.0.0.0")
	v.SetDefault("server.http.port", 8080)
	v.SetDefault("server.http.read_timeout", 30*time.Second)
	v.SetDefault("server.http.write_timeout", 30*time.Second)
	v.SetDefault("server.http.idle_timeout", 60*time.Second)
	v.SetDefault("server.http.shutdown_timeout", 10*time.Second)

	// CLI 配置
	v.SetDefault("server.cli.default_protocol", "auto")
	v.SetDefault("server.cli.default_output", "json")
	v.SetDefault("server.cli.verbose", false)

	// RDAP 配置
	v.SetDefault("engine.rdap.enabled", true)
	v.SetDefault("engine.rdap.timeout", 10*time.Second)
	v.SetDefault("engine.rdap.max_retries", 3)
	v.SetDefault("engine.rdap.retry_delay", 1*time.Second)
	v.SetDefault("engine.rdap.bootstrap_url", "https://data.iana.org/rdap/dns.json")
	v.SetDefault("engine.rdap.bootstrap_cache_ttl", 24*time.Hour)
	v.SetDefault("engine.rdap.user_agent", "go-whois/1.0")

	// WHOIS 配置
	v.SetDefault("engine.whois.enabled", true)
	v.SetDefault("engine.whois.timeout", 10*time.Second)
	v.SetDefault("engine.whois.max_retries", 3)
	v.SetDefault("engine.whois.retry_delay", 1*time.Second)
	v.SetDefault("engine.whois.default_port", 43)
	v.SetDefault("engine.whois.user_agent", "go-whois/1.0")

	// 协议优先级配置
	v.SetDefault("engine.priority.default", "rdap")

	// 缓存配置
	v.SetDefault("cache.memory.enabled", true)
	v.SetDefault("cache.memory.max_size", 10000)
	v.SetDefault("cache.memory.ttl", 1*time.Hour)
	v.SetDefault("cache.memory.cleanup_interval", 10*time.Minute)

	// Redis 配置
	v.SetDefault("cache.redis.enabled", false)
	v.SetDefault("cache.redis.host", "localhost")
	v.SetDefault("cache.redis.port", 6379)
	v.SetDefault("cache.redis.password", "")
	v.SetDefault("cache.redis.db", 0)
	v.SetDefault("cache.redis.ttl", 1*time.Hour)
	v.SetDefault("cache.redis.max_retries", 3)

	// 日志配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
	v.SetDefault("log.file_path", "logs/go-whois.log")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 3)
	v.SetDefault("log.max_age", 7)
	v.SetDefault("log.compress", true)

	// 限流配置
	v.SetDefault("ratelimit.enabled", true)
	v.SetDefault("ratelimit.rate", 100)
	v.SetDefault("ratelimit.burst", 200)
	v.SetDefault("ratelimit.cleanup_interval", 1*time.Minute)

	// 监控配置
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.port", 9090)

	// 健康检查配置
	v.SetDefault("health.path", "/health")
	v.SetDefault("health.timeout", 5*time.Second)
}
