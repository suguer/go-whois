package service

import (
	"context"
	"fmt"
	"time"

	"go-whois/internal/cache"
	"go-whois/internal/config"
	"go-whois/internal/engine"
	"go-whois/internal/model"
	"go-whois/pkg/validator"
)

// LookupService 定义查询服务接口
type LookupService interface {
	// Lookup 执行单域名查询
	Lookup(ctx context.Context, req *engine.QueryRequest) (*model.DomainInfo, error)
}

// LookupServiceImpl 实现 LookupService
type LookupServiceImpl struct {
	engines    map[engine.Protocol]engine.Engine
	normalizer engine.Normalizer
	cache      cache.CacheManager
	config     *config.Config
}

// NewLookupService 创建新的查询服务实例
func NewLookupService(
	engines map[engine.Protocol]engine.Engine,
	normalizer engine.Normalizer,
	cache cache.CacheManager,
	config *config.Config,
) LookupService {
	return &LookupServiceImpl{
		engines:    engines,
		normalizer: normalizer,
		cache:      cache,
		config:     config,
	}
}

// Lookup 实现单域名查询
func (s *LookupServiceImpl) Lookup(ctx context.Context, req *engine.QueryRequest) (*model.DomainInfo, error) {
	startTime := time.Now()

	// 验证域名
	if err := validator.ValidateDomain(req.Domain); err != nil {
		return nil, fmt.Errorf("域名验证失败: %w", err)
	}

	// 规范化域名
	req.Domain = validator.NormalizeDomain(req.Domain)

	// 确定查询协议
	protocol := s.determineProtocol(req)

	// 检查缓存
	cacheKey := cache.GenerateCacheKey(string(protocol), req.Domain)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		cached.DataSource = "cache"
		return cached, nil
	}

	// 执行查询
	result, err := s.executeQuery(ctx, req.Domain, protocol)
	if err != nil {
		// 如果是自动模式，尝试切换协议
		if req.Protocol == engine.ProtocolAuto {
			result, err = s.tryFallbackProtocol(ctx, req.Domain, protocol)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// 设置查询耗时
	result.QueryDuration = time.Since(startTime).Milliseconds()

	// 写入缓存
	if err := s.cache.Set(ctx, cacheKey, result, s.config.Cache.Memory.TTL); err != nil {
		// 缓存写入失败不影响查询结果
		fmt.Printf("缓存写入失败: %v\n", err)
	}

	return result, nil
}

// determineProtocol 确定查询协议
func (s *LookupServiceImpl) determineProtocol(req *engine.QueryRequest) engine.Protocol {
	// 如果调用方指定了协议，使用指定的协议
	if req.Protocol != engine.ProtocolAuto {
		return req.Protocol
	}

	// 检查 TLD 配置
	if tldProtocol, ok := s.config.Engine.Priority.TLDOverride[req.Domain]; ok {
		return engine.Protocol(tldProtocol)
	}

	// 使用默认协议
	return engine.Protocol(s.config.Engine.Priority.Default)
}

// executeQuery 执行查询
func (s *LookupServiceImpl) executeQuery(ctx context.Context, domain string, protocol engine.Protocol) (*model.DomainInfo, error) {
	eng, ok := s.engines[protocol]
	if !ok {
		return nil, fmt.Errorf("不支持的查询协议: %s", protocol)
	}

	if !eng.IsAvailable() {
		return nil, fmt.Errorf("查询引擎 %s 不可用", protocol)
	}

	return eng.Query(ctx, domain)
}

// tryFallbackProtocol 尝试备用协议
func (s *LookupServiceImpl) tryFallbackProtocol(ctx context.Context, domain string, failedProtocol engine.Protocol) (*model.DomainInfo, error) {
	// 确定备用协议
	var fallbackProtocol engine.Protocol
	if failedProtocol == engine.ProtocolRDAP {
		fallbackProtocol = engine.ProtocolWHOIS
	} else {
		fallbackProtocol = engine.ProtocolRDAP
	}

	// 尝试备用协议
	result, err := s.executeQuery(ctx, domain, fallbackProtocol)
	if err != nil {
		return nil, fmt.Errorf("所有查询协议都失败，RDAP错误: %v, WHOIS错误: %v", err, err)
	}

	// 标记使用了备用协议
	result.QueryProtocol = string(fallbackProtocol)

	return result, nil
}
