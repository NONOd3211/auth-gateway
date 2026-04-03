# Auth Gateway

一个轻量级的 API Token 认证网关，用于管理 CLIProxyAPI 的访问权限。

## 功能特性

- ✅ Token 创建、编辑、删除
- ✅ Token 过期时间设置
- ✅ Token 请求次数限制
- ✅ 使用统计和报表
- ✅ WebUI 管理界面
- ✅ Docker 一键部署

## 架构

```
用户请求 → Auth Gateway (8080) → CLIProxyAPI (8317)
              ↓
           SQLite DB
         (Token/Usage)
```

## 快速开始

### 1. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件
```

### 2. Docker 部署

```bash
docker-compose up -d
```

### 3. 访问管理面板

```
http://localhost:8080
```

默认管理员密码: `admin123`

## API 端点

### 管理接口 (需要管理员认证)

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/admin/tokens` | GET | 列出所有 Token |
| `/api/admin/tokens` | POST | 创建 Token |
| `/api/admin/tokens/:id` | GET | 获取 Token 详情 |
| `/api/admin/tokens/:id` | PUT | 更新 Token |
| `/api/admin/tokens/:id` | DELETE | 删除 Token |
| `/api/admin/tokens/:id/reset` | POST | 重置使用次数 |
| `/api/admin/usage` | GET | 获取使用统计 |
| `/api/admin/usage/daily` | GET | 每日统计 |

### 代理接口 (需要 Token 认证)

| 端点 | 说明 |
|------|------|
| `/v1/*` | OpenAI 兼容 API |
| `/v1beta/*` | Gemini 兼容 API |

## 使用方式

### 创建 Token

```bash
curl -X POST http://localhost:8080/api/admin/tokens \
  -H "Authorization: Bearer your-admin-password" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试 Token",
    "max_requests": 1000,
    "expires_at": "2026-12-31T23:59:59Z"
  }'
```

### 使用 Token 调用 API

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `UPSTREAM_URL` | `http://192.168.1.237:8317` | CLIProxyAPI 地址 |
| `PORT` | `8080` | 网关端口 |
| `DATABASE_URL` | `/data/gateway.db` | 数据库路径 |
| `JWT_SECRET` | - | JWT 密钥 |
| `ADMIN_PASSWORD` | `admin123` | 管理员密码 |
| `ALLOWED_ORIGINS` | `*` | CORS 允许的源 |

## WebUI 开发

```bash
cd webui
npm install
npm run dev
```

## 构建

```bash
# 构建后端
go build -o gateway .

# 构建 Docker 镜像
docker build -t auth-gateway .
```

## License

MIT
