package config

import (
	"time"
)

// Config 表示应用配置
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Engine    EngineConfig    `mapstructure:"engine"`
	Cache     CacheConfig     `mapstructure:"cache"`
	Log       LogConfig       `mapstructure:"log"`
	RateLimit RateLimitConfig `mapstructure:"ratelimit"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Health    HealthConfig    `mapstructure:"health"`
}

// ServerConfig 表示服务器配置
type ServerConfig struct {
	HTTP HTTPConfig `mapstructure:"http"`
	CLI  CLIConfig  `mapstructure:"cli"`
}

// HTTPConfig 表示 HTTP 服务器配置
type HTTPConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// CLIConfig 表示 CLI 配置
type CLIConfig struct {
	DefaultProtocol string `mapstructure:"default_protocol"`
	DefaultOutput   string `mapstructure:"default_output"`
	Verbose         bool   `mapstructure:"verbose"`
}

// EngineConfig 表示查询引擎配置
type EngineConfig struct {
	RDAP     RDAPConfig     `mapstructure:"rdap"`
	WHOIS    WHOISConfig    `mapstructure:"whois"`
	Priority PriorityConfig `mapstructure:"priority"`
}

// RDAPConfig 表示 RDAP 配置
type RDAPConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	Timeout           time.Duration `mapstructure:"timeout"`
	MaxRetries        int           `mapstructure:"max_retries"`
	RetryDelay        time.Duration `mapstructure:"retry_delay"`
	BootstrapURL      string        `mapstructure:"bootstrap_url"`
	BootstrapCacheTTL time.Duration `mapstructure:"bootstrap_cache_ttl"`
	UserAgent         string        `mapstructure:"user_agent"`
}

// WHOISConfig 表示 WHOIS 配置
type WHOISConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	Timeout     time.Duration `mapstructure:"timeout"`
	MaxRetries  int           `mapstructure:"max_retries"`
	RetryDelay  time.Duration `mapstructure:"retry_delay"`
	DefaultPort int           `mapstructure:"default_port"`
	UserAgent   string        `mapstructure:"user_agent"`
}

// PriorityConfig 表示协议优先级配置
type PriorityConfig struct {
	Default     string            `mapstructure:"default"`
	TLDOverride map[string]string `mapstructure:"tld_override"`
}

// CacheConfig 表示缓存配置
type CacheConfig struct {
	Memory MemoryCacheConfig `mapstructure:"memory"`
	Redis  RedisCacheConfig  `mapstructure:"redis"`
}

// MemoryCacheConfig 表示内存缓存配置
type MemoryCacheConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MaxSize         int           `mapstructure:"max_size"`
	TTL             time.Duration `mapstructure:"ttl"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// RedisCacheConfig 表示 Redis 缓存配置
type RedisCacheConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Host       string        `mapstructure:"host"`
	Port       int           `mapstructure:"port"`
	Password   string        `mapstructure:"password"`
	DB         int           `mapstructure:"db"`
	TTL        time.Duration `mapstructure:"ttl"`
	MaxRetries int           `mapstructure:"max_retries"`
}

// LogConfig 表示日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// RateLimitConfig 表示限流配置
type RateLimitConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Rate            int           `mapstructure:"rate"`
	Burst           int           `mapstructure:"burst"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// MetricsConfig 表示监控配置
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

// HealthConfig 表示健康检查配置
type HealthConfig struct {
	Path    string        `mapstructure:"path"`
	Timeout time.Duration `mapstructure:"timeout"`
}
