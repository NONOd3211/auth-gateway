# MiniMax Provider 集成设计

## 1. 概述

在 auth-gateway 中集成 MiniMax AI provider，替换对 CLIProxyAPI 上游的依赖。所有请求使用网关配置的 MiniMax API Key（支持多个组成共享池），Token 系统继续用于认证和限速。

## 2. 整体架构

```
用户请求 (sk-token) → TokenAuth 中间件 → ProxyHandler → ProviderManager → MiniMaxExecutor → MiniMax API
                          ↓                                      ↓
                      SQLite DB                          API Key 池
                   (Token 存储)                        (Key 状态管理)
```

### 请求流程
1. TokenAuth 验证 `sk-token` 并设置 `token_id`
2. ProxyHandler 调用 ProviderManager 获取可用 executor
3. ProviderManager 根据 token → key 映射选择 Key
4. MiniMaxExecutor 执行请求，失败时触发 failover

## 3. 目录结构

```
providers/
├── manager.go        # Provider 管理器（接口定义、Key 池）
├── minimax/
│   ├── executor.go   # MiniMax 请求执行器
│   ├── request.go    # 请求构建
│   └── response.go   # 响应解析
```

## 4. 数据库变更

### 新增表 `api_keys`
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 主键 (UUID) |
| key | string | MiniMax API Key（加密存储） |
| name | string | Key 名称/备注 |
| enabled | bool | 是否启用 |
| healthy | bool | 健康状态 |
| failed_at | time | 最近失败时间 |
| fail_count | int | 连续失败次数 |
| created_at | time | 创建时间 |

### 新增表 `token_key_mapping`
| 字段 | 类型 | 说明 |
|------|------|------|
| token_id | string | Token ID |
| api_key_id | string | 分配的 API Key ID |
| assigned_at | time | 分配时间 |

## 5. Provider 接口

```go
type Provider interface {
    Name() string
    Execute(ctx context.Context, req *http.Request, apiKey string) (*http.Response, error)
    IsQuotaError(resp *http.Response) bool
    GetQuotaInfo(resp *http.Response) (used, limit int64, err error)
}
```

## 6. ProviderManager

### 功能
- 管理 API Key 池（加载、选择、状态跟踪）
- Token 到 API Key 的固定映射（hash 分配）
- Failover 逻辑：检测到配额错误时自动切换 Key

### Key 选择策略
1. 根据 token_id hash 选择固定的 API Key
2. 若该 Key 不健康（`healthy=false`），尝试选择池中其他健康 Key
3. 若所有 Key 都不可用，返回错误

### Failover 策略
1. **检测触发**：收到 429 错误或响应中 quota 耗尽
2. **标记 Key**：设置 `healthy=false`，`failed_at=now`，`fail_count++`
3. **自动切换**：选择池中下一个健康的 Key
4. **冷却恢复**：5 分钟后重试该 Key，若成功则恢复

## 7. API Key 管理接口

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/admin/keys` | GET | 列出所有 API Key |
| `/api/admin/keys` | POST | 添加 API Key |
| `/api/admin/keys/:id` | PUT | 更新 API Key |
| `/api/admin/keys/:id` | DELETE | 删除 API Key |
| `/api/admin/keys/:id/enable` | POST | 启用 Key |
| `/api/admin/keys/:id/disable` | POST | 禁用 Key |

### POST /api/admin/keys 请求体
```json
{
  "key": "mmx-xxxxx",
  "name": "Key 1"
}
```

## 8. MiniMaxExecutor

### 请求转发
- 读取请求体，解析 model 等字段
- 使用分配的 API Key 构建 Authorization header
- 转发到 MiniMax API 端点

### 响应处理
- 透传响应给客户端
- 解析响应头/体中的 quota 信息
- 识别配额错误 (429)

## 9. Web UI 变更

新增 **API Key 管理页面** (`/keys`)：
- 列表视图：显示 Key 名称、状态、健康度、失败次数
- 添加表单：Key 值、名称
- 启用/禁用按钮
- Token 绑定查看

## 10. 配置

### 环境变量
| 变量 | 说明 |
|------|------|
| `MINIMAX_API_KEYS` | 逗号分隔的 API Keys（初始加载） |

## 11. 实现步骤

1. 新增数据库表和 model
2. 实现 Provider 接口和 Manager
3. 新增 API Key 管理 handler
4. 改造 proxy.go 使用 ProviderManager
5. 添加 Web UI 页面
6. 配置文件支持 MINIMAX_API_KEYS