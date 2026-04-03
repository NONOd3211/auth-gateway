# MiniMax Provider 集成实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 auth-gateway 中集成 MiniMax provider，替换 CLIProxyAPI 上游，支持多 API Key 共享池和自动 failover。

**Architecture:** 实现 Provider 接口抽象，创建 MiniMaxExecutor 和 ProviderManager 管理 API Key 池。请求通过 token_id hash 选择固定 Key，失败时自动切换到其他健康 Key。

**Tech Stack:** Go (gin, gorm, sqlite), React (TypeScript)

---

## 文件结构

```
models/
  apikey.go          # 新增: APIKey model

providers/
  manager.go         # 新增: ProviderManager 接口和实现
  minimax/
    executor.go      # 新增: MiniMaxExecutor 实现

handler/
  apikey.go          # 新增: APIKey CRUD handler

config/
  config.go          # 修改: 添加 MINIMAX_API_KEYS

database/
  database.go        # 修改: 注册新 model

proxy/
  client.go          # 不修改 (保留备用)

main.go              # 修改: 初始化 ProviderManager

webui/src/
  pages/
    ApiKeyList.tsx    # 新增: API Key 管理页面
    ApiKeyCreate.tsx  # 新增: 添加 API Key 表单
  components/
    Navbar.tsx        # 修改: 添加 API Key 菜单
  api/
    client.ts         # 修改: 添加 apiKey API
```

---

## Task 1: 添加数据库 Model

**Files:**
- Create: `models/apikey.go`
- Modify: `database/database.go:32`

- [ ] **Step 1: 创建 models/apikey.go**

```go
package models

import (
	"time"
)

type APIKey struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"size:200"`        // MiniMax API Key
	Name      string    `json:"name" gorm:"size:100"`       // 名称备注
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	Healthy   bool      `json:"healthy" gorm:"default:true"`
	FailedAt  *time.Time `json:"failed_at"`
	FailCount int       `json:"fail_count" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenKeyMapping struct {
	TokenID    string    `json:"token_id" gorm:"primaryKey"`
	APIKeyID   string    `json:"api_key_id" gorm:"primaryKey"`
	AssignedAt time.Time `json:"assigned_at"`
}
```

- [ ] **Step 2: 修改 database/database.go 注册新 model**

文件: `database/database.go:32`

将:
```go
if err = DB.AutoMigrate(&models.Token{}, &models.UsageRecord{}); err != nil {
```

改为:
```go
if err = DB.AutoMigrate(&models.Token{}, &models.UsageRecord{}, &models.APIKey{}, &models.TokenKeyMapping{}); err != nil {
```

- [ ] **Step 3: 提交**

```bash
git add models/apikey.go database/database.go
git commit -m "feat: add APIKey and TokenKeyMapping models"
```

---

## Task 2: 实现 Provider 接口和 ProviderManager

**Files:**
- Create: `providers/manager.go`
- Create: `providers/minimax/executor.go`
- Create: `providers/minimax/request.go`
- Create: `providers/minimax/response.go`

- [ ] **Step 1: 创建 providers/manager.go**

```go
package providers

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"fmt"
	"hash/fnv"
	"net/http"
	"sync"
	"time"
)

type Provider interface {
	Name() string
	Execute(req *http.Request, apiKey string) (*http.Response, error)
	IsQuotaError(resp *http.Response) bool
	GetQuotaInfo(resp *http.Response) (used, limit int64, err error)
}

type ProviderManager struct {
	providers map[string]Provider
	keys      []*models.APIKey
	mu        sync.RWMutex
}

func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]Provider),
		keys:       make([]*models.APIKey, 0),
	}
}

func (m *ProviderManager) RegisterProvider(p Provider) {
	m.providers[p.Name()] = p
}

func (m *ProviderManager) LoadAPIKeys() error {
	var keys []*models.APIKey
	if err := database.DB.Where("enabled = ?", true).Find(&keys).Error; err != nil {
		return err
	}
	m.mu.Lock()
	m.keys = keys
	m.mu.Unlock()
	return nil
}

func (m *ProviderManager) GetAPIKeyForToken(tokenID string) (*models.APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find existing mapping
	var mapping models.TokenKeyMapping
	if err := database.DB.Where("token_id = ?", tokenID).First(&mapping).Error; err == nil {
		// Find the key
		for _, k := range m.keys {
			if k.ID == mapping.APIKeyID && k.Enabled && k.Healthy {
				return k, nil
			}
		}
	}

	// No mapping or key unavailable - assign new one based on hash
	if len(m.keys) == 0 {
		return nil, ErrNoAPIKeysAvailable
	}

	// Hash tokenID to select key index
	h := fnv.New32a()
	h.Write([]byte(tokenID))
	index := int(h.Sum32()) % len(m.keys)
	selectedKey := m.keys[index]

	// Check if selected key is healthy
	if !selectedKey.Healthy {
		// Try other keys in sequence
		for i := 1; i < len(m.keys); i++ {
			idx := (index + i) % len(m.keys)
			if m.keys[idx].Healthy && m.keys[idx].Enabled {
				selectedKey = m.keys[idx]
				break
			}
		}
		if !selectedKey.Healthy {
			return nil, ErrNoAPIKeysAvailable
		}
	}

	// Create mapping
	mapping = models.TokenKeyMapping{
		TokenID:    tokenID,
		APIKeyID:   selectedKey.ID,
		AssignedAt:  time.Now(),
	}
	database.DB.Create(&mapping)

	return selectedKey, nil
}

func (m *ProviderManager) MarkKeyFailed(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", keyID).Error; err != nil {
		return err
	}

	now := time.Now()
	key.Healthy = false
	key.FailedAt = &now
	key.FailCount++

	// Update in-memory keys
	for _, k := range m.keys {
		if k.ID == keyID {
			k.Healthy = false
			k.FailedAt = &now
			k.FailCount++
			break
		}
	}

	return database.DB.Save(&key).Error
}

func (m *ProviderManager) MarkKeyHealthy(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", keyID).Error; err != nil {
		return err
	}

	key.Healthy = true
	key.FailedAt = nil
	key.FailCount = 0

	// Update in-memory keys
	for _, k := range m.keys {
		if k.ID == keyID {
			k.Healthy = true
			k.FailedAt = nil
			k.FailCount = 0
			break
		}
	}

	return database.DB.Save(&key).Error
}

func (m *ProviderManager) GetProvider(name string) Provider {
	return m.providers[name]
}

var ErrNoAPIKeysAvailable = fmt.Errorf("no available API keys")

import (
	"fmt"
)
```

- [ ] **Step 2: 创建 providers/minimax/request.go**

```go
package minimax

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type ChatRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream,omitempty"`
}

func BuildRequest(req *http.Request, apiKey string, upstreamURL string) (*http.Request, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	targetURL := upstreamURL + req.URL.Path
	proxyReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+apiKey)

	// Copy other headers except Host
	for key, values := range req.Header {
		if key == "Host" || key == "Content-Length" {
			continue
		}
		proxyReq.Header[key] = values
	}

	return proxyReq, nil
}
```

- [ ] **Step 3: 创建 providers/minimax/response.go**

```go
package minimax

import (
	"net/http"
	"strconv"
	"strings"
)

type QuotaInfo struct {
	Used  int64
	Limit int64
}

func IsQuotaError(resp *http.Response) bool {
	// 429 Too Many Requests
	if resp.StatusCode == 429 {
		return true
	}

	// Check response body for quota error message
	// (implement based on actual MiniMax API error format)
	return false
}

func GetQuotaInfo(resp *http.Response) (used, limit int64, err error) {
	// MiniMax API quota info in header or body
	// Return 0,0 if not available
	return 0, 0, nil
}
```

- [ ] **Step 4: 创建 providers/minimax/executor.go**

```go
package minimax

import (
	"auth-gateway/config"
	"io"
	"net/http"
	"time"
)

type Executor struct {
	baseURL   string
	timeout   time.Duration
}

func NewExecutor(cfg *config.Config) *Executor {
	return &Executor{
		baseURL: getMiniMaxBaseURL(cfg),
		timeout: 10 * time.Minute,
	}
}

func getMiniMaxBaseURL(cfg *config.Config) string {
	// MiniMax API base URL
	return "https://api.minimax.chat/v1"
}

func (e *Executor) Name() string {
	return "minimax"
}

func (e *Executor) Execute(req *http.Request, apiKey string) (*http.Response, error) {
	proxyReq, err := BuildRequest(req, apiKey, e.baseURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: e.timeout}
	return client.Do(proxyReq)
}

func (e *Executor) IsQuotaError(resp *http.Response) bool {
	return IsQuotaError(resp)
}

func (e *Executor) GetQuotaInfo(resp *http.Response) (used, limit int64, err error) {
	return GetQuotaInfo(resp)
}

// PassThroughResponse reads response body and returns it
func PassThroughResponse(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 5: 提交**

```bash
git add providers/manager.go providers/minimax/
git commit -m "feat: add Provider interface and MiniMax executor"
```

---

## Task 3: 添加 API Key 管理 Handler

**Files:**
- Create: `handler/apikey.go`

- [ ] **Step 1: 创建 handler/apikey.go**

```go
package handler

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ListAPIKeys(c *gin.Context) {
	var keys []models.APIKey
	if err := database.DB.Order("created_at DESC").Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func CreateAPIKey(c *gin.Context) {
	var req struct {
		Key  string `json:"key" binding:"required"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := models.APIKey{
		ID:        uuid.New().String(),
		Key:       req.Key,
		Name:      req.Name,
		Enabled:   true,
		Healthy:   true,
		FailCount: 0,
	}

	if err := database.DB.Create(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"key": key})
}

func UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")
	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	var req struct {
		Name    string `json:"name"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		key.Name = req.Name
	}
	if req.Enabled != nil {
		key.Enabled = *req.Enabled
	}

	if err := database.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}

func DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.APIKey{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func EnableAPIKey(c *gin.Context) {
	id := c.Param("id")
	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	key.Enabled = true
	key.Healthy = true
	if err := database.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}

func DisableAPIKey(c *gin.Context) {
	id := c.Param("id")
	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	key.Enabled = false
	if err := database.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}
```

- [ ] **Step 2: 提交**

```bash
git add handler/apikey.go
git commit -m "feat: add APIKey CRUD handlers"
```

---

## Task 4: 修改 Config 支持 MINIMAX_API_KEYS

**Files:**
- Modify: `config/config.go`

- [ ] **Step 1: 修改 config/config.go**

在 Config struct 添加:
```go
MiniMaxAPIKeys string
```

在 Load() 函数添加:
```go
MiniMaxAPIKeys: getEnv("MINIMAX_API_KEYS", ""),
```

- [ ] **Step 2: 提交**

```bash
git add config/config.go
git commit -m "feat: add MINIMAX_API_KEYS config"
```

---

## Task 5: 改造 Proxy 使用 ProviderManager

**Files:**
- Modify: `handler/proxy.go`
- Modify: `main.go`

- [ ] **Step 1: 创建全局 ProviderManager 实例**

在 `main.go` 添加:
```go
var providerManager *providers.ProviderManager

func init() {
    providerManager = providers.NewProviderManager()
    providerManager.RegisterProvider(minimax.NewExecutor(cfg))
}
```

- [ ] **Step 2: 修改 handler/proxy.go**

```go
// 替换原有的 proxy.Client 调用
// 1. 从 ProviderManager 获取 token 对应的 APIKey
// 2. 调用 MiniMaxExecutor 执行请求
// 3. 失败时调用 MarkKeyFailed 标记 Key 不健康
// 4. 成功时调用 MarkKeyHealthy 恢复 Key 健康状态

tokenID, _ := c.Get("token_id")
apiKey, err := providerManager.GetAPIKeyForToken(tokenID.(string))
if err != nil {
    c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available API keys"})
    return
}

provider := providerManager.GetProvider("minimax")
resp, err := provider.Execute(c.Request, apiKey.Key)
if err != nil {
    providerManager.MarkKeyFailed(apiKey.ID)
    recordUsage(tokenID.(string), c.Request.URL.Path, "", 0, 0, false, err.Error())
    c.JSON(http.StatusBadGateway, gin.H{"error": "upstream error: " + err.Error()})
    return
}

// Check quota error
if provider.IsQuotaError(resp) {
    providerManager.MarkKeyFailed(apiKey.ID)
}

// Pass through response
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
for key, values := range resp.Header {
    for _, value := range values {
        c.Header(key, value)
    }
}
c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)

if resp.StatusCode == 200 {
    providerManager.MarkKeyHealthy(apiKey.ID)
}
```

- [ ] **Step 3: 提交**

```bash
git add handler/proxy.go main.go
git commit -m "feat: integrate ProviderManager into proxy handler"
```

---

## Task 6: 初始化 API Keys 加载

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 在 main.go 启动时加载 API Keys**

在 `runProxy` 函数或初始化时添加:
```go
// Load API keys from config MINIMAX_API_KEYS
if cfg.MiniMaxAPIKeys != "" {
    keys := strings.Split(cfg.MiniMaxAPIKeys, ",")
    for _, key := range keys {
        key = strings.TrimSpace(key)
        if key == "" {
            continue
        }
        // Check if key already exists
        var existing models.APIKey
        if database.DB.Where("key = ?", key).First(&existing).Error != nil {
            // Create new
            apiKey := models.APIKey{
                ID:        uuid.New().String(),
                Key:       key,
                Name:      "Imported Key",
                Enabled:   true,
                Healthy:   true,
                FailCount: 0,
            }
            database.DB.Create(&apiKey)
        }
    }
}

// Load keys into manager
providerManager.LoadAPIKeys()
```

- [ ] **Step 2: 添加路由**

在 `main.go` admin panel 路由添加:
```go
admin.GET("/keys", handler.ListAPIKeys)
admin.POST("/keys", handler.CreateAPIKey)
admin.PUT("/keys/:id", handler.UpdateAPIKey)
admin.DELETE("/keys/:id", handler.DeleteAPIKey)
admin.POST("/keys/:id/enable", handler.EnableAPIKey)
admin.POST("/keys/:id/disable", handler.DisableAPIKey)
```

- [ ] **Step 3: 提交**

```bash
git add main.go
git commit -m "feat: initialize API keys from config on startup"
```

---

## Task 7: Web UI API Key 管理页面

**Files:**
- Create: `webui/src/pages/ApiKeyList.tsx`
- Create: `webui/src/pages/ApiKeyCreate.tsx`
- Create: `webui/src/components/ApiKeyForm.tsx`
- Modify: `webui/src/App.tsx`
- Modify: `webui/src/api/client.ts`
- Modify: `webui/src/components/Navbar.tsx`

- [ ] **Step 1: 添加 API 到 client.ts**

```typescript
export interface APIKey {
  id: string
  key: string
  name: string
  enabled: boolean
  healthy: boolean
  fail_count: number
  created_at: string
}

export const apiKeyApi = {
  list: () => api.get<{ keys: APIKey[] }>('/keys'),
  create: (data: { key: string; name: string }) => api.post('/keys', data),
  update: (id: string, data: Partial<APIKey>) => api.put(`/keys/${id}`, data),
  delete: (id: string) => api.delete(`/keys/${id}`),
  enable: (id: string) => api.post(`/keys/${id}/enable`),
  disable: (id: string) => api.post(`/keys/${id}/disable`),
}
```

- [ ] **Step 2: 创建 ApiKeyList.tsx**

类似 TokenList.tsx 的表格布局，显示:
- Key 名称 (name)
- Key 掩码 (key 的前6位 + *** + 后4位)
- 状态 (enabled)
- 健康状态 (healthy)
- 失败次数 (fail_count)
- 创建时间

操作按钮: 编辑、启用/禁用、删除

- [ ] **Step 3: 创建 ApiKeyCreate.tsx**

简单表单:
- Key 输入框 (required)
- Name 输入框 (optional)
- 提交/取消按钮

- [ ] **Step 4: 修改 App.tsx 添加路由**

```tsx
import { ApiKeyList } from './pages/ApiKeyList'
import { ApiKeyCreate } from './pages/ApiKeyCreate'

// Add route
<Route path="/keys" element={<ApiKeyList />} />
<Route path="/keys/create" element={<ApiKeyCreate />} />
```

- [ ] **Step 5: 修改 Navbar.tsx 添加菜单**

```tsx
<Link to="/keys">API Keys</Link>
```

- [ ] **Step 6: 提交**

```bash
git add webui/src/pages/ApiKeyList.tsx webui/src/pages/ApiKeyCreate.tsx
git add webui/src/api/client.ts webui/src/App.tsx webui/src/components/Navbar.tsx
git commit -m "feat: add API key management UI"
```

---

## Task 8: 整体测试

- [ ] **Step 1: 编译后端**

```bash
go build ./...
```

- [ ] **Step 2: 编译前端**

```bash
cd webui && npm run build
```

- [ ] **Step 3: 手动测试流程**

1. 启动服务
2. Web UI 添加 API Key
3. 创建 Token
4. 用 Token 请求 /v1/chat/completions
5. 检查响应和 usage 记录

---

## 依赖关系

```
Task 1 (数据库 Model)
    ↓
Task 2 (Provider 接口)
    ↓
Task 3 (API Key Handler)
    ↓
Task 4 (Config)
    ↓
Task 5 (Proxy 改造) ← Task 2, Task 3 完成后
Task 6 (初始化) ← Task 1, Task 2, Task 4 完成后
Task 7 (Web UI) ← Task 3 完成后
Task 8 (测试)
```

---

## 备注

- MiniMax API 的具体 endpoint 和认证方式需要根据 MiniMax 官方文档确认
- Quota 错误检测逻辑需要根据实际 API 响应格式调整
- 加密存储 API Key: 当前明文存储，生产环境应加密