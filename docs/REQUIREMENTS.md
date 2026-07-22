# Go-WHOIS 域名查询服务系统 - 需求开发文档

> 文档版本：v1.0  
> 创建日期：2026-07-17  
> 文档状态：待评审  
> 编写人：需求分析师

---

## 目录

1. [项目概述与目标](#1-项目概述与目标)
2. [用户角色与使用场景](#2-用户角色与使用场景)
3. [功能需求详细说明](#3-功能需求详细说明)
4. [非功能需求](#4-非功能需求)
5. [技术架构设计建议](#5-技术架构设计建议)
6. [数据结构定义](#6-数据结构定义)
7. [API接口规范](#7-api接口规范)
8. [业务流程图及说明](#8-业务流程图及说明)
9. [业务规则与计算逻辑](#9-业务规则与计算逻辑)
10. [部署方案](#10-部署方案)
11. [开发计划与里程碑](#11-开发计划与里程碑)
12. [风险与假设](#12-风险与假设)
13. [附录](#13-附录)

---

## 1. 项目概述与目标

### 1.1 项目背景

在域名注册、交易、监控等业务场景中，频繁需要查询域名的注册信息（注册商、注册时间、过期时间、域名状态等）。传统WHOIS查询存在协议老旧、响应格式不统一、部分注册商限制访问等问题。RDAP（Registration Data Access Protocol）作为WHOIS的现代化替代方案，提供结构化JSON响应、分页支持、访问控制等优势。

本项目旨在构建一个高性能、易扩展的域名信息查询服务，统一WHOIS和RDAP两种查询协议的输出格式，为上层业务系统提供标准化的域名信息查询能力。

### 1.2 系统定位

| 属性 | 描述 |
|------|------|
| **系统类型** | CLI命令行工具 + HTTP API服务 |
| **主要功能** | 域名注册信息查询（WHOIS/RDAP） |
| **目标用户** | 开发者、运维人员、自动化脚本、业务系统 |
| **查询对象** | 仅支持域名查询（不支持IP、ASN等其他对象） |
| **输出格式** | 统一JSON格式输出 |

### 1.3 项目目标

| 目标编号 | 目标描述 | 量化指标 |
|----------|----------|----------|
| G-01 | 提供统一的域名查询接口 | 支持CLI和HTTP两种访问方式 |
| G-02 | 支持RDAP和WHOIS双协议 | RDAP优先，可配置切换 |
| G-03 | 满足业务查询量需求 | 支持每日10,000+次查询 |
| G-04 | 降低外部依赖风险 | 多注册商轮询、失败自动切换 |
| G-05 | 提供良好的查询性能 | 单次查询响应时间 < 3秒（P95） |

### 1.4 项目范围

**包含范围（In Scope）**：
- 域名WHOIS查询
- 域名RDAP查询
- CLI命令行工具
- HTTP API服务
- 本地查询缓存
- 查询结果JSON标准化输出

**不包含范围（Out of Scope）**：
- IP地址查询
- ASN查询
- 域名注册/续费等写操作
- Web管理界面
- 用户认证系统
- 数据持久化存储（仅缓存）

---

## 2. 用户角色与使用场景

### 2.1 用户角色定义

| 角色 | 描述 | 使用方式 | 典型需求 |
|------|------|----------|----------|
| **开发者** | 集成域名查询能力到自有系统 | HTTP API调用 | 批量查询、API集成 |
| **运维人员** | 日常运维和故障排查 | CLI命令行 | 单次查询、快速验证 |
| **自动化脚本** | CI/CD流水线、监控脚本 | CLI/HTTP | 批量查询、定时任务 |
| **业务系统** | 域名交易平台、监控平台 | HTTP API | 高频查询、数据标准化 |

### 2.2 核心使用场景

#### 场景1：单域名快速查询
```
作为运维人员，
我想要通过命令行快速查询一个域名的注册信息，
以便在故障排查时快速确认域名状态和归属。
```

**验收标准**：
- 给定一个有效域名（如 example.com），当执行CLI查询命令时，系统在3秒内返回JSON格式的域名注册信息
- 返回信息包含：域名名称、注册商、注册时间、过期时间、域名状态、名称服务器

#### 场景2：指定RDAP协议查询
```
作为开发者，
我想要在调用API时指定使用RDAP协议查询，
以便获取更结构化的域名注册数据。
```

**验收标准**：
- 给定一个支持RDAP的域名（如 google.com），当请求中指定协议为rdap时，系统使用RDAP协议进行查询
- 返回结果中标记实际使用的查询协议为rdap

#### 场景3：批量域名查询（可选）
```
作为业务系统，
我想要通过API批量查询多个域名的注册信息，
以便提高查询效率、减少网络开销。
```

**验收标准**：
- 给定不超过50个域名的列表，当提交批量查询请求时，系统返回所有域名的查询结果
- 每个域名的查询结果独立返回，单个失败不影响其他域名
- 整批查询在30秒内完成

#### 场景4：缓存命中查询
```
作为开发者，
我想要在短时间内重复查询同一域名时直接获取缓存结果，
以便提高查询速度、降低对外部服务的依赖。
```

**验收标准**：
- 给定一个域名在缓存有效期内（如1小时），当再次查询该域名时，系统直接返回缓存结果
- 响应时间 < 50毫秒
- 响应中标记数据来源为cache

#### 场景5：协议自动切换
```
作为运维人员，
我希望当RDAP查询失败时系统能自动切换到WHOIS协议重试，
以便确保查询成功率。
```

**验收标准**：
- 给定一个RDAP服务不可用的域名，当系统尝试RDAP查询失败时，自动切换到WHOIS协议重试
- 最终返回成功的查询结果
- 响应中标记实际使用的协议为whois（回退）

---

## 3. 功能需求详细说明

### 3.1 功能需求总览

| 需求编号 | 功能模块 | 功能名称 | 优先级 | 故事级别 |
|----------|----------|----------|--------|----------|
| FR-01 | 查询引擎 | 域名WHOIS查询 | P0 | 必须 |
| FR-02 | 查询引擎 | 域名RDAP查询 | P0 | 必须 |
| FR-03 | 查询引擎 | RDAP优先策略 | P0 | 必须 |
| FR-04 | 查询引擎 | 协议自动切换 | P0 | 必须 |
| FR-05 | 查询引擎 | 后缀级别协议配置 | P1 | 必须 |
| FR-06 | CLI接口 | 命令行单域名查询 | P0 | 必须 |
| FR-07 | CLI接口 | 命令行批量查询 | P2 | 可选 |
| FR-08 | CLI接口 | 查询协议指定 | P0 | 必须 |
| FR-09 | CLI接口 | 输出格式选项 | P1 | 应当 |
| FR-10 | HTTP API | 单域名查询接口 | P0 | 必须 |
| FR-11 | HTTP API | 批量查询接口 | P2 | 可选 |
| FR-12 | HTTP API | 健康检查接口 | P0 | 必须 |
| FR-13 | 缓存模块 | 查询结果缓存 | P1 | 必须 |
| FR-14 | 缓存模块 | 缓存过期清理 | P1 | 必须 |
| FR-15 | 输出标准化 | WHOIS结果标准化 | P0 | 必须 |
| FR-16 | 输出标准化 | RDAP结果标准化 | P0 | 必须 |
| FR-17 | 配置管理 | 配置文件加载 | P0 | 必须 |
| FR-18 | 配置管理 | 环境变量支持 | P1 | 应当 |

### 3.2 FR-01：域名WHOIS查询

**用户故事**：
```
作为调用方，我想要通过WHOIS协议查询域名注册信息，以便获取域名的基本注册数据。
```

**功能描述**：
- 根据输入的域名，自动识别顶级域名后缀（TLD）
- 查询该TLD对应的WHOIS服务器
- 建立TCP连接（端口43）发送查询请求
- 接收并解析WHOIS文本响应
- 提取关键字段并结构化

**验收标准（GWT格式）**：

| 编号 | 给定条件 | 操作 | 期望结果 |
|------|----------|------|----------|
| AC-01-01 | 输入有效域名 example.com | 执行WHOIS查询 | 返回该域名的注册信息JSON |
| AC-01-02 | 输入未注册域名 xyzxyz123abc.com | 执行WHOIS查询 | 返回状态为"未注册" |
| AC-01-03 | 输入无效域名格式 "not-a-domain" | 执行WHOIS查询 | 返回错误信息：域名格式无效 |
| AC-01-04 | WHOIS服务器超时（>10秒） | 执行WHOIS查询 | 返回超时错误，自动尝试备用服务器 |
| AC-01-05 | WHOIS服务器拒绝连接 | 执行WHOIS查询 | 返回连接错误，触发协议切换 |

### 3.3 FR-02：域名RDAP查询

**用户故事**：
```
作为调用方，我想要通过RDAP协议查询域名注册信息，以便获取结构化的域名注册数据。
```

**功能描述**：
- 根据输入的域名，查询该TLD对应的RDAP服务端点
- 发送HTTP GET请求到 `/domain/{domainName}` 端点
- 接收JSON格式的RDAP响应
- 解析并提取标准化字段

**验收标准（GWT格式）**：

| 编号 | 给定条件 | 操作 | 期望结果 |
|------|----------|------|----------|
| AC-02-01 | 输入支持RDAP的域名 google.com | 执行RDAP查询 | 返回结构化域名信息JSON |
| AC-02-02 | 输入未注册域名 | 执行RDAP查询 | 返回404状态，标记为未注册 |
| AC-02-03 | RDAP服务返回429（限流） | 执行RDAP查询 | 等待Retry-After后重试，最多重试3次 |
| AC-02-04 | RDAP服务返回5xx | 执行RDAP查询 | 返回错误，触发协议切换到WHOIS |

### 3.4 FR-03：RDAP优先策略

**用户故事**：
```
作为调用方，我想要系统默认优先使用RDAP协议查询，以便获取更高质量的结构化数据。
```

**功能描述**：
- 默认查询策略：优先使用RDAP协议
- RDAP查询失败时，自动切换到WHOIS协议重试
- 调用方可通过参数显式指定使用哪种协议

**验收标准（GWT格式）**：

| 编号 | 给定条件 | 操作 | 期望结果 |
|------|----------|------|----------|
| AC-03-01 | 未指定协议，域名支持RDAP | 执行查询 | 使用RDAP协议查询，返回结果 |
| AC-03-02 | 未指定协议，RDAP查询失败 | 执行查询 | 自动切换到WHOIS重试，返回结果 |
| AC-03-03 | 指定协议为rdap | 执行查询 | 仅使用RDAP协议，失败不自动切换 |
| AC-03-04 | 指定协议为whois | 执行查询 | 仅使用WHOIS协议，不尝试RDAP |

### 3.5 FR-05：后缀级别协议配置

**用户故事**：
```
作为系统管理员，我想要为特定域名后缀配置优先使用的查询协议，以便优化不同后缀的查询成功率。
```

**功能描述**：
- 支持在配置文件中为特定TLD设置优先协议
- 配置格式：`tld_protocol_map`，如 `{"cn": "whois", "jp": "whois"}`
- 未配置的TLD使用默认策略（RDAP优先）

**验收标准（GWT格式）**：

| 编号 | 给定条件 | 操作 | 期望结果 |
|------|----------|------|----------|
| AC-05-01 | 配置 cn 后缀优先使用whois | 查询 example.cn | 优先使用WHOIS协议查询 |
| AC-05-02 | 未配置 com 后缀 | 查询 example.com | 使用默认策略（RDAP优先） |
| AC-05-03 | 配置了后缀优先级，但调用方指定协议 | 查询时指定rdap | 以调用方指定为准，忽略配置 |

### 3.6 FR-06/FR-07：CLI命令行查询

**用户故事**：
```
作为运维人员，我想要通过命令行快速查询域名信息，以便在终端中快速获取结果。
```

**功能描述**：
- 单域名查询：`go-whois lookup example.com`
- 批量查询：`go-whois lookup example.com google.com`
- 文件批量查询：`go-whois lookup --file domains.txt`
- 协议指定：`go-whois lookup --protocol rdap example.com`
- 输出格式：默认JSON，支持 `--pretty` 美化输出

**验收标准（GWT格式）**：

| 编号 | 给定条件 | 操作 | 期望结果 |
|------|----------|------|----------|
| AC-06-01 | 命令行输入 `go-whois lookup example.com` | 执行命令 | 输出该域名的JSON格式查询结果 |
| AC-06-02 | 命令行输入 `go-whois lookup --protocol whois example.com` | 执行命令 | 使用WHOIS协议查询并输出结果 |
| AC-06-03 | 输入 `go-whois lookup --file domains.txt`，文件包含10个域名 | 执行命令 | 输出所有域名的查询结果数组 |
| AC-06-04 | 输入 `go-whois lookup example.com --pretty` | 执行命令 | 输出美化格式的JSON（缩进、换行） |
| AC-06-05 | 查询失败的域名 | 批量查询 | 失败域名返回错误信息，不影响其他域名 |

### 3.7 FR-10/FR-11/FR-12：HTTP API接口

**用户故事**：
```
作为开发者，我想要通过HTTP API调用域名查询服务，以便将查询能力集成到我的系统中。
```

**功能描述**：
- 单域名查询：`GET /api/v1/lookup/{domain}`
- 批量查询：`POST /api/v1/lookup/batch`
- 健康检查：`GET /health`
- 协议参数：`?protocol=rdap|whois`
- 响应格式：统一JSON

**验收标准（GWT格式）**：

| 编号 | 给定条件 | 操作 | 期望结果 |
|------|----------|------|----------|
| AC-10-01 | 发送 GET /api/v1/lookup/example.com | API调用 | 返回200及域名查询结果JSON |
| AC-10-02 | 发送 GET /api/v1/lookup/example.com?protocol=whois | API调用 | 使用WHOIS协议查询并返回结果 |
| AC-10-03 | 发送 GET /api/v1/lookup/invalid-domain | API调用 | 返回400及错误信息 |
| AC-10-04 | 发送 POST /api/v1/lookup/batch（body含域名列表） | API调用 | 返回200及所有域名查询结果 |
| AC-10-05 | 发送 GET /health | API调用 | 返回200及服务状态信息 |
| AC-10-06 | 服务端口被占用 | 启动服务 | 输出端口占用错误信息 |

### 3.8 FR-13/FR-14：缓存模块

**用户故事**：
```
作为调用方，我想要重复查询同一域名时能快速获取结果，以便提高查询效率。
```

**功能描述**：
- 内存缓存：使用进程内LRU缓存
- 缓存键：`{protocol}:{domain}`
- 默认过期时间：1小时（可配置）
- 支持禁用缓存：`--no-cache` 参数或配置

**验收标准（GWT格式）**：

| 编号 | 给定条件 | 操作 | 期望结果 |
|------|----------|------|----------|
| AC-13-01 | 首次查询 example.com | 执行查询 | 正常查询并缓存结果 |
| AC-13-02 | 缓存未过期，再次查询 example.com | 执行查询 | 返回缓存结果，响应时间 < 50ms |
| AC-13-03 | 缓存已过期，查询 example.com | 执行查询 | 重新查询并更新缓存 |
| AC-13-04 | 使用 --no-cache 参数查询 | 执行查询 | 跳过缓存，直接查询 |
| AC-13-05 | 缓存达到最大容量 | 新增缓存 | 淘汰最久未使用的缓存项 |

---

## 4. 非功能需求

### 4.1 性能需求

| 需求编号 | 性能指标 | 目标值 | 测量方法 |
|----------|----------|--------|----------|
| NFR-01 | 单次查询响应时间（P50） | < 1.5秒 | API监控统计 |
| NFR-02 | 单次查询响应时间（P95） | < 3秒 | API监控统计 |
| NFR-03 | 单次查询响应时间（P99） | < 5秒 | API监控统计 |
| NFR-04 | 缓存命中响应时间 | < 50毫秒 | 日志统计 |
| NFR-05 | 每日查询吞吐量 | > 10,000次/日 | 监控统计 |
| NFR-06 | 并发查询能力 | > 50 QPS | 压力测试 |
| NFR-07 | 服务启动时间 | < 3秒 | 启动日志 |

### 4.2 可用性需求

| 需求编号 | 可用性指标 | 目标值 | 说明 |
|----------|------------|--------|------|
| NFR-08 | 服务可用率 | > 99.5% | 排除外部WHOIS/RDAP服务不可用 |
| NFR-09 | 查询成功率 | > 95% | 含协议自动切换后的最终成功率 |
| NFR-10 | 故障恢复时间 | < 5分钟 | 服务异常自动重启 |

### 4.3 扩展性需求

| 需求编号 | 扩展性指标 | 目标值 | 说明 |
|----------|------------|--------|------|
| NFR-11 | 新增TLD支持 | 配置化 | 无需代码修改，更新配置即可 |
| NFR-12 | 新增RDAP服务端点 | 配置化 | 支持动态添加RDAP bootstrap |
| NFR-13 | 缓存策略可配置 | 支持 | 可切换内存/Redis缓存 |
| NFR-14 | 日志级别可配置 | 支持 | debug/info/warn/error |

### 4.4 安全需求

| 需求编号 | 安全指标 | 目标值 | 说明 |
|----------|----------|--------|------|
| NFR-15 | 输入验证 | 严格 | 域名格式验证、防注入 |
| NFR-16 | 速率限制 | 可配置 | 防止单IP过度请求 |
| NFR-17 | 错误信息脱敏 | 必须 | 不暴露内部服务地址 |
| NFR-18 | 日志脱敏 | 必须 | 不记录敏感查询参数 |

### 4.5 可维护性需求

| 需求编号 | 维护性指标 | 目标值 | 说明 |
|----------|------------|--------|------|
| NFR-19 | 代码测试覆盖率 | > 70% | 单元测试 + 集成测试 |
| NFR-20 | 日志结构化 | 必须 | JSON格式日志，便于ELK收集 |
| NFR-21 | 指标暴露 | 必须 | 支持Prometheus指标导出 |
| NFR-22 | 配置热更新 | 可选 | 支持SIGHUP信号重载配置 |

---

## 5. 技术架构设计建议

### 5.1 整体架构图

```
+------------------------------------------------------------------+
|                          调用层 (Clients)                         |
+------------------------------------------------------------------+
|        CLI客户端              |         HTTP客户端                |
+-------------------------------+----------------------------------+
                                |
+------------------------------------------------------------------+
|                        接口层 (API Layer)                         |
+------------------------------------------------------------------+
|   CLI命令解析 (cobra)         |    HTTP服务器 (gin/chi)          |
|   - lookup 子命令             |    - /api/v1/lookup/{domain}     |
|   - 参数解析                  |    - /api/v1/lookup/batch        |
|   - 输出格式化                |    - /health                     |
+-------------------------------+----------------------------------+
                                |
+------------------------------------------------------------------+
|                       服务层 (Service Layer)                      |
+------------------------------------------------------------------+
|                    LookupService (查询调度器)                     |
|   - 接收查询请求                                                  |
|   - 确定查询协议策略                                              |
|   - 调用对应查询引擎                                              |
|   - 结果标准化处理                                                |
+------------------------------------------------------------------+
                                |
+------------------------------------------------------------------+
|                       引擎层 (Engine Layer)                       |
+------------------------------------------------------------------+
|   RDAP Engine               |        WHOIS Engine               |
|   - Bootstrap查询           |    - TLD WHOIS服务器查询          |
|   - HTTP客户端              |    - TCP连接管理                  |
|   - JSON响应解析            |    - 文本响应解析                 |
|   - 重试与限流处理          |    - 备用服务器切换               |
+-------------------------------+----------------------------------+
                                |
+------------------------------------------------------------------+
|                       缓存层 (Cache Layer)                        |
+------------------------------------------------------------------+
|                    CacheManager (缓存管理器)                      |
|   - 内存LRU缓存（默认）                                          |
|   - 可选Redis缓存                                                |
|   - TTL过期管理                                                  |
|   - 缓存键生成策略                                                |
+------------------------------------------------------------------+
                                |
+------------------------------------------------------------------+
|                      基础设施层 (Infrastructure)                   |
+------------------------------------------------------------------+
|   配置管理        |   日志系统        |   指标监控               |
|   (viper)         |   (zap/zerolog)   |   (prometheus)           |
+-------------------+-------------------+-------------------------+
```

### 5.2 核心模块说明

#### 5.2.1 查询调度器（LookupService）

**职责**：
- 接收上层查询请求
- 根据配置和请求参数确定查询策略
- 调用对应的查询引擎
- 处理查询失败时的协议切换
- 统一结果格式

**设计要点**：
- 使用策略模式实现协议选择
- 支持并发查询（批量场景）
- 记录查询耗时和结果统计

#### 5.2.2 RDAP引擎（RDAPEngine）

**职责**：
- 维护TLD到RDAP服务端点的映射（通过IANA Bootstrap文件）
- 发送HTTP请求到RDAP服务
- 解析RDAP JSON响应
- 处理RDAP特有的错误码（404未注册、429限流等）

**设计要点**：
- 定期更新IANA RDAP Bootstrap文件（每日或每周）
- 使用HTTP客户端池复用连接
- 支持RDAP扩展字段解析

#### 5.2.3 WHOIS引擎（WHOISEngine）

**职责**：
- 维护TLD到WHOIS服务器的映射
- 建立TCP连接发送查询
- 接收并解析WHOIS文本响应
- 处理WHOIS服务器的特殊响应格式

**设计要点**：
- 使用连接池管理TCP连接
- 设置合理的超时时间（连接超时5秒，读取超时10秒）
- 支持WHOIS服务器轮询（同一TLD多个服务器）

#### 5.2.4 结果标准化器（ResultNormalizer）

**职责**：
- 将不同来源（RDAP/WHOIS）的结果转换为统一格式
- 字段名称标准化
- 日期格式统一
- 状态值枚举化

#### 5.2.5 缓存管理器（CacheManager）

**职责**：
- 管理查询结果缓存
- 生成缓存键
- 处理缓存过期和淘汰
- 提供缓存统计信息

**设计要点**：
- 默认使用进程内LRU缓存（如 `groupcache` 或自实现）
- 缓存键格式：`{protocol}:{normalized_domain}`
- 支持缓存预热（常用域名提前查询）
- 可选Redis作为分布式缓存

### 5.3 技术栈建议

| 组件 | 推荐技术 | 备选方案 | 选择理由 |
|------|----------|----------|----------|
| **编程语言** | Go 1.21+ | - | 高并发、编译部署简单、标准库丰富 |
| **CLI框架** | cobra | urfave/cli | Go生态最成熟的CLI框架 |
| **HTTP框架** | gin | chi, echo | 性能优秀、中间件丰富、社区活跃 |
| **配置管理** | viper | envconfig | 支持多格式配置、环境变量、热更新 |
| **日志库** | zap | zerolog | 高性能结构化日志 |
| **缓存库** | groupcache | bigcache, ristretto | Google出品、无GC压力 |
| **HTTP客户端** | 标准库 net/http | resty | 标准库足够、无额外依赖 |
| **指标监控** | prometheus client | - | 行业标准、Grafana生态 |
| **RDAP Bootstrap** | IANA官方 | 自维护 | 权威数据源 |

### 5.4 项目目录结构建议

```
go-whois/
├── cmd/                          # 命令行入口
│   ├── root.go                   # 根命令
│   └── lookup.go                 # lookup子命令
├── internal/                     # 内部包（不对外暴露）
│   ├── config/                   # 配置管理
│   │   └── config.go
│   ├── service/                  # 业务服务层
│   │   └── lookup.go
│   ├── engine/                   # 查询引擎
│   │   ├── rdap.go
│   │   ├── whois.go
│   │   └── normalizer.go
│   ├── cache/                    # 缓存管理
│   │   └── cache.go
│   ├── api/                      # HTTP API
│   │   ├── handler.go
│   │   └── router.go
│   └── model/                    # 数据模型
│       └── domain.go
├── pkg/                          # 可复用的工具包
│   ├── validator/                # 域名验证
│   └── tld/                      # TLD工具
├── config/                       # 配置文件
│   ├── config.yaml               # 主配置文件
│   ├── tld_whois_servers.yaml    # TLD WHOIS服务器映射
│   └── rdap_bootstrap.json       # RDAP Bootstrap数据
├── main.go                       # 程序入口
├── go.mod
├── go.sum
└── Makefile
```

### 5.5 依赖关系图

```
main.go
  └── cmd/
        ├── root.go
        │     └── internal/config
        └── lookup.go
              └── internal/service/lookup
                    ├── internal/engine/rdap
                    │     └── internal/model
                    ├── internal/engine/whois
                    │     └── internal/model
                    ├── internal/engine/normalizer
                    │     └── internal/model
                    └── internal/cache
```

---

## 6. 数据结构定义

### 6.1 核心数据模型

#### 6.1.1 域名查询结果（DomainInfo）

```json
{
  "domain_name": "example.com",
  "query_protocol": "rdap",
  "query_time": "2026-07-17T10:30:00Z",
  "query_duration_ms": 1234,
  "data_source": "live",
  "registration": {
    "registrar": {
      "name": "Example Registrar Inc.",
      "url": "https://www.example-registrar.com",
      "iana_id": "1234"
    },
    "registrant": {
      "name": "REDACTED FOR PRIVACY",
      "organization": "Example Organization",
      "country": "US",
      "state": "CA",
      "city": "Los Angeles"
    },
    "registration_date": "1995-08-14T04:00:00Z",
    "expiration_date": "2026-08-13T04:00:00Z",
    "last_updated": "2025-08-14T04:00:00Z"
  },
  "status": [
    "clientDeleteProhibited",
    "clientTransferProhibited",
    "clientUpdateProhibited",
    "serverDeleteProhibited",
    "serverTransferProhibited",
    "serverUpdateProhibited"
  ],
  "name_servers": [
    "ns1.example.com",
    "ns2.example.com"
  ],
  "dnssec": {
    "signed": true,
    "delegation_signed": true
  },
  "raw_response": null
}
```

#### 6.1.2 字段说明

| 字段路径 | 类型 | 必填 | 说明 |
|----------|------|------|------|
| `domain_name` | string | 是 | 查询的域名（小写、去除末尾点号） |
| `query_protocol` | string | 是 | 实际使用的查询协议：`rdap` / `whois` |
| `query_time` | string (ISO8601) | 是 | 查询时间（UTC） |
| `query_duration_ms` | integer | 是 | 查询耗时（毫秒） |
| `data_source` | string | 是 | 数据来源：`live` / `cache` |
| `registration.registrar.name` | string | 否 | 注册商名称 |
| `registration.registrar.url` | string | 否 | 注册商网站 |
| `registration.registrar.iana_id` | string | 否 | 注册商IANA ID |
| `registration.registrant.name` | string | 否 | 注册人姓名（可能脱敏） |
| `registration.registrant.organization` | string | 否 | 注册人组织 |
| `registration.registrant.country` | string | 否 | 注册人国家代码（ISO 3166-1） |
| `registration.registration_date` | string (ISO8601) | 否 | 域名注册日期 |
| `registration.expiration_date` | string (ISO8601) | 否 | 域名过期日期 |
| `registration.last_updated` | string (ISO8601) | 否 | 最后更新日期 |
| `status` | string[] | 是 | 域名状态列表（EPP状态码） |
| `name_servers` | string[] | 否 | 名称服务器列表 |
| `dnssec.signed` | boolean | 否 | 是否启用DNSSEC签名 |
| `dnssec.delegation_signed` | boolean | 否 | 是否委托签名 |
| `raw_response` | string/null | 否 | 原始响应（调试模式下返回） |

#### 6.1.3 错误响应（ErrorResponse）

```json
{
  "error": {
    "code": "DOMAIN_NOT_FOUND",
    "message": "The requested domain was not found",
    "details": "Domain example.xyz is not registered",
    "request_id": "req_abc123def456"
  }
}
```

| 错误码 | HTTP状态码 | 说明 |
|--------|------------|------|
| `INVALID_DOMAIN` | 400 | 域名格式无效 |
| `DOMAIN_NOT_FOUND` | 404 | 域名未注册 |
| `QUERY_TIMEOUT` | 504 | 查询超时 |
| `PROTOCOL_ERROR` | 502 | 协议查询失败 |
| `RATE_LIMITED` | 429 | 请求过于频繁 |
| `INTERNAL_ERROR` | 500 | 内部服务器错误 |

#### 6.1.4 批量查询请求（BatchLookupRequest）

```json
{
  "domains": ["example.com", "google.com", "github.com"],
  "protocol": "auto",
  "include_raw": false
}
```

#### 6.1.5 批量查询响应（BatchLookupResponse）

```json
{
  "results": [
    {
      "domain": "example.com",
      "success": true,
      "data": { /* DomainInfo */ }
    },
    {
      "domain": "invalid-domain",
      "success": false,
      "error": {
        "code": "INVALID_DOMAIN",
        "message": "Domain format is invalid"
      }
    }
  ],
  "total": 3,
  "success_count": 2,
  "failure_count": 1,
  "query_duration_ms": 5678
}
```

#### 6.1.6 健康检查响应（HealthResponse）

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "cache": {
    "enabled": true,
    "size": 1234,
    "hit_rate": 0.85
  },
  "stats": {
    "total_queries": 100000,
    "success_queries": 95000,
    "avg_query_duration_ms": 1500
  }
}
```

### 6.2 配置数据结构

```yaml
# config.yaml 示例
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

query:
  default_protocol: "auto"          # auto | rdap | whois
  rdap_timeout: 10s
  whois_timeout: 10s
  max_retries: 2
  tld_protocol_map:                 # TLD级别协议覆盖
    cn: "whois"
    jp: "whois"
    tw: "whois"

cache:
  enabled: true
  type: "memory"                    # memory | redis
  ttl: 1h
  max_size: 10000
  # redis配置（type=redis时生效）
  # redis:
  #   addr: "localhost:6379"
  #   password: ""
  #   db: 0

rate_limit:
  enabled: true
  requests_per_second: 100
  burst: 200

logging:
  level: "info"                     # debug | info | warn | error
  format: "json"                    # json | text
  output: "stdout"                  # stdout | file

metrics:
  enabled: true
  path: "/metrics"
```

---

## 7. API接口规范

### 7.1 接口总览

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | `/api/v1/lookup/{domain}` | 单域名查询 | 可选 |
| POST | `/api/v1/lookup/batch` | 批量域名查询（可选） | 可选 |
| GET | `/health` | 健康检查 | 无 |
| GET | `/metrics` | 指标导出 | 无 |

### 7.2 单域名查询接口

**请求**：
```
GET /api/v1/lookup/{domain}?protocol={protocol}&include_raw={boolean}
```

**路径参数**：

| 参数 | 类型 | 必填 | 说明 | 示例 |
|------|------|------|------|------|
| `domain` | string | 是 | 要查询的域名 | `example.com` |

**查询参数**：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `protocol` | string | 否 | `auto` | 查询协议：`auto` / `rdap` / `whois` |
| `include_raw` | boolean | 否 | `false` | 是否包含原始响应 |

**响应示例（成功）**：
```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "domain_name": "example.com",
  "query_protocol": "rdap",
  "query_time": "2026-07-17T10:30:00Z",
  "query_duration_ms": 1234,
  "data_source": "live",
  "registration": {
    "registrar": {
      "name": "RESERVED-Internet Assigned Numbers Authority",
      "url": "http://www.iana.org",
      "iana_id": "376"
    },
    "registration_date": "1995-08-14T04:00:00Z",
    "expiration_date": "2026-08-13T04:00:00Z",
    "last_updated": "2025-08-14T04:00:00Z"
  },
  "status": [
    "clientDeleteProhibited",
    "clientTransferProhibited",
    "clientUpdateProhibited",
    "serverDeleteProhibited",
    "serverTransferProhibited",
    "serverUpdateProhibited"
  ],
  "name_servers": [
    "a.iana-servers.net",
    "b.iana-servers.net"
  ]
}
```

**响应示例（失败）**：
```json
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": {
    "code": "INVALID_DOMAIN",
    "message": "Domain format is invalid",
    "details": "The domain 'not-a-domain' does not match valid domain pattern",
    "request_id": "req_abc123def456"
  }
}
```

### 7.3 批量查询接口

**请求**：
```
POST /api/v1/lookup/batch
Content-Type: application/json

{
  "domains": ["example.com", "google.com", "github.com"],
  "protocol": "auto",
  "include_raw": false
}
```

**请求体参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `domains` | string[] | 是 | 域名列表，最多50个 |
| `protocol` | string | 否 | 查询协议，默认`auto` |
| `include_raw` | boolean | 否 | 是否包含原始响应，默认`false` |

**响应示例**：
```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "results": [
    {
      "domain": "example.com",
      "success": true,
      "data": { /* DomainInfo对象 */ }
    },
    {
      "domain": "google.com",
      "success": true,
      "data": { /* DomainInfo对象 */ }
    },
    {
      "domain": "notregistered123.com",
      "success": false,
      "error": {
        "code": "DOMAIN_NOT_FOUND",
        "message": "Domain is not registered"
      }
    }
  ],
  "total": 3,
  "success_count": 2,
  "failure_count": 1,
  "query_duration_ms": 5678
}
```

### 7.4 健康检查接口

**请求**：
```
GET /health
```

**响应示例**：
```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "cache": {
    "enabled": true,
    "size": 1234,
    "hit_rate": 0.85
  }
}
```

### 7.5 错误码汇总

| 错误码 | HTTP状态码 | 说明 | 处理建议 |
|--------|------------|------|----------|
| `INVALID_DOMAIN` | 400 | 域名格式无效 | 检查域名格式 |
| `DOMAIN_NOT_FOUND` | 404 | 域名未注册 | 正常业务状态 |
| `QUERY_TIMEOUT` | 504 | 查询超时 | 重试或切换协议 |
| `PROTOCOL_ERROR` | 502 | 协议查询失败 | 重试或切换协议 |
| `RATE_LIMITED` | 429 | 请求过于频繁 | 等待后重试 |
| `BATCH_SIZE_EXCEEDED` | 400 | 批量查询数量超限 | 减少批量数量 |
| `INTERNAL_ERROR` | 500 | 内部服务器错误 | 联系管理员 |

---

## 8. 业务流程图及说明

### 8.1 单域名查询主流程

```
开始
  |
  v
接收查询请求（域名 + 协议参数）
  |
  v
域名格式验证
  |
  +----无效----> 返回错误：INVALID_DOMAIN
  |
  +----有效---->
  |
  v
确定查询协议
  |
  +--- 调用方指定协议 ---> 使用指定协议
  |
  +--- 未指定（auto模式）--->
  |     |
  |     v
  |     检查TLD配置
  |     |
  |     +--- TLD配置了协议 ---> 使用配置的协议
  |     |
  |     +--- TLD未配置 ---> 使用默认策略（RDAP优先）
  |
  v
检查缓存
  |
  +--- 缓存命中且未过期 ---> 返回缓存结果（data_source=cache）
  |
  +--- 缓存未命中或已过期 --->
  |
  v
执行查询
  |
  +--- RDAP查询 --->
  |     |
  |     v
  |     查询RDAP Bootstrap获取端点
  |     |
  |     v
  |     发送HTTP请求
  |     |
  |     +--- 成功 ---> 解析响应 ---> 标准化结果
  |     |
  |     +--- 失败 --->
  |           |
  |           v
  |           检查是否为auto模式
  |           |
  |           +--- 是 ---> 切换到WHOIS协议重试
  |           |
  |           +--- 否 ---> 返回错误
  |
  +--- WHOIS查询 --->
        |
        v
        查询TLD对应的WHOIS服务器
        |
        v
        建立TCP连接发送查询
        |
        +--- 成功 ---> 解析响应 ---> 标准化结果
        |
        +--- 失败 ---> 返回错误
  |
  v
写入缓存
  |
  v
返回查询结果
  |
  v
结束
```

### 8.2 批量查询流程

```
开始
  |
  v
接收批量查询请求
  |
  v
验证域名数量（<=50）
  |
  +--- 超限 ---> 返回错误：BATCH_SIZE_EXCEEDED
  |
  +--- 合格 --->
  |
  v
并发启动多个查询协程（每域名一个）
  |
  v
等待所有协程完成（或超时）
  |
  v
汇总结果
  |
  v
返回批量查询响应
  |
  v
结束
```

### 8.3 RDAP查询详细流程

```
开始
  |
  v
提取域名TLD
  |
  v
从Bootstrap数据查询RDAP端点
  |
  +--- 未找到 ---> 返回错误：RDAP_NOT_SUPPORTED
  |
  +--- 找到 --->
  |
  v
构建RDAP请求URL
  (https://{rdap_server}/domain/{domain})
  |
  v
发送HTTP GET请求
  |
  +--- 200 OK --->
  |     |
  |     v
  |     解析JSON响应
  |     |
  |     v
  |     提取标准化字段
  |     |
  |     v
  |     返回结果
  |
  +--- 404 Not Found --->
  |     |
  |     v
  |     返回：域名未注册
  |
  +--- 429 Too Many Requests --->
  |     |
  |     v
  |     等待 Retry-After 秒
  |     |
  |     v
  |     重试（最多3次）
  |
  +--- 5xx Server Error --->
  |     |
  |     v
  |     返回错误：触发协议切换
  |
  +--- 超时/网络错误 --->
        |
        v
        返回错误：触发协议切换
```

### 8.4 WHOIS查询详细流程

```
开始
  |
  v
提取域名TLD
  |
  v
查询TLD对应的WHOIS服务器列表
  |
  +--- 未配置 ---> 返回错误：TLD_NOT_SUPPORTED
  |
  +--- 找到 --->
  |
  v
选择第一个服务器
  |
  v
建立TCP连接（端口43）
  |
  +--- 连接失败 --->
  |     |
  |     v
  |     尝试下一个服务器
  |     |
  |     +--- 还有服务器 ---> 重试连接
  |     |
  |     +--- 所有服务器失败 ---> 返回错误
  |
  +--- 连接成功 --->
  |
  v
发送查询请求（域名 + \r\n）
  |
  v
读取响应（带超时）
  |
  +--- 超时 ---> 返回错误：QUERY_TIMEOUT
  |
  +--- 读取成功 --->
  |
  v
解析WHOIS文本
  |
  +--- 解析成功 ---> 返回标准化结果
  |
  +--- 解析失败 ---> 返回原始文本（调试模式）
```

---

## 9. 业务规则与计算逻辑

### 9.1 域名验证规则

| 规则编号 | 规则描述 | 验证方式 |
|----------|----------|----------|
| BR-01 | 域名长度：1-253字符（含末尾点号） | 字符长度检查 |
| BR-02 | 标签长度：每段1-63字符 | 按"."分割后检查 |
| BR-03 | 字符范围：仅允许字母、数字、连字符 | 正则匹配 |
| BR-04 | 连字符不能在首尾 | 首尾字符检查 |
| BR-05 | 至少包含一个点号（TLD） | 点号存在检查 |
| BR-06 | 域名统一转为小写 | 大小写转换 |

**域名正则表达式**：
```
^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$
```

### 9.2 协议选择规则

```
确定查询协议：
  IF 调用方显式指定协议 THEN
    使用调用方指定的协议
  ELSE IF TLD在tld_protocol_map中配置 THEN
    使用配置的协议
  ELSE
    使用默认策略（RDAP优先）
```

### 9.3 协议切换规则

| 规则编号 | 触发条件 | 切换行为 |
|----------|----------|----------|
| BR-07 | RDAP返回5xx错误 | 自动切换到WHOIS（仅auto模式） |
| BR-08 | RDAP请求超时 | 自动切换到WHOIS（仅auto模式） |
| BR-09 | RDAP返回404 | 标记域名未注册，不切换 |
| BR-10 | RDAP返回429 | 等待重试，不切换协议 |
| BR-11 | 调用方指定协议 | 不自动切换，直接返回错误 |

### 9.4 缓存规则

| 规则编号 | 规则描述 |
|----------|----------|
| BR-12 | 缓存键格式：`{protocol}:{normalized_domain}` |
| BR-13 | 默认缓存TTL：1小时（3600秒） |
| BR-14 | 域名未注册的结果也缓存（避免重复查询未注册域名） |
| BR-15 | 缓存满时使用LRU策略淘汰 |
| BR-16 | 查询错误不缓存（除域名未注册） |

### 9.5 重试规则

| 规则编号 | 规则描述 |
|----------|----------|
| BR-17 | 最大重试次数：2次（可配置） |
| BR-18 | 重试间隔：指数退避（1秒、2秒） |
| BR-19 | 仅对临时性错误重试（超时、5xx、网络错误） |
| BR-20 | 不对永久性错误重试（400、404） |

### 9.6 WHOIS服务器轮询规则

| 规则编号 | 规则描述 |
|----------|----------|
| BR-21 | 每个TLD配置主服务器和备用服务器列表 |
| BR-22 | 默认优先使用主服务器 |
| BR-23 | 主服务器失败时按顺序尝试备用服务器 |
| BR-24 | 所有服务器都失败后返回最终错误 |

### 9.7 EPP状态码映射

| EPP状态码 | 中文说明 |
|-----------|----------|
| `clientDeleteProhibited` | 客户端禁止删除 |
| `clientHold` | 客户端暂停解析 |
| `clientRenewProhibited` | 客户端禁止续费 |
| `clientTransferProhibited` | 客户端禁止转移 |
| `clientUpdateProhibited` | 客户端禁止更新 |
| `serverDeleteProhibited` | 服务器禁止删除 |
| `serverHold` | 服务器暂停解析 |
| `serverRenewProhibited` | 服务器禁止续费 |
| `serverTransferProhibited` | 服务器禁止转移 |
| `serverUpdateProhibited` | 服务器禁止更新 |
| `ok` | 正常状态 |

---

## 10. 部署方案

### 10.1 部署方式

| 方式 | 适用场景 | 说明 |
|------|----------|------|
| **单二进制** | 开发/测试 | 编译后直接运行，无外部依赖 |
| **Docker容器** | 生产环境 | 标准化部署，易于管理 |
| **systemd服务** | Linux服务器 | 服务化管理，自动重启 |

### 10.2 Docker部署

**Dockerfile**：
```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o go-whois .

# 运行阶段
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/go-whois .
COPY --from=builder /app/config ./config
COPY --from=builder /app/data ./data
EXPOSE 8080
ENTRYPOINT ["./go-whois"]
CMD ["serve"]
```

**docker-compose.yml**：
```yaml
version: '3.8'
services:
  go-whois:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config:/app/config:ro
    environment:
      - GO_WHOIS_SERVER_PORT=8080
      - GO_WHOIS_CACHE_ENABLED=true
      - GO_WHOIS_LOG_LEVEL=info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

### 10.3 环境变量

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `GO_WHOIS_SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `GO_WHOIS_SERVER_PORT` | 监听端口 | `8080` |
| `GO_WHOIS_CACHE_ENABLED` | 是否启用缓存 | `true` |
| `GO_WHOIS_CACHE_TTL` | 缓存过期时间 | `1h` |
| `GO_WHOIS_LOG_LEVEL` | 日志级别 | `info` |
| `GO_WHOIS_DEFAULT_PROTOCOL` | 默认查询协议 | `auto` |
| `GO_WHOIS_RDAP_TIMEOUT` | RDAP查询超时 | `10s` |
| `GO_WHOIS_WHOIS_TIMEOUT` | WHOIS查询超时 | `10s` |

### 10.4 资源配置建议

| 资源 | 最低配置 | 推荐配置 |
|------|----------|----------|
| CPU | 1核 | 2核 |
| 内存 | 256MB | 512MB |
| 磁盘 | 100MB | 500MB |
| 网络 | 10Mbps | 100Mbps |

### 10.5 监控指标

| 指标名称 | 类型 | 说明 |
|----------|------|------|
| `go_whois_queries_total` | Counter | 查询总数（按协议、结果分） |
| `go_whois_query_duration_seconds` | Histogram | 查询耗时分布 |
| `go_whois_cache_hits_total` | Counter | 缓存命中总数 |
| `go_whois_cache_misses_total` | Counter | 缓存未命中总数 |
| `go_whois_errors_total` | Counter | 错误总数（按错误类型分） |
| `go_whois_active_connections` | Gauge | 当前活跃连接数 |

---

## 11. 开发计划与里程碑

### 11.1 开发阶段划分

| 阶段 | 名称 | 周期 | 目标 |
|------|------|------|------|
| **M1** | 基础框架 | 第1-2周 | 项目骨架、配置管理、CLI框架 |
| **M2** | 查询引擎 | 第3-4周 | WHOIS/RDAP查询引擎、结果标准化 |
| **M3** | API服务 | 第5-6周 | HTTP API、缓存模块 |
| **M4** | 完善优化 | 第7-8周 | 批量查询、监控、文档、测试 |

### 11.2 详细任务分解

#### M1：基础框架（第1-2周）

| 任务编号 | 任务描述 | 优先级 | 预估工时 |
|----------|----------|--------|----------|
| T-01 | 初始化Go项目（go mod、目录结构） | P0 | 2h |
| T-02 | 实现配置管理模块（viper） | P0 | 4h |
| T-03 | 实现日志模块（zap） | P0 | 4h |
| T-04 | 实现CLI框架（cobra） | P0 | 4h |
| T-05 | 实现域名验证工具 | P0 | 4h |
| T-06 | 定义核心数据模型 | P0 | 4h |
| T-07 | TLD配置数据维护 | P1 | 8h |
| T-08 | 编写单元测试 | P1 | 8h |

**里程碑交付物**：
- 可运行的CLI程序骨架
- 配置文件加载功能
- 域名验证功能
- 基础单元测试

#### M2：查询引擎（第3-4周）

| 任务编号 | 任务描述 | 优先级 | 预估工时 |
|----------|----------|--------|----------|
| T-09 | 实现WHOIS查询引擎 | P0 | 12h |
| T-10 | 实现RDAP查询引擎 | P0 | 12h |
| T-11 | 实现结果标准化器 | P0 | 8h |
| T-12 | 实现协议调度逻辑 | P0 | 8h |
| T-13 | 实现协议自动切换 | P0 | 4h |
| T-14 | 实现重试机制 | P1 | 4h |
| T-15 | 编写集成测试 | P1 | 8h |

**里程碑交付物**：
- 可通过CLI查询域名的完整功能
- 支持RDAP和WHOIS双协议
- 支持协议自动切换
- 查询结果JSON输出

#### M3：API服务（第5-6周）

| 任务编号 | 任务描述 | 优先级 | 预估工时 |
|----------|----------|--------|----------|
| T-16 | 实现HTTP服务器框架 | P0 | 8h |
| T-17 | 实现单域名查询API | P0 | 8h |
| T-18 | 实现健康检查API | P0 | 4h |
| T-19 | 实现内存缓存模块 | P0 | 8h |
| T-20 | 实现中间件（日志、限流、CORS） | P1 | 8h |
| T-21 | 实现批量查询API（可选） | P2 | 8h |
| T-22 | 编写API测试 | P1 | 8h |

**里程碑交付物**：
- 完整的HTTP API服务
- 缓存功能
- API文档
- API测试用例

#### M4：完善优化（第7-8周）

| 任务编号 | 任务描述 | 优先级 | 预估工时 |
|----------|----------|--------|----------|
| T-23 | 实现Prometheus指标 | P1 | 4h |
| T-24 | 实现优雅关闭 | P1 | 4h |
| T-25 | Docker镜像构建 | P1 | 4h |
| T-26 | 性能测试与优化 | P1 | 8h |
| T-27 | 编写部署文档 | P2 | 4h |
| T-28 | 编写使用文档 | P2 | 4h |
| T-29 | 代码审查与重构 | P1 | 8h |
| T-30 | 端到端测试 | P1 | 8h |

**里程碑交付物**：
- 生产就绪的Docker镜像
- 完整的监控指标
- 部署和使用文档
- 性能测试报告

### 11.3 里程碑验收标准

| 里程碑 | 验收标准 |
|--------|----------|
| **M1** | CLI程序可启动、配置可加载、域名验证通过 |
| **M2** | 通过CLI可查询.com/.cn等主流域名、返回标准化JSON |
| **M3** | HTTP API可正常调用、缓存生效、错误处理正确 |
| **M4** | Docker镜像可部署、监控指标正常、性能达标 |

### 11.4 风险缓冲

每个里程碑预留20%的缓冲时间用于：
- 未预见的技术问题
- 需求变更
- 代码审查反馈
- 文档补充

---

## 12. 风险与假设

### 12.1 风险识别

| 风险编号 | 风险描述 | 影响程度 | 发生概率 | 缓解措施 |
|----------|----------|----------|----------|----------|
| R-01 | 部分TLD的WHOIS服务器不稳定 | 中 | 高 | 配置多个备用服务器、协议自动切换 |
| R-02 | RDAP Bootstrap数据更新不及时 | 中 | 中 | 定期更新、手动配置覆盖 |
| R-03 | 部分注册商限制查询频率 | 高 | 中 | 实现速率限制、缓存机制 |
| R-04 | WHOIS文本解析格式不统一 | 中 | 高 | 持续维护解析规则、容错处理 |
| R-05 | 隐私保护导致部分字段不可获取 | 低 | 高 | 记录原始响应、字段标记为不可用 |
| R-06 | 大批量查询导致外部服务限流 | 中 | 中 | 实现队列机制、请求间隔控制 |

### 12.2 假设条件

| 假设编号 | 假设描述 | 验证方式 |
|----------|----------|----------|
| A-01 | 服务器可正常访问外部WHOIS/RDAP服务 | 部署前验证网络连通性 |
| A-02 | 查询量主要集中在.com/.net/.org等主流TLD | 分析历史查询数据 |
| A-03 | 单次查询的平均响应时间可接受（<3秒） | 性能测试验证 |
| A-04 | 内存缓存能满足查询量需求 | 容量规划计算 |
| A-05 | IANA RDAP Bootstrap文件可正常获取 | 验证文件下载 |

---

## 13. 附录

### 13.1 术语表

| 术语 | 英文 | 说明 |
|------|------|------|
| WHOIS | WHOIS | 互联网域名注册信息查询协议，基于TCP端口43 |
| RDAP | Registration Data Access Protocol | 注册数据访问协议，WHOIS的现代化替代方案 |
| TLD | Top-Level Domain | 顶级域名，如.com、.net、.cn |
| IANA | Internet Assigned Numbers Authority | 互联网号码分配机构 |
| EPP | Extensible Provisioning Protocol | 可扩展供应协议，域名注册管理协议 |
| Bootstrap | Bootstrap | 引导数据，用于发现服务端点的初始数据 |
| LRU | Least Recently Used | 最近最少使用，缓存淘汰策略 |
| QPS | Queries Per Second | 每秒查询数 |
| P50/P95/P99 | Percentile | 百分位数，用于衡量响应时间分布 |

### 13.2 参考资料

| 资料 | 链接 | 说明 |
|------|------|------|
| RFC 3912 | https://datatracker.ietf.org/doc/html/rfc3912 | WHOIS协议规范 |
| RFC 7480 | https://datatracker.ietf.org/doc/html/rfc7480 | RDAP HTTP用法 |
| RFC 7481 | https://datatracker.ietf.org/doc/html/rfc7481 | RDAP安全服务 |
| RFC 7482 | https://datatracker.ietf.org/doc/html/rfc7482 | RDAP查询格式 |
| RFC 7483 | https://datatracker.ietf.org/doc/html/rfc7483 | RDAP JSON响应 |
| IANA RDAP Bootstrap | https://www.iana.org/assignments/rdap-dns/rdap-dns.xhtml | RDAP引导数据 |
| EPP Status Codes | https://www.iana.org/assignments/epp-status-codes/epp-status-codes.xhtml | EPP状态码定义 |

### 13.3 待确认事项

| 编号 | 事项 | 状态 | 备注 |
|------|------|------|------|
| TBD-01 | 是否需要支持国际化域名（IDN） | 待确认 | 如支持punycode转换 |
| TBD-02 | 是否需要支持WHOIS over TLS（端口43加密） | 待确认 | 部分注册商要求 |
| TBD-03 | 批量查询的最大并发数限制 | 待确认 | 建议默认50 |
| TBD-04 | 缓存是否需要支持持久化（Redis） | 待确认 | 第一版仅内存缓存 |
| TBD-05 | 是否需要支持自定义User-Agent | 待确认 | 部分WHOIS服务器要求 |
| TBD-06 | 是否需要支持代理服务器 | 待确认 | 特殊网络环境需求 |

---

> **文档结束**  
> 本文档作为go-whois项目的开发基线，所有功能需求、非功能需求、接口规范均以此文档为准。  
> 如有需求变更，请通过变更评审流程更新本文档。
